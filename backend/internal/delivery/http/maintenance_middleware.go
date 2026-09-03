package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

// MaintenanceMiddleware intercepta peticiones no-admin cuando el tenant está en modo mantenimiento (ADR-031)
func MaintenanceMiddleware(tenantRepo domain.TenantRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Bypass para peticiones OPTIONS (CORS) y rutas administrativas /api/v1/admin/
			if r.Method == http.MethodOptions || strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
				next.ServeHTTP(w, r)
				return
			}

			// 2. Bypass si el usuario tiene rol de administrador
			userRole := r.Header.Get("X-User-Role")
			if userRole == "admin" {
				next.ServeHTTP(w, r)
				return
			}

			// 3. Obtener tenantID del contexto
			tenantID, _ := r.Context().Value(domain.TenantIDKey).(string)
			if tenantID == "" {
				tenantID = "00000000-0000-0000-0000-000000000001"
			}

			// 4. Consultar estado de mantenimiento
			status, err := tenantRepo.GetMaintenance(r.Context(), tenantID)
			if err == nil && status != nil && status.MaintenanceMode {
				// Verificar si la fecha límite sigue vigente
				if status.MaintenanceUntil == nil || time.Now().Before(*status.MaintenanceUntil) {
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusServiceUnavailable)

					resp := map[string]interface{}{
						"error":   "maintenance_mode",
						"message": "Plataforma en mantenimiento",
					}
					if status.MaintenanceUntil != nil {
						resp["until"] = status.MaintenanceUntil.Format(time.RFC3339)
					}
					if status.MaintenanceReason != "" {
						resp["reason"] = status.MaintenanceReason
					}

					_ = json.NewEncoder(w).Encode(resp)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
