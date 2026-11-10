package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type workspaceProjectRepoFake struct {
	project      *domain.WorkspaceProject
	draft        *domain.WorkspaceProjectDraft
	versions     []*domain.WorkspaceProjectVersion
	publishCalls int
	createCalls  int
	saveCalls    int
	saveErr      error
}

func (f *workspaceProjectRepoFake) GetProjectByExercise(context.Context, string, string) (*domain.WorkspaceProject, error) {
	if f.project == nil {
		return nil, domain.ErrWorkspaceProjectNotFound
	}
	return f.project, nil
}

func (f *workspaceProjectRepoFake) CreateProject(_ context.Context, project *domain.WorkspaceProject) (*domain.WorkspaceProject, error) {
	f.createCalls++
	project.ID = "project-1"
	f.project = project
	return project, nil
}

func (f *workspaceProjectRepoFake) GetDraftByProject(context.Context, string, string) (*domain.WorkspaceProjectDraft, error) {
	if f.draft == nil {
		return nil, domain.ErrWorkspaceProjectNotFound
	}
	return f.draft, nil
}

func (f *workspaceProjectRepoFake) SaveDraft(_ context.Context, draft *domain.WorkspaceProjectDraft, expectedRevision int64) (*domain.WorkspaceProjectDraft, error) {
	f.saveCalls++
	if f.saveErr != nil {
		return nil, f.saveErr
	}
	draft.Revision = expectedRevision + 1
	f.draft = draft
	return draft, nil
}

func (f *workspaceProjectRepoFake) Publish(context.Context, string, string, string) (*domain.WorkspaceProjectVersion, error) {
	f.publishCalls++
	return &domain.WorkspaceProjectVersion{
		ID:            "version-1",
		ProjectID:     "project-1",
		ExerciseID:    "exercise-1",
		VersionNumber: 1,
		PublishedAt:   time.Now(),
	}, nil
}

func (f *workspaceProjectRepoFake) ListVersions(context.Context, string, string) ([]*domain.WorkspaceProjectVersion, error) {
	return f.versions, nil
}

func (f *workspaceProjectRepoFake) GetVersion(context.Context, string, string) (*domain.WorkspaceProjectVersion, error) {
	return nil, domain.ErrWorkspaceProjectNotFound
}

func (f *workspaceProjectRepoFake) GetLegacyReference(context.Context, string, string) (*domain.WorkspaceLegacyProjectReference, error) {
	return nil, domain.ErrWorkspaceProjectNotFound
}

func validWorkspaceSnapshot() string {
	return "# === FILE: main.py ===\nprint('hello')\n"
}

func TestWorkspacePublishIsFailClosed(t *testing.T) {
	repo := &workspaceProjectRepoFake{
		project: &domain.WorkspaceProject{ID: "project-1", ExerciseID: "exercise-1"},
		draft:   &domain.WorkspaceProjectDraft{ProjectID: "project-1", TextSnapshot: validWorkspaceSnapshot()},
	}
	service := NewWorkspaceProjectService(repo, false)

	_, err := service.Publish(context.Background(), "tenant-1", "exercise-1", "user-1")
	if !errors.Is(err, domain.ErrWorkspacePublishingDisabled) {
		t.Fatalf("Publish con flag deshabilitado devolvió %v", err)
	}
	if repo.publishCalls != 0 {
		t.Fatalf("Publish tocó el repositorio con el flag deshabilitado")
	}
}

func TestWorkspaceGetProjectWithoutProjectIsAnEditableEmptyView(t *testing.T) {
	service := NewWorkspaceProjectService(&workspaceProjectRepoFake{}, true)
	view, err := service.GetProject(context.Background(), "tenant-1", "exercise-1")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if view.Project != nil || view.Draft != nil || len(view.Versions) != 0 {
		t.Fatalf("vista inicial inesperada: %+v", view)
	}
	if !view.PublishingEnabled {
		t.Fatal("la vista vacía debe informar el flag de publicación")
	}
}

func TestWorkspacePublishEnabledPublishesCurrentDraft(t *testing.T) {
	repo := &workspaceProjectRepoFake{
		project: &domain.WorkspaceProject{ID: "project-1", ExerciseID: "exercise-1"},
		draft:   &domain.WorkspaceProjectDraft{ProjectID: "project-1", TextSnapshot: validWorkspaceSnapshot()},
	}
	service := NewWorkspaceProjectService(repo, true)

	version, err := service.Publish(context.Background(), "tenant-1", "exercise-1", "user-1")
	if err != nil {
		t.Fatalf("Publish habilitado: %v", err)
	}
	if version.VersionNumber != 1 || repo.publishCalls != 1 {
		t.Fatalf("publicación inesperada: %+v calls=%d", version, repo.publishCalls)
	}
}

func TestWorkspaceSaveDraftRejectsInvalidSnapshots(t *testing.T) {
	service := NewWorkspaceProjectService(&workspaceProjectRepoFake{}, false)
	for name, snapshot := range map[string]string{
		"empty":       "   ",
		"no headers":  "print('hello')\n",
		"unsafe path": "# === FILE: ../secret.py ===\nprint('x')\n",
		"too large":   "# === FILE: main.py ===\n" + strings.Repeat("x", maxWorkspaceProjectBytes) + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := service.SaveDraft(context.Background(), "tenant-1", "b3f7c9a2-7a1f-4d01-8e5f-1a2b3c4d5e6f", "user-1", SaveWorkspaceDraftRequest{
				TextSnapshot: snapshot,
			})
			if err == nil {
				t.Fatal("SaveDraft aceptó un snapshot inválido")
			}
		})
	}
}

func TestWorkspaceSaveDraftCreatesProjectAndPreservesRevisionContract(t *testing.T) {
	repo := &workspaceProjectRepoFake{}
	service := NewWorkspaceProjectService(repo, false)

	draft, err := service.SaveDraft(context.Background(), "tenant-1", "b3f7c9a2-7a1f-4d01-8e5f-1a2b3c4d5e6f", "user-1", SaveWorkspaceDraftRequest{
		TextSnapshot: validWorkspaceSnapshot(),
	})
	if err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}
	if repo.createCalls != 1 || draft.Revision != 1 || draft.ProjectID != "project-1" {
		t.Fatalf("creación/revisión inesperada: %+v create=%d", draft, repo.createCalls)
	}

	repo.saveErr = domain.ErrWorkspaceDraftRevisionConflict
	_, err = service.SaveDraft(context.Background(), "tenant-1", "b3f7c9a2-7a1f-4d01-8e5f-1a2b3c4d5e6f", "user-1", SaveWorkspaceDraftRequest{
		TextSnapshot:     validWorkspaceSnapshot(),
		ExpectedRevision: 1,
	})
	if !errors.Is(err, domain.ErrWorkspaceDraftRevisionConflict) {
		t.Fatalf("conflicto de revisión no propagado: %v", err)
	}
}

func TestClassifyIDEBaseStatus(t *testing.T) {
	version := "version-1"
	cases := []struct {
		name string
		env  *domain.PracticeEnvironment
		want string
	}{
		{"none", &domain.PracticeEnvironment{WorkspaceSeedMode: domain.WorkspaceSeedModeNone}, domain.BaseStatusNoBase},
		{"version", &domain.PracticeEnvironment{WorkspaceSeedMode: domain.WorkspaceSeedModeProject, WorkspaceProjectVersionID: &version}, domain.BaseStatusVersion},
		{"project without version", &domain.PracticeEnvironment{WorkspaceSeedMode: domain.WorkspaceSeedModeProject}, domain.BaseStatusLegacyUnknown},
		{"legacy", &domain.PracticeEnvironment{WorkspaceSeedMode: domain.WorkspaceSeedModeLegacy}, domain.BaseStatusLegacyUnknown},
		{"old row", &domain.PracticeEnvironment{}, domain.BaseStatusLegacyUnknown},
		{"nil", nil, domain.BaseStatusLegacyUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyIDEBaseStatus(tc.env); got != tc.want {
				t.Fatalf("classifyIDEBaseStatus = %q, quería %q", got, tc.want)
			}
		})
	}
}
