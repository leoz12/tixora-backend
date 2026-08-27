package services

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tixora/internal/models"
	"tixora/internal/repository"
	"tixora/internal/utils"
)

const (
	snapSandboxBaseURL    = "https://app.sandbox.midtrans.com/snap/v1"
	snapProductionBaseURL = "https://app.midtrans.com/snap/v1"
)

// snapTransactionRequest is the request body for Midtrans' Snap "create
// transaction" endpoint. See: https://docs.midtrans.com/reference/create-transaction
type snapTransactionRequest struct {
	TransactionDetails snapTransactionDetails `json:"transaction_details"`
	CustomerDetails    snapCustomerDetails    `json:"customer_details"`
	ItemDetails        []snapItemDetail       `json:"item_details"`
}

type snapTransactionDetails struct {
	OrderID     string `json:"order_id"`
	GrossAmount int64  `json:"gross_amount"`
}

type snapCustomerDetails struct {
	FirstName string `json:"first_name"`
	Email     string `json:"email"`
}

type snapItemDetail struct {
	ID       string `json:"id"`
	Price    int64  `json:"price"`
	Quantity int    `json:"quantity"`
	Name     string `json:"name"`
}

type snapTransactionResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

// MidtransNotification carries the fields Midtrans includes in a payment
// notification (webhook) callback.
type MidtransNotification struct {
	OrderID           string
	StatusCode        string
	GrossAmount       string
	SignatureKey      string
	TransactionStatus string
	TransactionID     string
	PaymentType       string
	FraudStatus       string
}

// IPaymentService contains payment webhook and status business logic.
type IPaymentService interface {
	CreateTransaction(ctx context.Context, order *models.Order, itemName, customerName, customerEmail string) (redirectURL, snapToken string, err error)
	ProcessWebhook(ctx context.Context, notification MidtransNotification) error
	GetPaymentStatus(ctx context.Context, userID, transactionID string) (*models.Payment, error)
}

type PaymentService struct {
	paymentRepo       repository.IPaymentRepository
	orderRepo         repository.IOrderRepository
	eventRepo         repository.IEventRepository
	midtransServerKey string
	midtransIsSandbox bool
	httpClient        *http.Client
}

func NewPaymentService(paymentRepo repository.IPaymentRepository, orderRepo repository.IOrderRepository, eventRepo repository.IEventRepository, midtransServerKey string, midtransIsSandbox bool) IPaymentService {
	return &PaymentService{
		paymentRepo:       paymentRepo,
		orderRepo:         orderRepo,
		eventRepo:         eventRepo,
		midtransServerKey: midtransServerKey,
		midtransIsSandbox: midtransIsSandbox,
		httpClient:        &http.Client{Timeout: 10 * time.Second},
	}
}

// CreateTransaction requests a Snap payment page from Midtrans for order and
// returns the redirect URL the client should send the buyer to, along with
// the Snap token. See: https://docs.midtrans.com/reference/create-transaction
func (s *PaymentService) CreateTransaction(ctx context.Context, order *models.Order, itemName, customerName, customerEmail string) (string, string, error) {
	if order == nil {
		return "", "", fmt.Errorf("%w: order is required", utils.ErrInvalidInput)
	}

	reqBody := snapTransactionRequest{
		TransactionDetails: snapTransactionDetails{
			OrderID:     order.OrderID,
			GrossAmount: order.TotalPrice,
		},
		CustomerDetails: snapCustomerDetails{
			FirstName: customerName,
			Email:     customerEmail,
		},
		ItemDetails: []snapItemDetail{
			{ID: order.EventID, Price: order.UnitPrice, Quantity: order.Quantity, Name: itemName},
			{ID: "admin_fee", Price: order.AdminFee, Quantity: 1, Name: "Admin Fee"},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("failed to encode snap request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.snapBaseURL()+"/transactions", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("failed to build snap request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(s.midtransServerKey+":")))

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("failed to call midtrans snap api: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read snap response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return "", "", fmt.Errorf("midtrans snap api returned %d: %s", resp.StatusCode, string(respBody))
	}

	var snapResp snapTransactionResponse
	if err := json.Unmarshal(respBody, &snapResp); err != nil {
		return "", "", fmt.Errorf("failed to decode snap response: %w", err)
	}

	return snapResp.RedirectURL, snapResp.Token, nil
}

func (s *PaymentService) snapBaseURL() string {
	if s.midtransIsSandbox {
		return snapSandboxBaseURL
	}
	return snapProductionBaseURL
}

func (s *PaymentService) ProcessWebhook(ctx context.Context, n MidtransNotification) error {
	if n.OrderID == "" || n.TransactionID == "" || n.StatusCode == "" || n.GrossAmount == "" {
		return fmt.Errorf("%w: incomplete payment notification", utils.ErrInvalidInput)
	}

	if !s.verifySignature(n) {
		return fmt.Errorf("%w: invalid midtrans signature", utils.ErrUnauthorized)
	}

	// Refunds are not supported yet; ignore the notification rather than
	// mutating payment/order state we have no downstream flow for.
	if n.TransactionStatus == "refund" || n.TransactionStatus == "partial_refund" {
		return nil
	}

	order, err := s.orderRepo.GetByOrderID(ctx, n.OrderID)
	if err != nil {
		return fmt.Errorf("failed to fetch order: %w", err)
	}
	if order == nil {
		return fmt.Errorf("%w: order", utils.ErrNotFound)
	}

	amount, err := strconv.ParseFloat(n.GrossAmount, 64)
	if err != nil {
		return fmt.Errorf("%w: invalid gross amount", utils.ErrInvalidInput)
	}

	details, err := json.Marshal(n)
	if err != nil {
		return fmt.Errorf("failed to encode payment details: %w", err)
	}

	paymentStatus, orderStatus := mapMidtransStatus(n.TransactionStatus, n.FraudStatus)

	payment, err := s.paymentRepo.GetByTransactionID(ctx, n.TransactionID)
	if err != nil {
		return fmt.Errorf("failed to fetch payment: %w", err)
	}

	if payment == nil {
		payment = &models.Payment{
			ID:                    utils.GenerateUUID(),
			OrderID:               order.ID,
			MidtransTransactionID: n.TransactionID,
			Amount:                int64(amount),
			Status:                paymentStatus,
			PaymentDetails:        details,
		}
		if err := s.paymentRepo.Create(ctx, payment); err != nil {
			return fmt.Errorf("failed to create payment: %w", err)
		}
	} else if err := s.paymentRepo.UpdateStatus(ctx, payment.ID, paymentStatus); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return s.applyOrderTransition(ctx, order, orderStatus, n.PaymentType)
}

// applyOrderTransition moves order to orderStatus, guarding against
// re-processing a notification for an order that already moved on.
func (s *PaymentService) applyOrderTransition(ctx context.Context, order *models.Order, orderStatus models.OrderStatus, paymentType string) error {
	switch orderStatus {
	case models.OrderStatusPaid:
		if order.Status != models.OrderStatusPending {
			return nil
		}
		now := time.Now()
		order.Status = models.OrderStatusPaid
		order.PaidAt = &now
		order.PaymentMethod = paymentType
		if order.TicketReference == nil {
			ref := generateTicketReference()
			order.TicketReference = &ref
		}
		if err := s.orderRepo.Update(ctx, order); err != nil {
			return fmt.Errorf("failed to mark order paid: %w", err)
		}

	case models.OrderStatusExpired, models.OrderStatusCancelled:
		if order.Status != models.OrderStatusPending {
			return nil
		}
		if err := s.releaseTickets(ctx, order); err != nil {
			return err
		}
		if err := s.orderRepo.UpdateStatus(ctx, order.ID, orderStatus); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}
	}

	return nil
}

func generateTicketReference() string {
	raw := strings.ToUpper(strings.ReplaceAll(utils.GenerateUUID(), "-", ""))
	return fmt.Sprintf("TKT-%s", raw[:12])
}

func (s *PaymentService) releaseTickets(ctx context.Context, order *models.Order) error {
	event, err := s.eventRepo.GetByID(ctx, order.EventID)
	if err != nil {
		return fmt.Errorf("failed to fetch event: %w", err)
	}
	if event == nil {
		return nil
	}

	event.AvailableTickets += order.Quantity
	if err := s.eventRepo.Update(ctx, event); err != nil {
		return fmt.Errorf("failed to release tickets: %w", err)
	}
	return nil
}

func (s *PaymentService) GetPaymentStatus(ctx context.Context, userID, transactionID string) (*models.Payment, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("%w: transaction id is required", utils.ErrInvalidInput)
	}

	payment, err := s.paymentRepo.GetByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment: %w", err)
	}
	if payment == nil {
		return nil, fmt.Errorf("%w: payment", utils.ErrNotFound)
	}

	order, err := s.orderRepo.GetByID(ctx, payment.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}
	if order == nil || order.UserID != userID {
		return nil, fmt.Errorf("%w: payment does not belong to user", utils.ErrForbidden)
	}

	return payment, nil
}

// verifySignature checks the Midtrans notification signature: SHA512(order_id + status_code + gross_amount + ServerKey).
// See: https://docs.midtrans.com/docs/https-notification-webhooks
func (s *PaymentService) verifySignature(n MidtransNotification) bool {
	raw := n.OrderID + n.StatusCode + n.GrossAmount + s.midtransServerKey
	sum := sha512.Sum512([]byte(raw))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(n.SignatureKey)) == 1
}

// mapMidtransStatus translates a Midtrans transaction_status (and, for
// credit card captures, fraud_status) into our internal payment/order statuses.
func mapMidtransStatus(transactionStatus, fraudStatus string) (models.PaymentStatus, models.OrderStatus) {
	switch transactionStatus {
	case "capture":
		if fraudStatus == "accept" {
			return models.PaymentStatusSettled, models.OrderStatusPaid
		}
		return models.PaymentStatusPending, models.OrderStatusPending
	case "settlement":
		return models.PaymentStatusSettled, models.OrderStatusPaid
	case "deny":
		return models.PaymentStatusFailed, models.OrderStatusCancelled
	case "cancel":
		return models.PaymentStatusFailed, models.OrderStatusCancelled
	case "expire":
		return models.PaymentStatusExpired, models.OrderStatusExpired
	default: // "pending" and anything unrecognized
		return models.PaymentStatusPending, models.OrderStatusPending
	}
}
