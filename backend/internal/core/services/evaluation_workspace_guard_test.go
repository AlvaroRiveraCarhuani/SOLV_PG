package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type guardWorkspaceRepo struct {
	project *domain.WorkspaceProject
}

func (g *guardWorkspaceRepo) GetProjectByExercise(context.Context, string, string) (*domain.WorkspaceProject, error) {
	if g.project == nil {
		return nil, domain.ErrWorkspaceProjectNotFound
	}
	return g.project, nil
}

func (g *guardWorkspaceRepo) CreateProject(context.Context, *domain.WorkspaceProject) (*domain.WorkspaceProject, error) {
	panic("not used")
}

func (g *guardWorkspaceRepo) GetDraftByProject(context.Context, string, string) (*domain.WorkspaceProjectDraft, error) {
	panic("not used")
}

func (g *guardWorkspaceRepo) SaveDraft(context.Context, *domain.WorkspaceProjectDraft, int64) (*domain.WorkspaceProjectDraft, error) {
	panic("not used")
}

func (g *guardWorkspaceRepo) Publish(context.Context, string, string, string) (*domain.WorkspaceProjectVersion, error) {
	panic("not used")
}

func (g *guardWorkspaceRepo) ListVersions(context.Context, string, string) ([]*domain.WorkspaceProjectVersion, error) {
	panic("not used")
}

func (g *guardWorkspaceRepo) GetVersion(context.Context, string, string) (*domain.WorkspaceProjectVersion, error) {
	panic("not used")
}

func (g *guardWorkspaceRepo) GetLegacyReference(context.Context, string, string) (*domain.WorkspaceLegacyProjectReference, error) {
	panic("not used")
}

func TestUpdateExerciseRejectsManagedBoilerplate(t *testing.T) {
	const tenantID = "tenant-1"
	const exerciseID = "exercise-1"
	current := &domain.Exercise{
		ID:              exerciseID,
		TenantID:        tenantID,
		Title:           "Práctica con proyecto",
		Type:            domain.ExerciseTypeAlgorithm,
		EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		Language:        "python",
		Status:          "draft",
		TimeLimitMS:     1000,
		MemoryLimitMB:   128,
		Boilerplate:     "base publicada",
	}
	repo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{exerciseID: current}}
	evaluation := services.NewEvaluationService(repo, nil, nil, nil)
	evaluation.SetWorkspaceProjectService(services.NewWorkspaceProjectService(&guardWorkspaceRepo{
		project: &domain.WorkspaceProject{ID: "project-1", TenantID: tenantID, ExerciseID: exerciseID},
	}, false))

	updated := *current
	updated.Boilerplate = "intento de reemplazo"
	err := evaluation.UpdateExercise(context.Background(), &updated)
	if !errors.Is(err, domain.ErrWorkspaceProjectManagedBoilerplate) {
		t.Fatalf("UpdateExercise devolvió %v", err)
	}
	if repo.exercises[exerciseID].Boilerplate != "base publicada" {
		t.Fatalf("el repositorio recibió un boilerplate reemplazado")
	}
}

func TestUpdateExerciseAllowsUnchangedManagedBoilerplate(t *testing.T) {
	const tenantID = "tenant-1"
	const exerciseID = "exercise-2"
	current := &domain.Exercise{
		ID:              exerciseID,
		TenantID:        tenantID,
		Title:           "Práctica con proyecto",
		Type:            domain.ExerciseTypeAlgorithm,
		EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		Language:        "python",
		Status:          "draft",
		TimeLimitMS:     1000,
		MemoryLimitMB:   128,
		Boilerplate:     "base publicada",
	}
	repo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{exerciseID: current}}
	evaluation := services.NewEvaluationService(repo, nil, nil, nil)
	evaluation.SetWorkspaceProjectService(services.NewWorkspaceProjectService(&guardWorkspaceRepo{
		project: &domain.WorkspaceProject{ID: "project-2", TenantID: tenantID, ExerciseID: exerciseID},
	}, false))

	updated := *current
	updated.Title = "Título nuevo"
	if err := evaluation.UpdateExercise(context.Background(), &updated); err != nil {
		t.Fatalf("UpdateExercise sin cambiar boilerplate: %v", err)
	}
}
