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

// TestTeacherExerciseDuplicate valida que el docente copia un laboratorio a otra
// de sus materias: la copia nace en borrador, sin entregas, con sus casos de
// prueba, y no se permite copiar a una materia ajena.
func TestTeacherExerciseDuplicate(t *testing.T) {
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
	exerciseRepo := postgres.NewPostgresExerciseRepository(db.GetDB())
	evalService := services.NewEvaluationService(exerciseRepo, nil, nil, nil)
	teacherService := services.NewTeacherService(teacherRepo, nil)
	teacherService.SetEvaluationService(evalService)
	teacherHandler := httpdelivery.NewTeacherHandler(teacherService, nil)

	tenantMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), domain.TenantIDKey, "00000000-0000-0000-0000-000000000001")
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
	defer server.Close()

	tenantID := "00000000-0000-0000-0000-000000000001"
	teacherID := uuid.NewString()
	otherTeacherID := uuid.NewString()

	_, _ = db.GetDB().Exec(`
		INSERT INTO users (id, first_name, last_name, email, role, tenant_id)
		VALUES ($1, 'Profesor', 'Duplica', $2, 'teacher', $3),
		       ($4, 'Otro', 'Docente', $5, 'teacher', $3)
		ON CONFLICT (id) DO NOTHING;
	`, teacherID, fmt.Sprintf("prof_dup_%s@uab.edu.bo", teacherID[:8]), tenantID,
		otherTeacherID, fmt.Sprintf("otro_dup_%s@uab.edu.bo", otherTeacherID[:8]))

	subjectA := uuid.NewString()
	subjectB := uuid.NewString()
	subjectOther := uuid.NewString()
	_, _ = db.GetDB().Exec(`
		INSERT INTO subjects (id, tenant_id, name, code, teacher_id)
		VALUES ($1, $4, 'Materia A - Dup', 'DUP-A', $5),
		       ($2, $4, 'Materia B - Dup', 'DUP-B', $5),
		       ($3, $4, 'Materia Ajena - Dup', 'DUP-X', $6)
		ON CONFLICT (id) DO NOTHING;
	`, subjectA, subjectB, subjectOther, tenantID, teacherID, otherTeacherID)

	tplID := uuid.NewString()
	_, err = db.GetDB().Exec(`
		INSERT INTO lab_templates (id, tenant_id, name, docker_image, base_ram_mb, target_environment, services_config, resource_profile, setup_script, tools_declared, stack_config)
		VALUES ($1, $2, $3, 'python:3.11-slim', 128, 'JUEZ_EFIMERO', '{}'::jsonb, '{}'::jsonb, '', '{}'::jsonb, '{}'::jsonb)
		ON CONFLICT (id) DO NOTHING;
	`, tplID, tenantID, fmt.Sprintf("Dup Runner %s", tplID[:8]))
	if err != nil {
		t.Fatalf("failed to seed template: %v", err)
	}

	sourceID := uuid.NewString()
	_, err = db.GetDB().Exec(`
		INSERT INTO exercises (id, tenant_id, subject_id, template_id, title, type, environment_type, language, status, time_limit_ms, memory_limit_mb)
		VALUES ($1, $2, $3, $4, 'Lab Origen - Dup', 'algorithm', 'JUEZ_EFIMERO', 'python', 'published', 2000, 128)
		ON CONFLICT (id) DO NOTHING;
	`, sourceID, tenantID, subjectA, tplID)
	if err != nil {
		t.Fatalf("failed to seed source exercise: %v", err)
	}

	// Un caso de prueba en el origen para verificar que se copia.
	_, err = db.GetDB().Exec(`
		INSERT INTO exercise_test_cases (id, exercise_id, order_index, input, expected_output, visibility, weight)
		VALUES ($1, $2, 0, '1 2', '3', 'public', 1)
		ON CONFLICT (id) DO NOTHING;
	`, uuid.NewString(), sourceID)
	if err != nil {
		t.Fatalf("failed to seed test case: %v", err)
	}

	duplicate := func(exerciseID, targetSubject, asTeacher string) *http.Response {
		body, _ := json.Marshal(map[string]string{"subject_id": targetSubject})
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/teacher/exercises/"+exerciseID+"/duplicate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-Id", asTeacher)
		req.Header.Set("X-User-Role", "teacher")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		return resp
	}

	// 1) Copia válida a otra materia del mismo docente.
	resp := duplicate(sourceID, subjectB, teacherID)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	var copyTitle, copyStatus, copySubject string
	var copyCases, copySubs int
	err = db.GetDB().QueryRow(`
		SELECT e.title, e.status, e.subject_id::text,
		       (SELECT COUNT(*) FROM exercise_test_cases WHERE exercise_id = e.id),
		       (SELECT COUNT(*) FROM submissions WHERE exercise_id = e.id)
		FROM exercises e
		WHERE e.tenant_id = $1 AND e.subject_id = $2 AND e.title = 'Lab Origen - Dup' AND e.id <> $3
	`, tenantID, subjectB, sourceID).Scan(&copyTitle, &copyStatus, &copySubject, &copyCases, &copySubs)
	if err != nil {
		t.Fatalf("copy not found in target subject: %v", err)
	}
	if copyStatus != "draft" {
		t.Errorf("copy status = %q, want draft", copyStatus)
	}
	if copySubject != subjectB {
		t.Errorf("copy subject = %s, want %s", copySubject, subjectB)
	}
	if copyCases != 1 {
		t.Errorf("copy test cases = %d, want 1", copyCases)
	}
	if copySubs != 0 {
		t.Errorf("copy submissions = %d, want 0", copySubs)
	}

	// 2) Materia ajena al docente -> 404, sin crear nada.
	resp = duplicate(sourceID, subjectOther, teacherID)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("foreign subject: expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 3) Otro docente no puede copiar un laboratorio que no es suyo -> 404.
	resp = duplicate(sourceID, subjectB, otherTeacherID)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("foreign exercise: expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
