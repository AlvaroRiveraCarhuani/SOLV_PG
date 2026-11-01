package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// La lectura se resuelve con la identidad de la sesión: el entorno que se
// devuelve es siempre el del usuario autenticado en esa materia.
func TestWorkspaceForSubject(t *testing.T) {
	const subjectID = "22222222-2222-4222-a222-222222222222"
	const teacherID = "11111111-1111-4111-a111-111111111111"

	repo := newMockWorkspaceRepo()
	repo.workspaces[teacherID+":"+subjectID] = &domain.WorkspaceInstance{
		ID:            "ws-1",
		StudentID:     teacherID,
		SubjectID:     subjectID,
		Status:        domain.WorkspaceStatusRunning,
		AccessURL:     "http://localhost:3999",
		MemoryLimitMB: 512,
	}
	service := services.NewWorkspaceService(repo, nil, nil)

	t.Run("devuelve el entorno vigente de la materia", func(t *testing.T) {
		instance, err := service.WorkspaceForSubject(context.Background(), teacherID, subjectID)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if instance == nil {
			t.Fatal("esperaba el entorno de la materia")
		}
		if instance.AccessURL != "http://localhost:3999" {
			t.Fatalf("url inesperada: %s", instance.AccessURL)
		}
	})

	t.Run("sin entorno devuelve nulo y no un error", func(t *testing.T) {
		instance, err := service.WorkspaceForSubject(context.Background(), teacherID, "33333333-3333-4333-a333-333333333333")
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if instance != nil {
			t.Fatalf("esperaba nulo, llegó %s", instance.ID)
		}
	})

	t.Run("no revela el entorno de otro usuario", func(t *testing.T) {
		instance, err := service.WorkspaceForSubject(context.Background(), "44444444-4444-4444-a444-444444444444", subjectID)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if instance != nil {
			t.Fatalf("esperaba nulo para otro usuario, llegó %s", instance.ID)
		}
	})

	t.Run("sin materia responde error de negocio", func(t *testing.T) {
		_, err := service.WorkspaceForSubject(context.Background(), teacherID, "  ")
		if !errors.Is(err, services.ErrWorkspaceSubjectRequired) {
			t.Fatalf("esperaba ErrWorkspaceSubjectRequired, llegó %v", err)
		}
	})

	t.Run("una materia que no es uuid se rechaza", func(t *testing.T) {
		_, err := service.WorkspaceForSubject(context.Background(), teacherID, "programacion-1")
		if !errors.Is(err, domain.ErrInvalidResourceID) {
			t.Fatalf("esperaba ErrInvalidResourceID, llegó %v", err)
		}
	})
}
