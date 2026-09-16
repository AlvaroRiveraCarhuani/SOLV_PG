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
			var tokenString string
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			} else if cookie, err := r.Cookie("solv_session"); err == nil && cookie.Value != "" {
				tokenString = cookie.Value
			}

			if tokenString == "" {
				// Si ya viene pre-autenticado por ForwardAuth o proxy interno
				if userID := r.Header.Get("X-User-Id"); userID != "" {
					tenantID := r.Header.Get("X-Tenant-Id")
					if tenantID == "" {
						tenantID = "00000000-0000-0000-0000-000000000001"
					}
					ctx := context.WithValue(r.Context(), domain.TenantIDKey, tenantID)
					ctx = context.WithValue(ctx, domain.UserIDKey, userID)
					if role := r.Header.Get("X-User-Role"); role != "" {
						ctx = context.WithValue(ctx, domain.UserRoleKey, role)
					}
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				http.Error(w, "missing or invalid authorization header or session cookie", http.StatusUnauthorized)
				return
			}

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
			r.Header.Set("X-User-Id", userID)
			r.Header.Set("X-Tenant-Id", tenantID)

			if userRole, _ := claims["role"].(string); userRole != "" {
				ctx = context.WithValue(ctx, domain.UserRoleKey, userRole)
				r.Header.Set("X-User-Role", userRole)
			}
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
