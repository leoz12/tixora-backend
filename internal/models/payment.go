package models

import (
	"encoding/json"
	"time"
)

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusSettled PaymentStatus = "settled"
	PaymentStatusFailed  PaymentStatus = "failed"
	PaymentStatusExpired PaymentStatus = "expired"
)

type Payment struct {
	ID      string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	OrderID string `gorm:"column:order_id;index;type:varchar(36);not null" json:"order_id"`

	MidtransTransactionID string `gorm:"column:midtrans_transaction_id;uniqueIndex;type:varchar(100)" json:"midtrans_transaction_id"`

	Amount int64         `gorm:"not null" json:"amount"`
	Status PaymentStatus `gorm:"type:varchar(20);index;not null;default:pending" json:"status"`

	PaymentDetails json.RawMessage `gorm:"column:payment_details;type:json" json:"payment_details"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Payment) TableName() string {
	return "payments"
}
