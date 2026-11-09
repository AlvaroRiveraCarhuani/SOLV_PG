package middleware

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"solv-backend/internal/infrastructure/logging"
	"solv-backend/internal/infrastructure/metrics"
)

type responseWriterInterceptor struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *responseWriterInterceptor) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterInterceptor) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

// Hijack delegates to the underlying ResponseWriter when it implements
// http.Hijacker. WebSocket upgrades (gorilla/websocket) require this.
func (w *responseWriterInterceptor) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("responseWriterInterceptor: underlying ResponseWriter of type %T does not implement http.Hijacker", w.ResponseWriter)
	}
	return hijacker.Hijack()
}

// Flush delegates to the underlying ResponseWriter when it implements
// http.Flusher; otherwise it is a no-op so streaming handlers never panic.
func (w *responseWriterInterceptor) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap exposes the wrapped writer to http.ResponseController and helpers.
func (w *responseWriterInterceptor) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// ObservabilityMiddleware handles trace_id injection, Prometheus HTTP metrics, and structured request logging.
func ObservabilityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = r.Header.Get("X-Request-ID")
		}
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Inject trace_id into context and response header
		ctx := logging.WithTraceID(r.Context(), traceID)
		w.Header().Set("X-Trace-ID", traceID)

		interceptor := &responseWriterInterceptor{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(interceptor, r.WithContext(ctx))

		duration := time.Since(start)
		statusStr := strconv.Itoa(interceptor.statusCode)
		path := r.URL.Path

		// Record Prometheus Metrics
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
		metrics.HTTPRequestDurationSeconds.WithLabelValues(r.Method, path).Observe(duration.Seconds())

		// Structured JSON Logging via slog
		logging.FromContext(ctx).Info("HTTP request processed",
			slog.String("method", r.Method),
			slog.String("path", path),
			slog.Int("status", interceptor.statusCode),
			slog.Float64("duration_ms", float64(duration.Microseconds())/1000.0),
			slog.String("remote_addr", r.RemoteAddr),
		)
	})
}
