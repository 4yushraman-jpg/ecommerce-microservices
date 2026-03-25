package models

import "time"

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

type BaseModel struct {
	ID        int64      `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type Payment struct {
	BaseModel
	OrderID               int64         `json:"order_id"`
	UserID                int64         `json:"user_id"`
	Amount                int64         `json:"amount"`
	Currency              string        `json:"currency"`
	Status                PaymentStatus `json:"status"`
	Provider              string        `json:"provider"`
	ProviderTransactionID string        `json:"provider_transaction_id"`
}

type CreateCheckoutRequest struct {
	OrderID int64 `json:"order_id"`
}

type CheckoutResponse struct {
	SessionID  string `json:"session_id"`
	SessionURL string `json:"session_url"`
}
