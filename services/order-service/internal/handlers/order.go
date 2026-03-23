package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"order-service/internal/middleware"
	"order-service/internal/models"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderHandler struct {
	DB *pgxpool.Pool
}

func writeJSONError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (h *OrderHandler) GetMyOrdersHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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

	query := `
		SELECT 
			id, user_id, total_amount, currency, status, 
			shipping_street1, COALESCE(shipping_street2, ''), 
			shipping_city, shipping_state, shipping_zip, shipping_country, COALESCE(shipping_phone, ''), 
			COALESCE(payment_id, ''), COALESCE(tracking_number, ''), 
			created_at, updated_at
		FROM orders 
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3
	`

	rows, err := h.DB.Query(r.Context(), query, claims.UserID, limit, offset)
	if err != nil {
		log.Printf("GetMyOrdersHandler: query failed: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	orders := make([]models.Order, 0, limit)
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID,
			&o.UserID,
			&o.TotalAmount,
			&o.Currency,
			&o.Status,
			&o.ShippingAddress.Street1,
			&o.ShippingAddress.Street2,
			&o.ShippingAddress.City,
			&o.ShippingAddress.State,
			&o.ShippingAddress.Zip,
			&o.ShippingAddress.Country,
			&o.ShippingAddress.Phone,
			&o.PaymentID,
			&o.TrackingNumber,
			&o.CreatedAt,
			&o.UpdatedAt,
		); err != nil {
			log.Printf("GetMyOrdersHandler: scan failed: %v", err)
			writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		orders = append(orders, o)
	}

	if rows.Err() != nil {
		log.Printf("GetMyOrdersHandler: rows.Err: %v", rows.Err())
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orders)
}

func (h *OrderHandler) GetOrderDetailsHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || orderID <= 0 {
		writeJSONError(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	orderQuery := `
		SELECT 
			id, user_id, total_amount, currency, status, 
			shipping_street1, COALESCE(shipping_street2, ''), 
			shipping_city, shipping_state, shipping_zip, shipping_country, COALESCE(shipping_phone, ''), 
			COALESCE(payment_id, ''), COALESCE(tracking_number, ''), 
			created_at, updated_at
		FROM orders 
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	var o models.Order
	err = h.DB.QueryRow(r.Context(), orderQuery, orderID, claims.UserID).Scan(
		&o.ID, &o.UserID, &o.TotalAmount, &o.Currency, &o.Status,
		&o.ShippingAddress.Street1, &o.ShippingAddress.Street2,
		&o.ShippingAddress.City, &o.ShippingAddress.State,
		&o.ShippingAddress.Zip, &o.ShippingAddress.Country, &o.ShippingAddress.Phone,
		&o.PaymentID, &o.TrackingNumber,
		&o.CreatedAt, &o.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		writeJSONError(w, "Order not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("GetOrderDetails: parent query failed: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	itemsQuery := `
		SELECT id, order_id, product_id, quantity, price_at_purchase
		FROM order_items
		WHERE order_id = $1
	`
	rows, err := h.DB.Query(r.Context(), itemsQuery, orderID)
	if err != nil {
		log.Printf("GetOrderDetails: items query failed: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.PriceAtPurchase); err != nil {
			log.Printf("GetOrderDetails: item scan failed: %v", err)
			writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	if rows.Err() != nil {
		log.Printf("GetOrderDetails: rows iteration error: %v", rows.Err())
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	o.Items = items

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(o)
}

type CheckoutRequest struct {
	ShippingAddress models.Address `json:"shipping_address"`
}

type CartItemResponse struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type CartResponse struct {
	Items []CartItemResponse `json:"items"`
}

type ProductResponse struct {
	ID            int64 `json:"id"`
	Price         int64 `json:"price"`
	StockQuantity int   `json:"stock_quantity"`
}

func (h *OrderHandler) CheckoutHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	authHeader := r.Header.Get("Authorization")

	cartReq, _ := http.NewRequest("GET", "http://cart-service:8080/carts/me", nil)
	cartReq.Header.Set("Authorization", authHeader)

	cartRes, err := http.DefaultClient.Do(cartReq)
	if err != nil || cartRes.StatusCode != http.StatusOK {
		log.Printf("Checkout: Failed to fetch cart: %v", err)
		writeJSONError(w, "Failed to retrieve cart", http.StatusInternalServerError)
		return
	}
	defer cartRes.Body.Close()

	var cart CartResponse
	if err := json.NewDecoder(cartRes.Body).Decode(&cart); err != nil {
		writeJSONError(w, "Failed to parse cart data", http.StatusInternalServerError)
		return
	}

	if len(cart.Items) == 0 {
		writeJSONError(w, "Your cart is empty", http.StatusBadRequest)
		return
	}

	var totalAmount int64 = 0
	var finalOrderItems []models.OrderItem

	for _, item := range cart.Items {
		productURL := fmt.Sprintf("http://product-catalog-service:8080/products/%d", item.ProductID)
		prodRes, err := http.Get(productURL)

		if err != nil || prodRes.StatusCode != http.StatusOK {
			writeJSONError(w, fmt.Sprintf("Product %d is unavailable", item.ProductID), http.StatusBadRequest)
			return
		}

		var product ProductResponse
		json.NewDecoder(prodRes.Body).Decode(&product)
		prodRes.Body.Close()

		if product.StockQuantity < item.Quantity {
			writeJSONError(w, fmt.Sprintf("Insufficient stock for product %d", item.ProductID), http.StatusBadRequest)
			return
		}

		totalAmount += (product.Price * int64(item.Quantity))

		finalOrderItems = append(finalOrderItems, models.OrderItem{
			ProductID:       item.ProductID,
			Quantity:        item.Quantity,
			PriceAtPurchase: product.Price,
		})
	}

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(context.Background())

	orderQuery := `
		INSERT INTO orders (user_id, total_amount, currency, status, shipping_street1, shipping_street2, shipping_city, shipping_state, shipping_zip, shipping_country, shipping_phone)
		VALUES ($1, $2, 'INR', 'pending', $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	var orderID int64
	err = tx.QueryRow(r.Context(), orderQuery,
		claims.UserID, totalAmount,
		req.ShippingAddress.Street1, req.ShippingAddress.Street2,
		req.ShippingAddress.City, req.ShippingAddress.State,
		req.ShippingAddress.Zip, req.ShippingAddress.Country, req.ShippingAddress.Phone,
	).Scan(&orderID)

	if err != nil {
		log.Printf("Checkout: Order insert failed: %v", err)
		writeJSONError(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	itemQuery := `INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase) VALUES ($1, $2, $3, $4)`
	for _, item := range finalOrderItems {
		_, err = tx.Exec(r.Context(), itemQuery, orderID, item.ProductID, item.Quantity, item.PriceAtPurchase)
		if err != nil {
			log.Printf("Checkout: OrderItem insert failed: %v", err)
			writeJSONError(w, "Failed to save order items", http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(r.Context()); err != nil {
		log.Printf("Checkout: TX Commit failed: %v", err)
		writeJSONError(w, "Failed to finalize order", http.StatusInternalServerError)
		return
	}

	clearCartReq, _ := http.NewRequest("DELETE", "http://cart-service:8080/carts/me", nil)
	clearCartReq.Header.Set("Authorization", authHeader)
	http.DefaultClient.Do(clearCartReq)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Order placed successfully",
		"order_id": orderID,
		"status":   "pending",
		"total":    totalAmount,
	})
}

type UpdateOrderStatusRequest struct {
	Status models.OrderStatus `json:"status"`
}

func (h *OrderHandler) UpdateOrderStatusHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if claims.Role != "admin" {
		writeJSONError(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || orderID <= 0 {
		writeJSONError(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	var req UpdateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if !req.Status.IsValid() {
		writeJSONError(w, "Invalid order status provided", http.StatusBadRequest)
		return
	}

	var currentOrder models.Order
	query := `SELECT id, status FROM orders WHERE id = $1 AND deleted_at IS NULL`

	err = h.DB.QueryRow(r.Context(), query, orderID).Scan(&currentOrder.ID, &currentOrder.Status)
	if err == pgx.ErrNoRows {
		writeJSONError(w, "Order not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("UpdateOrderStatus: fetch query failed: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !currentOrder.CanTransitionTo(req.Status) {
		errMsg := "Invalid transition: cannot move order from '" + string(currentOrder.Status) + "' to '" + string(req.Status) + "'"
		writeJSONError(w, errMsg, http.StatusConflict)
		return
	}

	updateQuery := `UPDATE orders SET status = $1 WHERE id = $2`
	res, err := h.DB.Exec(r.Context(), updateQuery, req.Status, orderID)
	if err != nil {
		log.Printf("UpdateOrderStatus: update failed: %v", err)
		writeJSONError(w, "Failed to update order status", http.StatusInternalServerError)
		return
	}

	if res.RowsAffected() == 0 {
		writeJSONError(w, "Order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message":    "Order status updated successfully",
		"old_status": string(currentOrder.Status),
		"new_status": string(req.Status),
	})
}
