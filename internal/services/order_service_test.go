package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/services"
	"tixora/internal/utils"
)

func newOrderService(orderRepo *mockOrderRepo, eventRepo *mockEventRepo, userRepo *mockUserRepo, paymentSvc *mockPaymentService) services.IOrderService {
	return services.NewOrderService(orderRepo, eventRepo, userRepo, paymentSvc)
}

func validOrderInput() services.CreateOrderInput {
	return services.CreateOrderInput{
		EventID:    "evt-1",
		Quantity:   2,
		BuyerName:  "Jane Doe",
		BuyerEmail: "jane@example.com",
	}
}

func TestOrderService_CreateOrder_Success(t *testing.T) {
	orderRepo := new(mockOrderRepo)
	eventRepo := new(mockEventRepo)
	userRepo := new(mockUserRepo)
	paymentSvc := new(mockPaymentService)

	userRepo.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil)
	event := &models.Event{ID: "evt-1", Title: "Concert", Price: 100000, AvailableTickets: 10}
	eventRepo.On("GetByID", mock.Anything, "evt-1").Return(event, nil)
	eventRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Event")).
		Run(func(args mock.Arguments) {
			e := args.Get(1).(*models.Event)
			assert.Equal(t, 8, e.AvailableTickets, "2 tickets must be reserved from the 10 available")
		}).
		Return(nil)
	orderRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Order")).Return(nil)
	orderRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Order")).Return(nil)
	paymentSvc.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*models.Order"), "Concert", "Jane Doe", "jane@example.com").
		Return("https://snap.example/redirect", "snap-token-abc", nil)

	svc := newOrderService(orderRepo, eventRepo, userRepo, paymentSvc)

	order, err := svc.CreateOrder(context.Background(), "user-1", validOrderInput())
	require.NoError(t, err)
	assert.Equal(t, int64(200000), order.Subtotal)
	assert.Equal(t, int64(5000), order.AdminFee)
	assert.Equal(t, int64(205000), order.TotalPrice)
	assert.Equal(t, models.OrderStatusPending, order.Status)
	assert.Equal(t, "snap-token-abc", order.SnapToken)
	eventRepo.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
	paymentSvc.AssertExpectations(t)
}

func TestOrderService_CreateOrder_InvalidQuantityRejected(t *testing.T) {
	svc := newOrderService(new(mockOrderRepo), new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	input := validOrderInput()
	input.Quantity = 0

	_, err := svc.CreateOrder(context.Background(), "user-1", input)
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestOrderService_CreateOrder_InvalidBuyerEmailRejected(t *testing.T) {
	svc := newOrderService(new(mockOrderRepo), new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	input := validOrderInput()
	input.BuyerEmail = "not-an-email"

	_, err := svc.CreateOrder(context.Background(), "user-1", input)
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestOrderService_CreateOrder_UserNotFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	userRepo.On("GetByID", mock.Anything, "user-1").Return(nil, nil)

	svc := newOrderService(new(mockOrderRepo), new(mockEventRepo), userRepo, new(mockPaymentService))

	_, err := svc.CreateOrder(context.Background(), "user-1", validOrderInput())
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestOrderService_CreateOrder_EventNotFound(t *testing.T) {
	userRepo := new(mockUserRepo)
	userRepo.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil)
	eventRepo := new(mockEventRepo)
	eventRepo.On("GetByID", mock.Anything, "evt-1").Return(nil, nil)

	svc := newOrderService(new(mockOrderRepo), eventRepo, userRepo, new(mockPaymentService))

	_, err := svc.CreateOrder(context.Background(), "user-1", validOrderInput())
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestOrderService_CreateOrder_NotEnoughTickets(t *testing.T) {
	userRepo := new(mockUserRepo)
	userRepo.On("GetByID", mock.Anything, "user-1").Return(&models.User{ID: "user-1"}, nil)
	eventRepo := new(mockEventRepo)
	eventRepo.On("GetByID", mock.Anything, "evt-1").
		Return(&models.Event{ID: "evt-1", AvailableTickets: 1}, nil)

	svc := newOrderService(new(mockOrderRepo), eventRepo, userRepo, new(mockPaymentService))

	_, err := svc.CreateOrder(context.Background(), "user-1", validOrderInput()) // wants 2
	assert.ErrorIs(t, err, utils.ErrConflict)
}

func TestOrderService_GetOrderByID_ForbiddenForOtherUser(t *testing.T) {
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-1").
		Return(&models.Order{ID: "order-row-1", OrderID: "ORD-1", UserID: "owner"}, nil)

	svc := newOrderService(orderRepo, new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	_, err := svc.GetOrderByID(context.Background(), "someone-else", "ORD-1")
	assert.ErrorIs(t, err, utils.ErrForbidden)
}

func TestOrderService_GetOrderByID_NotFound(t *testing.T) {
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-missing").Return(nil, nil)

	svc := newOrderService(orderRepo, new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	_, err := svc.GetOrderByID(context.Background(), "user-1", "ORD-missing")
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestOrderService_CancelOrder_OnlyPendingCanBeCancelled(t *testing.T) {
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-1").
		Return(&models.Order{ID: "row-1", OrderID: "ORD-1", UserID: "user-1", Status: models.OrderStatusPaid}, nil)

	svc := newOrderService(orderRepo, new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	err := svc.CancelOrder(context.Background(), "user-1", "ORD-1")
	assert.ErrorIs(t, err, utils.ErrConflict)
}

func TestOrderService_CancelOrder_ReleasesTickets(t *testing.T) {
	order := &models.Order{ID: "row-1", OrderID: "ORD-1", UserID: "user-1", EventID: "evt-1", Quantity: 3, Status: models.OrderStatusPending}
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-1").Return(order, nil)
	orderRepo.On("UpdateStatus", mock.Anything, "row-1", models.OrderStatusCancelled).Return(nil)

	eventRepo := new(mockEventRepo)
	eventRepo.On("GetByID", mock.Anything, "evt-1").
		Return(&models.Event{ID: "evt-1", AvailableTickets: 5}, nil)
	eventRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Event")).
		Run(func(args mock.Arguments) {
			e := args.Get(1).(*models.Event)
			assert.Equal(t, 8, e.AvailableTickets)
		}).
		Return(nil)

	svc := newOrderService(orderRepo, eventRepo, new(mockUserRepo), new(mockPaymentService))

	err := svc.CancelOrder(context.Background(), "user-1", "ORD-1")
	require.NoError(t, err)
	eventRepo.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
}

func TestOrderService_MarkOrderPaid_OnlyPendingAllowed(t *testing.T) {
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByID", mock.Anything, "row-1").
		Return(&models.Order{ID: "row-1", Status: models.OrderStatusCancelled}, nil)

	svc := newOrderService(orderRepo, new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	err := svc.MarkOrderPaid(context.Background(), "row-1")
	assert.ErrorIs(t, err, utils.ErrConflict)
}

func TestOrderService_MarkOrderPaid_GeneratesTicketReference(t *testing.T) {
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByID", mock.Anything, "row-1").
		Return(&models.Order{ID: "row-1", Status: models.OrderStatusPending}, nil)
	orderRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Order")).
		Run(func(args mock.Arguments) {
			o := args.Get(1).(*models.Order)
			require.NotNil(t, o.TicketReference)
			assert.Contains(t, *o.TicketReference, "TKT-")
		}).
		Return(nil)
	orderRepo.On("UpdateStatus", mock.Anything, "row-1", models.OrderStatusPaid).Return(nil)

	svc := newOrderService(orderRepo, new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	err := svc.MarkOrderPaid(context.Background(), "row-1")
	require.NoError(t, err)
	orderRepo.AssertExpectations(t)
}

func TestOrderService_ContinuePayment_ExpiredOrderRejected(t *testing.T) {
	past := time.Now().Add(-time.Minute)
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-1").
		Return(&models.Order{ID: "row-1", OrderID: "ORD-1", UserID: "user-1", Status: models.OrderStatusPending, ExpiresAt: &past}, nil)

	svc := newOrderService(orderRepo, new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	_, err := svc.ContinuePayment(context.Background(), "user-1", "ORD-1")
	assert.ErrorIs(t, err, utils.ErrConflict)
}

func TestOrderService_ContinuePayment_ReturnsExistingTokenWithoutNewTransaction(t *testing.T) {
	future := time.Now().Add(time.Hour)
	order := &models.Order{
		ID: "row-1", OrderID: "ORD-1", UserID: "user-1", Status: models.OrderStatusPending,
		ExpiresAt: &future, SnapToken: "existing-token", PaymentURL: "https://snap.example/existing",
	}
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-1").Return(order, nil)
	paymentSvc := new(mockPaymentService)

	svc := newOrderService(orderRepo, new(mockEventRepo), new(mockUserRepo), paymentSvc)

	result, err := svc.ContinuePayment(context.Background(), "user-1", "ORD-1")
	require.NoError(t, err)
	assert.Equal(t, "existing-token", result.SnapToken)
	paymentSvc.AssertNotCalled(t, "CreateTransaction", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestOrderService_GenerateTicketPDF_OnlyPaidOrdersAllowed(t *testing.T) {
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-1").
		Return(&models.Order{ID: "row-1", OrderID: "ORD-1", UserID: "user-1", Status: models.OrderStatusPending}, nil)

	svc := newOrderService(orderRepo, new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	_, _, err := svc.GenerateTicketPDF(context.Background(), "user-1", "ORD-1")
	assert.ErrorIs(t, err, utils.ErrConflict)
}

func TestOrderService_GenerateTicketPDF_Success(t *testing.T) {
	orderRepo := new(mockOrderRepo)
	orderRepo.On("GetByOrderID", mock.Anything, "ORD-1").
		Return(&models.Order{ID: "row-1", OrderID: "ORD-1", UserID: "user-1", Status: models.OrderStatusPaid}, nil)

	svc := newOrderService(orderRepo, new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	order, pdf, err := svc.GenerateTicketPDF(context.Background(), "user-1", "ORD-1")
	require.NoError(t, err)
	assert.NotEmpty(t, pdf)
	assert.Equal(t, "ORD-1", order.OrderID)
}

func TestOrderService_GetUserOrderStats_EmptyUserIDRejected(t *testing.T) {
	svc := newOrderService(new(mockOrderRepo), new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	_, _, err := svc.GetUserOrderStats(context.Background(), "")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestOrderService_GetAllOrders_InvalidStatusRejected(t *testing.T) {
	svc := newOrderService(new(mockOrderRepo), new(mockEventRepo), new(mockUserRepo), new(mockPaymentService))

	_, _, err := svc.GetAllOrders(context.Background(), 1, 10, models.OrderStatus("bogus"))
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}
