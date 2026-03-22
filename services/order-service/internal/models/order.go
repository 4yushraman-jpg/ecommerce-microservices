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

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}

type Order struct {
	ID              int         `json:"id"`
	UserID          int         `json:"user_id"`
	TotalAmount     int         `json:"total_amount"`
	Status          OrderStatus `json:"status"`
	ShippingAddress Address     `json:"shipping_address"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID              int `json:"id"`
	OrderID         int `json:"order_id"`
	ProductID       int `json:"product_id"`
	Quantity        int `json:"quantity"`
	PriceAtPurchase int `json:"price_at_purchase"`
}
