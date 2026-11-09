package httpdelivery

import (
	"crypto/subtle"
	"net"
	"net/http"
	"os"
	"strings"
)

// metricsAccessMiddleware limita /metrics al scraper interno. Sin
// METRICS_TOKEN solo acepta conexiones loopback; con token exige el Bearer
// configurado. Así el endpoint no queda expuesto por el bind actual.
func metricsAccessMiddleware(next http.Handler) http.Handler {
	token := strings.TrimSpace(os.Getenv("METRICS_TOKEN"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			provided := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
		if err != nil {
			host = strings.TrimSpace(r.RemoteAddr)
		}
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
