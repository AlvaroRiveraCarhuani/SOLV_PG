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

func setupSlice14ReassignAndStudentsServer(t *testing.T) (*httptest.Server, *database.Database) {
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

func TestSlice14_CourseReassignmentAndStudentDirectory(t *testing.T) {
	server, db := setupSlice14ReassignAndStudentsServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	client := &http.Client{}

	// 1. Seed Docentes
	teacherA_ID := uuid.NewString()
	teacherB_ID := uuid.NewString()
	notATeacherID := uuid.NewString()

	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES 
			($1, 'Roberto', 'DocenteA', $4, 'teacher', $7),
			($2, 'Elena', 'DocenteB', $5, 'teacher', $7),
			($3, 'Pedro', 'EstudianteComun', $6, 'student', $7)
		ON CONFLICT (id) DO NOTHING;
	`, teacherA_ID, teacherB_ID, notATeacherID,
		fmt.Sprintf("docA_%s@uab.edu.bo", teacherA_ID[:6]),
		fmt.Sprintf("docB_%s@uab.edu.bo", teacherB_ID[:6]),
		fmt.Sprintf("pedro_%s@uab.edu.bo", notATeacherID[:6]),
		tenantID)

	// 2. Seed Materia asignada a Docente A
	subjectID := uuid.NewString()
	_, _ = db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Estructura de Datos', 'SIS-204', $3)
		ON CONFLICT (id) DO NOTHING;
	`, subjectID, tenantID, teacherA_ID)

	// =========================================================================
	// 1. TEST Reasignación de Docente Titular (ADR-036)
	// =========================================================================
	t.Run("1. Reasignación de Materia - Validación de Rol Docente, 422 y Cambio en BD", func(t *testing.T) {
		// 1.1 Intentar reasignar a un usuario que es rol student -> 422 Unprocessable Entity
		payloadStudent := []byte(fmt.Sprintf(`{"new_teacher_id": "%s", "reason": "Intento invalido"}`, notATeacherID))
		reqInvalid, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/courses/%s/reassign", server.URL, subjectID), bytes.NewBuffer(payloadStudent))
		reqInvalid.Header.Set("Content-Type", "application/json")
		reqInvalid.Header.Set("X-User-Role", "admin")
		reqInvalid.Header.Set("X-Tenant-Id", tenantID)

		respInvalid, _ := client.Do(reqInvalid)
		if respInvalid.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422 Unprocessable Entity when assigning non-teacher, got %d", respInvalid.StatusCode)
		}

		// 1.2 Reasignar exitosamente a Docente B -> 200 OK
		payloadValid := []byte(fmt.Sprintf(`{"new_teacher_id": "%s", "reason": "Cambio de titular por inicio de semestre"}`, teacherB_ID))
		reqValid, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/courses/%s/reassign", server.URL, subjectID), bytes.NewBuffer(payloadValid))
		reqValid.Header.Set("Content-Type", "application/json")
		reqValid.Header.Set("X-User-Role", "admin")
		reqValid.Header.Set("X-Tenant-Id", tenantID)

		respValid, err := client.Do(reqValid)
		if err != nil {
			t.Fatalf("Failed reassign request: %v", err)
		}
		if respValid.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK for valid course reassignment, got %d", respValid.StatusCode)
		}

		// 1.3 Verificar en base de datos que la materia ahora pertenece a Docente B
		var currentTeacher string
		_ = db.GetDB().Get(&currentTeacher, "SELECT teacher_id FROM subjects WHERE id = $1", subjectID)
		if currentTeacher != teacherB_ID {
			t.Errorf("Expected subject teacher_id to be %s, got %s", teacherB_ID, currentTeacher)
		}

		// 1.4 Seguridad: Estudiante intentando reasignar curso -> 403 Forbidden
		reqStudent, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/courses/%s/reassign", server.URL, subjectID), bytes.NewBuffer(payloadValid))
		reqStudent.Header.Set("X-User-Role", "student")
		respStudent, _ := client.Do(reqStudent)
		if respStudent.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for student course reassign, got %d", respStudent.StatusCode)
		}
	})

	// =========================================================================
	// 2. TEST Directorio de Estudiantes y Reset OOM (ADR-033)
	// =========================================================================
	t.Run("2. Directorio de Estudiantes - Métricas Agregadas, Búsqueda y Filtros", func(t *testing.T) {
		// Crear 2 materias adicionales
		sub2 := uuid.NewString()
		_, _ = db.GetDB().Exec(`
			INSERT INTO subjects (id, tenant_id, name, code)
			VALUES ($1, $2, 'Sistemas Operativos', 'SIS-301')
			ON CONFLICT (id) DO NOTHING;
		`, sub2, tenantID)

		// Crear Estudiante 1: Matriculado en 2 materias, 1 workspace 'running', 0 strikes
		stu1ID := uuid.NewString()
		_, _ = db.GetDB().Exec(`
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, 'Juan', 'Perez', $2, 'student', $3)
			ON CONFLICT (id) DO NOTHING;
		`, stu1ID, fmt.Sprintf("juan_%s@uab.edu.bo", stu1ID[:6]), tenantID)

		_, _ = db.GetDB().Exec(`
			INSERT INTO enrollments (tenant_id, student_id, subject_id)
			VALUES ($1, $2, $3), ($1, $2, $4)
			ON CONFLICT DO NOTHING;
		`, tenantID, stu1ID, subjectID, sub2)

		ws1ID := uuid.NewString()
		_, _ = db.GetDB().Exec(`
			INSERT INTO workspaces (id, student_id, subject_id, type, status, access_url, memory_limit_mb, oom_strike_count, tenant_id)
			VALUES ($1, $2, $3, 'IDE_PERSISTENTE', 'running', 'http://ws1.local', 256, 0, $4)
			ON CONFLICT (id) DO NOTHING;
		`, ws1ID, stu1ID, subjectID, tenantID)

		// Crear Estudiante 2: 1 materia, 0 workspaces running, 2 strikes OOM
		stu2ID := uuid.NewString()
		uniqueLastName := fmt.Sprintf("GomezUniq_%s", stu2ID[:6])
		_, _ = db.GetDB().Exec(`
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, 'Maria', $2, $3, 'student', $4)
			ON CONFLICT (id) DO NOTHING;
		`, stu2ID, uniqueLastName, fmt.Sprintf("maria_%s@uab.edu.bo", stu2ID[:6]), tenantID)

		_, _ = db.GetDB().Exec(`
			INSERT INTO enrollments (tenant_id, student_id, subject_id)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING;
		`, tenantID, stu2ID, subjectID)

		ws2ID := uuid.NewString()
		_, _ = db.GetDB().Exec(`
			INSERT INTO workspaces (id, student_id, subject_id, type, status, access_url, memory_limit_mb, oom_strike_count, last_oom_killed_at, tenant_id)
			VALUES ($1, $2, $3, 'IDE_PERSISTENTE', 'hibernated', 'http://ws2.local', 256, 2, NOW() - INTERVAL '1 hour', $4)
			ON CONFLICT (id) DO NOTHING;
		`, ws2ID, stu2ID, subjectID, tenantID)

		// 2.1 Consultar directorio completo -> 200 OK
		reqDir, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/students", nil)
		reqDir.Header.Set("X-User-Role", "admin")
		reqDir.Header.Set("X-Tenant-Id", tenantID)

		respDir, err := client.Do(reqDir)
		if err != nil {
			t.Fatalf("Failed student directory request: %v", err)
		}
		if respDir.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK for student directory, got %d", respDir.StatusCode)
		}

		var dirBody map[string]interface{}
		json.NewDecoder(respDir.Body).Decode(&dirBody)
		students := dirBody["data"].([]interface{})
		if len(students) < 2 {
			t.Fatalf("Expected at least 2 students in directory, got %d", len(students))
		}

		// Validar métricas de Juan Perez
		var juanData map[string]interface{}
		for _, s := range students {
			sMap := s.(map[string]interface{})
			if sMap["id"] == stu1ID {
				juanData = sMap
				break
			}
		}
		if juanData == nil {
			t.Fatalf("Juan Perez not found in directory")
		}
		if int(juanData["enrolled_courses_count"].(float64)) != 2 {
			t.Errorf("Juan expected 2 enrolled courses, got %v", juanData["enrolled_courses_count"])
		}
		if int(juanData["active_workspaces_count"].(float64)) != 1 {
			t.Errorf("Juan expected 1 active workspace, got %v", juanData["active_workspaces_count"])
		}
		if int(juanData["oom_strike_count"].(float64)) != 0 {
			t.Errorf("Juan expected 0 oom strikes, got %v", juanData["oom_strike_count"])
		}

		// 2.2 Filtro de Búsqueda: search=uniqueLastName -> Solo Maria GomezUniq
		reqSearch, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/admin/students?search=%s", server.URL, uniqueLastName), nil)
		reqSearch.Header.Set("X-User-Role", "admin")
		reqSearch.Header.Set("X-Tenant-Id", tenantID)

		respSearch, _ := client.Do(reqSearch)
		var searchBody map[string]interface{}
		json.NewDecoder(respSearch.Body).Decode(&searchBody)
		searchList := searchBody["data"].([]interface{})
		if len(searchList) != 1 {
			t.Errorf("Expected 1 student for search=%s, got %d", uniqueLastName, len(searchList))
		}

		// 2.3 Filtro de Estado: status=strikes -> Devuelve a Maria Gomez (strike count > 0)
		reqStrikes, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/students?status=strikes", nil)
		reqStrikes.Header.Set("X-User-Role", "admin")
		reqStrikes.Header.Set("X-Tenant-Id", tenantID)

		respStrikes, _ := client.Do(reqStrikes)
		var strikesBody map[string]interface{}
		json.NewDecoder(respStrikes.Body).Decode(&strikesBody)
		strikesList := strikesBody["data"].([]interface{})
		foundMaria := false
		for _, s := range strikesList {
			sMap := s.(map[string]interface{})
			if sMap["id"] == stu2ID {
				foundMaria = true
				if int(sMap["oom_strike_count"].(float64)) != 2 {
					t.Errorf("Maria expected 2 strikes, got %v", sMap["oom_strike_count"])
				}
			}
		}
		if !foundMaria {
			t.Errorf("Maria Gomez not found in status=strikes filter")
		}

		// =====================================================================
		// 2.4 TEST Reset de Penalizaciones OOM-Killed (POST /students/{id}/reset-oom)
		// =====================================================================
		// Justificación menor a 10 caracteres -> 422
		payloadShort := []byte(`{"reason": "perdon"}`)
		reqShort, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/students/%s/reset-oom", server.URL, stu2ID), bytes.NewBuffer(payloadShort))
		reqShort.Header.Set("Content-Type", "application/json")
		reqShort.Header.Set("X-User-Role", "admin")
		reqShort.Header.Set("X-Tenant-Id", tenantID)

		respShort, _ := client.Do(reqShort)
		if respShort.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422 for reason < 10 chars, got %d", respShort.StatusCode)
		}

		// Justificación válida >= 10 caracteres -> 200 OK
		payloadReset := []byte(`{"reason": "El alumno solucionó la fuga de memoria en sus punteros C++"}`)
		reqReset, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/admin/students/%s/reset-oom", server.URL, stu2ID), bytes.NewBuffer(payloadReset))
		reqReset.Header.Set("Content-Type", "application/json")
		reqReset.Header.Set("X-User-Role", "admin")
		reqReset.Header.Set("X-Tenant-Id", tenantID)

		respReset, err := client.Do(reqReset)
		if err != nil {
			t.Fatalf("Failed reset oom request: %v", err)
		}
		if respReset.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK resetting OOM strikes, got %d", respReset.StatusCode)
		}

		var resetBody map[string]interface{}
		json.NewDecoder(respReset.Body).Decode(&resetBody)
		resetData := resetBody["data"].(map[string]interface{})
		if int(resetData["workspaces_reset_count"].(float64)) != 1 {
			t.Errorf("Expected 1 workspace reset, got %v", resetData["workspaces_reset_count"])
		}

		// Verificar en BD que los strikes de Maria Gómez volvieron a 0 y last_oom_killed_at es NULL
		var strikesAfter int
		var lastOOMAfter *time.Time
		_ = db.GetDB().QueryRow("SELECT oom_strike_count, last_oom_killed_at FROM workspaces WHERE id = $1", ws2ID).Scan(&strikesAfter, &lastOOMAfter)
		if strikesAfter != 0 {
			t.Errorf("Expected oom_strike_count = 0 after reset, got %d", strikesAfter)
		}
		if lastOOMAfter != nil {
			t.Errorf("Expected last_oom_killed_at to be NULL after reset, got %v", lastOOMAfter)
		}
	})
}
