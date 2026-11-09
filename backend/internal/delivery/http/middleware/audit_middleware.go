package middleware

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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

// Hijack delegates to the underlying ResponseWriter when it implements
// http.Hijacker. WebSocket upgrades (gorilla/websocket) require this.
func (w *statusCapturingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("statusCapturingResponseWriter: underlying ResponseWriter of type %T does not implement http.Hijacker", w.ResponseWriter)
	}
	return hijacker.Hijack()
}

// Flush delegates to the underlying ResponseWriter when it implements
// http.Flusher; otherwise it is a no-op so streaming handlers never panic.
func (w *statusCapturingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap exposes the wrapped writer to http.ResponseController and helpers.
func (w *statusCapturingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
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
				Metadata:     sanitizeAuditBody(bodyBytes),
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

// sanitizeAuditBody elimina secretos del cuerpo antes de persistirlo. Los
// cuerpos que no son JSON se descartan: no se puede garantizar una redacción
// segura sobre texto libre y el log conserva igualmente acción, actor y estado.
func sanitizeAuditBody(body []byte) []byte {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil
	}

	var payload any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil
	}

	redacted := redactAuditValue(payload)
	encoded, err := json.Marshal(redacted)
	if err != nil {
		return nil
	}
	return encoded
}

func redactAuditValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			if isSensitiveAuditKey(key) {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = redactAuditValue(nested)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, nested := range typed {
			out[i] = redactAuditValue(nested)
		}
		return out
	default:
		return value
	}
}

func isSensitiveAuditKey(key string) bool {
	compact := strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	if compact == "" {
		return false
	}
	switch compact {
	case "apikey", "encryptedapikey", "accesstoken", "refreshtoken", "idtoken", "bearertoken", "token", "password", "passwd", "secret", "clientsecret", "authorization":
		return true
	}
	return strings.Contains(compact, "apikey") ||
		strings.Contains(compact, "secret") ||
		strings.HasSuffix(compact, "password") ||
		strings.HasSuffix(compact, "token")
}
