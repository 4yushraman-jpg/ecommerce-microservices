package middleware

import (
	"cart-service/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
				writeJSONError(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeJSONError(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			claims := &models.Claims{}

			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				writeJSONError(w, "Invalid or expired token", http.StatusUnauthorized)
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
			writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, ok := ctxValue.(UserClaims)
		if !ok {
			writeJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if claims.Role != "admin" {
			writeJSONError(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func GetUserClaims(r *http.Request) (UserClaims, bool) {
	v := r.Context().Value(UserContextKey)
	if v == nil {
		return UserClaims{}, false
	}
	claims, ok := v.(UserClaims)
	return claims, ok
}

func writeJSONError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
