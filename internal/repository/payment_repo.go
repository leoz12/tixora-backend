package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"tixora/internal/models"
)

type IPaymentRepository interface {
	Create(ctx context.Context, payment *models.Payment) error
	GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error)
	UpdateStatus(ctx context.Context, id string, status models.PaymentStatus) error
}

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) IPaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	if err := r.db.WithContext(ctx).Create(payment).Error; err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	return nil
}

func (r *PaymentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error) {
	var payment models.Payment
	if err := r.db.WithContext(ctx).
		Where("midtrans_transaction_id = ?", transactionID).
		First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment by transaction id: %w", err)
	}
	return &payment, nil
}

func (r *PaymentRepository) UpdateStatus(ctx context.Context, id string, status models.PaymentStatus) error {
	result := r.db.WithContext(ctx).Model(&models.Payment{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("failed to update payment status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("update payment status: %w", gorm.ErrRecordNotFound)
	}
	return nil
}
