package models

import "time"

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusExpired   OrderStatus = "expired"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// IsValidOrderStatus reports whether status is a known OrderStatus value.
func IsValidOrderStatus(status OrderStatus) bool {
	switch status {
	case OrderStatusPending, OrderStatusPaid, OrderStatusExpired, OrderStatusCancelled:
		return true
	default:
		return false
	}
}

type Order struct {
	ID      string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	OrderID string `gorm:"column:order_id;uniqueIndex;type:varchar(50);not null" json:"order_id"`
	UserID  string `gorm:"column:user_id;index;type:varchar(36);not null" json:"user_id"`
	EventID string `gorm:"column:event_id;index;type:varchar(36);not null" json:"event_id"`

	Quantity int `gorm:"not null" json:"quantity"`

	BuyerName  string `gorm:"column:buyer_name;type:varchar(255)" json:"buyer_name"`
	BuyerEmail string `gorm:"column:buyer_email;type:varchar(255)" json:"buyer_email"`

	UnitPrice  int64 `gorm:"column:unit_price;not null" json:"unit_price"`
	Subtotal   int64 `gorm:"not null" json:"subtotal"`
	AdminFee   int64 `gorm:"column:admin_fee;not null" json:"admin_fee"`
	TotalPrice int64 `gorm:"column:total_price;not null" json:"total_price"`

	Status OrderStatus `gorm:"type:varchar(20);index;not null;default:pending" json:"status"`

	PaymentURL    string     `gorm:"column:payment_url;type:varchar(512)" json:"payment_url"`
	SnapToken     string     `gorm:"column:snap_token;type:varchar(128)" json:"snap_token"`
	PaymentMethod string     `gorm:"column:payment_method;type:varchar(50)" json:"payment_method"`
	PaidAt        *time.Time `gorm:"column:paid_at" json:"paid_at"`
	ExpiresAt     *time.Time `gorm:"column:expires_at" json:"expires_at"`

	TicketReference *string `gorm:"column:ticket_reference;uniqueIndex;type:varchar(100)" json:"ticket_reference"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	User     User      `gorm:"foreignKey:UserID" json:"-"`
	Event    Event     `gorm:"foreignKey:EventID" json:"-"`
	Payments []Payment `gorm:"foreignKey:OrderID" json:"-"`
}

func (Order) TableName() string {
	return "orders"
}
