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

func setupSlice14MaintenancePeriodsServer(t *testing.T) (*httptest.Server, *database.Database, *services.AcademicPeriodService, *services.MaintenanceService) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://solv_user:solv_password@127.0.0.1:5432/solv_db?sslmode=disable"
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
		return nil, nil, nil, nil
	}

	_ = db.RunInitialMigrations()

	tenantRepo := postgres.NewPostgresTenantRepository(db.GetDB())
	academicPeriodRepo := postgres.NewPostgresAcademicPeriodRepository(db.GetDB())
	subjectRepo := postgres.NewPostgresSubjectRepository(db.GetDB())
	workspaceRepo := postgres.NewPostgresWorkspaceRepository(db.GetDB())
	exerciseRepo := postgres.NewPostgresExerciseRepository(db.GetDB())
	submissionRepo := postgres.NewPostgresSubmissionRepository(db.GetDB())

	academicPeriodService := services.NewAcademicPeriodService(academicPeriodRepo)
	maintenanceService := services.NewMaintenanceService(tenantRepo)

	adminAcademicHandler := httpdelivery.NewAdminAcademicHandler(academicPeriodService, maintenanceService, nil)
	studentHandler := httpdelivery.NewStudentHandler(subjectRepo, workspaceRepo, submissionRepo, exerciseRepo)

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

	maintenanceMiddleware := httpdelivery.MaintenanceMiddleware(tenantRepo)

	handlers := httpdelivery.Handlers{
		AdminAcademicHandler:  adminAcademicHandler,
		StudentHandler:        studentHandler,
		TenantMiddleware:      tenantMiddleware,
		MaintenanceMiddleware: maintenanceMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	// Envolver el multiplexor con tenantMiddleware y maintenanceMiddleware
	server := httptest.NewServer(tenantMiddleware(maintenanceMiddleware(mux)))
	return server, db, academicPeriodService, maintenanceService
}

func TestSlice14_MaintenancePeriods(t *testing.T) {
	server, db, academicPeriodService, maintenanceService := setupSlice14MaintenancePeriodsServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	tenantB := uuid.NewString()

	// Seed Tenant B
	_, _ = db.GetDB().Exec(`
		INSERT INTO tenants (id, name, slug) VALUES ($1, 'Tenant Secundario', 'tenant-sec')
		ON CONFLICT (id) DO NOTHING;
	`, tenantB)

	// Seed Student
	studentID := uuid.NewString()
	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Estudiante', 'Mantenimiento', $2, 'student', $3)
		ON CONFLICT (id) DO NOTHING;
	`, studentID, fmt.Sprintf("student_maint_%s@uab.edu.bo", studentID[:8]), tenantID)

	client := &http.Client{}

	// =========================================================================
	// 1. TEST Modo Mantenimiento Global (ADR-031)
	// =========================================================================
	t.Run("1. Modo Mantenimiento - Activación, Bloqueo 503 y Bypass Administrativo", func(t *testing.T) {
		// 1.1 Activar mantenimiento para las próximas 2 horas
		untilTime := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
		enablePayload := []byte(fmt.Sprintf(`{
			"until": "%s",
			"reason": "Actualización programada de base de datos"
		}`, untilTime))

		req, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/maintenance/enable", bytes.NewBuffer(enablePayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-Role", "admin")
		req.Header.Set("X-Tenant-Id", tenantID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to enable maintenance: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK enabling maintenance, got %d", resp.StatusCode)
		}

		// 1.2 Petición de estudiante a ruta no-admin -> 503 Service Unavailable
		reqStudent, _ := http.NewRequest("GET", server.URL+"/api/v1/student/dashboard", nil)
		reqStudent.Header.Set("X-User-Id", studentID)
		reqStudent.Header.Set("X-User-Role", "student")
		reqStudent.Header.Set("X-Tenant-Id", tenantID)

		respStudent, err := client.Do(reqStudent)
		if err != nil {
			t.Fatalf("Failed student request during maintenance: %v", err)
		}
		if respStudent.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected 503 Service Unavailable for student during maintenance, got %d", respStudent.StatusCode)
		}

		var maintResp map[string]interface{}
		json.NewDecoder(respStudent.Body).Decode(&maintResp)
		if maintResp["error"] != "maintenance_mode" {
			t.Errorf("Expected error 'maintenance_mode', got %v", maintResp["error"])
		}

		// 1.3 Bypass: Petición a ruta administrativa -> 200 OK
		reqAdminStatus, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/maintenance/status", nil)
		reqAdminStatus.Header.Set("X-User-Role", "admin")
		reqAdminStatus.Header.Set("X-Tenant-Id", tenantID)

		respAdminStatus, err := client.Do(reqAdminStatus)
		if err != nil {
			t.Fatalf("Failed admin status request: %v", err)
		}
		if respAdminStatus.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 OK for admin bypass during maintenance, got %d", respAdminStatus.StatusCode)
		}

		// 1.4 Bypass: Admin accediendo a ruta protegida no-admin -> Pasa el middleware
		reqAdminOnStudent, _ := http.NewRequest("GET", server.URL+"/api/v1/student/dashboard", nil)
		reqAdminOnStudent.Header.Set("X-User-Id", studentID)
		reqAdminOnStudent.Header.Set("X-User-Role", "admin")
		reqAdminOnStudent.Header.Set("X-Tenant-Id", tenantID)

		respAdminOnStudent, _ := client.Do(reqAdminOnStudent)
		if respAdminOnStudent.StatusCode == http.StatusServiceUnavailable {
			t.Errorf("Admin user role must bypass 503 maintenance mode!")
		}

		// 1.5 Desactivar mantenimiento
		reqDisable, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/maintenance/disable", nil)
		reqDisable.Header.Set("X-User-Role", "admin")
		reqDisable.Header.Set("X-Tenant-Id", tenantID)

		respDisable, _ := client.Do(reqDisable)
		if respDisable.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK disabling maintenance, got %d", respDisable.StatusCode)
		}

		// 1.6 Estudiante vuelve a acceder normalmente (no da 503)
		respStudentAfter, _ := client.Do(reqStudent)
		if respStudentAfter.StatusCode == http.StatusServiceUnavailable {
			t.Errorf("Student should not receive 503 after maintenance is disabled")
		}

		// 1.7 Mantenimiento con fecha expirada en el pasado -> No bloquea
		pastUntil := time.Now().Add(-1 * time.Hour)
		_ = maintenanceService.EnableMaintenance(context.Background(), tenantID, domain.EnableMaintenanceDTO{
			Until:  pastUntil.Format(time.RFC3339),
			Reason: "Mantenimiento viejo",
		})

		respExpired, _ := client.Do(reqStudent)
		if respExpired.StatusCode == http.StatusServiceUnavailable {
			t.Errorf("Expired maintenance until should not block requests!")
		}

		// Limpiar estado
		_ = maintenanceService.DisableMaintenance(context.Background(), tenantID)
	})

	// =========================================================================
	// 2. TEST Periodos Académicos (ADR-029)
	// =========================================================================
	t.Run("2. Periodos Académicos - CRUD, Validaciones de Fechas y Constraint de Materias", func(t *testing.T) {
		// 2.1 Validación 422: end_date < start_date
		invalidPayload := []byte(`{
			"name": "Semestre Inválido",
			"code": "INV-2026",
			"start_date": "2026-08-01",
			"end_date": "2026-02-01"
		}`)
		reqInvalid, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/academic-periods", bytes.NewBuffer(invalidPayload))
		reqInvalid.Header.Set("Content-Type", "application/json")
		reqInvalid.Header.Set("X-User-Role", "admin")
		reqInvalid.Header.Set("X-Tenant-Id", tenantID)

		respInvalid, _ := client.Do(reqInvalid)
		if respInvalid.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("Expected 422 Unprocessable Entity for invalid date range, got %d", respInvalid.StatusCode)
		}

		// 2.2 Crear Periodo Válido -> 201 Created
		validPayload := []byte(`{
			"name": "Primer Semestre 2026",
			"code": "I-2026",
			"start_date": "2026-02-01",
			"end_date": "2026-06-30",
			"is_active": true
		}`)
		reqCreate, _ := http.NewRequest("POST", server.URL+"/api/v1/admin/academic-periods", bytes.NewBuffer(validPayload))
		reqCreate.Header.Set("Content-Type", "application/json")
		reqCreate.Header.Set("X-User-Role", "admin")
		reqCreate.Header.Set("X-Tenant-Id", tenantID)

		respCreate, err := client.Do(reqCreate)
		if err != nil {
			t.Fatalf("Failed to create academic period: %v", err)
		}
		if respCreate.StatusCode != http.StatusCreated {
			t.Fatalf("Expected 201 Created for academic period, got %d", respCreate.StatusCode)
		}

		var createResp map[string]interface{}
		json.NewDecoder(respCreate.Body).Decode(&createResp)
		periodData := createResp["data"].(map[string]interface{})
		periodID := periodData["id"].(string)

		// 2.3 Listar periodos -> 200 OK
		reqList, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/academic-periods", nil)
		reqList.Header.Set("X-User-Role", "admin")
		reqList.Header.Set("X-Tenant-Id", tenantID)

		respList, _ := client.Do(reqList)
		if respList.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK listing periods, got %d", respList.StatusCode)
		}

		var listResp map[string]interface{}
		json.NewDecoder(respList.Body).Decode(&listResp)
		periods := listResp["data"].([]interface{})
		if len(periods) == 0 {
			t.Fatalf("Expected at least 1 academic period in list")
		}

		// 2.4 Aislamiento Cross-Tenant: Admin de Tenant B no ve el periodo de Tenant A
		reqListB, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/academic-periods", nil)
		reqListB.Header.Set("X-User-Role", "admin")
		reqListB.Header.Set("X-Tenant-Id", tenantB)

		respListB, _ := client.Do(reqListB)
		var listRespB map[string]interface{}
		json.NewDecoder(respListB.Body).Decode(&listRespB)
		periodsB := listRespB["data"].([]interface{})
		for _, p := range periodsB {
			pMap := p.(map[string]interface{})
			if pMap["id"] == periodID {
				t.Fatalf("Cross-tenant violation: Tenant B accessed academic period of Tenant A!")
			}
		}

		// 2.5 Editar periodo -> 200 OK
		updatePayload := []byte(`{
			"name": "Primer Semestre 2026 (Actualizado)",
			"code": "I-2026",
			"start_date": "2026-02-01",
			"end_date": "2026-07-15",
			"is_active": true
		}`)
		reqUpdate, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/admin/academic-periods/%s", server.URL, periodID), bytes.NewBuffer(updatePayload))
		reqUpdate.Header.Set("Content-Type", "application/json")
		reqUpdate.Header.Set("X-User-Role", "admin")
		reqUpdate.Header.Set("X-Tenant-Id", tenantID)

		respUpdate, _ := client.Do(reqUpdate)
		if respUpdate.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK updating academic period, got %d", respUpdate.StatusCode)
		}

		// 2.6 Intentar eliminar periodo con materia asociada -> 409 Conflict
		subjectID := uuid.NewString()
		_, _ = db.GetDB().Exec(`
			INSERT INTO subjects (id, tenant_id, name, code, academic_period_id)
			VALUES ($1, $2, 'Redes Avanzadas', 'SIS-405', $3)
		`, subjectID, tenantID, periodID)

		reqDeleteConflict, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/admin/academic-periods/%s", server.URL, periodID), nil)
		reqDeleteConflict.Header.Set("X-User-Role", "admin")
		reqDeleteConflict.Header.Set("X-Tenant-Id", tenantID)

		respDeleteConflict, _ := client.Do(reqDeleteConflict)
		if respDeleteConflict.StatusCode != http.StatusConflict {
			t.Errorf("Expected 409 Conflict when deleting academic period with linked subjects, got %d", respDeleteConflict.StatusCode)
		}

		// 2.7 Desvincular materia y eliminar -> 204 No Content
		_, _ = db.GetDB().Exec(`DELETE FROM subjects WHERE id = $1`, subjectID)

		reqDeleteSuccess, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/admin/academic-periods/%s", server.URL, periodID), nil)
		reqDeleteSuccess.Header.Set("X-User-Role", "admin")
		reqDeleteSuccess.Header.Set("X-Tenant-Id", tenantID)

		respDeleteSuccess, _ := client.Do(reqDeleteSuccess)
		if respDeleteSuccess.StatusCode != http.StatusNoContent {
			t.Errorf("Expected 204 No Content deleting unlinked academic period, got %d", respDeleteSuccess.StatusCode)
		}
	})

	// =========================================================================
	// 3. TEST Archivado Automático por Cron Worker (ADR-029)
	// =========================================================================
	t.Run("3. Worker Cron - Archivado Automático de Periodos y Materias Vencidas", func(t *testing.T) {
		// Crear periodo ya expirado (end_date en el pasado)
		pastPeriodID := uuid.NewString()
		_, err := db.GetDB().Exec(`
			INSERT INTO academic_periods (id, tenant_id, name, code, start_date, end_date, is_active)
			VALUES ($1, $2, 'Semestre Pasado', $3, '2025-01-01', '2025-06-30', true)
		`, pastPeriodID, tenantID, fmt.Sprintf("HIST-%s", pastPeriodID[:6]))
		if err != nil {
			t.Fatalf("Failed to seed past academic period: %v", err)
		}

		// Crear materia asociada no archivada
		pastSubID := uuid.NewString()
		_, err = db.GetDB().Exec(`
			INSERT INTO subjects (id, tenant_id, name, code, academic_period_id, is_archived)
			VALUES ($1, $2, 'Materia Antigua', $3, $4, false)
		`, pastSubID, tenantID, fmt.Sprintf("ANT-%s", pastSubID[:6]), pastPeriodID)
		if err != nil {
			t.Fatalf("Failed to seed past subject: %v", err)
		}

		// Ejecutar la función de archivado del worker
		archivedCount, err := academicPeriodService.ArchiveExpiredPeriods(context.Background())
		if err != nil {
			t.Fatalf("Failed to execute archive worker: %v", err)
		}
		if archivedCount == 0 {
			t.Errorf("Expected at least 1 archived period, got %d", archivedCount)
		}

		// Verificar que el periodo ahora tiene is_active = false
		var isActive bool
		_ = db.GetDB().Get(&isActive, "SELECT is_active FROM academic_periods WHERE id = $1", pastPeriodID)
		if isActive {
			t.Errorf("Expected academic period to be archived (is_active = false)")
		}

		// Verificar que la materia ahora tiene is_archived = true
		var isArchived bool
		_ = db.GetDB().Get(&isArchived, "SELECT is_archived FROM subjects WHERE id = $1", pastSubID)
		if !isArchived {
			t.Errorf("Expected subject to be marked as is_archived = true")
		}
	})
}
