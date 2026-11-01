package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// Un examen cerrado no se reabre por recargar la página: el servicio devuelve
// ErrWorkspaceFrozen antes de tocar Docker (ADR-005 §5).
func TestWorkspaceService_FrozenWorkspaceCannotBeResumed(t *testing.T) {
	wsRepo := newMockWorkspaceRepo()
	frozenAt := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	wsRepo.workspaces["student-1:subj-101"] = &domain.WorkspaceInstance{
		ID:        "ws-1",
		StudentID: "student-1",
		SubjectID: "subj-101",
		Status:    domain.WorkspaceStatusFrozen,
		FrozenAt:  &frozenAt,
		AccessURL: "http://127.0.0.1:3999",
		TenantID:  domain.DefaultTenantID,
		UpdatedAt: frozenAt,
		CreatedAt: frozenAt,
	}

	orch := &mockOrchestrator{}
	svc := services.NewWorkspaceService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(
			&mockSubjectRepo{subjects: map[string]*domain.Subject{}},
			&mockTemplateRepo{templates: map[string]*domain.Template{}},
		)

	_, err := svc.StartWorkspace(context.Background(), "student-1", "subj-101")
	if !errors.Is(err, domain.ErrWorkspaceFrozen) {
		t.Fatalf("se esperaba ErrWorkspaceFrozen, fue: %v", err)
	}
	if orch.lastConfig != nil {
		t.Fatal("un laboratorio congelado no debía levantar un contenedor")
	}
}

// Un laboratorio congelado que además quedó pausado por inactividad tampoco se
// despausa: la reanudación mira el estado del examen antes que el contenedor.
func TestWorkspaceService_ReactivationRespectsFrozen(t *testing.T) {
	wsRepo := newMockWorkspaceRepo()
	frozenAt := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	containerID := "container-1"
	wsRepo.workspaces["student-2:subj-202"] = &domain.WorkspaceInstance{
		ID:          "ws-2",
		StudentID:   "student-2",
		SubjectID:   "subj-202",
		Status:      domain.WorkspaceStatusFrozen,
		FrozenAt:    &frozenAt,
		ContainerID: &containerID,
		TenantID:    domain.DefaultTenantID,
	}

	orch := &mockOrchestrator{}
	svc := services.NewWorkspaceService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(
			&mockSubjectRepo{subjects: map[string]*domain.Subject{}},
			&mockTemplateRepo{templates: map[string]*domain.Template{}},
		)

	if _, err := svc.StartWorkspace(context.Background(), "student-2", "subj-202"); !errors.Is(err, domain.ErrWorkspaceFrozen) {
		t.Fatalf("se esperaba ErrWorkspaceFrozen, fue: %v", err)
	}
}
