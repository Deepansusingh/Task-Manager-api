package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func response(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	body, _ := json.Marshal(data)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(status)
	w.Write(body)
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Step 1 — get the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response(w, http.StatusUnauthorized, map[string]string{"error": "authorization header required"})
			return
		}

		// Step 2 — check format is "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response(w, http.StatusUnauthorized, map[string]string{"error": "invalid authorization format, use: Bearer <token>"})
			return
		}

		tokenString := parts[1]

		// Step 3 — parse and verify JWT signature
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil {
			response(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
			return
		}

		// Step 4 — extract claims from verified token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			response(w, http.StatusUnauthorized, map[string]string{"error": "invalid token claims"})
			return
		}

		// Step 5 — extract user_id (JWT stores numbers as float64)
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			response(w, http.StatusUnauthorized, map[string]string{"error": "invalid token: missing user_id"})
			return
		}

		userID := uint(userIDFloat)

		// Step 6 — attach user_id to request context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// Step 7 — call the actual handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}