package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func setupSlice14EmergencyServer(t *testing.T) (*httptest.Server, *database.Database) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable"
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
		return nil, nil
	}

	_ = db.RunInitialMigrations()

	tenantRepo := postgres.NewPostgresTenantRepository(db.GetDB())
	academicPeriodRepo := postgres.NewPostgresAcademicPeriodRepository(db.GetDB())
	subjectRepo := postgres.NewPostgresSubjectRepository(db.GetDB())
	govRepo := postgres.NewPostgresAdminGovernanceRepository(db.GetDB())

	academicPeriodService := services.NewAcademicPeriodService(academicPeriodRepo)
	maintenanceService := services.NewMaintenanceService(tenantRepo)
	govService := services.NewAdminGovernanceService(subjectRepo, govRepo)

	adminAcademicHandler := httpdelivery.NewAdminAcademicHandler(academicPeriodService, maintenanceService, govService)

	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := r.Header.Get("X-Tenant-Id")
			if tenantID == "" {
				tenantID = "00000000-0000-0000-0000-000000000001"
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

func TestSlice14_EmergencyActions(t *testing.T) {
	server, db := setupSlice14EmergencyServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	adminID := uuid.NewString()
	client := &http.Client{}

	// Seed Admin
	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Admin', 'Seguridad', $2, 'admin', $3)
		ON CONFLICT (id) DO NOTHING;
	`, adminID, fmt.Sprintf("secadmin_%s@uab.edu.bo", adminID[:6]), tenantID)

	// Seed Alumno y Materia para Workspaces de Prueba
	stuID := uuid.NewString()
	subID := uuid.NewString()
	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Estudiante', 'Pruebas', $2, 'student', $3)
		ON CONFLICT (id) DO NOTHING;
	`, stuID, fmt.Sprintf("test_stu_%s@uab.edu.bo", stuID[:6]), tenantID)

	_, _ = db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code)
		VALUES ($1, $2, 'Redes y Seguridad', 'RED-401')
		ON CONFLICT (id) DO NOTHING;
	`, subID, tenantID)

	// =========================================================================
	// 1. TEST Validaciones de Frase y Seguridad (ADR-032)
	// =========================================================================
	t.Run("1. Validaciones - Frase Errónea, Acción Inexistente y Control de Rol", func(t *testing.T) {
		// 1.1 Frase de confirmación errónea -> 422 Unprocessable Entity
		payloadWrong := []byte(`{"confirmation_phrase": "terminar todo por favor", "reason": "emergencia"}`)
		reqWrong, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/emergency/terminate_all_workspaces", bytes.NewBuffer(payloadWrong))
		reqWrong.Header.Set("Content-Type", "application/json")
		reqWrong.Header.Set("X-User-Role", "admin")
		reqWrong.Header.Set("X-Tenant-Id", tenantID)

		respWrong, _ := client.Do(reqWrong)
		if respWrong.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422 for incorrect confirmation phrase, got %d", respWrong.StatusCode)
		}

		// 1.2 Acción desconocida -> 422 Unprocessable Entity
		payloadUnknown := []byte(`{"confirmation_phrase": "ACCION_INEXISTENTE"}`)
		reqUnknown, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/emergency/reboot_datacenter", bytes.NewBuffer(payloadUnknown))
		reqUnknown.Header.Set("Content-Type", "application/json")
		reqUnknown.Header.Set("X-User-Role", "admin")
		reqUnknown.Header.Set("X-Tenant-Id", tenantID)

		respUnknown, _ := client.Do(reqUnknown)
		if respUnknown.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422 for unknown emergency action, got %d", respUnknown.StatusCode)
		}

		// 1.3 Seguridad: Estudiante o docente intentando acción de pánico -> 403 Forbidden
		payloadValid := []byte(`{"confirmation_phrase": "TERMINAR TODOS LOS WORKSPACES"}`)
		reqStudent, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/emergency/terminate_all_workspaces", bytes.NewBuffer(payloadValid))
		reqStudent.Header.Set("Content-Type", "application/json")
		reqStudent.Header.Set("X-User-Role", "student")
		reqStudent.Header.Set("X-Tenant-Id", tenantID)

		respStudent, _ := client.Do(reqStudent)
		if respStudent.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for student, got %d", respStudent.StatusCode)
		}
	})

	// =========================================================================
	// 2. TEST terminate_all_workspaces (ADR-032)
	// =========================================================================
	t.Run("2. Terminar Todos los Workspaces - Apagado Masivo Inmediato", func(t *testing.T) {
		// Sembrar 2 workspaces activos ('running' y 'pending')
		ws1 := uuid.NewString()
		ws2 := uuid.NewString()
		_, _ = db.GetDB().Exec(`
			INSERT INTO workspaces (id, student_id, subject_id, type, status, access_url, memory_limit_mb, tenant_id)
			VALUES 
				($1, $3, $4, 'IDE_PERSISTENTE', 'running', 'http://ws1.local', 512, $5),
				($2, $3, $4, 'IDE_PERSISTENTE', 'pending', 'http://ws2.local', 512, $5)
			ON CONFLICT (id) DO NOTHING;
		`, ws1, ws2, stuID, subID, tenantID)

		payloadTerm := []byte(`{"confirmation_phrase": "TERMINAR TODOS LOS WORKSPACES", "reason": "Saturación crítica de memoria en host"}`)
		reqTerm, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/emergency/terminate_all_workspaces", bytes.NewBuffer(payloadTerm))
		reqTerm.Header.Set("Content-Type", "application/json")
		reqTerm.Header.Set("X-User-Role", "admin")
		reqTerm.Header.Set("X-User-Id", adminID)
		reqTerm.Header.Set("X-Tenant-Id", tenantID)

		respTerm, err := client.Do(reqTerm)
		if err != nil {
			t.Fatalf("Failed terminate all request: %v", err)
		}
		if respTerm.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK executing terminate_all_workspaces, got %d", respTerm.StatusCode)
		}

		var bodyTerm map[string]interface{}
		json.NewDecoder(respTerm.Body).Decode(&bodyTerm)
		data := bodyTerm["data"].(map[string]interface{})
		if int(data["affected_count"].(float64)) < 2 {
			t.Errorf("Expected at least 2 affected workspaces, got %v", data["affected_count"])
		}

		// Verificar en BD que los workspaces pasaron a 'failed'
		var status1, status2 string
		_ = db.GetDB().QueryRow("SELECT status FROM workspaces WHERE id = $1", ws1).Scan(&status1)
		_ = db.GetDB().QueryRow("SELECT status FROM workspaces WHERE id = $1", ws2).Scan(&status2)
		if status1 != "failed" || status2 != "failed" {
			t.Errorf("Expected workspaces to be terminated/failed, got ws1=%s, ws2=%s", status1, status2)
		}
	})

	// =========================================================================
	// 3. TEST hibernate_all_workspaces & kill_zombies (ADR-032)
	// =========================================================================
	t.Run("3. Hibernar Workspaces y Limpieza de Zombies Docker", func(t *testing.T) {
		// Sembrar workspace 'running'
		ws3 := uuid.NewString()
		_, _ = db.GetDB().Exec(`
			INSERT INTO workspaces (id, student_id, subject_id, type, status, access_url, memory_limit_mb, tenant_id)
			VALUES ($1, $2, $3, 'IDE_PERSISTENTE', 'running', 'http://ws3.local', 512, $4)
			ON CONFLICT (id) DO NOTHING;
		`, ws3, stuID, subID, tenantID)

		// 3.1 Hibernar todos
		payloadHib := []byte(`{"confirmation_phrase": "HIBERNAR TODOS LOS WORKSPACES"}`)
		reqHib, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/emergency/hibernate_all_workspaces", bytes.NewBuffer(payloadHib))
		reqHib.Header.Set("Content-Type", "application/json")
		reqHib.Header.Set("X-User-Role", "admin")
		reqHib.Header.Set("X-Tenant-Id", tenantID)

		respHib, _ := client.Do(reqHib)
		if respHib.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK executing hibernate_all_workspaces, got %d", respHib.StatusCode)
		}

		var status3 string
		_ = db.GetDB().QueryRow("SELECT status FROM workspaces WHERE id = $1", ws3).Scan(&status3)
		if status3 != "hibernated" {
			t.Errorf("Expected ws3 status to be hibernated, got %s", status3)
		}

		// 3.2 Limpieza de Zombies
		payloadZombies := []byte(`{"confirmation_phrase": "LIMPIAR ZOMBIES DOCKER"}`)
		reqZombies, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/emergency/kill_zombies", bytes.NewBuffer(payloadZombies))
		reqZombies.Header.Set("Content-Type", "application/json")
		reqZombies.Header.Set("X-User-Role", "admin")
		reqZombies.Header.Set("X-Tenant-Id", tenantID)

		respZombies, _ := client.Do(reqZombies)
		if respZombies.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK executing kill_zombies, got %d", respZombies.StatusCode)
		}
	})
}
