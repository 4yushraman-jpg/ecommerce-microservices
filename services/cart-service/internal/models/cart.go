package models

import "time"

type Cart struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	Items     []CartItem `json:"items"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CartItem struct {
	ID        int `json:"id"`
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type AddItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type UpdateItemRequest struct {
	Quantity int `json:"quantity"`
}
