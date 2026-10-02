package repositories

import (
	"context"
	"errors"
	"time"

	entities "github.com/ortizdavid/go-bank-core-api/core/entities/accounts"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{
		db: db,
	}
}

func (repo *AccountRepository) Create(ctx context.Context, account *entities.Account) error {
	return repo.db.WithContext(ctx).Create(account).Error
}

// UpdateBalance writes only the balance, so it never overwrites other columns
// with stale values. Callers must hold the row lock from GetByNumbersForUpdate.
func (repo *AccountRepository) UpdateBalance(ctx context.Context, accountId int64, balance float64, updatedAt time.Time) error {
	return repo.db.WithContext(ctx).
		Model(&entities.Account{}).
		Where("account_id = ?", accountId).
		Updates(map[string]any{"balance": balance, "updated_at": updatedAt}).Error
}

// UpdateStatus writes only the status, leaving the balance untouched.
func (repo *AccountRepository) UpdateStatus(ctx context.Context, accountId int64, status entities.AccountStatus, updatedAt time.Time) error {
	return repo.db.WithContext(ctx).
		Model(&entities.Account{}).
		Where("account_id = ?", accountId).
		Updates(map[string]any{"account_status": status, "updated_at": updatedAt}).Error
}

// GetByNumbersForUpdate returns the matching accounts and locks their rows
// until the surrounding database transaction ends. Rows are locked in
// account_id order so concurrent transfers in opposite directions cannot deadlock.
func (repo *AccountRepository) GetByNumbersForUpdate(ctx context.Context, accountNumbers ...string) ([]entities.Account, error) {
	var accounts []entities.Account
	result := repo.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("account_number IN ?", accountNumbers).
		Order("account_id").
		Find(&accounts)
	return accounts, result.Error
}

func (repo *AccountRepository) GetById(ctx context.Context, accountId int64) (entities.Account, error) {
	var account entities.Account
	result := repo.db.WithContext(ctx).Table("accounts").First(&account, accountId)
	if result.Error != nil {
		return entities.Account{}, result.Error
	}
	return account, nil
}

func (repo *AccountRepository) GetByNumber(ctx context.Context, accountNumber string) (entities.Account, error) {
	var account entities.Account
	result := repo.db.WithContext(ctx).Table("accounts").Where("account_number = ?", accountNumber).First(&account)
	if result.Error != nil {
		return entities.Account{}, result.Error
	}
	return account, nil
}

func (repo *AccountRepository) GetDataById(ctx context.Context, accountId int64) (entities.AccountData, error) {
	var account entities.AccountData
	result := repo.db.WithContext(ctx).Table("view_account_data").Where("account_id = ?", accountId).First(&account)
	if result.Error != nil {
		return entities.AccountData{}, result.Error
	}
	return account, nil
}

func (repo *AccountRepository) GetDataByNumber(ctx context.Context, accountNumber string) (entities.AccountData, error) {
	var account entities.AccountData
	result := repo.db.WithContext(ctx).Table("view_account_data").Where("account_number = ?", accountNumber).First(&account)
	if result.Error != nil {
		return entities.AccountData{}, result.Error
	}
	return account, nil
}

func (repo *AccountRepository) GetAll(ctx context.Context, limit int, offset int) ([]entities.AccountData, error) {
	var accounts []entities.AccountData
	result := repo.db.WithContext(ctx).
		Table("view_account_data").
		Limit(limit).Offset(offset).
		Find(&accounts)
	return accounts, result.Error
}

func (repo *AccountRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	result := repo.db.WithContext(ctx).Table("accounts").Count(&count)
	return count, result.Error
}

func (repo *AccountRepository) ExistsByNumber(ctx context.Context, accountNumber string) (bool, error) {
	var count int64
	result := repo.db.WithContext(ctx).Table("accounts").Where("account_number = ?", accountNumber).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
