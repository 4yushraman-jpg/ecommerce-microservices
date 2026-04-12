package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"payment-service/internal/database"
	"payment-service/internal/events"
	"payment-service/internal/handlers"
	"payment-service/internal/middleware"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, relying on environment variables")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Fatal("STRIPE_WEBHOOK_SECRET is required")
	}

	pool, err := database.ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	brokerURL := os.Getenv("KAFKA_BROKER_URL")
	if brokerURL == "" {
		brokerURL = "kafka:9092"
	}

	kafkaProducer := events.NewKafkaProducer(brokerURL)
	defer kafkaProducer.Close()

	paymentHandler := &handlers.PaymentHandler{
		DB:            pool,
		JWTSecret:     []byte(jwtSecret),
		WebhookSecret: webhookSecret,
		KafkaProducer: kafkaProducer,
	}

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Post("/payments/webhook", paymentHandler.HandleStripeWebhook)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware([]byte(jwtSecret)))
		r.Post("/payments/checkout-session", paymentHandler.CreateCheckoutSessionHandler)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Create a channel to listen for OS signals (like Ctrl+C or Docker stop)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Start the server in a separate goroutine so it doesn't block
	go func() {
		log.Printf("payment-service listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Block here until a signal is received
	<-stopChan
	log.Println("Shutting down gracefully...")

	// Give the server 5 seconds to finish active requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
