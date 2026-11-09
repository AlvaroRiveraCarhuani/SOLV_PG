package integration

import (
	"encoding/json"
	"fmt"
	"io"
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

// TestSlice13_TeacherExerciseBank valida el read model del banco de ejercicios:
// agrega laboratorios de todas las materias del docente con datos reales
// (environment_type, lenguaje, límites) y la semántica de revisión por entorno
// (juez: veredictos por revisar; IDE: entregas PENDING en la cola de SpeedGrader).
func TestSlice13_TeacherExerciseBank(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = getTestDSN()
	}

	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
		return
	}

	teacherRepo := postgres.NewPostgresTeacherRepository(db.GetDB())
	teacherService := services.NewTeacherService(teacherRepo, nil)
	teacherHandler := httpdelivery.NewTeacherHandler(teacherService, nil)

	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, withTestIdentityFromHeaders(r))
		})
	}

	handlers := httpdelivery.Handlers{
		TeacherHandler:   teacherHandler,
		TenantMiddleware: tenantMiddleware,
	}

	mux := http.NewServeMux()
	httpdelivery.SetupRoutes(mux, &handlers)
	server := httptest.NewServer(mux)
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	teacherID := uuid.NewString()

	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Profesora', 'Banco', $2, 'teacher', $3)
		ON CONFLICT (id) DO NOTHING;
	`, teacherID, fmt.Sprintf("prof_bank_%s@uab.edu.bo", teacherID[:8]), tenantID)

	// Dos materias: una con laboratorio de juez y otra con laboratorio IDE.
	subjectJudge := uuid.NewString()
	subjectIDE := uuid.NewString()
	_, _ = db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $4, 'Algoritmos - Banco', 'BAN-101', $3),
		       ($2, $4, 'Web Fullstack - Banco', 'BAN-202', $3)
		ON CONFLICT (id) DO NOTHING;
	`, subjectJudge, subjectIDE, teacherID, tenantID)

	judgeEx := uuid.NewString()
	ideEx := uuid.NewString()
	tplID := uuid.NewString()
	_, err = db.GetDB().Exec(`
		INSERT INTO lab_templates (id, tenant_id, name, docker_image, base_ram_mb, target_environment, services_config, resource_profile, setup_script, tools_declared, stack_config)
		VALUES ($1, $2, $3, 'python:3.11-slim', 128, 'JUEZ_EFIMERO', '{}'::jsonb, '{}'::jsonb, '', '{}'::jsonb, '{}'::jsonb)
		ON CONFLICT (id) DO NOTHING;
	`, tplID, tenantID, fmt.Sprintf("Banco Test Runner %s", tplID[:8]))
	if err != nil {
		t.Fatalf("failed to seed template: %v", err)
	}

	_, err = db.GetDB().Exec(`
		INSERT INTO exercises (id, tenant_id, subject_id, template_id, title, type, environment_type, language, status, time_limit_ms, memory_limit_mb)
		VALUES
			($1, $4, $2, $6, 'Búsqueda Binaria - Banco', 'algorithm', 'JUEZ_EFIMERO', 'python', 'published', 2000, 128),
			($3, $4, $5, $6, 'API REST - Banco', 'project', 'IDE_PERSISTENTE', 'typescript', 'published', 5000, 512)
		ON CONFLICT (id) DO NOTHING;
	`, judgeEx, subjectJudge, ideEx, tenantID, subjectIDE, tplID)
	if err != nil {
		t.Fatalf("failed to seed exercises: %v", err)
	}

	student := uuid.NewString()
	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Alumno', 'Banco', $2, 'student', $3)
		ON CONFLICT (id) DO NOTHING;
	`, student, fmt.Sprintf("alumno_bank_%s@uab.edu.bo", student[:8]), tenantID)
	_, _ = db.GetDB().Exec(`
		INSERT INTO enrollments (tenant_id, student_id, subject_id)
		VALUES ($1, $2, $3)
	`, tenantID, student, subjectIDE)

	// Entregas: 1 WA sin override (juez -> por revisar) y 1 proyecto de IDE sin
	// veredicto que espera corrección (review_status pending).
	_, _ = db.GetDB().Exec(`
		INSERT INTO submissions (id, tenant_id, exercise_id, student_id, code, verdict, review_status, execution_time_ms, memory_used_mb, manual_override)
		VALUES
			($1, $4, $2, $5, 'code', 'WA', 'not_required', 10, 5, FALSE),
			($3, $4, $6, $5, 'code', NULL, 'pending', 0, 0, FALSE)
		ON CONFLICT (id) DO NOTHING;
	`, uuid.NewString(), judgeEx, uuid.NewString(), tenantID, student, ideEx)

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/teacher/exercises/bank", nil)
	req.Header.Set("X-Test-User-Id", teacherID)
	req.Header.Set("X-Test-User-Role", "teacher")
	req.Header.Set("X-Test-Tenant-Id", tenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /teacher/exercises/bank failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 OK, got %d: %s", resp.StatusCode, string(b))
	}

	var body struct {
		Data []domain.TeacherExerciseBankItem `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	items := body.Data
	if len(items) != 2 {
		t.Fatalf("expected 2 bank items, got %d: %+v", len(items), items)
	}

	byID := map[string]*domain.TeacherExerciseBankItem{}
	for i := range items {
		byID[items[i].ID] = &items[i]
	}

	judgeItem := byID[judgeEx]
	if judgeItem == nil {
		t.Fatalf("juez exercise missing from bank: %+v", byID)
	}
	if judgeItem.EnvironmentType != "JUEZ_EFIMERO" || judgeItem.Language != "python" {
		t.Errorf("juez item: esperado JUEZ_EFIMERO/python, got %s/%s", judgeItem.EnvironmentType, judgeItem.Language)
	}
	if judgeItem.MemoryLimitMB != 128 || judgeItem.TimeLimitMS != 2000 {
		t.Errorf("juez item límites incorrectos: %d MB / %d ms", judgeItem.MemoryLimitMB, judgeItem.TimeLimitMS)
	}
	// Semántica juez: WA sin override cuenta como por revisar.
	if judgeItem.ReviewPending != 1 {
		t.Errorf("juez item review_pending esperado 1 (WA), got %d", judgeItem.ReviewPending)
	}

	ideItem := byID[ideEx]
	if ideItem == nil {
		t.Fatalf("IDE exercise missing from bank: %+v", byID)
	}
	if ideItem.EnvironmentType != "IDE_PERSISTENTE" || ideItem.Language != "typescript" {
		t.Errorf("IDE item: esperado IDE_PERSISTENTE/typescript, got %s/%s", ideItem.EnvironmentType, ideItem.Language)
	}
	// Semántica IDE: lo pendiente lo declara review_status, no el veredicto.
	if ideItem.ReviewPending != 1 {
		t.Errorf("IDE item review_pending esperado 1 (review_status pending), got %d", ideItem.ReviewPending)
	}
	if ideItem.StudentsCount != 1 {
		t.Errorf("IDE item students_count esperado 1, got %d", ideItem.StudentsCount)
	}
}
