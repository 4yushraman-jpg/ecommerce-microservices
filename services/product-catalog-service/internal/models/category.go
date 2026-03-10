package models

import "time"

type Category struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	ParentID  *int      `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
}

type CategoryByID struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	ParentID      *int      `json:"parent_id"`
	CreatedAt     time.Time `json:"created_at"`
	Products      []Product `json:"products"`
	ProductsTotal int       `json:"products_total"`
	ProductsPage  int       `json:"products_page"`
	ProductsLimit int       `json:"products_limit"`
}
