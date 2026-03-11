package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"product-catalog-service/internal/middleware"
	"product-catalog-service/internal/models"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (h *CategoryHandler) CreateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if claims.Role != "admin" {
		writeJSONError(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req models.CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))

	if req.Name == "" || req.Slug == "" {
		writeJSONError(w, "Name and slug are required", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO categories (name, slug, parent_id) VALUES ($1, $2, $3)`
	_, err := h.DB.Exec(r.Context(), query, req.Name, req.Slug, req.ParentID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeJSONError(w, "Category slug already exists", http.StatusConflict)
			return
		}
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Category created",
	})
}

func (h *CategoryHandler) UpdateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if claims.Role != "admin" {
		writeJSONError(w, "Forbidden", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSONError(w, "Invalid category id", http.StatusBadRequest)
		return
	}

	var req models.UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ParentID != nil && *req.ParentID == id {
		writeJSONError(w, "A category cannot be its own parent", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if req.Name == "" || req.Slug == "" {
		writeJSONError(w, "Name and slug are required", http.StatusBadRequest)
		return
	}

	query := `UPDATE categories SET name = $1, slug = $2, parent_id = $3 WHERE id = $4`

	res, err := h.DB.Exec(r.Context(), query, req.Name, req.Slug, req.ParentID, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeJSONError(w, "Category slug already exists", http.StatusConflict)
			return
		}

		log.Printf("UpdateCategoryHandler: update failed: %v", err)
		writeJSONError(w, "Failed to update category", http.StatusInternalServerError)
		return
	}

	rowsAffected := res.RowsAffected()
	if rowsAffected == 0 {
		writeJSONError(w, "Category not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Category updated",
	})
}

func (h *CategoryHandler) DeleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if claims.Role != "admin" {
		writeJSONError(w, "Forbidden", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSONError(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	query := `DELETE FROM categories WHERE id = $1`
	res, err := h.DB.Exec(r.Context(), query, id)
	if err != nil {
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	rowsAffected := res.RowsAffected()
	if rowsAffected == 0 {
		writeJSONError(w, "Category not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
