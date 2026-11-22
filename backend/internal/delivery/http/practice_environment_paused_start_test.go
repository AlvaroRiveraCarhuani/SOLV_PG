package httpdelivery_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

// pausedStartLinker responde la pausa del aula para el arranque del entorno.
type pausedStartLinker struct {
	paused bool
}

func (l *pausedStartLinker) AttachPracticeEnvironmentToOpenSession(context.Context, string, string, string, string) error {
	return nil
}

func (l *pausedStartLinker) StudentSessionPaused(context.Context, string, string, string, string) (bool, error) {
	return l.paused, nil
}

// newFrozenExamStartFixture arma un examen ya congelado para el arranque del
// alumno. Con aulaPaused modela la pausa en vivo; sin ella, el cierre.
func newFrozenExamStartFixture(t *testing.T, aulaPaused bool) (*httpdelivery.PracticeEnvironmentHandler, *ideOrchestrator) {
	t.Helper()

	exercise := ideExercise(string(domain.EnvironmentTypeIDEPersistente))
	exercise.Purpose = string(domain.ExercisePurposeExam)
	exerciseID := exercise.ID
	frozenAt := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)

	repo := &idePracticeEnvironmentRepo{
		existing: &domain.PracticeEnvironment{
			ID:                 "f7f7f7f7-0000-4000-8000-000000000001",
			TenantID:           ideTenant,
			StudentID:          "student-paused",
			SubjectID:          ideSubjectID,
			ExerciseID:         &exerciseID,
			Status:             domain.PracticeEnvironmentStatusFrozen,
			FrozenAt:           &frozenAt,
			EnvironmentProfile: domain.EnvironmentProfileExam,
		},
	}
	orch := &ideOrchestrator{}
	svc := services.NewPracticeEnvironmentService(repo, orch, &ideHostMonitor{}).
		WithProvisioning(
			&ideSubjectRepo{subjects: map[string]*domain.Subject{
				ideSubjectID: {ID: ideSubjectID, Name: "Bases de Datos"},
			}},
			nil,
		).
		WithExerciseSubmission(
			&ideExerciseRepo{exercises: map[string]*domain.Exercise{idePractice: exercise}},
			nil,
		).
		WithClassSessions(&pausedStartLinker{paused: aulaPaused})

	return httpdelivery.NewPracticeEnvironmentHandler(svc, validator.New()), orch
}

func callStartPracticeEnvironment(t *testing.T, handler *httpdelivery.PracticeEnvironmentHandler) (*httptest.ResponseRecorder, ideResponse) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/environments/start", handler.StartPracticeEnvironment)

	payload := fmt.Sprintf(`{"subject_id":%q,"exercise_id":%q}`, ideSubjectID, idePractice)
	req := httptest.NewRequestWithContext(
		ideaCtx("student-paused", "student"),
		http.MethodPost,
		"/api/v1/environments/start",
		strings.NewReader(payload),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body ideResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON (%d): %s", rec.Code, rec.Body.String())
	}
	return rec, body
}

// 409: el aula pausada deja el examen en solo lectura, pero no lo termina. El
// mensaje tiene que hablar de la pausa y no anunciar un cierre.
func TestStartPracticeEnvironment_PausedExamReportsPause(t *testing.T) {
	handler, orch := newFrozenExamStartFixture(t, true)

	rec, body := callStartPracticeEnvironment(t, handler)
	if rec.Code != http.StatusConflict {
		t.Fatalf("esperado 409, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if body.Error != "environment_frozen_exam_paused" {
		t.Errorf("código de error inesperado: %q", body.Error)
	}
	if !strings.Contains(body.Message, "pausó el examen") {
		t.Errorf("el mensaje debía explicar la pausa del examen: %q", body.Message)
	}
	if strings.Contains(body.Message, "terminó") {
		t.Errorf("un examen pausado no terminó: %q", body.Message)
	}
	if orch.started != nil {
		t.Error("un examen pausado no debía levantarse desde el arranque del alumno")
	}
}

// 409: un examen cerrado conserva el mensaje de cierre y no se confunde con la
// pausa en vivo.
func TestStartPracticeEnvironment_ClosedExamReportsEnd(t *testing.T) {
	handler, orch := newFrozenExamStartFixture(t, false)

	rec, body := callStartPracticeEnvironment(t, handler)
	if rec.Code != http.StatusConflict {
		t.Fatalf("esperado 409, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if body.Error != "environment_frozen" {
		t.Errorf("código de error inesperado: %q", body.Error)
	}
	if !strings.Contains(body.Message, "El examen terminó") {
		t.Errorf("el mensaje debía anunciar el cierre del examen: %q", body.Message)
	}
	if strings.Contains(body.Message, "pausó") {
		t.Errorf("un examen cerrado no está pausado: %q", body.Message)
	}
	if orch.started != nil {
		t.Error("un examen cerrado no debía levantarse de nuevo")
	}
}
