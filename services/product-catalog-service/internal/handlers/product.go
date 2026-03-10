package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"product-catalog-service/internal/models"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProductHandler struct {
	DB *pgxpool.Pool
}

func (h *ProductHandler) GetProductsHandler(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	offset := 0
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 1 {
			offset = (n - 1) * limit
		}
	}

	var args []interface{}
	var whereClauses []string
	argNum := 1

	if categoryID := r.URL.Query().Get("category_id"); categoryID != "" {
		if catID, err := strconv.Atoi(categoryID); err == nil && catID > 0 {
			whereClauses = append(whereClauses, "category_id = $"+strconv.Itoa(argNum))
			args = append(args, catID)
			argNum++
		}
	}

	search := strings.TrimSpace(r.URL.Query().Get("q"))
	if search != "" {
		searchPattern := "%" + search + "%"
		whereClauses = append(whereClauses, "(name ILIKE $"+strconv.Itoa(argNum)+" OR COALESCE(description,'') ILIKE $"+strconv.Itoa(argNum)+")")
		args = append(args, searchPattern)
		argNum++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	args = append(args, limit, offset)
	query := `SELECT id, name, COALESCE(description, ''), price, sku, category_id, stock_quantity, created_at, updated_at
		FROM products` + whereSQL + ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argNum) + ` OFFSET $` + strconv.Itoa(argNum+1)

	rows, err := h.DB.Query(r.Context(), query, args...)
	if err != nil {
		log.Printf("GetProductsHandler: query failed: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	products := make([]models.Product, 0, limit)
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.SKU, &p.CategoryID, &p.StockQuantity, &p.CreatedAt, &p.UpdatedAt); err != nil {
			log.Printf("GetProductsHandler: scan failed: %v", err)
			writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}
	if rows.Err() != nil {
		log.Printf("GetProductsHandler: rows.Err: %v", rows.Err())
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(products)
}

func writeJSONError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (h *ProductHandler) GetProductByIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSONError(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	query := `SELECT id, name, COALESCE(description, ''), price, sku, category_id, stock_quantity, created_at, updated_at FROM products WHERE id = $1`

	var product models.Product

	err = h.DB.QueryRow(r.Context(), query, id).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.SKU,
		&product.CategoryID,
		&product.StockQuantity,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		writeJSONError(w, "Product not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(product)
}
