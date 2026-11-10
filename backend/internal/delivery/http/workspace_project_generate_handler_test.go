package httpdelivery

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type handlerGenerationExerciseRepo struct {
	domain.ExerciseRepository
	byID map[string]*domain.Exercise
}

func (r *handlerGenerationExerciseRepo) GetByIDAndTenant(_ context.Context, id, tenantID string) (*domain.Exercise, error) {
	exercise, ok := r.byID[id]
	if !ok || exercise == nil || exercise.TenantID != tenantID {
		return nil, sql.ErrNoRows
	}
	return exercise, nil
}

type handlerGenerationSubjectRepo struct {
	domain.SubjectRepository
	subjects map[string]*domain.Subject
}

func (r *handlerGenerationSubjectRepo) GetByID(_ context.Context, tenantID, id string) (*domain.Subject, error) {
	subject, ok := r.subjects[id]
	if !ok || subject == nil {
		return nil, sql.ErrNoRows
	}
	return subject, nil
}

func generationHandler(t *testing.T, content string) (*WorkspaceProjectHandler, string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": content}}},
		})
	}))
	t.Cleanup(server.Close)

	settings := services.NewAISettingsService(&mockAISettingsTenantRepo{configs: map[string][]byte{"t-gen": []byte(`{}`)}}).
		WithTransport(&aiTestRewriteTransport{base: mustParseAITestURL(t, server.URL)})
	if _, err := settings.Update(context.Background(), "t-gen", domain.UpdateAISettingsDTO{
		Enabled:            true,
		Provider:           domain.AIProviderOpenAI,
		Model:              "gpt-4o-mini",
		APIKey:             "sk-test-generation-0001",
		HourlyQuotaPerUser: 15,
		TimeoutSeconds:     30,
	}); err != nil {
		t.Fatalf("no se pudo habilitar la IA de prueba: %v", err)
	}

	subjectID := "subject-1"
	teacher := "teacher-1"
	exerciseRepo := &handlerGenerationExerciseRepo{byID: map[string]*domain.Exercise{
		"ex-ide": {
			ID:              "ex-ide",
			TenantID:        "t-gen",
			SubjectID:       &subjectID,
			Title:           "Práctica de proyecto",
			EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		},
		"ex-juez": {
			ID:              "ex-juez",
			TenantID:        "t-gen",
			SubjectID:       &subjectID,
			Title:           "Práctica de juez",
			EnvironmentType: string(domain.EnvironmentTypeJuezEfimero),
		},
		"ex-otro-tenant": {
			ID:              "ex-otro-tenant",
			TenantID:        "t-otro",
			SubjectID:       &subjectID,
			Title:           "Práctica ajena",
			EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		},
	}}
	subjectRepo := &handlerGenerationSubjectRepo{subjects: map[string]*domain.Subject{
		subjectID: {ID: subjectID, TenantID: "t-gen", TeacherID: &teacher},
	}}

	svc := services.NewWorkspaceProjectService(nil, false)
	svc.SetAccessRepositories(exerciseRepo, subjectRepo)
	svc.SetAISettingsService(settings)
	return NewWorkspaceProjectHandler(svc), "t-gen"
}

func generateRequest(role, exerciseID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/exercises/"+exerciseID+"/project/generate",
		strings.NewReader(`{"description":"Un proyecto que imprima un saludo"}`))
	req.SetPathValue("id", exerciseID)
	req = withTestIdentity(req, "teacher-1", "t-gen", role)
	return req.WithContext(context.WithValue(req.Context(), domain.TenantIDKey, "t-gen"))
}

func TestWorkspaceGenerateHandlerReturnsValidatedProposal(t *testing.T) {
	raw := `{"files":[{"path":"main.py","content":"print('hola')\n"}],"summary":"Proyecto mínimo"}`
	handler, _ := generationHandler(t, raw)

	rec := httptest.NewRecorder()
	handler.GenerateProject(rec, generateRequest("teacher", "ex-ide"))
	if rec.Code != http.StatusOK {
		t.Fatalf("esperaba 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var envelope struct {
		Data services.GenerateWorkspaceProjectProposal `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("respuesta ilegible: %v", err)
	}
	if len(envelope.Data.Files) != 1 || envelope.Data.Files[0].Path != "main.py" {
		t.Fatalf("propuesta inesperada: %#v", envelope.Data.Files)
	}
	if !strings.Contains(envelope.Data.Files[0].Content, "print") {
		t.Errorf("el contenido generado debe llegar al editor: %q", envelope.Data.Files[0].Content)
	}
}

func TestWorkspaceGenerateHandlerRejectsUnauthorizedAndWrongEnvironment(t *testing.T) {
	handler, _ := generationHandler(t, `{"files":[],"summary":""}`)

	student := httptest.NewRecorder()
	handler.GenerateProject(student, generateRequest("student", "ex-ide"))
	if student.Code != http.StatusForbidden {
		t.Fatalf("un estudiante no genera proyectos: got %d", student.Code)
	}

	foreign := httptest.NewRecorder()
	handler.GenerateProject(foreign, generateRequest("teacher", "ex-otro-tenant"))
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("un ejercicio de otro tenant debe ser 404, got %d: %s", foreign.Code, foreign.Body.String())
	}

	judge := httptest.NewRecorder()
	handler.GenerateProject(judge, generateRequest("teacher", "ex-juez"))
	if judge.Code != http.StatusUnprocessableEntity {
		t.Fatalf("el juez efímero no usa proyectos iniciales: got %d: %s", judge.Code, judge.Body.String())
	}
	var envelope GlobalResponse
	if err := json.Unmarshal(judge.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("respuesta ilegible: %v", err)
	}
	if envelope.Error != "workspace_project_not_ide" {
		t.Errorf("código de error inesperado: %q", envelope.Error)
	}
}

func TestWorkspaceGenerateHandlerReportsInvalidRequestAndResponse(t *testing.T) {
	handler, _ := generationHandler(t, `{"files":[{"path":"../escape.py","content":"x\n"}],"summary":"s"}`)

	malformed := httptest.NewRequest(http.MethodPost, "/api/v1/teacher/exercises/ex-ide/project/generate",
		strings.NewReader(`{"description":`))
	malformed.SetPathValue("id", "ex-ide")
	malformed = withTestIdentity(malformed, "teacher-1", "t-gen", "teacher")
	rec := httptest.NewRecorder()
	handler.GenerateProject(rec, malformed)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperaba 400, got %d", rec.Code)
	}

	unsafe := httptest.NewRecorder()
	handler.GenerateProject(unsafe, generateRequest("teacher", "ex-ide"))
	if unsafe.Code != http.StatusUnprocessableEntity {
		t.Fatalf("una respuesta con traversal debe rechazarse con 422, got %d: %s", unsafe.Code, unsafe.Body.String())
	}
}

func TestWorkspaceGenerateHandlerReportsDisabledAI(t *testing.T) {
	subjectID := "subject-1"
	teacher := "teacher-1"
	exerciseRepo := &handlerGenerationExerciseRepo{byID: map[string]*domain.Exercise{
		"ex-ide": {
			ID:              "ex-ide",
			TenantID:        "t-gen",
			SubjectID:       &subjectID,
			Title:           "Práctica",
			EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		},
	}}
	subjectRepo := &handlerGenerationSubjectRepo{subjects: map[string]*domain.Subject{
		subjectID: {ID: subjectID, TenantID: "t-gen", TeacherID: &teacher},
	}}
	svc := services.NewWorkspaceProjectService(nil, false)
	svc.SetAccessRepositories(exerciseRepo, subjectRepo)
	svc.SetAISettingsService(services.NewAISettingsService(&mockAISettingsTenantRepo{configs: map[string][]byte{"t-gen": []byte(`{}`)}}))
	handler := NewWorkspaceProjectHandler(svc)

	rec := httptest.NewRecorder()
	handler.GenerateProject(rec, generateRequest("teacher", "ex-ide"))
	if rec.Code != http.StatusConflict {
		t.Fatalf("esperaba 409 ai_disabled, got %d: %s", rec.Code, rec.Body.String())
	}
}
