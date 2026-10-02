package repositories

import (
	"context"
	"errors"

	entities "github.com/ortizdavid/go-bank-core-api/core/entities/transactions"
	"gorm.io/gorm"
)

type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{
		db: db,
	}
}

func (repo *TransactionRepository) Create(ctx context.Context, transaction *entities.Transaction) error {
	return repo.db.WithContext(ctx).Create(transaction).Error
}

func (repo *TransactionRepository) GetDataById(ctx context.Context, transactionId int64) (entities.TransactionData, error) {
	var transaction entities.TransactionData
	result := repo.db.WithContext(ctx).Table("view_transaction_data").Where("transaction_id = ?", transactionId).First(&transaction)
	if result.Error != nil {
		return entities.TransactionData{}, result.Error
	}
	return transaction, nil
}

// historyOrder lists newest transactions first. transaction_id breaks ties so
// pages stay stable when several rows share a timestamp.
const historyOrder = "created_at DESC, transaction_id DESC"

func (repo *TransactionRepository) GetAll(ctx context.Context, limit int, offset int) ([]entities.TransactionData, error) {
	var transactions []entities.TransactionData
	result := repo.db.WithContext(ctx).
		Table("view_transaction_data").
		Order(historyOrder).
		Limit(limit).Offset(offset).
		Find(&transactions)
	return transactions, result.Error
}

func (repo *TransactionRepository) GetByAccountId(ctx context.Context, accountId int64, limit int, offset int) ([]entities.TransactionData, error) {
	var transactions []entities.TransactionData
	result := repo.db.WithContext(ctx).
		Table("view_transaction_data").
		Where("source_account_id = ? OR destination_account_id = ?", accountId, accountId).
		Order(historyOrder).
		Limit(limit).Offset(offset).
		Find(&transactions)
	return transactions, result.Error
}

func (repo *TransactionRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	result := repo.db.WithContext(ctx).Table("transactions").Count(&count)
	return count, result.Error
}

func (repo *TransactionRepository) CountByAccountId(ctx context.Context, accountId int64) (int64, error) {
	var count int64
	result := repo.db.WithContext(ctx).
		Table("transactions").
		Where("source_id = ? OR destination_id = ?", accountId, accountId).
		Count(&count)
	return count, result.Error
}

// LockIdempotencyKey makes requests that share a key run one at a time. The
// lock is released when the surrounding database transaction ends, so a waiting
// request only continues once the first one has committed or rolled back.
func (repo *TransactionRepository) LockIdempotencyKey(ctx context.Context, key string) error {
	return repo.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error
}

// GetByIdempotencyKey returns the transaction previously recorded for the key.
func (repo *TransactionRepository) GetByIdempotencyKey(ctx context.Context, key string) (entities.Transaction, error) {
	var transaction entities.Transaction
	result := repo.db.WithContext(ctx).
		Joins("JOIN idempotency_keys ON idempotency_keys.transaction_id = transactions.transaction_id").
		Where("idempotency_keys.idempotency_key = ?", key).
		First(&transaction)
	if result.Error != nil {
		return entities.Transaction{}, result.Error
	}
	return transaction, nil
}

func (repo *TransactionRepository) CreateIdempotencyKey(ctx context.Context, record *entities.IdempotencyKey) error {
	return repo.db.WithContext(ctx).Create(record).Error
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
