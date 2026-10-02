package entities

import "time"

type IdempotencyKey struct {
	IdempotencyKey string    `gorm:"column:idempotency_key;primaryKey"`
	TransactionId  int64     `gorm:"column:transaction_id"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (IdempotencyKey) TableName() string {
	return "idempotency_keys"
}
