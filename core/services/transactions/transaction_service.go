package services

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/ortizdavid/go-bank-core-api/common/apperrors"
	"github.com/ortizdavid/go-bank-core-api/common/helpers"
	accountEntities "github.com/ortizdavid/go-bank-core-api/core/entities/accounts"
	entities "github.com/ortizdavid/go-bank-core-api/core/entities/transactions"
	accountRepo "github.com/ortizdavid/go-bank-core-api/core/repositories/accounts"
	transactionRepo "github.com/ortizdavid/go-bank-core-api/core/repositories/transactions"
	"github.com/ortizdavid/go-nopain/encryption"
	"github.com/ortizdavid/go-nopain/httputils"
	"github.com/ortizdavid/go-nopain/serialization"
	"gorm.io/gorm"
)

type TransactionService struct {
	repository        *transactionRepo.TransactionRepository
	accountRepository *accountRepo.AccountRepository
	db                *gorm.DB
}

// maxIdempotencyKeyLength matches the idempotency_keys.idempotency_key column.
const maxIdempotencyKeyLength = 255

var transactionLabels = map[entities.TransactionType]struct{ prefix, description string }{
	entities.TransactionTypeDeposit:    {"DEP", "Account deposit"},
	entities.TransactionTypeWithdrawal: {"WDR", "Account withdrawal"},
	entities.TransactionTypeTransfer:   {"TRF", "Transfer between accounts"},
}

func NewTransactionService(db *gorm.DB) *TransactionService {
	return &TransactionService{
		repository:        transactionRepo.NewTransactionRepository(db),
		accountRepository: accountRepo.NewAccountRepository(db),
		db:                db,
	}
}

func (s *TransactionService) Transfer(r *http.Request, ctx context.Context, request entities.TransferNumberRequest) (entities.Transaction, error) {
	if err := serialization.DecodeJson(r.Body, &request); err != nil {
		return entities.Transaction{}, apperrors.NewBadRequestError("invalid request body")
	}
	if err := request.Validate(); err != nil {
		return entities.Transaction{}, apperrors.NewBadRequestError(err.Error())
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(idempotencyKey) > maxIdempotencyKeyLength {
		return entities.Transaction{}, apperrors.NewBadRequestError("Idempotency-Key must be at most 255 characters")
	}

	var transaction entities.Transaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		accounts := accountRepo.NewAccountRepository(tx)
		transactions := transactionRepo.NewTransactionRepository(tx)

		// Requests sharing a key wait here until the first one commits, so the
		// lookup below sees its result instead of racing with it.
		if idempotencyKey != "" {
			if err := transactions.LockIdempotencyKey(ctx, idempotencyKey); err != nil {
				return err
			}
		}
		source, destination, err := lockTransferAccounts(ctx, accounts, request)
		if err != nil {
			return err
		}
		if idempotencyKey != "" {
			existing, err := transactions.GetByIdempotencyKey(ctx, idempotencyKey)
			if err == nil {
				if !isSameTransfer(existing, source, destination, request) {
					return apperrors.NewUnprocessableEntityError("Idempotency-Key was already used with a different request")
				}
				transaction = existing
				return nil
			}
			if !transactionRepo.IsNotFound(err) {
				return err
			}
		}

		if source.AccountStatus != accountEntities.AccountStatusActive {
			return apperrors.NewUnprocessableEntityError("source account is not active")
		}
		if destination.AccountStatus != accountEntities.AccountStatusActive {
			return apperrors.NewUnprocessableEntityError("destination account is not active")
		}
		if !strings.EqualFold(request.Currency, source.Currency) || source.Currency != destination.Currency {
			return apperrors.NewUnprocessableEntityError("currency must match the currency of both accounts")
		}
		if source.Balance < request.Amount {
			return apperrors.NewUnprocessableEntityError("insufficient balance")
		}

		now := time.Now().UTC()
		sourceBalance := roundCents(source.Balance - request.Amount)
		if err := accounts.UpdateBalance(ctx, source.AccountId, sourceBalance, now); err != nil {
			return err
		}
		if err := accounts.UpdateBalance(ctx, destination.AccountId, roundCents(destination.Balance+request.Amount), now); err != nil {
			return err
		}
		transaction = newTransaction(entities.TransactionTypeTransfer, source, destination, request.Amount, sourceBalance, now)
		if err := transactions.Create(ctx, &transaction); err != nil {
			return err
		}
		if idempotencyKey == "" {
			return nil
		}
		return transactions.CreateIdempotencyKey(ctx, &entities.IdempotencyKey{
			IdempotencyKey: idempotencyKey,
			TransactionId:  transaction.TransactionId,
			CreatedAt:      now,
		})
	})
	if err != nil {
		return entities.Transaction{}, err
	}
	return transaction, nil
}

func (s *TransactionService) Deposit(r *http.Request, ctx context.Context, request entities.DepositRequest) (entities.Transaction, error) {
	if err := serialization.DecodeJson(r.Body, &request); err != nil {
		return entities.Transaction{}, apperrors.NewBadRequestError("invalid request body")
	}
	if err := request.Validate(); err != nil {
		return entities.Transaction{}, apperrors.NewBadRequestError(err.Error())
	}
	return s.changeBalance(ctx, entities.TransactionTypeDeposit, request.AccountNumber, request.Amount, request.Currency)
}

func (s *TransactionService) Withdraw(r *http.Request, ctx context.Context, request entities.WithdrawRequest) (entities.Transaction, error) {
	if err := serialization.DecodeJson(r.Body, &request); err != nil {
		return entities.Transaction{}, apperrors.NewBadRequestError("invalid request body")
	}
	if err := request.Validate(); err != nil {
		return entities.Transaction{}, apperrors.NewBadRequestError(err.Error())
	}
	return s.changeBalance(ctx, entities.TransactionTypeWithdrawal, request.AccountNumber, request.Amount, request.Currency)
}

// changeBalance applies a deposit or withdrawal to one account and records it
// in a single database transaction, holding the account row lock throughout.
func (s *TransactionService) changeBalance(ctx context.Context, txType entities.TransactionType, accountNumber string, amount float64, currency string) (entities.Transaction, error) {
	var transaction entities.Transaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		accounts := accountRepo.NewAccountRepository(tx)
		locked, err := accounts.GetByNumbersForUpdate(ctx, accountNumber)
		if err != nil {
			return err
		}
		if len(locked) == 0 {
			return apperrors.NewNotFoundError("account not found")
		}
		account := locked[0]
		if account.AccountStatus != accountEntities.AccountStatusActive {
			return apperrors.NewUnprocessableEntityError("account is not active")
		}
		if !strings.EqualFold(currency, account.Currency) {
			return apperrors.NewUnprocessableEntityError("currency must match the account currency")
		}
		balance := roundCents(account.Balance + amount)
		if txType == entities.TransactionTypeWithdrawal {
			if account.Balance < amount {
				return apperrors.NewUnprocessableEntityError("insufficient balance")
			}
			balance = roundCents(account.Balance - amount)
		}

		now := time.Now().UTC()
		if err := accounts.UpdateBalance(ctx, account.AccountId, balance, now); err != nil {
			return err
		}
		transaction = newTransaction(txType, account, account, amount, balance, now)
		return transactionRepo.NewTransactionRepository(tx).Create(ctx, &transaction)
	})
	if err != nil {
		return entities.Transaction{}, err
	}
	return transaction, nil
}

func (s *TransactionService) GetAllTransactions(r *http.Request, ctx context.Context, params helpers.PaginationParams) (*httputils.Pagination[entities.TransactionData], error) {
	page := params.CurrentPage
	if page < 1 {
		page = 1
	}
	if params.Limit < 1 {
		return nil, apperrors.NewBadRequestError("limit must be greater than 0")
	}
	offset := (page - 1) * params.Limit
	count, err := s.repository.Count(ctx)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to count transactions")
	}
	transactions, err := s.repository.GetAll(ctx, params.Limit, offset)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list transactions")
	}
	pagination, err := httputils.NewPagination(r, transactions, count, page, params.Limit)
	if err != nil {
		return nil, err
	}
	return pagination, nil
}

func (s *TransactionService) GetAccountTransactions(r *http.Request, ctx context.Context, accountNumber string, params helpers.PaginationParams) (*httputils.Pagination[entities.TransactionData], error) {
	account, err := s.accountRepository.GetByNumber(ctx, accountNumber)
	if accountRepo.IsNotFound(err) {
		return nil, apperrors.NewNotFoundError("account not found")
	}
	if err != nil {
		return nil, err
	}
	page := params.CurrentPage
	if page < 1 {
		page = 1
	}
	if params.Limit < 1 {
		return nil, apperrors.NewBadRequestError("limit must be greater than 0")
	}
	offset := (page - 1) * params.Limit
	count, err := s.repository.CountByAccountId(ctx, account.AccountId)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to count transactions")
	}
	transactions, err := s.repository.GetByAccountId(ctx, account.AccountId, params.Limit, offset)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list transactions")
	}
	pagination, err := httputils.NewPagination(r, transactions, count, page, params.Limit)
	if err != nil {
		return nil, err
	}
	return pagination, nil
}

// lockTransferAccounts loads both accounts and locks their rows for the rest of
// the database transaction.
func lockTransferAccounts(ctx context.Context, accounts *accountRepo.AccountRepository, request entities.TransferNumberRequest) (source, destination accountEntities.Account, err error) {
	locked, err := accounts.GetByNumbersForUpdate(ctx, request.SourceNumber, request.DestinationNumber)
	if err != nil {
		return source, destination, err
	}
	byNumber := make(map[string]accountEntities.Account, len(locked))
	for _, account := range locked {
		byNumber[account.AccountNumber] = account
	}
	source, found := byNumber[request.SourceNumber]
	if !found {
		return source, destination, apperrors.NewNotFoundError("source account not found")
	}
	destination, found = byNumber[request.DestinationNumber]
	if !found {
		return source, destination, apperrors.NewNotFoundError("destination account not found")
	}
	return source, destination, nil
}

// isSameTransfer reports whether a stored transfer matches the incoming request,
// so a reused Idempotency-Key cannot return the result of a different transfer.
func isSameTransfer(existing entities.Transaction, source, destination accountEntities.Account, request entities.TransferNumberRequest) bool {
	return existing.TransactionType == entities.TransactionTypeTransfer &&
		existing.SourceId == source.AccountId &&
		existing.DestinationId == destination.AccountId &&
		existing.Amount == request.Amount &&
		strings.EqualFold(existing.Currency, request.Currency)
}

// newTransaction builds a completed transaction. balanceAfter is the source
// account balance once the amount has been applied.
func newTransaction(txType entities.TransactionType, source, destination accountEntities.Account, amount, balanceAfter float64, now time.Time) entities.Transaction {
	label := transactionLabels[txType]
	return entities.Transaction{
		SourceId:          source.AccountId,
		DestinationId:     destination.AccountId,
		TransactionType:   txType,
		TransactionStatus: entities.TransactionStatusCompleted,
		Code:              generateTransactionCode(label.prefix),
		Amount:            amount,
		Currency:          source.Currency,
		BalanceBefore:     source.Balance,
		BalanceAfter:      balanceAfter,
		Description:       label.description,
		TransactionDate:   now,
		UniqueId:          encryption.GenerateUUID(),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// roundCents keeps balances on whole cents, matching the DECIMAL(18, 2) columns.
func roundCents(value float64) float64 {
	return math.Round(value*100) / 100
}

func generateTransactionCode(prefix string) string {
	id := strings.ReplaceAll(encryption.GenerateUUID(), "-", "")
	return fmt.Sprintf("%s%s", prefix, strings.ToUpper(id[:16]))
}
