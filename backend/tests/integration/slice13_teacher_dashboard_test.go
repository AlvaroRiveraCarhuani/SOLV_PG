package integration

import (
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

func setupTeacherDashboardTestServer(t *testing.T) (*httptest.Server, *database.Database) {
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
	teacherService := services.NewTeacherService(teacherRepo)
	teacherHandler := httpdelivery.NewTeacherHandler(teacherService)

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
		TeacherHandler:   teacherHandler,
		TenantMiddleware: tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)

	server := httptest.NewServer(mux)
	return server, db
}

func TestSlice13_TeacherDashboard_CompleteSuite(t *testing.T) {
	server, db := setupTeacherDashboardTestServer(t)
	if server == nil {
		return
	}
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	tenantB := "00000000-0000-0000-0000-000000000002"

	// Crear Tenant B si no existe
	_, _ = db.GetDB().Exec(`
		INSERT INTO tenants (id, name, slug)
		VALUES ($1, 'Tenant Secundario', 'tenant-b')
		ON CONFLICT (id) DO NOTHING;
	`, tenantB)

	teacherID := uuid.NewString()
	otherTeacherID := uuid.NewString()

	// Insertar docentes
	_, err := db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES 
			($1, 'Profesor', 'Principal', $3, 'teacher', $5),
			($2, 'Profesor', 'Vacio', $4, 'teacher', $5)
		ON CONFLICT (id) DO NOTHING;
	`, teacherID, otherTeacherID, fmt.Sprintf("prof_dash_%s@uab.edu.bo", teacherID[:8]), fmt.Sprintf("prof_vacio_%s@uab.edu.bo", otherTeacherID[:8]), tenantID)
	if err != nil {
		t.Fatalf("Failed to seed teachers: %v", err)
	}

	// 1 Docente con 2 materias
	subject1ID := uuid.NewString()
	subject2ID := uuid.NewString()

	_, err = db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES 
			($1, $3, 'Programación II - Docente', 'PROG-201', $4),
			($2, $3, 'Estructuras de Datos - Docente', 'ED-301', $4)
		ON CONFLICT (id) DO NOTHING;
	`, subject1ID, subject2ID, tenantID, teacherID)
	if err != nil {
		t.Fatalf("Failed to seed subjects: %v", err)
	}

	// Crear 6 estudiantes por materia (Total 12 estudiantes)
	var studentsSub1 []string
	var studentsSub2 []string

	for i := 1; i <= 6; i++ {
		sID := uuid.NewString()
		studentsSub1 = append(studentsSub1, sID)
		_, _ = db.GetDB().Exec(`
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, $2, 'Estudiante', $3, 'student', $4)
		`, sID, fmt.Sprintf("AlumnoA%d", i), fmt.Sprintf("alumno_a%d_%s@uab.edu.bo", i, sID[:8]), tenantID)

		_, _ = db.GetDB().Exec(`
			INSERT INTO enrollments (tenant_id, student_id, subject_id)
			VALUES ($1, $2, $3)
		`, tenantID, sID, subject1ID)
	}

	for i := 1; i <= 6; i++ {
		sID := uuid.NewString()
		studentsSub2 = append(studentsSub2, sID)
		_, _ = db.GetDB().Exec(`
			INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
			VALUES ($1, $2, 'Estudiante', $3, 'student', $4)
		`, sID, fmt.Sprintf("AlumnoB%d", i), fmt.Sprintf("alumno_b%d_%s@uab.edu.bo", i, sID[:8]), tenantID)

		_, _ = db.GetDB().Exec(`
			INSERT INTO enrollments (tenant_id, student_id, subject_id)
			VALUES ($1, $2, $3)
		`, tenantID, sID, subject2ID)
	}

	// Configurar Workspaces para Materia 1:
	// - 2 estudiantes con heartbeat fresco (< 1 minuto) -> active_now = 2
	// - 1 estudiante con workspace OOM_Killed
	// - 1 estudiante con heartbeat viejo (> 5 min)
	wsFresh1 := uuid.NewString()
	wsFresh2 := uuid.NewString()
	wsOOM := uuid.NewString()
	wsOld := uuid.NewString()

	_, err = db.GetDB().Exec(`
		INSERT INTO workspaces (id, student_id, subject_id, status, type, access_url, last_heartbeat_at, last_oom_killed_at, tenant_id)
		VALUES 
			($1, $5, $9, 'running', 'IDE_PERSISTENTE', 'http://ws1.local', NOW(), NULL, $10),
			($2, $6, $9, 'running', 'IDE_PERSISTENTE', 'http://ws2.local', NOW(), NULL, $10),
			($3, $7, $9, 'oom_killed', 'IDE_PERSISTENTE', 'http://ws3.local', NOW() - INTERVAL '10 minutes', NOW(), $10),
			($4, $8, $9, 'running', 'IDE_PERSISTENTE', 'http://ws4.local', NOW() - INTERVAL '10 minutes', NULL, $10)
	`, wsFresh1, wsFresh2, wsOOM, wsOld, studentsSub1[0], studentsSub1[1], studentsSub1[2], studentsSub1[3], subject1ID, tenantID)
	if err != nil {
		t.Fatalf("Failed to insert test workspaces: %v", err)
	}

	// Crear Laboratorio 1 (con due_date a menos de 24h: 12h en el futuro)
	now := time.Now()
	lab1ID := uuid.NewString()
	dueIn12h := now.Add(12 * time.Hour)
	_, err = db.GetDB().Exec(`
		INSERT INTO exercises (id, subject_id, title, description, type, status, due_date, config, tenant_id)
		VALUES ($1, $2, 'Lab #01: Árboles AVL', 'Balanceo de árboles', 'algorithm', 'published', $3, '{}'::jsonb, $4)
	`, lab1ID, subject1ID, dueIn12h, tenantID)
	if err != nil {
		t.Fatalf("Failed to create lab 1: %v", err)
	}

	// Submissions para Lab 1:
	// - Estudiante 1: AC
	// - Estudiante 2: WA (pending review)
	// - Estudiante 3: AST_BLOCKED
	// - Estudiante 4: RE (pending review)
	// - Estudiante 5: Tiene submission AC previa
	// - Estudiante 6: NO TIENE SUBMISSION Y NO TIENE HEARTBEAT -> AT RISK = 1 !
	sub1 := uuid.NewString()
	sub2 := uuid.NewString()
	sub3 := uuid.NewString()
	sub4 := uuid.NewString()
	sub5 := uuid.NewString()

	_, _ = db.GetDB().Exec(`
		INSERT INTO submissions (id, tenant_id, exercise_id, student_id, code, verdict, ast_result, submitted_at, manual_override)
		VALUES 
			($1, $6, $7, $8, 'code1', 'AC', '{}'::jsonb, NOW(), FALSE),
			($2, $6, $7, $9, 'code2', 'WA', '{}'::jsonb, NOW(), FALSE),
			($3, $6, $7, $10, 'code3', 'AST_BLOCKED', '{"rule_id": "no_import_os"}'::jsonb, NOW(), FALSE),
			($4, $6, $7, $11, 'code4', 'RE', '{}'::jsonb, NOW(), FALSE),
			($5, $6, $7, $12, 'code5', 'AC', '{}'::jsonb, NOW(), FALSE)
	`, sub1, sub2, sub3, sub4, sub5, tenantID, lab1ID, studentsSub1[0], studentsSub1[1], studentsSub1[2], studentsSub1[3], studentsSub1[4])

	client := &http.Client{}

	// =========================================================================
	// 1. TEST GET /api/v1/teacher/courses
	// =========================================================================
	t.Run("1. GET /api/v1/teacher/courses - Metricas Agregadas Exactas", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/teacher/courses", nil)
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
		data, ok := body["data"].([]interface{})
		if !ok || len(data) != 2 {
			t.Fatalf("Expected exactly 2 courses for teacher, got %v", body)
		}

		// Buscar Materia 1
		var course1 map[string]interface{}
		for _, item := range data {
			c := item.(map[string]interface{})
			if c["id"] == subject1ID {
				course1 = c
				break
			}
		}
		if course1 == nil {
			t.Fatalf("Course 1 (%s) not found in response", subject1ID)
		}

		// Afirmaciones exactas (no sólo > 0)
		if int(course1["students_count"].(float64)) != 6 {
			t.Errorf("Expected students_count = 6, got %v", course1["students_count"])
		}
		if int(course1["active_now"].(float64)) != 2 {
			t.Errorf("Expected active_now = 2, got %v", course1["active_now"])
		}
		// WA y RE = 2 pending_review
		if int(course1["pending_review"].(float64)) != 2 {
			t.Errorf("Expected pending_review = 2 (WA y RE), got %v", course1["pending_review"])
		}
		// Estudiante 6 = exactly 1 at_risk
		if int(course1["at_risk"].(float64)) != 1 {
			t.Errorf("Expected at_risk = 1 (Student 6), got %v", course1["at_risk"])
		}
	})

	// =========================================================================
	// 2. TEST GET /api/v1/teacher/attention
	// =========================================================================
	t.Run("2. GET /api/v1/teacher/attention - Clasificación por Severidad", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/teacher/attention", nil)
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
		data := body["data"].(map[string]interface{})

		critical := data["critical"].([]interface{})
		warning := data["warning"].([]interface{})
		standard := data["standard"].([]interface{})

		// Verificar Critical (OOM_Killed)
		if len(critical) == 0 {
			t.Errorf("Expected at least 1 critical alert (OOM_Killed), got 0")
		} else {
			crit1 := critical[0].(map[string]interface{})
			if crit1["type"] != "oom_killed" {
				t.Errorf("Expected type 'oom_killed', got %v", crit1["type"])
			}
			if crit1["workspace_id"] != wsOOM {
				t.Errorf("Expected workspace_id %s, got %v", wsOOM, crit1["workspace_id"])
			}
		}

		// Verificar Warning (AST_BLOCKED)
		if len(warning) == 0 {
			t.Errorf("Expected at least 1 warning alert (AST_BLOCKED), got 0")
		} else {
			warn1 := warning[0].(map[string]interface{})
			if warn1["type"] != "ast_blocked" {
				t.Errorf("Expected type 'ast_blocked', got %v", warn1["type"])
			}
			if warn1["rule_violated"] != "no_import_os" {
				t.Errorf("Expected rule_violated 'no_import_os', got %v", warn1["rule_violated"])
			}
		}

		// Verificar Standard (WA y RE pendientes)
		if len(standard) < 2 {
			t.Errorf("Expected at least 2 standard alerts (WA y RE), got %d", len(standard))
		}
	})

	// =========================================================================
	// 3. TEST GET /api/v1/teacher/courses/{id}/labs
	// =========================================================================
	t.Run("3. GET /api/v1/teacher/courses/{id}/labs - Desglose de Laboratorio", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/teacher/courses/"+subject1ID+"/labs", nil)
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
		labs := body["data"].([]interface{})
		if len(labs) != 1 {
			t.Fatalf("Expected exactly 1 lab for Subject 1, got %d", len(labs))
		}

		labStat := labs[0].(map[string]interface{})
		if labStat["id"] != lab1ID {
			t.Errorf("Expected lab ID %s, got %v", lab1ID, labStat["id"])
		}
		if int(labStat["students_count"].(float64)) != 6 {
			t.Errorf("Expected students_count = 6, got %v", labStat["students_count"])
		}
		if int(labStat["submissions_count"].(float64)) != 5 {
			t.Errorf("Expected submissions_count = 5, got %v", labStat["submissions_count"])
		}
		if int(labStat["auto_graded"].(float64)) != 2 {
			t.Errorf("Expected auto_graded = 2 (AC), got %v", labStat["auto_graded"])
		}
		if int(labStat["pending_review"].(float64)) != 2 {
			t.Errorf("Expected pending_review = 2 (WA y RE), got %v", labStat["pending_review"])
		}
		if int(labStat["at_risk"].(float64)) != 1 {
			t.Errorf("Expected at_risk = 1, got %v", labStat["at_risk"])
		}

		verdicts := labStat["verdicts_summary"].(map[string]interface{})
		if int(verdicts["AC"].(float64)) != 2 {
			t.Errorf("Expected AC = 2, got %v", verdicts["AC"])
		}
		if int(verdicts["WA"].(float64)) != 1 {
			t.Errorf("Expected WA = 1, got %v", verdicts["WA"])
		}
		if int(verdicts["AST_BLOCKED"].(float64)) != 1 {
			t.Errorf("Expected AST_BLOCKED = 1, got %v", verdicts["AST_BLOCKED"])
		}
		if int(verdicts["RE"].(float64)) != 1 {
			t.Errorf("Expected RE = 1, got %v", verdicts["RE"])
		}
	})

	// =========================================================================
	// 4. TEST Lógica de Frontera y Casos Negativos
	// =========================================================================
	t.Run("4. Casos Negativos y Frontera - Permisos, Tenant y Vacíos", func(t *testing.T) {
		// 4.1 Rol student -> 403 Forbidden
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/teacher/courses", nil)
		req.Header.Set("X-User-Role", "student")
		resp, _ := client.Do(req)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for student role, got %d", resp.StatusCode)
		}

		// 4.2 Docente sin materias -> Array vacío [] (no 500 ni null)
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/teacher/courses", nil)
		req.Header.Set("X-User-Id", otherTeacherID)
		req.Header.Set("X-User-Role", "teacher")
		resp, _ = client.Do(req)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK for empty teacher courses, got %d", resp.StatusCode)
		}
		var emptyBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&emptyBody)
		emptyCourses, ok := emptyBody["data"].([]interface{})
		if !ok || len(emptyCourses) != 0 {
			t.Errorf("Expected empty courses array [], got %v", emptyBody["data"])
		}

		// 4.3 Aislamiento Cross-Tenant: Docente en Tenant B consultando materia de Tenant A -> 404
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/teacher/courses/"+subject1ID+"/labs", nil)
		req.Header.Set("X-User-Id", teacherID)
		req.Header.Set("X-User-Role", "teacher")
		req.Header.Set("X-Tenant-Id", tenantB)
		resp, _ = client.Do(req)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found for cross-tenant course lookup, got %d", resp.StatusCode)
		}

		// 4.4 Materia 2 (sin laboratorios) -> Array vacío [] (no null)
		req, _ = http.NewRequest("GET", server.URL+"/api/v1/teacher/courses/"+subject2ID+"/labs", nil)
		req.Header.Set("X-User-Id", teacherID)
		req.Header.Set("X-User-Role", "teacher")
		req.Header.Set("X-Tenant-Id", tenantID)
		resp, _ = client.Do(req)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 OK for empty labs subject, got %d", resp.StatusCode)
		}
		var emptyLabsBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&emptyLabsBody)
		emptyLabs, ok := emptyLabsBody["data"].([]interface{})
		if !ok || len(emptyLabs) != 0 {
			t.Errorf("Expected empty labs array [], got %v", emptyLabsBody["data"])
		}
	})
}
