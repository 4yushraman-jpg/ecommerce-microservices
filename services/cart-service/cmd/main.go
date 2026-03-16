package main

import (
	"log"
	"net/http"
	"os"

	"cart-service/internal/database"
	"cart-service/internal/handlers"
	"cart-service/internal/middleware"

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
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	cartHandler := &handlers.CartHandler{
		DB: pool,
	}

	r := chi.NewRouter()

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Authenticated cart routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware([]byte(jwtSecret)))

		r.Get("/carts/me", cartHandler.GetUserCart)
		r.Post("/carts/me/items", cartHandler.AddItemHandler)
		r.Put("/carts/me/items/{id}", cartHandler.UpdateItemHandler)
		r.Delete("/carts/me/items/{id}", cartHandler.DeleteItemHandler)
		r.Delete("/carts/me", cartHandler.DeleteCartHandler)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("cart-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
