package httpdelivery

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"solv-backend/internal/core/domain"
)
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// key type para inyectar en el contexto sin colisiones
type contextKey string

const UserIDKey contextKey = "user_id"

func WithAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenString string
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else if cookie, err := r.Cookie("solv_session"); err == nil && cookie.Value != "" {
			tokenString = cookie.Value
		}

		if tokenString == "" {
			// Si el request ya viene autenticado aguas arriba por ForwardAuth
			if r.Header.Get("X-User-Id") != "" {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "missing or invalid authorization header or session cookie", http.StatusUnauthorized)
			return
		}

		jwtSecret := os.Getenv("JWT_SECRET")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || token == nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			http.Error(w, "missing user_id in token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, domain.UserIDKey, userID)
		r.Header.Set("X-User-Id", userID)

		if tenantID, ok := claims["tenant_id"].(string); ok && tenantID != "" {
			ctx = context.WithValue(ctx, domain.TenantIDKey, tenantID)
			r.Header.Set("X-Tenant-Id", tenantID)
		}
		if role, ok := claims["role"].(string); ok && role != "" {
			ctx = context.WithValue(ctx, domain.UserRoleKey, role)
			r.Header.Set("X-User-Role", role)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
