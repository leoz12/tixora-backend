package dto

import (
	"time"

	"tixora/internal/models"
)

// CreateOrderRequest is the request body for creating an order.
type CreateOrderRequest struct {
	EventID    string `json:"event_id" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,min=1,max=100"`
	BuyerEmail string `json:"buyer_email" binding:"required,email"`
	BuyerName  string `json:"buyer_name" binding:"required,min=3"`
}

// OrderResponse is the API representation of an order.
type OrderResponse struct {
	ID               string     `json:"id"`
	OrderID          string     `json:"order_id"`
	EventID          string     `json:"event_id"`
	EventTitle       string     `json:"event_title,omitempty"`
	EventImage       string     `json:"event_image,omitempty"`
	EventDescription string     `json:"event_description,omitempty"`
	EventDate        *time.Time `json:"event_date,omitempty"`
	EventLocation    string     `json:"event_location,omitempty"`
	Quantity         int        `json:"quantity"`
	UnitPrice        int64      `json:"unit_price"`
	Subtotal         int64      `json:"subtotal"`
	AdminFee         int64      `json:"admin_fee"`
	TotalPrice       int64      `json:"total_price"`
	Status           string     `json:"status"`
	PaymentURL       string     `json:"payment_url"`
	SnapToken        string     `json:"snap_token"`
	PaymentMethod    string     `json:"payment_method"`
	TicketReference  *string    `json:"ticket_reference"`
	BuyerName        string     `json:"buyer_name,omitempty"`
	BuyerEmail       string     `json:"buyer_email,omitempty"`
	PaidAt           *time.Time `json:"paid_at"`
	ExpiresAt        *time.Time `json:"expires_at"`
	PurchasedAt      time.Time  `json:"purchased_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func NewOrderResponse(order *models.Order, imageBaseURL string) OrderResponse {
	resp := OrderResponse{
		ID:              order.ID,
		OrderID:         order.OrderID,
		EventID:         order.EventID,
		Quantity:        order.Quantity,
		UnitPrice:       order.UnitPrice,
		Subtotal:        order.Subtotal,
		AdminFee:        order.AdminFee,
		TotalPrice:      order.TotalPrice,
		Status:          string(order.Status),
		PaymentURL:      order.PaymentURL,
		SnapToken:       order.SnapToken,
		PaymentMethod:   order.PaymentMethod,
		TicketReference: order.TicketReference,
		BuyerName:       order.BuyerName,
		BuyerEmail:      order.BuyerEmail,
		PaidAt:          order.PaidAt,
		ExpiresAt:       order.ExpiresAt,
		PurchasedAt:     order.CreatedAt,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}

	if order.Event.ID != "" {
		resp.EventTitle = order.Event.Title
		resp.EventDescription = order.Event.Description
		if order.Event.Image != nil {
			resp.EventImage = imageBaseURL + "/" + order.Event.Image.ObjectKey
		}
		eventDate := order.Event.EventDate
		resp.EventDate = &eventDate
		resp.EventLocation = order.Event.Location
	}

	if order.User.ID != "" {
		if resp.BuyerName == "" {
			resp.BuyerName = order.User.Name
		}
		if resp.BuyerEmail == "" {
			resp.BuyerEmail = order.User.Email
		}
	}

	return resp
}

func NewOrderResponseList(orders []models.Order, imageBaseURL string) []OrderResponse {
	responses := make([]OrderResponse, 0, len(orders))
	for _, order := range orders {
		responses = append(responses, NewOrderResponse(&order, imageBaseURL))
	}
	return responses
}

// OrderStatusResponse is a lightweight representation of an order's payment
// status, intended for polling.
type OrderStatusResponse struct {
	ID        string     `json:"id"`
	OrderID   string     `json:"order_id"`
	Status    string     `json:"status"`
	PaidAt    *time.Time `json:"paid_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func NewOrderStatusResponse(order *models.Order) OrderStatusResponse {
	return OrderStatusResponse{
		ID:        order.ID,
		OrderID:   order.OrderID,
		Status:    string(order.Status),
		PaidAt:    order.PaidAt,
		ExpiresAt: order.ExpiresAt,
	}
}
