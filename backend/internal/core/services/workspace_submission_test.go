package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type fakeVolumeOrchestrator struct {
	*mockOrchestrator
	volume         string
	writtenFiles   map[string]string
	snapshotFiles  []domain.WorkspaceVolumeFile
	snapshotCalled bool
}

func (f *fakeVolumeOrchestrator) WriteWorkspaceVolumeFiles(ctx context.Context, volumeName string, files map[string]string) error {
	f.volume = volumeName
	f.writtenFiles = files
	return nil
}

func (f *fakeVolumeOrchestrator) SnapshotWorkspaceVolume(ctx context.Context, volumeName string, maxBytes int64) ([]domain.WorkspaceVolumeFile, error) {
	f.volume = volumeName
	f.snapshotCalled = true
	return f.snapshotFiles, nil
}

type fakeSubmissionRepo struct {
	created *domain.Submission
}

func (f *fakeSubmissionRepo) Create(ctx context.Context, submission *domain.Submission) error {
	f.created = submission
	return nil
}

func (f *fakeSubmissionRepo) GetByID(ctx context.Context, tenantID, id string) (*domain.Submission, error) {
	return nil, nil
}

func (f *fakeSubmissionRepo) ListByExerciseAndStudent(ctx context.Context, tenantID, exerciseID, studentID string) ([]*domain.Submission, error) {
	return nil, nil
}

func (f *fakeSubmissionRepo) ListByExercise(ctx context.Context, tenantID, exerciseID string) ([]*domain.Submission, error) {
	return nil, nil
}

func (f *fakeSubmissionRepo) ListByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.Submission, error) {
	return nil, nil
}

func (f *fakeSubmissionRepo) UpdateOverride(ctx context.Context, tenantID, id, verdict, reason string, score *int, gradedBy *string) error {
	return nil
}

const testExerciseID = "11111111-1111-1111-1111-111111111111"

func TestSubmitWorkspaceProject_CreatesPendingSubmissionWithSnapshot(t *testing.T) {
	wsRepo := newMockWorkspaceRepo()
	orch := &fakeVolumeOrchestrator{
		mockOrchestrator: &mockOrchestrator{},
		snapshotFiles: []domain.WorkspaceVolumeFile{
			{Path: "utils.py", Content: []byte("x = 2\n")},
			{Path: "main.py", Content: []byte("print(1)\n")},
		},
	}
	subRepo := &fakeSubmissionRepo{}
	svc := services.NewWorkspaceService(wsRepo, orch, &mockHostMonitor{}).
		WithExerciseSubmission(nil, subRepo)

	workspace := &domain.WorkspaceInstance{
		ID:        "22222222-2222-2222-2222-222222222222",
		TenantID:  domain.DefaultTenantID,
		StudentID: "student-1",
		SubjectID: "subject-1",
		Status:    domain.WorkspaceStatusRunning,
	}
	if err := wsRepo.Create(context.Background(), workspace); err != nil {
		t.Fatalf("no se pudo preparar el workspace: %v", err)
	}

	_, err := svc.SubmitWorkspaceProject(context.Background(), workspace.ID, "student-1", testExerciseID, map[string]string{
		"helpers/extra.py": "print(3)\n",
	})
	if err != nil {
		t.Fatalf("error inesperado al entregar el proyecto: %v", err)
	}

	if !orch.snapshotCalled {
		t.Error("esperado snapshot del volumen")
	}
	if orch.volume != "solv_workspace_student-1_subject-1" {
		t.Errorf("volumen inesperado: %s", orch.volume)
	}
	if _, ok := orch.writtenFiles["helpers/extra.py"]; !ok {
		t.Error("esperado que los archivos enviados se escriban en el volumen")
	}
	if subRepo.created == nil {
		t.Fatal("esperada submission persistida")
	}
	if subRepo.created.Verdict != "PENDING" {
		t.Errorf("esperado verdict PENDING, obtenido: %s", subRepo.created.Verdict)
	}
	if subRepo.created.Score == nil || *subRepo.created.Score != 0 {
		t.Errorf("esperado score 0, obtenido: %v", subRepo.created.Score)
	}
	if subRepo.created.WorkspaceID == nil || *subRepo.created.WorkspaceID != workspace.ID {
		t.Errorf("esperado workspace_id %s, obtenido: %v", workspace.ID, subRepo.created.WorkspaceID)
	}
	if subRepo.created.ExerciseID != testExerciseID {
		t.Errorf("esperado exercise_id %s, obtenido: %s", testExerciseID, subRepo.created.ExerciseID)
	}
	if !strings.HasPrefix(subRepo.created.Code, "# === FILE: main.py ===\n") {
		t.Errorf("snapshot sin cabecera determinista: %q", subRepo.created.Code)
	}
	if strings.Index(subRepo.created.Code, "main.py") > strings.Index(subRepo.created.Code, "utils.py") {
		t.Errorf("el snapshot no quedó ordenado alfabéticamente: %q", subRepo.created.Code)
	}
}

func TestSubmitWorkspaceProject_RejectsForeignWorkspace(t *testing.T) {
	wsRepo := newMockWorkspaceRepo()
	orch := &fakeVolumeOrchestrator{mockOrchestrator: &mockOrchestrator{}}
	svc := services.NewWorkspaceService(wsRepo, orch, &mockHostMonitor{}).
		WithExerciseSubmission(nil, &fakeSubmissionRepo{})

	workspace := &domain.WorkspaceInstance{
		ID:        "33333333-3333-3333-3333-333333333333",
		StudentID: "student-owner",
		SubjectID: "subject-1",
	}
	if err := wsRepo.Create(context.Background(), workspace); err != nil {
		t.Fatalf("no se pudo preparar el workspace: %v", err)
	}

	_, err := svc.SubmitWorkspaceProject(context.Background(), workspace.ID, "student-impostor", testExerciseID, nil)
	if !errors.Is(err, services.ErrWorkspaceForbidden) {
		t.Fatalf("esperado ErrWorkspaceForbidden, obtenido: %v", err)
	}
}
