package main

import (
	"log"
	"net/http"
	"os"

	"order-service/internal/database"
	"order-service/internal/handlers"
	"order-service/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	pool, err := database.ConnectDB()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()

	orderHandler := &handlers.OrderHandler{
		DB: pool,
	}

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware([]byte(jwtSecret)))

		r.Post("/orders/checkout", orderHandler.CheckoutHandler)
		r.Get("/orders/me", orderHandler.GetMyOrdersHandler)
		r.Get("/orders/me/{id}", orderHandler.GetOrderDetailsHandler)

		r.Group(func(r chi.Router) {
			r.Use(middleware.AdminOnlyMiddleware)

			r.Patch("/orders/{id}/status", orderHandler.UpdateOrderStatusHandler)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("order-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
