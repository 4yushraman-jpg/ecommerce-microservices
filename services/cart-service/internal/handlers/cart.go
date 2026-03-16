package handlers

import (
	"cart-service/internal/middleware"
	"cart-service/internal/models"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CartHandler struct {
	DB *pgxpool.Pool
}

func (h *CartHandler) GetUserCart(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := `SELECT id, created_at, updated_at FROM carts WHERE user_id = $1`
	var cart models.Cart

	err := h.DB.QueryRow(r.Context(), query, claims.UserID).Scan(
		&cart.ID,
		&cart.CreatedAt,
		&cart.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		writeJSONError(w, "Cart not found", http.StatusNotFound)
		return
	}
	if err != nil {
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	query = `SELECT id, product_id, quantity FROM cart_items WHERE cart_id = $1 ORDER BY id`
	rows, err := h.DB.Query(r.Context(), query, cart.ID)
	if err != nil {
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	cartItems := make([]models.CartItem, 0)
	for rows.Next() {
		var c models.CartItem
		if err := rows.Scan(&c.ID, &c.ProductID, &c.Quantity); err != nil {
			writeJSONError(w, "Error fetching cart items", http.StatusInternalServerError)
			return
		}
		cartItems = append(cartItems, c)
	}
	if rows.Err() != nil {
		writeJSONError(w, "Error iterating cart items", http.StatusInternalServerError)
		return
	}

	cart.UserID = claims.UserID
	cart.Items = cartItems

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(cart)
}

func (h *CartHandler) AddItemHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.AddItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ProductID <= 0 || req.Quantity <= 0 {
		writeJSONError(w, "Invalid product ID or quantity", http.StatusBadRequest)
		return
	}

	var cartID int
	err := h.DB.QueryRow(r.Context(), `SELECT id FROM carts WHERE user_id = $1`, claims.UserID).Scan(&cartID)

	if errors.Is(err, pgx.ErrNoRows) {
		err = h.DB.QueryRow(r.Context(), `INSERT INTO carts (user_id) VALUES ($1) RETURNING id`, claims.UserID).Scan(&cartID)
		if err != nil {
			log.Printf("AddItemHandler: failed to create cart: %v", err)
			writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		log.Printf("AddItemHandler: failed to fetch cart: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	query := `
        INSERT INTO cart_items (cart_id, product_id, quantity) 
        VALUES ($1, $2, $3)
        ON CONFLICT (cart_id, product_id) 
        DO UPDATE SET quantity = EXCLUDED.quantity`

	_, err = h.DB.Exec(r.Context(), query, cartID, req.ProductID, req.Quantity)
	if err != nil {
		log.Printf("AddItemHandler: failed to add item: %v", err)
		writeJSONError(w, "Failed to add item to cart", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Product added to cart successfully",
	})
}

func (h *CartHandler) UpdateItemHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Quantity == 0 {
		h.DeleteItemHandler(w, r)
		return
	} else if req.Quantity < 0 {
		writeJSONError(w, "Quantity cannot be negative", http.StatusBadRequest)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSONError(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	query := `
        UPDATE cart_items 
        SET quantity = $1 
        WHERE product_id = $2 
        AND cart_id = (SELECT id FROM carts WHERE user_id = $3)
    `

	res, err := h.DB.Exec(r.Context(), query, req.Quantity, id, claims.UserID)
	if err != nil {
		log.Printf("UpdateItemHandler: update failed: %v", err)
		writeJSONError(w, "Failed to update quantity", http.StatusInternalServerError)
		return
	}

	if res.RowsAffected() == 0 {
		writeJSONError(w, "Item not found in cart", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Product quantity updated",
	})
}

func (h *CartHandler) DeleteItemHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSONError(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	query := `DELETE FROM cart_items WHERE product_id = $1 AND cart_id = (SELECT id FROM carts WHERE user_id = $2)`
	res, err := h.DB.Exec(r.Context(), query, id, claims.UserID)
	if err != nil {
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if res.RowsAffected() == 0 {
		writeJSONError(w, "Product not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CartHandler) DeleteCartHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := `DELETE FROM carts WHERE user_id = $1`
	res, err := h.DB.Exec(r.Context(), query, claims.UserID)
	if err != nil {
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if res.RowsAffected() == 0 {
		writeJSONError(w, "Cart not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSONError(w http.ResponseWriter, errMsg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": errMsg,
	})
}
