package main

import (
	"log"
	"net/http"
	"os"

	"user-service/internal/database"
	"user-service/internal/handlers"
	"user-service/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
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

	handler := &handlers.UserHandler{
		DB:        pool,
		JWTSecret: []byte(jwtSecret),
	}

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Public routes
	r.Post("/signup", handler.SignupUserHandler)
	r.Post("/login", handler.LoginUserHandler)

	// Health check
	r.Get("/health", handlers.HealthHandler)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware([]byte(jwtSecret)))
		r.Get("/users/me", handler.GetProfileHandler)
		r.Put("/users/me", handler.UpdateProfileHandler)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("user-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
