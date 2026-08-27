package dto

import (
	"time"

	"tixora/internal/models"
)

// MidtransWebhookRequest is the payload Midtrans posts to the payment webhook.
type MidtransWebhookRequest struct {
	OrderID           string `json:"order_id" binding:"required"`
	StatusCode        string `json:"status_code" binding:"required"`
	GrossAmount       string `json:"gross_amount" binding:"required"`
	SignatureKey      string `json:"signature_key" binding:"required"`
	TransactionStatus string `json:"transaction_status" binding:"required"`
	TransactionID     string `json:"transaction_id" binding:"required"`
	PaymentType       string `json:"payment_type"`
	FraudStatus       string `json:"fraud_status"`
}

// PaymentStatusResponse is the API representation of a payment's status.
type PaymentStatusResponse struct {
	TransactionID string    `json:"transaction_id"`
	OrderID       string    `json:"order_id"`
	Status        string    `json:"status"`
	Amount        int64     `json:"amount"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func NewPaymentStatusResponse(payment *models.Payment) PaymentStatusResponse {
	return PaymentStatusResponse{
		TransactionID: payment.MidtransTransactionID,
		OrderID:       payment.OrderID,
		Status:        string(payment.Status),
		Amount:        payment.Amount,
		UpdatedAt:     payment.UpdatedAt,
	}
}
