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
// ErrPracticeEnvironmentFrozen antes de tocar Docker (ADR-005 §5).
func TestPracticeEnvironmentService_FrozenPracticeEnvironmentCannotBeResumed(t *testing.T) {
	wsRepo := newMockPracticeEnvironmentRepo()
	frozenAt := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	practiceID := testPracticeID
	wsRepo.environments["student-1:subj-101:"+testPracticeID] = &domain.PracticeEnvironment{
		ID:         "ws-1",
		StudentID:  "student-1",
		SubjectID:  "subj-101",
		ExerciseID: &practiceID,
		Status:     domain.PracticeEnvironmentStatusFrozen,
		FrozenAt:   &frozenAt,
		AccessURL:  "http://127.0.0.1:3999",
		TenantID:   domain.DefaultTenantID,
		UpdatedAt:  frozenAt,
		CreatedAt:  frozenAt,
	}

	orch := &mockOrchestrator{}
	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(
			&mockSubjectRepo{subjects: map[string]*domain.Subject{}},
			&mockTemplateRepo{templates: map[string]*domain.Template{}},
		)

	_, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-1", "subj-101", testPracticeID)
	if !errors.Is(err, domain.ErrPracticeEnvironmentFrozen) {
		t.Fatalf("se esperaba ErrPracticeEnvironmentFrozen, fue: %v", err)
	}
	if orch.lastConfig != nil {
		t.Fatal("un laboratorio congelado no debía levantar un contenedor")
	}
}

// Un laboratorio congelado que además quedó pausado por inactividad tampoco se
// despausa: la reanudación mira el estado del examen antes que el contenedor.
func TestPracticeEnvironmentService_ReactivationRespectsFrozen(t *testing.T) {
	wsRepo := newMockPracticeEnvironmentRepo()
	frozenAt := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	containerID := "container-1"
	practiceID := testPracticeID
	wsRepo.environments["student-2:subj-202:"+testPracticeID] = &domain.PracticeEnvironment{
		ID:          "ws-2",
		StudentID:   "student-2",
		SubjectID:   "subj-202",
		ExerciseID:  &practiceID,
		Status:      domain.PracticeEnvironmentStatusFrozen,
		FrozenAt:    &frozenAt,
		ContainerID: &containerID,
		TenantID:    domain.DefaultTenantID,
	}

	orch := &mockOrchestrator{}
	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(
			&mockSubjectRepo{subjects: map[string]*domain.Subject{}},
			&mockTemplateRepo{templates: map[string]*domain.Template{}},
		)

	if _, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-2", "subj-202", testPracticeID); !errors.Is(err, domain.ErrPracticeEnvironmentFrozen) {
		t.Fatalf("se esperaba ErrPracticeEnvironmentFrozen, fue: %v", err)
	}
}
