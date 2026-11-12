package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
	"solv-backend/internal/infrastructure/database"
	"solv-backend/internal/infrastructure/storage/postgres"
)

func setupTeacherEnrollTestServer(t *testing.T) (*httptest.Server, *database.Database) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
		return nil, nil
	}

	teacherRepo := postgres.NewPostgresTeacherRepository(db.GetDB())
	teacherService := services.NewTeacherService(teacherRepo, nil)
	teacherHandler := httpdelivery.NewTeacherHandler(teacherService, nil)

	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, withTestIdentityFromHeaders(r))
		})
	}

	auditMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	handlers := httpdelivery.Handlers{
		TeacherHandler:   teacherHandler,
		TenantMiddleware: tenantMiddleware,
		AuditMiddleware:  auditMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	server := httptest.NewServer(mux)
	return server, db
}

func TestTeacher_EnrollStudentInCourse_Integration(t *testing.T) {
	server, db := setupTeacherEnrollTestServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	teacher1ID := uuid.New().String()
	teacher2ID := uuid.New().String()
	student1ID := uuid.New().String()
	student2ID := uuid.New().String()
	subject1ID := uuid.New().String()

	// Cleanup
	defer func() {
		_, _ = db.GetDB().Exec(`DELETE FROM enrollments WHERE subject_id = $1`, subject1ID)
		_, _ = db.GetDB().Exec(`DELETE FROM subjects WHERE id = $1`, subject1ID)
		_, _ = db.GetDB().Exec(`DELETE FROM users WHERE id IN ($1, $2, $3, $4)`, teacher1ID, teacher2ID, student1ID, student2ID)
	}()

	// 1. Insertar docentes y estudiantes
	mustExec := func(q string, args ...interface{}) {
		_, err := db.GetDB().Exec(q, args...)
		if err != nil {
			t.Fatalf("Failed to execute query %q: %v", q, err)
		}
	}

	mustExec(`
		INSERT INTO users (id, tenant_id, first_name, last_name, email, role, status)
		VALUES 
			($1, $2, 'Docente', 'Uno', 'docente1@institucion.edu', 'teacher', 'active'),
			($3, $2, 'Docente', 'Dos', 'docente2@institucion.edu', 'teacher', 'active'),
			($4, $2, 'Alumno', 'YaInscripto', 'alumno1@institucion.edu', 'student', 'active'),
			($5, $2, 'Alumno', 'Disponible', 'alumno2@institucion.edu', 'student', 'active')
		ON CONFLICT (id) DO NOTHING
	`, teacher1ID, tenantID, teacher2ID, student1ID, student2ID)

	// 2. Insertar materia del Docente 1
	mustExec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Algoritmos y Complejidad', 'ALG-101', $3)
		ON CONFLICT (id) DO NOTHING
	`, subject1ID, tenantID, teacher1ID)

	// 3. Matricular al Alumno 1 en la materia
	mustExec(`
		INSERT INTO enrollments (id, tenant_id, student_id, subject_id, enrolled_at)
		VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		ON CONFLICT DO NOTHING
	`, tenantID, student1ID, subject1ID)

	// Helper para requests autenticadas
	doRequest := func(method, url, userID, role string, body []byte) (*http.Response, map[string]interface{}) {
		var req *http.Request
		var err error
		if body != nil {
			req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, err = http.NewRequest(method, url, nil)
		}
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("X-Test-Tenant-ID", tenantID)
		req.Header.Set("X-Test-User-ID", userID)
		req.Header.Set("X-Test-Role", role)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		var parsed map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&parsed)
		return resp, parsed
	}

	// Caso A: Docente 1 busca estudiantes disponibles (debe salir Alumno 2, pero NO Alumno 1)
	t.Run("Docente busca estudiantes disponibles", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/teacher/courses/%s/available-students?q=alumno", server.URL, subject1ID)
		resp, data := doRequest(http.MethodGet, url, teacher1ID, "teacher", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d: %v", resp.StatusCode, data)
		}

		rawItems, ok := data["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected array in data: %v", data)
		}
		if len(rawItems) != 1 {
			t.Fatalf("Expected exactly 1 available student, got %d", len(rawItems))
		}
		item := rawItems[0].(map[string]interface{})
		if item["id"] != student2ID {
			t.Errorf("Expected available student to be student2 (%s), got %v", student2ID, item["id"])
		}
	})

	// Caso B: Docente 2 intenta matricular en materia ajena (debe dar 404 Not Found)
	t.Run("Docente ajeno recibe 404 al intentar matricular en curso ajeno", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/teacher/courses/%s/students", server.URL, subject1ID)
		body, _ := json.Marshal(map[string]string{"student_id": student2ID})
		resp, data := doRequest(http.MethodPost, url, teacher2ID, "teacher", body)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Expected 404 Not Found, got %d: %v", resp.StatusCode, data)
		}
	})

	// Caso C: Docente 1 matricula al Alumno 2 exitosamente
	t.Run("Docente titular matricula estudiante con éxito", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/teacher/courses/%s/students", server.URL, subject1ID)
		body, _ := json.Marshal(map[string]string{"student_id": student2ID})
		resp, data := doRequest(http.MethodPost, url, teacher1ID, "teacher", body)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected 201 Created, got %d: %v", resp.StatusCode, data)
		}

		// Verificar que ahora existe en la base de datos
		var enrolledExists bool
		err := db.GetDB().Get(&enrolledExists, `
			SELECT EXISTS(SELECT 1 FROM enrollments WHERE tenant_id = $1 AND student_id = $2 AND subject_id = $3)
		`, tenantID, student2ID, subject1ID)
		if err != nil || !enrolledExists {
			t.Fatalf("Enrollment was not persisted in database: %v", err)
		}
	})

	// Caso D: Docente 1 vuelve a buscar disponibles (ya no debe figurar Alumno 2)
	t.Run("Alumno recién matriculado ya no figura como disponible", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/teacher/courses/%s/available-students?q=alumno", server.URL, subject1ID)
		resp, data := doRequest(http.MethodGet, url, teacher1ID, "teacher", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d: %v", resp.StatusCode, data)
		}

		rawItems, ok := data["data"].([]interface{})
		if !ok || len(rawItems) != 0 {
			t.Fatalf("Expected 0 available students, got %v", rawItems)
		}
	})
}
