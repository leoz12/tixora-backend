package services_test

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/services"
	"tixora/internal/utils"
)

const testMidtransServerKey = "SB-Mid-server-test-key"

func newPaymentService(paymentRepo *mockPaymentRepo, orderRepo *mockOrderRepo, eventRepo *mockEventRepo) services.IPaymentService {
	return services.NewPaymentService(paymentRepo, orderRepo, eventRepo, testMidtransServerKey, true)
}

// validSignature computes the Midtrans notification signature the same way
// PaymentService.verifySignature does: SHA512(order_id+status_code+gross_amount+ServerKey).
func validSignature(orderID, statusCode, grossAmount string) string {
	sum := sha512.Sum512([]byte(orderID + statusCode + grossAmount + testMidtransServerKey))
	return hex.EncodeToString(sum[:])
}

func TestPaymentService_CreateTransaction_NilOrderRejected(t *testing.T) {
	svc := newPaymentService(new(mockPaymentRepo), new(mockOrderRepo), new(mockEventRepo))

	_, _, err := svc.CreateTransaction(context.Background(), nil, "Concert", "Jane", "jane@example.com")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestPaymentService_ProcessWebhook_IncompleteNotificationRejected(t *testing.T) {
	svc := newPaymentService(new(mockPaymentRepo), new(mockOrderRepo), new(mockEventRepo))

	err := svc.ProcessWebhook(context.Background(), services.MidtransNotification{OrderID: "ORD-1"})
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestPaymentService_ProcessWebhook_InvalidSignatureRejected(t *testing.T) {
	svc := newPaymentService(new(mockPaymentRepo), new(mockOrderRepo), new(mockEventRepo))

	n := services.MidtransNotification{
		OrderID: "ORD-1", TransactionID: "txn-1", StatusCode: "200", GrossAmount: "205000.00",
		TransactionStatus: "settlement", SignatureKey: "not-the-right-signature",
	}

	err := svc.ProcessWebhook(context.Background(), n)
	assert.ErrorIs(t, err, utils.ErrUnauthorized)
}

func TestPaymentService_ProcessWebhook_RefundIgnored(t *testing.T) {
	orderRepo := new(mockOrderRepo)
	svc := newPaymentService(new(mockPaymentRepo), orderRepo, new(mockEventRepo))

	n := services.MidtransNotification{
		OrderID: "ORD-1", TransactionID: "txn-1", StatusCode: "200", GrossAmount: "205000.00",
		TransactionStatus: "refund",
	}
	n.SignatureKey = validSignature(n.OrderID, n.StatusCode, n.GrossAmount)

	err := svc.ProcessWebhook(context.Background(), n)
	require.NoError(t, err)
	orderRepo.AssertNotCalled(t, "GetByOrderID", mock.Anything, mock.Anything)
}

func TestPaymentService_ProcessWebhook_OrderNotFound(t *testing.T) {
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-missing").Return(nil, nil)

	svc := newPaymentService(new(mockPaymentRepo), orderRepo, new(mockEventRepo))

	n := services.MidtransNotification{
		OrderID: "ORD-missing", TransactionID: "txn-1", StatusCode: "200", GrossAmount: "205000.00",
		TransactionStatus: "settlement",
	}
	n.SignatureKey = validSignature(n.OrderID, n.StatusCode, n.GrossAmount)

	err := svc.ProcessWebhook(context.Background(), n)
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestPaymentService_ProcessWebhook_SettlementMarksOrderPaid(t *testing.T) {
	order := &models.Order{ID: "row-1", OrderID: "ORD-1", Status: models.OrderStatusPending}
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-1").Return(order, nil)
	orderRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Order")).
		Run(func(args mock.Arguments) {
			o := args.Get(1).(*models.Order)
			assert.Equal(t, models.OrderStatusPaid, o.Status)
			assert.NotNil(t, o.PaidAt)
			require.NotNil(t, o.TicketReference)
			assert.Contains(t, *o.TicketReference, "TKT-")
		}).
		Return(nil)

	paymentRepo := new(mockPaymentRepo)
	paymentRepo.On("GetByTransactionID", mock.Anything, "txn-1").Return(nil, nil)
	paymentRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Payment")).
		Run(func(args mock.Arguments) {
			p := args.Get(1).(*models.Payment)
			assert.Equal(t, models.PaymentStatusSettled, p.Status)
		}).
		Return(nil)

	svc := newPaymentService(paymentRepo, orderRepo, new(mockEventRepo))

	n := services.MidtransNotification{
		OrderID: "ORD-1", TransactionID: "txn-1", StatusCode: "200", GrossAmount: "205000.00",
		TransactionStatus: "settlement", PaymentType: "bank_transfer",
	}
	n.SignatureKey = validSignature(n.OrderID, n.StatusCode, n.GrossAmount)

	err := svc.ProcessWebhook(context.Background(), n)
	require.NoError(t, err)
	orderRepo.AssertExpectations(t)
	paymentRepo.AssertExpectations(t)
}

func TestPaymentService_ProcessWebhook_SettlementIsIdempotent(t *testing.T) {
	// Order already paid - a duplicate/late notification must not re-apply
	// the transition (Update must not be called again).
	order := &models.Order{ID: "row-1", OrderID: "ORD-1", Status: models.OrderStatusPaid}
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-1").Return(order, nil)

	paymentRepo := new(mockPaymentRepo)
	paymentRepo.On("GetByTransactionID", mock.Anything, "txn-1").
		Return(&models.Payment{ID: "pay-1", Status: models.PaymentStatusSettled}, nil)
	paymentRepo.On("UpdateStatus", mock.Anything, "pay-1", models.PaymentStatusSettled).Return(nil)

	svc := newPaymentService(paymentRepo, orderRepo, new(mockEventRepo))

	n := services.MidtransNotification{
		OrderID: "ORD-1", TransactionID: "txn-1", StatusCode: "200", GrossAmount: "205000.00",
		TransactionStatus: "settlement",
	}
	n.SignatureKey = validSignature(n.OrderID, n.StatusCode, n.GrossAmount)

	err := svc.ProcessWebhook(context.Background(), n)
	require.NoError(t, err)
	orderRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestPaymentService_ProcessWebhook_ExpireReleasesTickets(t *testing.T) {
	order := &models.Order{ID: "row-1", OrderID: "ORD-1", EventID: "evt-1", Quantity: 2, Status: models.OrderStatusPending}
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-1").Return(order, nil)
	orderRepo.On("UpdateStatus", mock.Anything, "row-1", models.OrderStatusExpired).Return(nil)

	eventRepo := new(mockEventRepo)
	eventRepo.On("GetByID", mock.Anything, "evt-1").
		Return(&models.Event{ID: "evt-1", AvailableTickets: 3}, nil)
	eventRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Event")).
		Run(func(args mock.Arguments) {
			e := args.Get(1).(*models.Event)
			assert.Equal(t, 5, e.AvailableTickets)
		}).
		Return(nil)

	paymentRepo := new(mockPaymentRepo)
	paymentRepo.On("GetByTransactionID", mock.Anything, "txn-1").Return(nil, nil)
	paymentRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Payment")).Return(nil)

	svc := newPaymentService(paymentRepo, orderRepo, eventRepo)

	n := services.MidtransNotification{
		OrderID: "ORD-1", TransactionID: "txn-1", StatusCode: "200", GrossAmount: "205000.00",
		TransactionStatus: "expire",
	}
	n.SignatureKey = validSignature(n.OrderID, n.StatusCode, n.GrossAmount)

	err := svc.ProcessWebhook(context.Background(), n)
	require.NoError(t, err)
	eventRepo.AssertExpectations(t)
}

func TestPaymentService_ProcessWebhook_CapturePendingFraudStaysUnpaid(t *testing.T) {
	order := &models.Order{ID: "row-1", OrderID: "ORD-1", Status: models.OrderStatusPending}
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-1").Return(order, nil)

	paymentRepo := new(mockPaymentRepo)
	paymentRepo.On("GetByTransactionID", mock.Anything, "txn-1").Return(nil, nil)
	paymentRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Payment")).
		Run(func(args mock.Arguments) {
			p := args.Get(1).(*models.Payment)
			assert.Equal(t, models.PaymentStatusPending, p.Status)
		}).
		Return(nil)

	svc := newPaymentService(paymentRepo, orderRepo, new(mockEventRepo))

	n := services.MidtransNotification{
		OrderID: "ORD-1", TransactionID: "txn-1", StatusCode: "200", GrossAmount: "205000.00",
		TransactionStatus: "capture", FraudStatus: "challenge",
	}
	n.SignatureKey = validSignature(n.OrderID, n.StatusCode, n.GrossAmount)

	err := svc.ProcessWebhook(context.Background(), n)
	require.NoError(t, err)
	orderRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestPaymentService_GetPaymentStatus_EmptyTransactionIDRejected(t *testing.T) {
	svc := newPaymentService(new(mockPaymentRepo), new(mockOrderRepo), new(mockEventRepo))

	_, err := svc.GetPaymentStatus(context.Background(), "user-1", "")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestPaymentService_GetPaymentStatus_NotFound(t *testing.T) {
	paymentRepo := new(mockPaymentRepo)
	paymentRepo.On("GetByTransactionID", mock.Anything, "txn-1").Return(nil, nil)

	svc := newPaymentService(paymentRepo, new(mockOrderRepo), new(mockEventRepo))

	_, err := svc.GetPaymentStatus(context.Background(), "user-1", "txn-1")
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestPaymentService_GetPaymentStatus_ForbiddenForOtherUser(t *testing.T) {
	paymentRepo := new(mockPaymentRepo)
	paymentRepo.On("GetByTransactionID", mock.Anything, "txn-1").
		Return(&models.Payment{ID: "pay-1", OrderID: "row-1"}, nil)

	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByID", mock.Anything, "row-1").
		Return(&models.Order{ID: "row-1", UserID: "owner"}, nil)

	svc := newPaymentService(paymentRepo, orderRepo, new(mockEventRepo))

	_, err := svc.GetPaymentStatus(context.Background(), "someone-else", "txn-1")
	assert.ErrorIs(t, err, utils.ErrForbidden)
}

func TestPaymentService_GetPaymentStatus_Success(t *testing.T) {
	paymentRepo := new(mockPaymentRepo)
	paymentRepo.On("GetByTransactionID", mock.Anything, "txn-1").
		Return(&models.Payment{ID: "pay-1", OrderID: "row-1", Status: models.PaymentStatusSettled}, nil)

	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByID", mock.Anything, "row-1").
		Return(&models.Order{ID: "row-1", UserID: "user-1"}, nil)

	svc := newPaymentService(paymentRepo, orderRepo, new(mockEventRepo))

	payment, err := svc.GetPaymentStatus(context.Background(), "user-1", "txn-1")
	require.NoError(t, err)
	assert.Equal(t, models.PaymentStatusSettled, payment.Status)
}
