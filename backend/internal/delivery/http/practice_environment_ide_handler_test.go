package httpdelivery_test

import (
	"context"
	"encoding/json"
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

// Los dobles embeben la interfaz completa y solo implementan lo que el camino del
// IDE toca: así el test falla con un panic explícito si el handler empieza a
// depender de otra operación sin que este archivo se entere.
type idePracticeEnvironmentRepo struct {
	domain.PracticeEnvironmentRepository
	existing *domain.PracticeEnvironment
	created  *domain.PracticeEnvironment
	// getByIDErr modela la lectura por id de los endpoints de ciclo de vida. Sin
	// valor, el entorno no existe.
	getByIDErr error
}

func (r *idePracticeEnvironmentRepo) GetByID(_ context.Context, _ string) (*domain.PracticeEnvironment, error) {
	if r.getByIDErr != nil {
		return nil, r.getByIDErr
	}
	return nil, domain.ErrNotFound
}

func (r *idePracticeEnvironmentRepo) GetByStudentSubjectExercise(_ context.Context, _, _, _ string) (*domain.PracticeEnvironment, error) {
	return r.existing, nil
}

func (r *idePracticeEnvironmentRepo) GetActivePracticeEnvironments(_ context.Context) ([]*domain.PracticeEnvironment, error) {
	return nil, nil
}

func (r *idePracticeEnvironmentRepo) Create(_ context.Context, environment *domain.PracticeEnvironment) error {
	r.created = environment
	return nil
}

func (r *idePracticeEnvironmentRepo) UpdateContainerID(_ context.Context, _, _ string) error {
	return nil
}

func (r *idePracticeEnvironmentRepo) UpdateStatus(_ context.Context, _, _ string) error { return nil }

func (r *idePracticeEnvironmentRepo) RecordHeartbeat(_ context.Context, _ string) error { return nil }

type ideOrchestrator struct {
	domain.PracticeEnvironmentOrchestrator
	started *domain.PracticeEnvironmentContainerConfig
}

func (o *ideOrchestrator) EnsureVolumeExists(_ context.Context, _ string) error { return nil }

func (o *ideOrchestrator) EnsureICCDisabledNetworkExists(_ context.Context, _ string) error {
	return nil
}

func (o *ideOrchestrator) StartPracticeEnvironmentContainer(_ context.Context, config domain.PracticeEnvironmentContainerConfig) (string, error) {
	cfg := config
	o.started = &cfg
	return "container-ide", nil
}

type ideHostMonitor struct {
	domain.HostMonitor
}

func (m *ideHostMonitor) CanAllocateMemory(_ int64) bool { return true }

type ideExerciseRepo struct {
	domain.ExerciseRepository
	exercises map[string]*domain.Exercise
}

func (r *ideExerciseRepo) GetByIDAndTenant(_ context.Context, id, _ string) (*domain.Exercise, error) {
	return r.exercises[id], nil
}

func (r *ideExerciseRepo) GetByID(_ context.Context, id string) (*domain.Exercise, error) {
	return r.exercises[id], nil
}

type ideSubjectRepo struct {
	domain.SubjectRepository
	subjects map[string]*domain.Subject
}

func (r *ideSubjectRepo) GetByID(_ context.Context, _, id string) (*domain.Subject, error) {
	return r.subjects[id], nil
}

const (
	ideOwnerID    = "teacher-ide-owner"
	ideIntruderID = "teacher-ide-intruder"
	ideTenant     = "00000000-0000-0000-0000-000000000001"
	ideSubjectID  = "a1de0000-0000-4000-8000-000000000901"
	idePractice   = "a1de0000-0000-4000-8000-000000000902"
	ideJudgeID    = "a1de0000-0000-4000-8000-000000000903"
	ideMissingID  = "a1de0000-0000-4000-8000-0000000009ff"
)

func ideExercise(environmentType string) *domain.Exercise {
	subjectID := ideSubjectID
	return &domain.Exercise{
		ID:              idePractice,
		SubjectID:       &subjectID,
		Title:           "Práctica con IDE",
		EnvironmentType: environmentType,
		TenantID:        ideTenant,
	}
}

type ideFixture struct {
	handler  *httpdelivery.PracticeEnvironmentHandler
	repo     *idePracticeEnvironmentRepo
	orch     *ideOrchestrator
	exercise *domain.Exercise
}

func newIDEFixture(t *testing.T, owner string, environmentType string) *ideFixture {
	t.Helper()

	exercise := ideExercise(environmentType)
	judgeExercise := ideExercise(string(domain.EnvironmentTypeJuezEfimero))
	judgeExercise.ID = ideJudgeID

	repo := &idePracticeEnvironmentRepo{}
	orch := &ideOrchestrator{}
	svc := services.NewPracticeEnvironmentService(repo, orch, &ideHostMonitor{}).
		WithProvisioning(
			&ideSubjectRepo{subjects: map[string]*domain.Subject{
				ideSubjectID: {ID: ideSubjectID, Name: "Bases de Datos", TeacherID: &owner},
			}},
			nil,
		).
		WithExerciseSubmission(
			&ideExerciseRepo{exercises: map[string]*domain.Exercise{
				idePractice: exercise,
				ideJudgeID:  judgeExercise,
			}},
			nil,
		)

	return &ideFixture{
		handler:  httpdelivery.NewPracticeEnvironmentHandler(svc, validator.New()),
		repo:     repo,
		orch:     orch,
		exercise: exercise,
	}
}

func ideaCtx(userID, role string) context.Context {
	ctx := context.WithValue(context.Background(), domain.UserIDKey, userID)
	ctx = context.WithValue(ctx, domain.TenantIDKey, ideTenant)
	return context.WithValue(ctx, domain.UserRoleKey, role)
}

type ideResponse struct {
	Data struct {
		EnvironmentID string  `json:"environment_id"`
		ExerciseID    *string `json:"exercise_id"`
		SubjectID     string  `json:"subject_id"`
		Status        string  `json:"status"`
		AccessURL     string  `json:"access_url"`
		VolumeName    string  `json:"volume_name"`
	} `json:"data"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func callOpenPracticeIDE(t *testing.T, f *ideFixture, ctx context.Context, exerciseID string) (*httptest.ResponseRecorder, ideResponse) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/ide", f.handler.OpenPracticeIDE)

	req := httptest.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"/api/v1/teacher/courses/"+ideSubjectID+"/exercises/"+exerciseID+"/ide",
		nil,
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var body ideResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON (%d): %s", rec.Code, rec.Body.String())
	}
	return rec, body
}

// 200: el docente recibe la URL del IDE y el entorno queda acotado a la práctica.
func TestOpenPracticeIDE_ReturnsAccessURLForOwnedPractice(t *testing.T) {
	f := newIDEFixture(t, ideOwnerID, string(domain.EnvironmentTypeIDEPersistente))

	rec, body := callOpenPracticeIDE(t, f, ideaCtx(ideOwnerID, "teacher"), idePractice)
	if rec.Code != http.StatusOK {
		t.Fatalf("esperado 200, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if body.Data.EnvironmentID == "" {
		t.Error("la respuesta debía incluir environment_id")
	}
	if body.Data.ExerciseID == nil || *body.Data.ExerciseID != idePractice {
		t.Errorf("el entorno debía quedar acotado a la práctica %s: %v", idePractice, body.Data.ExerciseID)
	}
	if !strings.HasPrefix(body.Data.AccessURL, "http://localhost:") {
		t.Errorf("access_url inesperada: %q", body.Data.AccessURL)
	}
	wantVolume := practiceVolumeName(ideOwnerID, ideSubjectID, idePractice)
	if body.Data.VolumeName != wantVolume {
		t.Errorf("volumen inesperado: %s, esperado: %s", body.Data.VolumeName, wantVolume)
	}
	if f.repo.created == nil {
		t.Error("el entorno debía registrarse")
	}
	if f.orch.started == nil || f.orch.started.VolumeName != wantVolume {
		t.Errorf("el contenedor debía montar el volumen de la práctica: %+v", f.orch.started)
	}
}

// 400: una práctica del juez no tiene repositorio persistente que abrir.
func TestOpenPracticeIDE_RejectsJudgePractice(t *testing.T) {
	f := newIDEFixture(t, ideOwnerID, string(domain.EnvironmentTypeJuezEfimero))

	rec, body := callOpenPracticeIDE(t, f, ideaCtx(ideOwnerID, "teacher"), ideJudgeID)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if body.Error != "not_project_practice" {
		t.Errorf("código de error inesperado: %q", body.Error)
	}
	if f.orch.started != nil {
		t.Error("una práctica del juez no debía levantar un contenedor")
	}
}

// 403: la materia de otro docente no se abre desde una sesión docente.
func TestOpenPracticeIDE_RejectsForeignTeacher(t *testing.T) {
	f := newIDEFixture(t, ideOwnerID, string(domain.EnvironmentTypeIDEPersistente))

	rec, body := callOpenPracticeIDE(t, f, ideaCtx(ideIntruderID, "teacher"), idePractice)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperado 403, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if body.Error != "forbidden" {
		t.Errorf("código de error inesperado: %q", body.Error)
	}
	if f.repo.created != nil {
		t.Error("un docente ajeno no debía crear un entorno")
	}
}

// 403: el rol alumno no abre el IDE del docente aunque conozca la práctica.
func TestOpenPracticeIDE_RejectsStudentRole(t *testing.T) {
	f := newIDEFixture(t, ideOwnerID, string(domain.EnvironmentTypeIDEPersistente))

	rec, body := callOpenPracticeIDE(t, f, ideaCtx("student-1", "student"), idePractice)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperado 403, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if body.Error != "forbidden" {
		t.Errorf("código de error inesperado: %q", body.Error)
	}
}

// 404: una práctica inexistente no se confunde con un fallo del entorno.
func TestOpenPracticeIDE_ReturnsNotFoundForMissingPractice(t *testing.T) {
	f := newIDEFixture(t, ideOwnerID, string(domain.EnvironmentTypeIDEPersistente))

	rec, body := callOpenPracticeIDE(t, f, ideaCtx(ideOwnerID, "teacher"), ideMissingID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("esperado 404, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if body.Error != "practice_not_found" {
		t.Errorf("código de error inesperado: %q", body.Error)
	}
}

// 409: un volumen congelado por examen cerrado no se reabre desde la ficha.
func TestOpenPracticeIDE_RejectsFrozenEnvironment(t *testing.T) {
	f := newIDEFixture(t, ideOwnerID, string(domain.EnvironmentTypeIDEPersistente))
	frozenAt := time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)
	exerciseID := idePractice
	f.repo.existing = &domain.PracticeEnvironment{
		ID:         "b7c1e5f2-9a53-4f0e-9a5b-3f5b0f0e7c11",
		TenantID:   ideTenant,
		StudentID:  ideOwnerID,
		SubjectID:  ideSubjectID,
		ExerciseID: &exerciseID,
		Status:     domain.PracticeEnvironmentStatusFrozen,
		FrozenAt:   &frozenAt,
	}

	rec, body := callOpenPracticeIDE(t, f, ideaCtx(ideOwnerID, "teacher"), idePractice)
	if rec.Code != http.StatusConflict {
		t.Fatalf("esperado 409, obtenido %d: %s", rec.Code, rec.Body.String())
	}
	if body.Error != "environment_frozen" {
		t.Errorf("código de error inesperado: %q", body.Error)
	}
	if f.orch.started != nil {
		t.Error("un volumen congelado no debía levantarse de nuevo")
	}
}

// Un laboratorio que ya no existe responde 404 en el ciclo de vida, no un 500:
// el servicio distingue "no hay fila" de "no se pudo leer".
func TestPracticeEnvironmentHandler_LifecycleOnMissingPracticeEnvironmentReturnsNotFound(t *testing.T) {
	missing := "dddddddd-4444-4444-8444-444444444444"

	cases := []struct {
		name    string
		method  string
		pattern string
		handle  func(http.ResponseWriter, *http.Request)
	}{
		{
			name:    "terminate",
			method:  http.MethodDelete,
			pattern: "DELETE /api/v1/environments/{id}",
		},
		{
			name:    "pause",
			method:  http.MethodPost,
			pattern: "POST /api/v1/environments/{id}/pause",
		},
		{
			name:    "restart",
			method:  http.MethodPost,
			pattern: "POST /api/v1/environments/{id}/restart",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newIDEFixture(t, ideOwnerID, string(domain.EnvironmentTypeIDEPersistente))

			mux := http.NewServeMux()
			switch tc.name {
			case "terminate":
				mux.HandleFunc(tc.pattern, f.handler.TerminatePracticeEnvironment)
			case "pause":
				mux.HandleFunc(tc.pattern, f.handler.PausePracticeEnvironment)
			case "restart":
				mux.HandleFunc(tc.pattern, f.handler.RestartPracticeEnvironment)
			}

			path := "/api/v1/environments/" + missing
			if tc.name != "terminate" {
				path += "/" + tc.name
			}
			req := httptest.NewRequestWithContext(ideaCtx(ideOwnerID, "teacher"), tc.method, path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("esperado 404, obtenido %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

// practiceVolumeName replica el nombre público del volumen sin exportar el detalle
// de armado: si cambia, este test lo delata.
func practiceVolumeName(studentID, subjectID, exerciseID string) string {
	return "solv_workspace_" + studentID + "_" + subjectID + "_" + exerciseID
}
