package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockExamOfflineVerifier struct {
	result *domain.ExamProjectOfflineCheckResult
	err    error
}

func (m *mockExamOfflineVerifier) VerifyExamProjectOffline(ctx context.Context, check domain.ExamProjectOfflineCheck) (*domain.ExamProjectOfflineCheckResult, error) {
	return m.result, m.err
}

type mockProjectRepoForOffline struct {
	project *domain.WorkspaceProject
	draft   *domain.WorkspaceProjectDraft
}

func (m *mockProjectRepoForOffline) GetProjectByExercise(ctx context.Context, tenantID, exerciseID string) (*domain.WorkspaceProject, error) {
	if m.project == nil {
		return nil, domain.ErrWorkspaceProjectNotFound
	}
	return m.project, nil
}
func (m *mockProjectRepoForOffline) CreateProject(ctx context.Context, p *domain.WorkspaceProject) (*domain.WorkspaceProject, error) {
	return p, nil
}
func (m *mockProjectRepoForOffline) GetDraftByProject(ctx context.Context, tenantID, projectID string) (*domain.WorkspaceProjectDraft, error) {
	if m.draft != nil {
		return m.draft, nil
	}
	return nil, domain.ErrWorkspaceProjectNotFound
}
func (m *mockProjectRepoForOffline) SaveDraft(ctx context.Context, d *domain.WorkspaceProjectDraft, v int64) (*domain.WorkspaceProjectDraft, error) {
	return d, nil
}
func (m *mockProjectRepoForOffline) Publish(ctx context.Context, tenantID, projectID, userID string, manifest []domain.WorkspaceProjectManifestEntry) (*domain.WorkspaceProjectVersion, error) {
	return nil, nil
}
func (m *mockProjectRepoForOffline) ListVersions(ctx context.Context, tenantID, projectID string) ([]*domain.WorkspaceProjectVersion, error) {
	return nil, nil
}
func (m *mockProjectRepoForOffline) GetVersion(ctx context.Context, tenantID, versionID string) (*domain.WorkspaceProjectVersion, error) {
	return nil, nil
}
func (m *mockProjectRepoForOffline) GetLegacyReference(ctx context.Context, tenantID, exerciseID string) (*domain.WorkspaceLegacyProjectReference, error) {
	return nil, nil
}

func sampleTextSnapshot() string {
	return "# === FILE: package.json ===\n{\n  \"name\": \"test-app\",\n  \"scripts\": { \"start\": \"node index.js\" }\n}\n# === FILE: index.js ===\nconsole.log(\"hello offline\");\n"
}

func TestVerifyExamProjectOffline_RejectsNetworkDependentProject(t *testing.T) {
	exRepo := &mockExerciseRepoGov{exercises: make(map[string]*domain.Exercise)}
	govRepo := &mockGovRepoForTemplate{
		templates: map[string]*domain.AdminTemplateReviewItem{
			"tpl-offline": {
				ID:                "tpl-offline",
				DockerImage:       "node:20-bookworm-slim",
				Status:            "approved",
				TargetEnvironment: "IDE_PERSISTENTE",
				BaseRamMB:         512,
			},
		},
	}

	verifier := &mockExamOfflineVerifier{
		result: &domain.ExamProjectOfflineCheckResult{
			Status: domain.ExamOfflineFailed,
			Step:   "setup_script",
			Output: "npm ERR! code ENOTFOUND\nnpm ERR! getaddrinfo EAI_AGAIN registry.npmjs.org",
		},
	}

	projectRepo := &mockProjectRepoForOffline{
		project: &domain.WorkspaceProject{ID: "proj-1", ExerciseID: "ex-offline-fail"},
		draft:   &domain.WorkspaceProjectDraft{ID: "draft-1", ProjectID: "proj-1", TextSnapshot: sampleTextSnapshot()},
	}

	svc := services.NewEvaluationService(exRepo, nil, nil, nil)
	svc.SetAdminGovernanceRepository(govRepo)
	svc.SetExamOfflineVerifier(verifier)
	svc.SetWorkspaceProjectService(services.NewWorkspaceProjectService(projectRepo, true))

	ex := &domain.Exercise{
		ID:              "ex-offline-fail",
		Title:           "Examen con dependencia online",
		Type:            domain.ExerciseTypeProject,
		EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		Purpose:         string(domain.ExercisePurposeExam),
		ExamNetwork:     domain.ExamNetworkNone,
		TemplateID:      "tpl-offline",
	}
	if err := svc.CreateExercise(context.Background(), ex); err != nil {
		t.Fatalf("no se pudo crear el ejercicio: %v", err)
	}

	ex.Config = domain.ExerciseConfig{
		IDE: &domain.IDEConfig{
			AppRunCommand: "npm start",
		},
	}

	_, err := svc.PublishExercise(context.Background(), ex.ID, ex.TenantID)
	if err == nil {
		t.Fatal("se esperaba rechazo al publicar ejercicio que necesita red exterior")
	}

	var offlineErr *domain.ExamOfflineVerificationError
	if !errors.As(err, &offlineErr) {
		t.Fatalf("se esperaba error ExamOfflineVerificationError, obtenido: %v", err)
	}
	if offlineErr.Step != "setup_script" {
		t.Errorf("paso fallido inesperado: %q", offlineErr.Step)
	}
}

func TestVerifyExamProjectOffline_AllowsOfflineReadyProject(t *testing.T) {
	exRepo := &mockExerciseRepoGov{exercises: make(map[string]*domain.Exercise)}
	govRepo := &mockGovRepoForTemplate{
		templates: map[string]*domain.AdminTemplateReviewItem{
			"tpl-offline-ok": {
				ID:                "tpl-offline-ok",
				DockerImage:       "python:3.12-slim-bookworm",
				Status:            "approved",
				TargetEnvironment: "IDE_PERSISTENTE",
				BaseRamMB:         512,
			},
		},
	}

	verifier := &mockExamOfflineVerifier{
		result: &domain.ExamProjectOfflineCheckResult{
			Status: domain.ExamOfflinePassed,
			Step:   "run_command",
			Output: "Application listening on 0.0.0.0:3000",
		},
	}

	projectRepo := &mockProjectRepoForOffline{
		project: &domain.WorkspaceProject{ID: "proj-2", ExerciseID: "ex-offline-ok"},
		draft:   &domain.WorkspaceProjectDraft{ID: "draft-2", ProjectID: "proj-2", TextSnapshot: sampleTextSnapshot()},
	}

	svc := services.NewEvaluationService(exRepo, nil, nil, nil)
	svc.SetAdminGovernanceRepository(govRepo)
	svc.SetExamOfflineVerifier(verifier)
	svc.SetWorkspaceProjectService(services.NewWorkspaceProjectService(projectRepo, true))

	ex := &domain.Exercise{
		ID:              "ex-offline-ok",
		Title:           "Examen autocontenido sin red",
		Type:            domain.ExerciseTypeProject,
		EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		Purpose:         string(domain.ExercisePurposeExam),
		ExamNetwork:     domain.ExamNetworkNone,
		TemplateID:      "tpl-offline-ok",
	}
	if err := svc.CreateExercise(context.Background(), ex); err != nil {
		t.Fatalf("no se pudo crear el ejercicio: %v", err)
	}

	published, err := svc.PublishExercise(context.Background(), ex.ID, ex.TenantID)
	if err != nil {
		t.Fatalf("no se esperaba error al publicar ejercicio autocontenido: %v", err)
	}
	if published.Status != "published" {
		t.Fatalf("estado esperado published, obtenido: %q", published.Status)
	}
}
