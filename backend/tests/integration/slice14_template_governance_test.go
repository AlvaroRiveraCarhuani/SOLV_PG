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
	"time"

	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func setupSlice14TemplateGovServer(t *testing.T) (*httptest.Server, *database.Database) {
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

func TestSlice14_DockerTemplateGovernance(t *testing.T) {
	server, db := setupSlice14TemplateGovServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	adminID := uuid.NewString()
	teacherID := uuid.NewString()
	client := &http.Client{}

	// 1. Seed Usuarios
	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES 
			($1, 'Admin', 'Root', $3, 'admin', $5),
			($2, 'Profesor', 'Docker', $4, 'teacher', $5)
		ON CONFLICT (id) DO NOTHING;
	`, adminID, teacherID,
		fmt.Sprintf("admin_%s@uab.edu.bo", adminID[:6]),
		fmt.Sprintf("prof_%s@uab.edu.bo", teacherID[:6]),
		tenantID)

	// 2. Seed Plantillas: 1 aprobada (sistema) y 1 solicitada por docente (pendiente)
	tplApprovedID := uuid.NewString()
	tplPendingID := uuid.NewString()
	tplToRejectID := uuid.NewString()

	_, _ = db.GetDB().Exec(`
		INSERT INTO lab_templates (id, name, docker_image, base_ram_mb, status, tenant_id)
		VALUES ($1, 'Ubuntu Base C++', 'solv/cpp:22.04', 512, 'approved', $2)
		ON CONFLICT (id) DO NOTHING;
	`, tplApprovedID, tenantID)

	_, _ = db.GetDB().Exec(`
		INSERT INTO lab_templates (id, name, docker_image, base_ram_mb, status, requested_by, description, tenant_id)
		VALUES ($1, 'Rust Async Lab', 'solv/rust:1.75', 512, 'pending', $2, 'Plantilla para laboratorio concurrente', $3)
		ON CONFLICT (id) DO NOTHING;
	`, tplPendingID, teacherID, tenantID)

	_, _ = db.GetDB().Exec(`
		INSERT INTO lab_templates (id, name, docker_image, base_ram_mb, status, requested_by, description, tenant_id)
		VALUES ($1, 'Insecure Custom Lab', 'custom/unsafe:latest', 256, 'pending', $2, 'Plantilla con root no saneada', $3)
		ON CONFLICT (id) DO NOTHING;
	`, tplToRejectID, teacherID, tenantID)

	// =========================================================================
	// 1. TEST Listado de Plantillas y Filtros (GET /admin/templates)
	// =========================================================================
	t.Run("1. Listado de Plantillas - Consulta Completa, Filtros por Estado y Seguridad", func(t *testing.T) {
		// 1.1 Listar todas las plantillas
		reqAll, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/templates", nil)
		reqAll.Header.Set("X-User-Role", "admin")
		reqAll.Header.Set("X-Tenant-Id", tenantID)

		respAll, err := client.Do(reqAll)
		if err != nil {
			t.Fatalf("Failed list all templates request: %v", err)
		}
		if respAll.StatusCode != http.StatusOK {
			var errBody map[string]interface{}
			json.NewDecoder(respAll.Body).Decode(&errBody)
			t.Fatalf("Expected 200 OK listing templates, got %d. Body: %v", respAll.StatusCode, errBody)
		}

		var bodyAll map[string]interface{}
		json.NewDecoder(respAll.Body).Decode(&bodyAll)
		items := bodyAll["data"].([]interface{})
		if len(items) < 3 {
			t.Fatalf("Expected at least 3 templates, got %d", len(items))
		}

		// 1.2 Filtro status=pending
		reqPending, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/templates?status=pending", nil)
		reqPending.Header.Set("X-User-Role", "admin")
		reqPending.Header.Set("X-Tenant-Id", tenantID)

		respPending, _ := client.Do(reqPending)
		var bodyPending map[string]interface{}
		json.NewDecoder(respPending.Body).Decode(&bodyPending)
		pendingItems := bodyPending["data"].([]interface{})
		for _, item := range pendingItems {
			pMap := item.(map[string]interface{})
			if pMap["status"] != "pending" {
				t.Errorf("Expected only pending templates, got status=%v", pMap["status"])
			}
		}

		// 1.3 Seguridad: Estudiante no puede consultar endpoint administrativo
		reqStudent, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/templates", nil)
		reqStudent.Header.Set("X-User-Role", "student")
		respStudent, _ := client.Do(reqStudent)
		if respStudent.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for student, got %d", respStudent.StatusCode)
		}
	})

	// =========================================================================
	// 2. TEST Aprobación de Plantilla (PUT /admin/templates/{id}/review)
	// =========================================================================
	t.Run("2. Aprobación de Plantilla - Actualización de Estado, RAM y Auditoría", func(t *testing.T) {
		newRam := 1024
		reviewDTO := domain.ReviewTemplateDTO{
			Status:    "approved",
			BaseRamMB: &newRam,
		}
		reviewBytes, _ := json.Marshal(reviewDTO)

		reqApprove, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/admin/templates/%s/review", server.URL, tplPendingID), bytes.NewBuffer(reviewBytes))
		reqApprove.Header.Set("Content-Type", "application/json")
		reqApprove.Header.Set("X-User-Role", "admin")
		reqApprove.Header.Set("X-User-Id", adminID)
		reqApprove.Header.Set("X-Tenant-Id", tenantID)

		respApprove, err := client.Do(reqApprove)
		if err != nil {
			t.Fatalf("Failed approve request: %v", err)
		}
		if respApprove.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK approving template, got %d", respApprove.StatusCode)
		}

		var approveBody map[string]interface{}
		json.NewDecoder(respApprove.Body).Decode(&approveBody)
		data := approveBody["data"].(map[string]interface{})
		if data["status"] != "approved" {
			t.Errorf("Expected status = approved, got %v", data["status"])
		}
		if int(data["base_ram_mb"].(float64)) != 1024 {
			t.Errorf("Expected base_ram_mb = 1024, got %v", data["base_ram_mb"])
		}

		// Verificar en BD
		var statusInDB string
		var ramInDB int
		var reviewedByInDB string
		var reviewedAtInDB *time.Time
		_ = db.GetDB().QueryRow("SELECT status, base_ram_mb, reviewed_by, reviewed_at FROM lab_templates WHERE id = $1", tplPendingID).
			Scan(&statusInDB, &ramInDB, &reviewedByInDB, &reviewedAtInDB)

		if statusInDB != "approved" {
			t.Errorf("Expected status in DB to be approved, got %s", statusInDB)
		}
		if ramInDB != 1024 {
			t.Errorf("Expected base_ram_mb in DB to be 1024, got %d", ramInDB)
		}
		if reviewedByInDB != adminID {
			t.Errorf("Expected reviewed_by in DB to be %s, got %s", adminID, reviewedByInDB)
		}
		if reviewedAtInDB == nil {
			t.Errorf("Expected reviewed_at in DB to be non-null")
		}
	})

	// =========================================================================
	// 3. TEST Rechazo de Plantilla y Validaciones de Rechazo (ADR-030)
	// =========================================================================
	t.Run("3. Rechazo de Plantilla - Validación de Motivo Obligatorio y Persistencia", func(t *testing.T) {
		// 3.1 Status inválido -> 422
		reqInvalid, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/admin/templates/%s/review", server.URL, tplToRejectID), bytes.NewBuffer([]byte(`{"status": "unknown"}`)))
		reqInvalid.Header.Set("Content-Type", "application/json")
		reqInvalid.Header.Set("X-User-Role", "admin")
		reqInvalid.Header.Set("X-Tenant-Id", tenantID)

		respInvalid, _ := client.Do(reqInvalid)
		if respInvalid.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422 for invalid status, got %d", respInvalid.StatusCode)
		}

		// 3.2 Rechazar sin motivo -> 422 Unprocessable Entity
		reqNoReason, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/admin/templates/%s/review", server.URL, tplToRejectID), bytes.NewBuffer([]byte(`{"status": "rejected", "rejection_reason": ""}`)))
		reqNoReason.Header.Set("Content-Type", "application/json")
		reqNoReason.Header.Set("X-User-Role", "admin")
		reqNoReason.Header.Set("X-Tenant-Id", tenantID)

		respNoReason, _ := client.Do(reqNoReason)
		if respNoReason.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422 for rejection without reason, got %d", respNoReason.StatusCode)
		}

		// 3.3 Rechazar con motivo válido -> 200 OK
		rejectionPayload := []byte(`{"status": "rejected", "rejection_reason": "La imagen Docker no cumple con las políticas de seguridad institucional"}`)
		reqReject, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/admin/templates/%s/review", server.URL, tplToRejectID), bytes.NewBuffer(rejectionPayload))
		reqReject.Header.Set("Content-Type", "application/json")
		reqReject.Header.Set("X-User-Role", "admin")
		reqReject.Header.Set("X-User-Id", adminID)
		reqReject.Header.Set("X-Tenant-Id", tenantID)

		respReject, err := client.Do(reqReject)
		if err != nil {
			t.Fatalf("Failed rejection request: %v", err)
		}
		if respReject.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK rejecting template with reason, got %d", respReject.StatusCode)
		}

		// Verificar en base de datos
		var statusAfter string
		var reasonAfter string
		_ = db.GetDB().QueryRow("SELECT status, rejection_reason FROM lab_templates WHERE id = $1", tplToRejectID).Scan(&statusAfter, &reasonAfter)
		if statusAfter != "rejected" {
			t.Errorf("Expected status in DB = rejected, got %s", statusAfter)
		}
		if reasonAfter != "La imagen Docker no cumple con las políticas de seguridad institucional" {
			t.Errorf("Rejection reason not persisted correctly: %s", reasonAfter)
		}
	})
}
