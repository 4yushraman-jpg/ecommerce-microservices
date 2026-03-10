package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"product-catalog-service/internal/models"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryHandler struct {
	DB *pgxpool.Pool
}

func (h *CategoryHandler) GetCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, name, slug, parent_id, created_at FROM categories ORDER BY created_at DESC`

	rows, err := h.DB.Query(r.Context(), query)
	if err != nil {
		log.Printf("GetCategoriesHandler: query failed: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	categories := make([]models.Category, 0, 50)
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.CreatedAt); err != nil {
			log.Printf("GetCategoriesHandler: scan failed: %v", err)
			writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		categories = append(categories, c)
	}
	if rows.Err() != nil {
		log.Printf("GetCategoriesHandler: rows.Err: %v", rows.Err())
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(categories)
}

func (h *CategoryHandler) GetCategoryByIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSONError(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	offset := 0
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 1 {
			page = n
			offset = (n - 1) * limit
		}
	}

	categoryQuery := `SELECT id, name, slug, parent_id, created_at FROM categories WHERE id = $1`
	var category models.CategoryByID

	err = h.DB.QueryRow(r.Context(), categoryQuery, id).Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.ParentID,
		&category.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		writeJSONError(w, "Category not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("GetCategoryByIDHandler: category query failed: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	productsQuery := `
		SELECT 
			id, name, COALESCE(description, ''), price, sku, category_id, stock_quantity, created_at, updated_at,
			COUNT(*) OVER() AS total_count
		FROM products 
		WHERE category_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3`

	rows, err := h.DB.Query(r.Context(), productsQuery, id, limit, offset)
	if err != nil {
		log.Printf("GetCategoryByIDHandler: products query failed: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var productsTotal int
	products := make([]models.Product, 0, limit)

	for rows.Next() {
		var p models.Product
		var rowTotal int

		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.SKU,
			&p.CategoryID, &p.StockQuantity, &p.CreatedAt, &p.UpdatedAt,
			&rowTotal,
		); err != nil {
			log.Printf("GetCategoryByIDHandler: scan failed: %v", err)
			writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		products = append(products, p)
		productsTotal = rowTotal
	}
	if rows.Err() != nil {
		log.Printf("GetCategoryByIDHandler: rows.Err: %v", rows.Err())
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	category.Products = products
	category.ProductsTotal = productsTotal
	category.ProductsPage = page
	category.ProductsLimit = limit

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(category)
}
