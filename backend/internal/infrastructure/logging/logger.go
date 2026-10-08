package logging

import (
	"context"
	"log/slog"
	"os"

	"github.com/google/uuid"
)

type contextKey string

const (
	TraceIDKey contextKey = "trace_id"
)

var defaultLogger *slog.Logger

func init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// SetDefault sets a custom global logger.
func SetDefault(l *slog.Logger) {
	defaultLogger = l
	slog.SetDefault(l)
}

// WithTraceID returns a context with the given trace_id.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID extracts trace_id from context or returns an empty string.
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(TraceIDKey).(string); ok {
		return val
	}
	return ""
}

// GenerateTraceID generates a unique UUID v4 string for tracing.
func GenerateTraceID() string {
	return uuid.New().String()
}

// FromContext returns a slog.Logger enriched with the trace_id from the context if present.
func FromContext(ctx context.Context) *slog.Logger {
	traceID := GetTraceID(ctx)
	if traceID != "" {
		return defaultLogger.With(slog.String("trace_id", traceID))
	}
	return defaultLogger
}

// Info logs an informational message using context's trace_id if available.
func Info(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Info(msg, args...)
}

// Error logs an error message using context's trace_id if available.
func Error(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Error(msg, args...)
}

// Warn logs a warning message using context's trace_id if available.
func Warn(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Warn(msg, args...)
}

// Debug logs a debug message using context's trace_id if available.
func Debug(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Debug(msg, args...)
}
