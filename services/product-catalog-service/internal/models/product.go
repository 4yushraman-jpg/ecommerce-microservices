package models

import "time"

type Product struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Price         int       `json:"price"`
	SKU           string    `json:"sku"`
	CategoryID    int       `json:"category_id"`
	StockQuantity int       `json:"stock_quantity"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateProductRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Price         int    `json:"price"`
	SKU           string `json:"sku"`
	CategoryID    int    `json:"category_id"`
	StockQuantity int    `json:"stock_quantity"`
}
