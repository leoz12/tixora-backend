package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/repository"
)

func TestPaymentRepository_CreateAndGetByTransactionID(t *testing.T) {
	db := requireDB(t)
	userID, eventID := seedUserAndEvent(t, db, "pay1")
	require.NoError(t, repository.NewOrderRepository(db).Create(context.Background(),
		newTestOrder("row-1", "ORD-PAY1", userID, eventID, models.OrderStatusPending)))

	repo := repository.NewPaymentRepository(db)
	ctx := context.Background()

	payment := &models.Payment{
		ID: "pay-1", OrderID: "row-1", MidtransTransactionID: "txn-1",
		Amount: 205000, Status: models.PaymentStatusPending,
	}
	require.NoError(t, repo.Create(ctx, payment))

	fetched, err := repo.GetByTransactionID(ctx, "txn-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, models.PaymentStatusPending, fetched.Status)
}

func TestPaymentRepository_GetByTransactionID_NotFound(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewPaymentRepository(db)

	fetched, err := repo.GetByTransactionID(context.Background(), "missing")
	require.NoError(t, err)
	assert.Nil(t, fetched)
}

func TestPaymentRepository_UniqueTransactionIDConstraint(t *testing.T) {
	db := requireDB(t)
	userID, eventID := seedUserAndEvent(t, db, "pay2")
	require.NoError(t, repository.NewOrderRepository(db).Create(context.Background(),
		newTestOrder("row-1", "ORD-PAY2", userID, eventID, models.OrderStatusPending)))

	repo := repository.NewPaymentRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Payment{
		ID: "pay-1", OrderID: "row-1", MidtransTransactionID: "dup-txn", Amount: 1000, Status: models.PaymentStatusPending,
	}))

	err := repo.Create(ctx, &models.Payment{
		ID: "pay-2", OrderID: "row-1", MidtransTransactionID: "dup-txn", Amount: 1000, Status: models.PaymentStatusPending,
	})
	assert.Error(t, err)
}

func TestPaymentRepository_UpdateStatus(t *testing.T) {
	db := requireDB(t)
	userID, eventID := seedUserAndEvent(t, db, "pay3")
	require.NoError(t, repository.NewOrderRepository(db).Create(context.Background(),
		newTestOrder("row-1", "ORD-PAY3", userID, eventID, models.OrderStatusPending)))

	repo := repository.NewPaymentRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Payment{
		ID: "pay-1", OrderID: "row-1", MidtransTransactionID: "txn-3", Amount: 1000, Status: models.PaymentStatusPending,
	}))

	require.NoError(t, repo.UpdateStatus(ctx, "pay-1", models.PaymentStatusSettled))

	fetched, err := repo.GetByTransactionID(ctx, "txn-3")
	require.NoError(t, err)
	assert.Equal(t, models.PaymentStatusSettled, fetched.Status)
}

func TestPaymentRepository_UpdateStatus_NonexistentRowReturnsError(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewPaymentRepository(db)

	err := repo.UpdateStatus(context.Background(), "does-not-exist", models.PaymentStatusSettled)
	assert.Error(t, err)
}
