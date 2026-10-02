package entities

import "time"

type Account struct {
	AccountId     int64         `gorm:"primaryKey;autoIncrement"`
	CustomerId    int64         `gorm:"column:customer_id"`
	AccountStatus AccountStatus `gorm:"column:account_status"`
	AccountType   AccountType   `gorm:"column:account_type"`
	AccountNumber string        `gorm:"column:account_number"`
	Iban          string        `gorm:"column:iban"`
	HolderName    string        `gorm:"column:holder_name"`
	Balance       float64       `gorm:"column:balance"`
	Currency      string        `gorm:"column:currency"`
	UniqueId      string        `gorm:"column:unique_id"`
	CreatedAt     time.Time     `gorm:"column:created_at"`
	UpdatedAt     time.Time     `gorm:"column:updated_at"`
}

func (*Account) TableName() string {
	return "accounts"
}
