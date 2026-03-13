package main

import (
	"log"
	"net/http"
	"os"
	"product-catalog-service/internal/database"
	"product-catalog-service/internal/handlers"
	"product-catalog-service/internal/middleware"

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

	productHandler := &handlers.ProductHandler{
		DB: pool,
	}

	categoryHandler := &handlers.CategoryHandler{
		DB: pool,
	}

	r := chi.NewRouter()

	r.Get("/products", productHandler.GetProductsHandler)
	r.Get("/products/{id}", productHandler.GetProductByIDHandler)
	r.Get("/categories", categoryHandler.GetCategoriesHandler)
	r.Get("/categories/{id}", categoryHandler.GetCategoryByIDHandler)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware([]byte(jwtSecret)))

		r.Group(func(r chi.Router) {
			r.Use(middleware.AdminOnlyMiddleware)

			r.Post("/products", productHandler.CreateProductHandler)
			r.Put("/products/{id}", productHandler.UpdateProductHandler)
			r.Delete("/products/{id}", productHandler.DeleteProductHandler)

			r.Post("/categories", categoryHandler.CreateCategoryHandler)
			r.Put("/categories/{id}", categoryHandler.UpdateCategoryHandler)
			r.Delete("/categories/{id}", categoryHandler.DeleteCategoryHandler)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("product-catalog-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
