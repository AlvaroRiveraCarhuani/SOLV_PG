package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"solv-backend/internal/core/domain"
)

type contextKey string

func WithTenant(tenantRepo domain.TenantRepository, jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "missing or invalid authorization header", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return jwtSecret, nil
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

			tenantID, ok := claims["tenant_id"].(string)
			if !ok || tenantID == "" {
				http.Error(w, "unauthorized: missing tenant_id in token", http.StatusUnauthorized)
				return
			}

			userID, ok := claims["user_id"].(string)
			if !ok || userID == "" {
				http.Error(w, "unauthorized: missing user_id in token", http.StatusUnauthorized)
				return
			}

			// Validar si el tenant_id existe en la BD (Punto 4 del ajuste obligatorio)
			_, err = tenantRepo.GetByID(r.Context(), tenantID)
			if err != nil {
				http.Error(w, "unauthorized: tenant not found", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), domain.TenantIDKey, tenantID)
			ctx = context.WithValue(ctx, domain.UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetTenantIDFromContext(ctx context.Context) (string, error) {
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		return "", domain.ErrTenantIDMissing
	}
	return tenantID, nil
}
