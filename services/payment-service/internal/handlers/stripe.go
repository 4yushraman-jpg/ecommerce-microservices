package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"payment-service/internal/middleware"
	"payment-service/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
)

type PaymentHandler struct {
	DB            *pgxpool.Pool
	JWTSecret     []byte
	WebhookSecret string
	HTTPClient    *http.Client
}

type OrderResponse struct {
	ID          int64  `json:"id"`
	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`
	UserID      int64  `json:"user_id"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, msg string, status int) {
	writeJSON(w, status, map[string]string{
		"error": msg,
	})
}

func (h *PaymentHandler) CreateCheckoutSessionHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	claims, ok := middleware.GetUserClaims(r)
	if !ok {
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var req models.CreateCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if req.OrderID <= 0 {
		writeJSONError(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	orderData, err := h.fetchOrder(ctx, r, req.OrderID)
	if err != nil {
		log.Printf("order fetch failed: %v", err)
		writeJSONError(w, "Failed to verify order", http.StatusBadRequest)
		return
	}

	if orderData.UserID != int64(claims.UserID) {
		writeJSONError(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := validateOrder(orderData); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		log.Println("missing STRIPE_SECRET_KEY")
		writeJSONError(w, "Payment configuration error", http.StatusInternalServerError)
		return
	}

	domain := os.Getenv("FRONTEND_URL")
	if domain == "" {
		domain = "http://localhost:3000"
	}

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
		ClientReferenceID:  stripe.String(fmt.Sprintf("%d", orderData.ID)),

		SuccessURL: stripe.String(domain + "/checkout/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(domain + "/checkout/canceled"),

		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(orderData.Currency),
					UnitAmount: stripe.Int64(orderData.TotalAmount),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(fmt.Sprintf("Order #%d", orderData.ID)),
					},
				},
			},
		},
	}

	idempotencyKey := fmt.Sprintf("checkout_order_%d_user_%d", orderData.ID, claims.UserID)
	params.SetIdempotencyKey(idempotencyKey)

	s, err := session.New(params)
	if err != nil {
		log.Printf("stripe session creation failed: %v", err)
		writeJSONError(w, "Failed to initialize payment", http.StatusInternalServerError)
		return
	}

	if err := h.insertPayment(ctx, orderData, int64(claims.UserID), s.ID); err != nil {
		log.Printf("DB insert failed: %v", err)

		writeJSONError(w, "Failed to persist payment", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, models.CheckoutResponse{
		SessionID:  s.ID,
		SessionURL: s.URL,
	})
}

func (h *PaymentHandler) fetchOrder(ctx context.Context, r *http.Request, orderID int64) (*OrderResponse, error) {
	url := fmt.Sprintf("http://order-service:8080/orders/me/%d", orderID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", r.Header.Get("Authorization"))

	client := h.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("order service returned %d", res.StatusCode)
	}

	var order OrderResponse
	if err := json.NewDecoder(res.Body).Decode(&order); err != nil {
		return nil, err
	}

	return &order, nil
}

func validateOrder(order *OrderResponse) error {
	if order.TotalAmount <= 0 {
		return errors.New("invalid order amount")
	}
	if order.Currency == "" {
		return errors.New("invalid currency")
	}
	return nil
}

func (h *PaymentHandler) insertPayment(ctx context.Context, order *OrderResponse, userID int64, sessionID string) error {
	query := `
		INSERT INTO payments 
		(order_id, user_id, amount, currency, status, provider, provider_transaction_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := h.DB.Exec(ctx, query,
		order.ID,
		userID,
		order.TotalAmount,
		order.Currency,
		models.PaymentStatusPending,
		"stripe",
		sessionID,
	)

	return err
}

func (h *PaymentHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")

	event, err := webhook.ConstructEvent(payload, sigHeader, h.WebhookSecret)
	if err != nil {
		writeJSONError(w, "Signature verification failed", http.StatusBadRequest)
		return
	}

	processed, err := h.isEventProcessed(ctx, event.ID)
	if err != nil {
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if processed {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.processEvent(ctx, &event); err != nil {
		log.Printf("event processing failed: event=%s type=%s err=%v", event.ID, event.Type, err)
		writeJSONError(w, "Failed to process event", http.StatusInternalServerError)
		return
	}

	if err := h.storeEvent(ctx, event); err != nil {
		log.Printf("failed to store event: %v", err)
		writeJSONError(w, "Failed to persist event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *PaymentHandler) processEvent(ctx context.Context, event *stripe.Event) error {
	switch event.Type {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return err
		}
		return h.handleCheckoutCompleted(ctx, &session)

	case "checkout.session.async_payment_failed", "checkout.session.expired":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return err
		}
		return h.handlePaymentFailed(ctx, &session)

	default:
		log.Printf("unhandled event type: %s", event.Type)
		return nil
	}
}

func (h *PaymentHandler) handleCheckoutCompleted(ctx context.Context, s *stripe.CheckoutSession) error {
	if s.ID == "" {
		return errors.New("Missing session ID")
	}

	query := `
		UPDATE payments
		SET status = $1, updated_at = NOW()
		WHERE provider = 'stripe'
		AND provider_transaction_id = $2
		AND status = $3
	`

	res, err := h.DB.Exec(ctx, query, models.PaymentStatusSucceeded, s.ID, models.PaymentStatusPending)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		log.Printf("no rows updated (already processed?): session_id=%s", s.ID)
	}

	// TODO: Notify order-service here (async preferred)

	return nil
}

func (h *PaymentHandler) handlePaymentFailed(ctx context.Context, s *stripe.CheckoutSession) error {
	if s.ID == "" {
		return errors.New("Missing session ID")
	}

	query := `
		UPDATE payments
		SET status = $1, updated_at = NOW()
		WHERE provider = 'stripe'
		AND provider_transaction_id = $2
		AND status = $3
	`

	_, err := h.DB.Exec(ctx, query, models.PaymentStatusFailed, s.ID, models.PaymentStatusPending)

	return err
}

func (h *PaymentHandler) isEventProcessed(ctx context.Context, eventID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM payment_events WHERE event_id = $1)`
	err := h.DB.QueryRow(ctx, query, eventID).Scan(&exists)
	return exists, err
}

func (h *PaymentHandler) storeEvent(ctx context.Context, event stripe.Event) error {
	query := `
		INSERT INTO payment_events (event_id, type, payload)
		VALUES ($1, $2, $3)
	`

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = h.DB.Exec(ctx, query, event.ID, event.Type, payload)
	return err
}
