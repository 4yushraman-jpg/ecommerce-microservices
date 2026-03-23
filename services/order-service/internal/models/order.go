package models

import "time"

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusPaid      OrderStatus = "paid"
	StatusShipped   OrderStatus = "shipped"
	StatusDelivered OrderStatus = "delivered"
	StatusReturned  OrderStatus = "returned"
	StatusCancelled OrderStatus = "cancelled"
)

func (s OrderStatus) IsValid() bool {
	switch s {
	case StatusPending,
		StatusPaid,
		StatusShipped,
		StatusDelivered,
		StatusReturned,
		StatusCancelled:
		return true
	default:
		return false
	}
}

type Address struct {
	Street1 string `json:"street1"`
	Street2 string `json:"street2,omitempty"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
	Phone   string `json:"phone,omitempty"`
}

type Order struct {
	ID              int64       `json:"id"`
	UserID          int64       `json:"user_id"`
	TotalAmount     int64       `json:"total_amount"`
	Currency        string      `json:"currency"`
	Status          OrderStatus `json:"status"`
	ShippingAddress Address     `json:"shipping_address"`
	Items           []OrderItem `json:"items,omitempty"`
	PaymentID       string      `json:"payment_id,omitempty"`
	TrackingNumber  string      `json:"tracking_number,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	DeletedAt       *time.Time  `json:"deleted_at,omitempty"`
}

func (o *Order) CanTransitionTo(next OrderStatus) bool {
	switch o.Status {
	case StatusPending:
		return next == StatusPaid || next == StatusCancelled
	case StatusPaid:
		return next == StatusShipped || next == StatusCancelled
	case StatusShipped:
		return next == StatusDelivered || next == StatusReturned
	case StatusDelivered:
		return next == StatusReturned
	case StatusReturned:
		return false
	case StatusCancelled:
		return false
	default:
		return false
	}
}

type OrderItem struct {
	ID              int64 `json:"id"`
	OrderID         int64 `json:"order_id"`
	ProductID       int64 `json:"product_id"`
	Quantity        int   `json:"quantity"`
	PriceAtPurchase int64 `json:"price_at_purchase"`
}

type BaseModel struct {
	ID        int64      `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
