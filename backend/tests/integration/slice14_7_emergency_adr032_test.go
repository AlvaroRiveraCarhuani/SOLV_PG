package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

// setupSlice147EmergencyServer arma el servidor de emergencias del submódulo
// 14.7 con el repositorio de auditoría y executors deterministas inyectados
// (docker-prune y reset-pools no dependen del daemon real en tests).
func setupSlice147EmergencyServer(t *testing.T) (*httptest.Server, *database.Database) {
	db, err := database.NewPostgresDB(getTestDSN())
	if err != nil {
		t.Skipf("Skipping integration test: PostgreSQL DB connection failed: %v", err)
		return nil, nil
	}

	tenantRepo := postgres.NewPostgresTenantRepository(db.GetDB())
	academicPeriodRepo := postgres.NewPostgresAcademicPeriodRepository(db.GetDB())
	subjectRepo := postgres.NewPostgresSubjectRepository(db.GetDB())
	govRepo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())
	auditRepo := postgres.NewAuditLogRepository(db.GetDB())

	academicPeriodService := services.NewAcademicPeriodService(academicPeriodRepo)
	maintenanceService := services.NewMaintenanceService(tenantRepo)
	govService := services.NewAdminGovernanceService(subjectRepo, govRepo)
	govService.SetAuditRepo(auditRepo)
	govService.SetDockerPruner(func(ctx context.Context) (int64, int64, error) {
		return 3, 120, nil
	})
	govService.SetPoolResetter(func(ctx context.Context) (int64, error) {
		return 4, nil
	})

	adminAcademicHandler := httpdelivery.NewAdminAcademicHandler(academicPeriodService, maintenanceService, govService)

	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := r.Header.Get("X-Tenant-Id")
			if tenantID == "" {
				tenantID = domain.DefaultTenantID
			}
			ctx := context.WithValue(r.Context(), domain.TenantIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	handlers := httpdelivery.Handlers{
		AdminAcademicHandler: adminAcademicHandler,
		TenantMiddleware:     tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	server := httptest.NewServer(mux)
	return server, db
}

// TestSlice147_EmergencyActionsADR032 verifica el catálogo completo del
// ADR-032 en el submódulo 14.7:
//  1. Las 5 acciones (nombres reales del sistema) exigen su frase exacta y
//     responden 200.
//  2. Toda ejecución registra el evento EMERGENCY_* correspondiente en
//     audit_logs con el motivo y el impacto.
//  3. Una frase inválida se rechaza con 422 y NO registra auditoría.
func TestSlice147_EmergencyActionsADR032(t *testing.T) {
	server, db := setupSlice147EmergencyServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	ctx := context.Background()
	client := &http.Client{}
	auditRepo := postgres.NewAuditLogRepository(db.GetDB())

	cases := []struct {
		action     string
		phrase     string
		auditEvent string
		affected   int64 // -1 = no determinista (depende de la BD)
	}{
		{"terminate_all_workspaces", "TERMINAR TODOS LOS WORKSPACES", "EMERGENCY_TERMINATE_ALL", -1},
		{"hibernate_all_workspaces", "HIBERNAR TODOS LOS WORKSPACES", "EMERGENCY_HIBERNATE_ALL", -1},
		{"kill_zombies", "LIMPIAR ZOMBIES DOCKER", "EMERGENCY_KILL_ZOMBIES", -1},
		{"docker_prune", "PURGAR CAPAS HUERFANAS", "EMERGENCY_PRUNE_DOCKER", 3},
		{"reset_pools", "REINICIAR POOLS", "EMERGENCY_RESET_POOLS", 4},
	}

	const reason = "Verificacion de integracion del submodulo 14.7"

	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			payload := fmt.Sprintf(`{"confirmation_phrase": %q, "reason": %q}`, tc.phrase, reason)
			req, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/emergency/"+tc.action, bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-Role", "admin")
			req.Header.Set("X-Tenant-Id", tenantID)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to call %s: %v", tc.action, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("Expected 200 OK for %s with exact phrase, got %d", tc.action, resp.StatusCode)
			}

			events, err := auditRepo.ListFiltered(ctx, tenantID, "", tc.auditEvent, 10, 0)
			if err != nil {
				t.Fatalf("Failed to read audit events for %s: %v", tc.auditEvent, err)
			}
			if len(events) == 0 {
				t.Fatalf("Expected at least 1 %s event in audit_logs", tc.auditEvent)
			}

			latest := events[0]
			if latest.ActorEmail == "" {
				t.Errorf("ActorEmail must never be empty (resolved email or actor_id fallback)")
			}

			var meta map[string]any
			if err := json.Unmarshal(latest.Metadata, &meta); err != nil {
				t.Fatalf("Failed to unmarshal audit metadata: %v", err)
			}
			if meta["reason"] != reason {
				t.Errorf("Expected audit reason %q, got %v", reason, meta["reason"])
			}
			if tc.affected >= 0 {
				if af, ok := meta["affected_count"].(float64); !ok || int64(af) != tc.affected {
					t.Errorf("Expected affected_count=%d for %s, got %v", tc.affected, tc.action, meta["affected_count"])
				}
			}
		})
	}

	// Frase inválida en acción conocida: 422 y SIN registro de auditoría.
	pruneEventsBefore, _ := auditRepo.ListFiltered(ctx, tenantID, "", "EMERGENCY_PRUNE_DOCKER", 10, 0)

	badPayload := `{"confirmation_phrase": "por favor borrar todo", "reason": "prueba de frase invalida"}`
	reqBad, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/emergency/docker_prune", bytes.NewBufferString(badPayload))
	reqBad.Header.Set("Content-Type", "application/json")
	reqBad.Header.Set("X-User-Role", "admin")
	reqBad.Header.Set("X-Tenant-Id", tenantID)

	respBad, err := client.Do(reqBad)
	if err != nil {
		t.Fatalf("Failed to call docker_prune with invalid phrase: %v", err)
	}
	defer respBad.Body.Close()
	if respBad.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("Expected 422 for invalid confirmation phrase, got %d", respBad.StatusCode)
	}

	pruneEventsAfter, _ := auditRepo.ListFiltered(ctx, tenantID, "", "EMERGENCY_PRUNE_DOCKER", 10, 0)
	if len(pruneEventsAfter) != len(pruneEventsBefore) {
		t.Errorf("Rejected execution must not write audit events: before=%d after=%d", len(pruneEventsBefore), len(pruneEventsAfter))
	}

	// Acción desconocida: 422.
	unknownPayload := `{"confirmation_phrase": "ACCION_INEXISTENTE"}`
	reqUnknown, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/emergency/reboot_datacenter", bytes.NewBufferString(unknownPayload))
	reqUnknown.Header.Set("Content-Type", "application/json")
	reqUnknown.Header.Set("X-User-Role", "admin")
	reqUnknown.Header.Set("X-Tenant-Id", tenantID)

	respUnknown, err := client.Do(reqUnknown)
	if err != nil {
		t.Fatalf("Failed to call unknown action: %v", err)
	}
	defer respUnknown.Body.Close()
	if respUnknown.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("Expected 422 for unknown emergency action, got %d", respUnknown.StatusCode)
	}
}
