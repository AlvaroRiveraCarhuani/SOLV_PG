package httpdelivery_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"solv-backend/internal/core/domain"
	httpdelivery "solv-backend/internal/delivery/http"
)

type stubRubricService struct {
	calls int
	actor domain.RubricActor
}

func (s *stubRubricService) GetByExerciseID(context.Context, uuid.UUID) (*domain.Rubric, error) {
	return nil, domain.ErrRubricNotFound
}

func (s *stubRubricService) CreateOrUpdate(context.Context, *domain.Rubric) error { return nil }

func (s *stubRubricService) CalculateScore([]domain.RubricCriterion, map[string]string) (float64, []domain.CriterionBreakdown, error) {
	return 0, nil, nil
}

func (s *stubRubricService) EvaluateSubmission(_ context.Context, _ uuid.UUID, _ domain.RubricEvaluationRequest, actor domain.RubricActor) (*domain.RubricEvaluationResponse, error) {
	s.calls++
	s.actor = actor
	return &domain.RubricEvaluationResponse{Score: 50}, nil
}

func callRubricEvaluation(t *testing.T, service domain.RubricService, role, userID string) *httptest.ResponseRecorder {
	t.Helper()

	handler := httpdelivery.NewRubricHandler(service)
	router := http.NewServeMux()
	router.HandleFunc("POST /api/v1/submissions/{id}/rubric-evaluation", handler.EvaluateSubmission)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/submissions/"+uuid.NewString()+"/rubric-evaluation",
		strings.NewReader(`{"selections":{"c1":"Logrado"},"comments":"Corrección del docente"}`),
	)
	if role != "" {
		req.Header.Set("X-User-Role", role)
	}
	if userID != "" {
		req.Header.Set("X-User-Id", userID)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// Calificar con rúbrica no es una acción del alumno: el rol se rechaza antes
// de tocar el servicio.
func TestRubricEvaluation_RejectsStudentAndAnonymous(t *testing.T) {
	for name, headers := range map[string][2]string{
		"alumno":          {"student", "student-1"},
		"sin rol":         {"", "user-1"},
		"rol desconocido": {"assistant", "user-1"},
	} {
		t.Run(name, func(t *testing.T) {
			service := &stubRubricService{}
			rr := callRubricEvaluation(t, service, headers[0], headers[1])
			if rr.Code != http.StatusForbidden {
				t.Fatalf("esperado 403, obtenido %d: %s", rr.Code, rr.Body.String())
			}
			if service.calls != 0 {
				t.Fatalf("el servicio no debe invocarse para un rol sin permiso; llamadas: %d", service.calls)
			}
		})
	}
}

func TestRubricEvaluation_PassesTeacherAndAdminIdentity(t *testing.T) {
	service := &stubRubricService{}
	rr := callRubricEvaluation(t, service, "teacher", "teacher-1")
	if rr.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d: %s", rr.Code, rr.Body.String())
	}
	if service.calls != 1 || service.actor.UserID != "teacher-1" || service.actor.IsAdmin {
		t.Fatalf("actor inesperado: %+v (llamadas %d)", service.actor, service.calls)
	}

	adminService := &stubRubricService{}
	rr = callRubricEvaluation(t, adminService, "admin", "admin-1")
	if rr.Code != http.StatusOK {
		t.Fatalf("un administrador debe poder calificar, obtenido %d: %s", rr.Code, rr.Body.String())
	}
	if !adminService.actor.IsAdmin {
		t.Fatalf("el actor administrador debe declararse como tal: %+v", adminService.actor)
	}
}

// Una petición sin identidad no puede atribuirse a ningún docente.
func TestRubricEvaluation_RejectsTeacherWithoutIdentity(t *testing.T) {
	service := &stubRubricService{}
	rr := callRubricEvaluation(t, service, "teacher", "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("esperado 401, obtenido %d: %s", rr.Code, rr.Body.String())
	}
	if service.calls != 0 {
		t.Fatalf("el servicio no debe invocarse sin identidad; llamadas: %d", service.calls)
	}
}
