package httpdelivery_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
)

func callEnvironmentLookup(t *testing.T, f *ideFixture, ctxPath string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/student/environments", f.handler.GetStudentPracticeEnvironment)

	req := httptest.NewRequestWithContext(ideaCtx(ideOwnerID, "student"), http.MethodGet, ctxPath, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON (%d): %s", rec.Code, rec.Body.String())
	}
	return rec, body
}

// 200: el alumno con entorno previo recibe su id sin que se arranque nada.
func TestGetStudentPracticeEnvironment_ReturnsOwnedEnvironmentWithoutStarting(t *testing.T) {
	f := newIDEFixture(t, ideOwnerID, string(domain.EnvironmentTypeIDEPersistente))
	exerciseID := idePractice
	f.repo.existing = &domain.PracticeEnvironment{
		ID:         "environment-previous",
		StudentID:  ideOwnerID,
		SubjectID:  ideSubjectID,
		ExerciseID: &exerciseID,
		Status:     domain.PracticeEnvironmentStatusHibernated,
	}

	rec, body := callEnvironmentLookup(t, f,
		"/api/v1/student/environments?subject_id="+ideSubjectID+"&exercise_id="+idePractice)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	data, _ := body["data"].(map[string]any)
	environment, _ := data["environment"].(map[string]any)
	if environment == nil {
		t.Fatalf("debía devolver el entorno existente: %s", rec.Body.String())
	}
	if environment["environment_id"] != "environment-previous" {
		t.Errorf("entorno inesperado: %v", environment["environment_id"])
	}
	if environment["status"] != domain.PracticeEnvironmentStatusHibernated {
		t.Errorf("estado inesperado: %v", environment["status"])
	}
	if f.orch.started != nil {
		t.Error("la consulta no debe arrancar el contenedor")
	}
	if f.repo.created != nil {
		t.Error("la consulta no debe crear un entorno")
	}
}

// 200 con null: sin entorno previo, la pantalla pide abrirlo una vez.
func TestGetStudentPracticeEnvironment_ReturnsNullWithoutPreviousEnvironment(t *testing.T) {
	f := newIDEFixture(t, ideOwnerID, string(domain.EnvironmentTypeIDEPersistente))

	rec, body := callEnvironmentLookup(t, f,
		"/api/v1/student/environments?subject_id="+ideSubjectID+"&exercise_id="+idePractice)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	data, _ := body["data"].(map[string]any)
	if environment, ok := data["environment"]; !ok || environment != nil {
		t.Errorf("sin entorno previo debía viajar null: %s", rec.Body.String())
	}
	if f.orch.started != nil {
		t.Error("la consulta no debe arrancar el contenedor")
	}
}

// 400: la consulta necesita la práctica completa, no un id suelto.
func TestGetStudentPracticeEnvironment_RequiresPracticeIDs(t *testing.T) {
	f := newIDEFixture(t, ideOwnerID, string(domain.EnvironmentTypeIDEPersistente))

	rec, body := callEnvironmentLookup(t, f, "/api/v1/student/environments?exercise_id="+idePractice)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if body["error"] != "missing_practice" {
		t.Errorf("código de error inesperado: %v", body["error"])
	}
}
