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

func setupTeacherReviewTestServer(t *testing.T) (*httptest.Server, *database.Database) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://solv_user:solv_secure_password@localhost:5432/solv_db?sslmode=disable"
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
		return nil, nil
	}

	_ = db.RunInitialMigrations()

	teacherRepo := postgres.NewPostgresTeacherRepository(db.GetDB())
	submissionRepo := postgres.NewPostgresSubmissionRepository(db.GetDB())
	teacherService := services.NewTeacherService(teacherRepo, submissionRepo)
	submissionService := services.NewSubmissionService(submissionRepo)

	teacherHandler := httpdelivery.NewTeacherHandler(teacherService)
	subHandler := httpdelivery.NewSubmissionHandler(submissionService)

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
		TeacherHandler:    teacherHandler,
		SubmissionHandler: subHandler,
		TenantMiddleware:  tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	server := httptest.NewServer(mux)
	return server, db
}

func TestSlice13_TeacherReviewAndSpeedGraderSuite(t *testing.T) {
	server, db := setupTeacherReviewTestServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	teacherID := uuid.NewString()

	// 1. Crear Docente
	_, err := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Profesor', 'Revisor', $2, 'teacher', $3)
		ON CONFLICT (id) DO NOTHING;
	`, teacherID, fmt.Sprintf("prof_rev_%s@uab.edu.bo", teacherID[:8]), tenantID)
	if err != nil {
		t.Fatalf("Failed to seed teacher: %v", err)
	}

	// 2. Crear Materia
	subjectID := uuid.NewString()
	_, err = db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $2, 'Algoritmos Avanzados - Docente', 'ALG-401', $3)
		ON CONFLICT (id) DO NOTHING;
	`, subjectID, tenantID, teacherID)
	if err != nil {
		t.Fatalf("Failed to seed subject: %v", err)
	}

	// 3. Crear 3 Estudiantes
	student1ID := uuid.NewString()
	student2ID := uuid.NewString()
	student3ID := uuid.NewString()

	for idx, sID := range []string{student1ID, student2ID, student3ID} {
		_, _ = db.GetDB().Exec(`
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, $2, 'Alumno', $3, 'student', $4)
			ON CONFLICT (id) DO NOTHING;
		`, sID, fmt.Sprintf("Estudiante %d", idx+1), fmt.Sprintf("est_%s@uab.edu.bo", sID[:8]), tenantID)

		_, _ = db.GetDB().Exec(`
			INSERT INTO enrollments (tenant_id, student_id, subject_id)
			VALUES ($1, $2, $3)
		`, tenantID, sID, subjectID)
	}

	// 4. Crear Ejercicio con casos de prueba públicos y privados
	exerciseID := uuid.NewString()
	exConfigJSON := `{
		"algorithm": {
			"test_cases": [
				{"input": "5", "expected_output": "120", "is_hidden": false},
				{"input": "0", "expected_output": "1", "is_hidden": false},
				{"input": "12", "expected_output": "479001600", "is_hidden": true},
				{"input": "20", "expected_output": "2432902008176640000", "is_hidden": true}
			]
		}
	}`

	_, err = db.GetDB().Exec(`
		INSERT INTO exercises (id, subject_id, title, description, type, status, config, tenant_id)
		VALUES ($1, $2, 'Lab #02: Factorial Recursivo', 'Calcular factorial', 'algorithm', 'published', $3::jsonb, $4)
	`, exerciseID, subjectID, exConfigJSON, tenantID)
	if err != nil {
		t.Fatalf("Failed to create exercise: %v", err)
	}

	// 5. Crear 3 Submissions en secuencia temporal (para SpeedGrader prev/next):
	// - Sub 1: Student 1 (WA) - hace 30 minutos
	// - Sub 2: Student 2 (AC) - hace 20 minutos
	// - Sub 3: Student 3 (WA) - hace 10 minutos
	sub1ID := uuid.NewString()
	sub2ID := uuid.NewString()
	sub3ID := uuid.NewString()

	t1 := time.Now().Add(-30 * time.Minute)
	t2 := time.Now().Add(-20 * time.Minute)
	t3 := time.Now().Add(-10 * time.Minute)

	_, err = db.GetDB().Exec(`
		INSERT INTO submissions (id, tenant_id, exercise_id, student_id, code, verdict, ast_result, execution_time_ms, memory_used_mb, submitted_at)
		VALUES 
			($1, $4, $5, $6, 'def fact(n): return 1 if n<=1 else n*fact(n-1) # sub1', 'WA', '{"rule_id": "none"}'::jsonb, 12, 24, $7),
			($2, $4, $5, $8, 'def fact(n): return 1 if n<=1 else n*fact(n-1) # sub2', 'AC', '{"rule_id": "none"}'::jsonb, 10, 22, $9),
			($3, $4, $5, $10, 'def fact(n): return 1 if n<=1 else n*fact(n-1) # sub3', 'WA', '{"rule_id": "none"}'::jsonb, 15, 25, $11)
	`, sub1ID, sub2ID, sub3ID, tenantID, exerciseID, student1ID, t1, student2ID, t2, student3ID, t3)
	if err != nil {
		t.Fatalf("Failed to insert submissions: %v", err)
	}

	client := &http.Client{}

	// =========================================================================
	// 1. TEST GET /api/v1/teacher/courses/{id}/submissions (Cola de Revisión)
	// =========================================================================
	t.Run("1. GET /api/v1/teacher/courses/{id}/submissions - Cola con Filtros", func(t *testing.T) {
		// 1.1 Listar todas las entregas del curso
		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/teacher/courses/%s/submissions", server.URL, subjectID), nil)
		req.Header.Set("X-User-Id", teacherID)
		req.Header.Set("X-User-Role", "teacher")
		req.Header.Set("X-Tenant-Id", tenantID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		items := body["data"].([]interface{})
		if len(items) != 3 {
			t.Fatalf("Expected 3 submissions in course queue, got %d", len(items))
		}

		// 1.2 Filtrar por verdict=WA (debe retornar exactamente 2 entregas)
		reqWA, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/teacher/courses/%s/submissions?verdict=WA", server.URL, subjectID), nil)
		reqWA.Header.Set("X-User-Id", teacherID)
		reqWA.Header.Set("X-User-Role", "teacher")
		reqWA.Header.Set("X-Tenant-Id", tenantID)

		respWA, _ := client.Do(reqWA)
		var bodyWA map[string]interface{}
		json.NewDecoder(respWA.Body).Decode(&bodyWA)
		itemsWA := bodyWA["data"].([]interface{})
		if len(itemsWA) != 2 {
			t.Fatalf("Expected 2 submissions with WA verdict, got %d", len(itemsWA))
		}
	})

	// =========================================================================
	// 2. TEST GET /api/v1/teacher/submissions/{id}/review (SpeedGrader & Unmasked)
	// =========================================================================
	t.Run("2. GET /api/v1/teacher/submissions/{id}/review - Desenmascaramiento y Punteros SpeedGrader", func(t *testing.T) {
		// 2.1 Consulta por docente para Sub 2 (intermedia en tiempo)
		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/teacher/submissions/%s/review", server.URL, sub2ID), nil)
		req.Header.Set("X-User-Id", teacherID)
		req.Header.Set("X-User-Role", "teacher")
		req.Header.Set("X-Tenant-Id", tenantID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		review := body["data"].(map[string]interface{})

		// Verificar que se desenmascararon los 4 casos de prueba (incluyendo los 2 privados)
		testCases := review["test_cases"].([]interface{})
		if len(testCases) != 4 {
			t.Fatalf("Expected 4 test cases unmasked for teacher, got %d", len(testCases))
		}

		var privateCount int
		for _, tc := range testCases {
			tcMap := tc.(map[string]interface{})
			if tcMap["is_hidden"] == true {
				privateCount++
				if tcMap["input"] == "" || tcMap["expected_output"] == "" {
					t.Errorf("Private test case had empty input or output: %v", tcMap)
				}
			}
		}
		if privateCount != 2 {
			t.Errorf("Expected exactly 2 private test cases, got %d", privateCount)
		}

		// Verificar punteros de navegación SpeedGrader
		// Para Sub 2 (t2): Sub 1 (t1) es prev, Sub 3 (t3) es next
		if review["prev_submission_id"] != sub1ID {
			t.Errorf("Expected prev_submission_id to be %s, got %v", sub1ID, review["prev_submission_id"])
		}
		if review["next_submission_id"] != sub3ID {
			t.Errorf("Expected next_submission_id to be %s, got %v", sub3ID, review["next_submission_id"])
		}

		// 2.2 Seguridad: Estudiante intentando acceder a review docente -> 403 Forbidden
		reqStudent, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/teacher/submissions/%s/review", server.URL, sub2ID), nil)
		reqStudent.Header.Set("X-User-Role", "student")
		respStudent, _ := client.Do(reqStudent)
		if respStudent.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for student accessing teacher review, got %d", respStudent.StatusCode)
		}
	})

	// =========================================================================
	// 3. TEST POST /api/v1/submissions/{id}/override (Override Manual con Validación >= 10 chars)
	// =========================================================================
	t.Run("3. POST /api/v1/submissions/{id}/override - Convalidación Docente y Validación 422", func(t *testing.T) {
		// 3.1 Frontera 422: Justificación demasiado corta (< 10 caracteres)
		shortPayload := []byte(`{"verdict": "AC", "override_reason": "ok", "score": 100}`)
		reqShort, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/submissions/%s/override", server.URL, sub1ID), bytes.NewBuffer(shortPayload))
		reqShort.Header.Set("Content-Type", "application/json")
		reqShort.Header.Set("X-User-Id", teacherID)
		reqShort.Header.Set("X-User-Role", "teacher")
		reqShort.Header.Set("X-Tenant-Id", tenantID)

		respShort, _ := client.Do(reqShort)
		if respShort.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected 422 Unprocessable Entity for short reason, got %d", respShort.StatusCode)
		}

		// 3.2 Seguridad: Estudiante intentando hacer override -> 403 Forbidden
		reqStuOverride, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/submissions/%s/override", server.URL, sub1ID), bytes.NewBuffer(shortPayload))
		reqStuOverride.Header.Set("Content-Type", "application/json")
		reqStuOverride.Header.Set("X-User-Role", "student")
		respStuOverride, _ := client.Do(reqStuOverride)
		if respStuOverride.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for student override attempt, got %d", respStuOverride.StatusCode)
		}

		// 3.3 Override Exitoso (>= 10 caracteres)
		validPayload := []byte(`{"verdict": "AC", "override_reason": "Solución correcta, contempló caso borde correctamente.", "score": 95}`)
		reqValid, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/submissions/%s/override", server.URL, sub1ID), bytes.NewBuffer(validPayload))
		reqValid.Header.Set("Content-Type", "application/json")
		reqValid.Header.Set("X-User-Id", teacherID)
		reqValid.Header.Set("X-User-Role", "teacher")
		reqValid.Header.Set("X-Tenant-Id", tenantID)

		respValid, err := client.Do(reqValid)
		if err != nil {
			t.Fatalf("Failed request: %v", err)
		}
		if respValid.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK for valid override, got %d", respValid.StatusCode)
		}

		// Verificar persistencia en base de datos
		var subUpdated struct {
			Verdict        string `db:"verdict"`
			ManualOverride bool   `db:"manual_override"`
			Score          int    `db:"score"`
			OverrideReason string `db:"override_reason"`
			GradedBy       string `db:"graded_by"`
		}
		err = db.GetDB().Get(&subUpdated, `
			SELECT verdict, manual_override, score, override_reason, graded_by
			FROM submissions WHERE id = $1
		`, sub1ID)
		if err != nil {
			t.Fatalf("Failed to fetch updated submission: %v", err)
		}

		if subUpdated.Verdict != "AC" {
			t.Errorf("Expected verdict 'AC', got %s", subUpdated.Verdict)
		}
		if !subUpdated.ManualOverride {
			t.Errorf("Expected manual_override = true")
		}
		if subUpdated.Score != 95 {
			t.Errorf("Expected score = 95, got %d", subUpdated.Score)
		}
		if subUpdated.GradedBy != teacherID {
			t.Errorf("Expected graded_by %s, got %s", teacherID, subUpdated.GradedBy)
		}
	})

	// =========================================================================
	// 4. TEST Comentarios In-line Anclados a Código (POST & GET)
	// =========================================================================
	t.Run("4. Comentarios In-line Anclados a Líneas de Código", func(t *testing.T) {
		// 4.1 Docente agrega comentario en línea 15
		c1Payload := []byte(`{"line_number": 15, "comment": "Ojo con el caso base cuando n == 0"}`)
		reqC1, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/teacher/submissions/%s/comments", server.URL, sub1ID), bytes.NewBuffer(c1Payload))
		reqC1.Header.Set("Content-Type", "application/json")
		reqC1.Header.Set("X-User-Id", teacherID)
		reqC1.Header.Set("X-User-Role", "teacher")
		reqC1.Header.Set("X-Tenant-Id", tenantID)

		respC1, err := client.Do(reqC1)
		if err != nil {
			t.Fatalf("Failed request: %v", err)
		}
		if respC1.StatusCode != http.StatusCreated {
			t.Fatalf("Expected 201 Created for comment 1, got %d", respC1.StatusCode)
		}

		// 4.2 Docente agrega comentario en línea 28
		c2Payload := []byte(`{"line_number": 28, "comment": "Excelente optimización de memoria"}`)
		reqC2, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/teacher/submissions/%s/comments", server.URL, sub1ID), bytes.NewBuffer(c2Payload))
		reqC2.Header.Set("Content-Type", "application/json")
		reqC2.Header.Set("X-User-Id", teacherID)
		reqC2.Header.Set("X-User-Role", "teacher")
		reqC2.Header.Set("X-Tenant-Id", tenantID)

		respC2, _ := client.Do(reqC2)
		if respC2.StatusCode != http.StatusCreated {
			t.Fatalf("Expected 201 Created for comment 2, got %d", respC2.StatusCode)
		}

		// 4.3 Consultar comentarios de la entrega
		reqList, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/teacher/submissions/%s/comments", server.URL, sub1ID), nil)
		reqList.Header.Set("X-Tenant-Id", tenantID)

		respList, _ := client.Do(reqList)
		if respList.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK for comments list, got %d", respList.StatusCode)
		}

		var bodyList map[string]interface{}
		json.NewDecoder(respList.Body).Decode(&bodyList)
		comments := bodyList["data"].([]interface{})
		if len(comments) != 2 {
			t.Fatalf("Expected exactly 2 comments, got %d", len(comments))
		}

		comment1 := comments[0].(map[string]interface{})
		if int(comment1["line_number"].(float64)) != 15 {
			t.Errorf("Expected line_number = 15, got %v", comment1["line_number"])
		}

		comment2 := comments[1].(map[string]interface{})
		if int(comment2["line_number"].(float64)) != 28 {
			t.Errorf("Expected line_number = 28, got %v", comment2["line_number"])
		}
	})
}
