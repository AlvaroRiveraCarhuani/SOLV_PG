package middleware

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
)

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCapturingResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusCapturingResponseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func AuditMiddleware(pool *AuditWorkerPool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Solo auditar métodos de mutación (POST, PUT, DELETE)
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
				next.ServeHTTP(w, r)
				return
			}

			// Opcional: clonar el body para metadata si existe
			var bodyBytes []byte
			if r.Body != nil {
				bodyBytes, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			rec := &statusCapturingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rec, r)

			// Verificar rol de usuario
			role := domain.GetUserRole(r.Context())
			if role != "teacher" && role != "admin" {
				return
			}

			actorID := domain.GetUserID(r.Context())
			tenantID := domain.GetTenantID(r.Context())
			if tenantID == "" {
				tenantID = domain.DefaultTenantID
			}

			resourceType := extractResourceType(r.URL.Path)
			ipAddress := extractIP(r)

			logEntry := &domain.AuditLog{
				TenantID:     tenantID,
				ActorID:      actorID,
				Action:       r.Method + " " + r.URL.Path,
				ResourceType: resourceType,
				StatusCode:   rec.statusCode,
				Metadata:     bodyBytes,
				IPAddress:    ipAddress,
				UserAgent:    r.Header.Get("User-Agent"),
			}

			pool.Enqueue(logEntry)
		})
	}
}

func extractResourceType(path string) string {
	cleanPath := strings.TrimPrefix(path, "/api/v1/")
	parts := strings.Split(cleanPath, "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "resource"
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
