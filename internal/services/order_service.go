package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tixora/internal/dto"
	"tixora/internal/models"
	"tixora/internal/repository"
	"tixora/internal/utils"
)

const (
	defaultOrderPageLimit = 12
	maxOrderPageLimit     = 50

	orderExpiryDuration = 30 * time.Minute
)

// CreateOrderInput carries the caller-supplied fields needed to place an order.
type CreateOrderInput struct {
	EventID    string
	Quantity   int
	BuyerName  string
	BuyerEmail string
}

// IOrderService contains order creation and lifecycle business logic.
type IOrderService interface {
	CreateOrder(ctx context.Context, userID string, input CreateOrderInput) (*models.Order, error)
	GetOrderByID(ctx context.Context, userID, id string) (*models.Order, error)
	GetUserOrders(ctx context.Context, userID string, page, limit int, status models.OrderStatus) ([]models.Order, int64, error)
	CancelOrder(ctx context.Context, userID, id string) error
	MarkOrderPaid(ctx context.Context, id string) error
	ContinuePayment(ctx context.Context, userID, id string) (*models.Order, error)
	GetAllOrders(ctx context.Context, page, limit int, status models.OrderStatus) ([]models.Order, int64, error)
	GetOrderByOrderIDAdmin(ctx context.Context, id string) (*models.Order, error)
	GenerateTicketPDF(ctx context.Context, userID, id string) (*models.Order, []byte, error)
	GetUserOrderStats(ctx context.Context, userID string) (totalOrders int64, totalSpent int64, err error)
}

type OrderService struct {
	orderRepo      repository.IOrderRepository
	eventRepo      repository.IEventRepository
	userRepo       repository.IUserRepository
	paymentService IPaymentService
}

func NewOrderService(orderRepo repository.IOrderRepository, eventRepo repository.IEventRepository, userRepo repository.IUserRepository, paymentService IPaymentService) IOrderService {
	return &OrderService{orderRepo: orderRepo, eventRepo: eventRepo, userRepo: userRepo, paymentService: paymentService}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID string, input CreateOrderInput) (*models.Order, error) {
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", utils.ErrInvalidInput)
	}
	if input.EventID == "" {
		return nil, fmt.Errorf("%w: event id is required", utils.ErrInvalidInput)
	}
	if !utils.ValidateQuantity(input.Quantity) {
		return nil, fmt.Errorf("%w: quantity must be between 1 and 100", utils.ErrInvalidInput)
	}
	if !utils.ValidateName(input.BuyerName) {
		return nil, fmt.Errorf("%w: buyer name is required", utils.ErrInvalidInput)
	}
	if !utils.ValidateEmail(input.BuyerEmail) {
		return nil, fmt.Errorf("%w: a valid buyer email is required", utils.ErrInvalidInput)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("%w: user", utils.ErrNotFound)
	}

	event, err := s.eventRepo.GetByID(ctx, input.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch event: %w", err)
	}
	if event == nil {
		return nil, fmt.Errorf("%w: event", utils.ErrNotFound)
	}
	if event.AvailableTickets < input.Quantity {
		return nil, fmt.Errorf("%w: not enough tickets available", utils.ErrConflict)
	}

	event.AvailableTickets -= input.Quantity
	if err := s.eventRepo.Update(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to reserve tickets: %w", err)
	}

	subtotal := event.Price * int64(input.Quantity)
	adminFee := dto.AdminFeeFlat
	expiresAt := time.Now().Add(orderExpiryDuration)

	order := &models.Order{
		ID:         utils.GenerateUUID(),
		OrderID:    generateOrderID(),
		UserID:     userID,
		EventID:    event.ID,
		Quantity:   input.Quantity,
		BuyerName:  input.BuyerName,
		BuyerEmail: input.BuyerEmail,
		UnitPrice:  event.Price,
		Subtotal:   subtotal,
		AdminFee:   adminFee,
		TotalPrice: subtotal + adminFee,
		Status:     models.OrderStatusPending,
		ExpiresAt:  &expiresAt,
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	redirectURL, snapToken, err := s.paymentService.CreateTransaction(ctx, order, event.Title, order.BuyerName, order.BuyerEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment transaction: %w", err)
	}

	order.PaymentURL = redirectURL
	order.SnapToken = snapToken
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save payment url: %w", err)
	}

	return order, nil
}

func (s *OrderService) GetOrderByID(ctx context.Context, userID, id string) (*models.Order, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: order id is required", utils.ErrInvalidInput)
	}

	order, err := s.orderRepo.GetByOrderID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}
	if order == nil {
		return nil, fmt.Errorf("%w: order", utils.ErrNotFound)
	}
	if order.UserID != userID {
		return nil, fmt.Errorf("%w: order does not belong to user", utils.ErrForbidden)
	}

	return order, nil
}

func (s *OrderService) GetUserOrders(ctx context.Context, userID string, page, limit int, status models.OrderStatus) ([]models.Order, int64, error) {
	if userID == "" {
		return nil, 0, fmt.Errorf("%w: user id is required", utils.ErrInvalidInput)
	}
	if status != "" && !models.IsValidOrderStatus(status) {
		return nil, 0, fmt.Errorf("%w: invalid order status", utils.ErrInvalidInput)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxOrderPageLimit {
		limit = defaultOrderPageLimit
	}

	offset := (page - 1) * limit
	orders, total, err := s.orderRepo.GetByUserID(ctx, userID, status, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch orders: %w", err)
	}

	return orders, total, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, userID, id string) error {
	order, err := s.GetOrderByID(ctx, userID, id)
	if err != nil {
		return err
	}
	if order.Status != models.OrderStatusPending {
		return fmt.Errorf("%w: only pending orders can be cancelled", utils.ErrConflict)
	}

	event, err := s.eventRepo.GetByID(ctx, order.EventID)
	if err != nil {
		return fmt.Errorf("failed to fetch event: %w", err)
	}
	if event != nil {
		event.AvailableTickets += order.Quantity
		if err := s.eventRepo.Update(ctx, event); err != nil {
			return fmt.Errorf("failed to release tickets: %w", err)
		}
	}

	if err := s.orderRepo.UpdateStatus(ctx, order.ID, models.OrderStatusCancelled); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}

func (s *OrderService) MarkOrderPaid(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: order id is required", utils.ErrInvalidInput)
	}

	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to fetch order: %w", err)
	}
	if order == nil {
		return fmt.Errorf("%w: order", utils.ErrNotFound)
	}
	if order.Status != models.OrderStatusPending {
		return fmt.Errorf("%w: only pending orders can be marked paid", utils.ErrConflict)
	}

	if order.TicketReference == nil {
		ref := generateTicketReference()
		order.TicketReference = &ref
		if err := s.orderRepo.Update(ctx, order); err != nil {
			return fmt.Errorf("failed to save ticket reference: %w", err)
		}
	}

	if err := s.orderRepo.UpdateStatus(ctx, order.ID, models.OrderStatusPaid); err != nil {
		return fmt.Errorf("failed to mark order paid: %w", err)
	}

	return nil
}

// ContinuePayment returns the payment URL/Snap token for a still-pending,
// unexpired order, generating a fresh Midtrans transaction if one was never
// created for it.
func (s *OrderService) ContinuePayment(ctx context.Context, userID, id string) (*models.Order, error) {
	order, err := s.GetOrderByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if order.Status != models.OrderStatusPending {
		return nil, fmt.Errorf("%w: only pending orders can continue payment", utils.ErrConflict)
	}
	if order.ExpiresAt != nil && time.Now().After(*order.ExpiresAt) {
		return nil, fmt.Errorf("%w: order has expired", utils.ErrConflict)
	}

	if order.SnapToken != "" && order.PaymentURL != "" {
		return order, nil
	}

	buyerName, buyerEmail := order.BuyerName, order.BuyerEmail
	if buyerName == "" || buyerEmail == "" {
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user: %w", err)
		}
		if user == nil {
			return nil, fmt.Errorf("%w: user", utils.ErrNotFound)
		}
		buyerName, buyerEmail = user.Name, user.Email
	}

	event, err := s.eventRepo.GetByID(ctx, order.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch event: %w", err)
	}
	if event == nil {
		return nil, fmt.Errorf("%w: event", utils.ErrNotFound)
	}

	redirectURL, snapToken, err := s.paymentService.CreateTransaction(ctx, order, event.Title, buyerName, buyerEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment transaction: %w", err)
	}

	order.PaymentURL = redirectURL
	order.SnapToken = snapToken
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save payment url: %w", err)
	}

	return order, nil
}

// GetAllOrders returns a paginated list of orders across all users, for
// admin use. Unlike GetUserOrders, it is not scoped to a single buyer.
func (s *OrderService) GetAllOrders(ctx context.Context, page, limit int, status models.OrderStatus) ([]models.Order, int64, error) {
	if status != "" && !models.IsValidOrderStatus(status) {
		return nil, 0, fmt.Errorf("%w: invalid order status", utils.ErrInvalidInput)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxOrderPageLimit {
		limit = defaultOrderPageLimit
	}

	offset := (page - 1) * limit
	orders, total, err := s.orderRepo.GetWithPagination(ctx, status, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch orders: %w", err)
	}

	return orders, total, nil
}

// GetOrderByOrderIDAdmin fetches a single order by its human-readable order
// ID without the ownership check GetOrderByID applies, for admin use.
func (s *OrderService) GetOrderByOrderIDAdmin(ctx context.Context, id string) (*models.Order, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: order id is required", utils.ErrInvalidInput)
	}

	order, err := s.orderRepo.GetByOrderID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}
	if order == nil {
		return nil, fmt.Errorf("%w: order", utils.ErrNotFound)
	}

	return order, nil
}

// GenerateTicketPDF returns a paid order owned by userID along with a
// generated PDF e-ticket for it. Tickets are only available once an order
// has been paid.
func (s *OrderService) GenerateTicketPDF(ctx context.Context, userID, id string) (*models.Order, []byte, error) {
	order, err := s.GetOrderByID(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	if order.Status != models.OrderStatusPaid {
		return nil, nil, fmt.Errorf("%w: ticket is only available for paid orders", utils.ErrConflict)
	}

	return order, buildTicketPDF(order), nil
}

// GetUserOrderStats returns the number of paid orders and total amount spent
// by userID. Pending/expired/cancelled orders don't count toward either.
func (s *OrderService) GetUserOrderStats(ctx context.Context, userID string) (int64, int64, error) {
	if userID == "" {
		return 0, 0, fmt.Errorf("%w: user id is required", utils.ErrInvalidInput)
	}

	totalOrders, totalSpent, err := s.orderRepo.GetUserOrderStats(ctx, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch order stats: %w", err)
	}

	return totalOrders, totalSpent, nil
}

// buildTicketPDF and formatRupiah live in ticket_pdf.go.

func generateOrderID() string {
	raw := strings.ToUpper(strings.ReplaceAll(utils.GenerateUUID(), "-", ""))
	return fmt.Sprintf("ORD-%s", raw[:12])
}
