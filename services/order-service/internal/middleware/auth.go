package middleware

import (
	"context"
	"fmt"
	"net/http"
	"order-service/internal/models"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "userContext"

type UserClaims struct {
	UserID int
	Role   string
}

func AuthMiddleware(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing auth header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid auth header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			claims := &models.Claims{}

			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			userCtxPayload := UserClaims{
				UserID: claims.UserID,
				Role:   claims.Role,
			}

			ctx := context.WithValue(r.Context(), UserContextKey, userCtxPayload)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxValue := r.Context().Value(UserContextKey)
		if ctxValue == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, ok := ctxValue.(UserClaims)
		if !ok {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if claims.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func GetUserClaims(r *http.Request) (UserClaims, bool) {
	ctxValue := r.Context().Value(UserContextKey)
	if ctxValue == nil {
		return UserClaims{}, false
	}
	claims, ok := ctxValue.(UserClaims)
	return claims, ok
}
