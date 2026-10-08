package middleware

import (
	"net/http"
	"os"
	"strings"
)

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedEnv := os.Getenv("ALLOWED_ORIGINS")

		if origin != "" {
			allowed := false
			if allowedEnv != "" {
				allowedOrigins := strings.Split(allowedEnv, ",")
				for _, o := range allowedOrigins {
					if strings.TrimSpace(o) == origin {
						allowed = true
						break
					}
				}
			} else {
				if origin == "http://localhost:4200" || origin == "http://127.0.0.1:4200" || origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000" {
					allowed = true
				}
			}

			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-Id, X-User-Id, X-User-Role")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
