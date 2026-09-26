package httpdelivery

import (
	"encoding/json"
	"net/http"

	"solv-backend/internal/core/domain"
)

type GlobalResponse struct {
	Data    any    `json:"data"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func SendJSON(w http.ResponseWriter, status int, data any, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(GlobalResponse{
		Data:    data,
		Error:   "",
		Message: msg,
	})
}

func SendError(w http.ResponseWriter, status int, err string, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(GlobalResponse{
		Data:    nil,
		Error:   err,
		Message: msg,
	})
}

// getTenantFromCtx resolves the tenant ID from the request context, the
// X-Tenant-Id header, or falls back to DefaultTenantID (single-node default).
// All handlers in this package must use this helper instead of duplicating the
// three-step resolution inline.
func getTenantFromCtx(r *http.Request) string {
	if tenantID, _ := r.Context().Value(domain.TenantIDKey).(string); tenantID != "" {
		return tenantID
	}
	if tenantID := r.Header.Get("X-Tenant-Id"); tenantID != "" {
		return tenantID
	}
	return domain.DefaultTenantID
}

// getUserIDFromCtx resolves the user ID from the X-User-Id header or falls
// back to DefaultTenantID as a sentinel for anonymous/system callers.
func getUserIDFromCtx(r *http.Request) string {
	if userID := r.Header.Get("X-User-Id"); userID != "" {
		return userID
	}
	return domain.DefaultTenantID
}
