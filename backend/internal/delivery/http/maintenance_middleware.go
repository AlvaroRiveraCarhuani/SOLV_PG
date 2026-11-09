package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// MaintenanceMiddleware intercepta peticiones no-admin cuando el tenant está en modo mantenimiento (ADR-031)
func MaintenanceMiddleware(tenantRepo domain.TenantRepository, authService *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Bypass para peticiones OPTIONS (CORS) y rutas administrativas /api/v1/admin/
			if r.Method == http.MethodOptions || strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
				next.ServeHTTP(w, r)
				return
			}

			// 2. Resolver identidad desde la sesión. Este middleware corre antes
			// del tenant middleware de ruta, así que no puede depender de headers
			// del cliente ni de un contexto que todavía no existe.
			userRole := requestRole(r)
			tenantID := requestTenantID(r)
			if authService != nil {
				if token := sessionTokenFromRequest(r); token != "" {
					if claims, err := authService.ValidateSessionTokenContext(r.Context(), token); err == nil {
						if role, ok := claims["role"].(string); ok {
							userRole = strings.TrimSpace(role)
						}
						if claimTenant, ok := claims["tenant_id"].(string); ok && strings.TrimSpace(claimTenant) != "" {
							tenantID = strings.TrimSpace(claimTenant)
						}
					}
				}
			}
			if userRole == "admin" {
				next.ServeHTTP(w, r)
				return
			}

			if tenantID == "" {
				tenantID = domain.DefaultTenantID
			}

			// 4. Lazy auto-off: una vigencia vencida se persiste como off en la
			// primera lectura (sin sweeper). El evento MAINTENANCE_AUTO_DISABLED
			// se audita en GetMaintenanceStatus, que sí tiene auditLogRepo.
			_, _ = services.NewMaintenanceService(tenantRepo).ClearExpiredMaintenance(r.Context(), tenantID)

			// 5. Consultar estado de mantenimiento
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
