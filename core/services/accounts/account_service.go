package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/ortizdavid/go-bank-core-api/common/apperrors"
	"github.com/ortizdavid/go-bank-core-api/common/helpers"
	entities "github.com/ortizdavid/go-bank-core-api/core/entities/accounts"
	accountRepo "github.com/ortizdavid/go-bank-core-api/core/repositories/accounts"
	customerRepo "github.com/ortizdavid/go-bank-core-api/core/repositories/customers"
	"github.com/ortizdavid/go-nopain/encryption"
	"github.com/ortizdavid/go-nopain/httputils"
	"github.com/ortizdavid/go-nopain/serialization"
	"gorm.io/gorm"
)

type AccountService struct {
	repository         *accountRepo.AccountRepository
	customerRepository *customerRepo.CustomerRepository
}

func NewAccountService(db *gorm.DB) *AccountService {
	return &AccountService{
		repository:         accountRepo.NewAccountRepository(db),
		customerRepository: customerRepo.NewCustomerRepository(db),
	}
}

func (s *AccountService) CreateAccount(r *http.Request, ctx context.Context, request entities.CreateAccountRequest) (entities.Account, error) {
	if err := serialization.DecodeJson(r.Body, &request); err != nil {
		return entities.Account{}, apperrors.NewBadRequestError("invalid request body")
	}
	if err := request.Validate(); err != nil {
		return entities.Account{}, apperrors.NewBadRequestError("invalid request data")
	}
	customer, err := s.customerRepository.GetById(ctx, request.CustomerId)
	if err != nil {
		return entities.Account{}, apperrors.NewNotFoundError("customer not found")
	}
	account := entities.Account{
		CustomerId:    request.CustomerId,
		AccountType:   request.AccountType,
		AccountStatus: entities.AccountStatusActive,
		AccountNumber: generateAccountNumber(),
		Iban:          generateIBAN(),
		HolderName:    customer.CustomerName,
		Balance:       0,
		Currency:      request.Currency,
		UniqueId:      encryption.GenerateUUID(),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := s.repository.Create(ctx, &account); err != nil {
		return entities.Account{}, apperrors.NewInternalError("failed to create account")
	}
	return account, nil
}

func (s *AccountService) ChangeAccountStatus(r *http.Request, ctx context.Context, request entities.ChangeAccountStatusRequest) error {
	if err := serialization.DecodeJson(r.Body, &request); err != nil {
		return apperrors.NewBadRequestError("invalid request body")
	}
	if err := request.Validate(); err != nil {
		return apperrors.NewBadRequestError("invalid request data")
	}
	account, err := s.repository.GetByNumber(ctx, request.AccountNumber)
	if accountRepo.IsNotFound(err) {
		return apperrors.NewNotFoundError("account not found")
	}
	if err != nil {
		return err
	}
	return s.repository.UpdateStatus(ctx, account.AccountId, request.AccountStatus, time.Now().UTC())
}

func (s *AccountService) GetAllAccounts(r *http.Request, ctx context.Context, params helpers.PaginationParams) (*httputils.Pagination[entities.AccountData], error) {
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
		return nil, apperrors.NewInternalError("failed to count accounts")
	}
	accounts, err := s.repository.GetAll(ctx, params.Limit, offset)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list accounts")
	}
	pagination, err := httputils.NewPagination(r, accounts, count, page, params.Limit)
	if err != nil {
		return nil, err
	}
	return pagination, nil
}

func (s *AccountService) GetAccountById(ctx context.Context, accountId int64) (entities.AccountData, error) {
	if accountId < 1 {
		return entities.AccountData{}, apperrors.NewBadRequestError("account id must be greater than 0")
	}
	account, err := s.repository.GetDataById(ctx, accountId)
	if err != nil {
		return entities.AccountData{}, apperrors.NewNotFoundError("account not found")
	}
	return account, nil
}

func (s *AccountService) GetAccountByNumber(ctx context.Context, accountNumber string) (entities.AccountData, error) {
	if accountNumber == "" {
		return entities.AccountData{}, apperrors.NewBadRequestError("account number is required")
	}
	account, err := s.repository.GetDataByNumber(ctx, accountNumber)
	if err != nil {
		return entities.AccountData{}, apperrors.NewNotFoundError("account not found")
	}
	return account, nil
}

func generateAccountNumber() string {
	n, err := rand.Int(rand.Reader, big.NewInt(9000000000))
	if err != nil {
		return fmt.Sprintf("%010d", time.Now().UnixNano()%10000000000)
	}
	return fmt.Sprintf("%010d", n.Int64()+1000000000)
}

func generateIBAN() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1e16))
	if err != nil {
		return fmt.Sprintf("US00%016d", time.Now().UnixNano()%10000000000000000)
	}
	return fmt.Sprintf("US00%016d", n.Int64())
}
