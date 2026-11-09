package middleware

import (
	"net/http"
	"strings"

	"solv-backend/internal/core/services"
)

// SessionRevocationMiddleware rechaza los JWT que el logout ya revocó. El
// chequeo se hace antes de las rutas para cubrir tanto las APIs HTTP como el
// upgrade de WebSocket; los handlers siguen validando firma, expiración y
// claims. Los headers de identidad no participan.
func SessionRevocationMiddleware(authService *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenString string
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			} else if cookie, err := r.Cookie("solv_session"); err == nil {
				tokenString = strings.TrimSpace(cookie.Value)
			}
			if tokenString != "" {
				if authService == nil {
					http.Error(w, "session revocation unavailable", http.StatusServiceUnavailable)
					return
				}
				revoked, err := authService.IsSessionRevokedContext(r.Context(), tokenString)
				if err != nil {
					http.Error(w, "session revocation unavailable", http.StatusServiceUnavailable)
					return
				}
				if revoked {
					http.Error(w, "session revoked", http.StatusUnauthorized)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
