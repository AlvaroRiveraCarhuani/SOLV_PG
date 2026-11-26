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
	exerciseRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		practiceID: {ID: practiceID, Purpose: string(domain.ExercisePurposeExam)},
	}}
	wsRepo.environments["student-1:subj-101:"+testPracticeID] = &domain.PracticeEnvironment{
		ID:                 "ws-1",
		StudentID:          "student-1",
		SubjectID:          "subj-101",
		ExerciseID:         &practiceID,
		Status:             domain.PracticeEnvironmentStatusFrozen,
		EnvironmentProfile: domain.EnvironmentProfileExam,
		FrozenAt:           &frozenAt,
		AccessURL:          "http://127.0.0.1:3999",
		TenantID:           domain.DefaultTenantID,
		UpdatedAt:          frozenAt,
		CreatedAt:          frozenAt,
	}

	orch := &mockOrchestrator{}
	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(
			&mockSubjectRepo{subjects: map[string]*domain.Subject{}},
			&mockTemplateRepo{templates: map[string]*domain.Template{}},
		).
		WithExerciseSubmission(exerciseRepo, nil)

	_, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-1", "subj-101", testPracticeID)
	if !errors.Is(err, domain.ErrPracticeEnvironmentFrozen) {
		t.Fatalf("se esperaba ErrPracticeEnvironmentFrozen, fue: %v", err)
	}
	if orch.lastConfig != nil {
		t.Fatal("un laboratorio congelado no debía levantar un contenedor")
	}
}

// Un examen congelado tampoco se despausa al volver: la reanudación mira el
// estado del examen antes que el contenedor.
func TestPracticeEnvironmentService_ReactivationRespectsFrozen(t *testing.T) {
	wsRepo := newMockPracticeEnvironmentRepo()
	frozenAt := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	containerID := "container-1"
	practiceID := testPracticeID
	exerciseRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		practiceID: {ID: practiceID, Purpose: string(domain.ExercisePurposeExam)},
	}}
	wsRepo.environments["student-2:subj-202:"+testPracticeID] = &domain.PracticeEnvironment{
		ID:                 "ws-2",
		StudentID:          "student-2",
		SubjectID:          "subj-202",
		ExerciseID:         &practiceID,
		Status:             domain.PracticeEnvironmentStatusFrozen,
		EnvironmentProfile: domain.EnvironmentProfileExam,
		FrozenAt:           &frozenAt,
		ContainerID:        &containerID,
		TenantID:           domain.DefaultTenantID,
	}

	orch := &mockOrchestrator{}
	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(
			&mockSubjectRepo{subjects: map[string]*domain.Subject{}},
			&mockTemplateRepo{templates: map[string]*domain.Template{}},
		).
		WithExerciseSubmission(exerciseRepo, nil)

	if _, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-2", "subj-202", testPracticeID); !errors.Is(err, domain.ErrPracticeEnvironmentFrozen) {
		t.Fatalf("se esperaba ErrPracticeEnvironmentFrozen, fue: %v", err)
	}
}

// Una práctica marcada por una pausa que ya terminó vuelve a escritura al
// abrirse: reanudar, cerrar o dejar vencer el aula no puede dejar al alumno en
// solo lectura para siempre.
func TestPracticeEnvironmentService_EndedClassPauseThawsPractice(t *testing.T) {
	wsRepo := newMockPracticeEnvironmentRepo()
	frozenAt := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	practiceID := testPracticeID
	stored := &domain.PracticeEnvironment{
		ID:                 "ws-thaw",
		StudentID:          "student-thaw",
		SubjectID:          "subj-303",
		ExerciseID:         &practiceID,
		Status:             domain.PracticeEnvironmentStatusFrozen,
		EnvironmentProfile: domain.EnvironmentProfilePractice,
		FrozenAt:           &frozenAt,
		TenantID:           domain.DefaultTenantID,
	}
	wsRepo.environments["student-thaw:subj-303:"+testPracticeID] = stored
	wsRepo.environments[stored.ID] = stored

	orch := &mockOrchestrator{}
	linker := &pausedAulaLinker{paused: false}
	svc := pausedAulaService(wsRepo, orch, linker)

	instance, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-thaw", "subj-303", testPracticeID)
	if err != nil {
		t.Fatalf("una pausa terminada no debía bloquear el entorno: %v", err)
	}
	if instance.FrozenAt != nil || instance.Status != domain.PracticeEnvironmentStatusRunning {
		t.Fatalf("el entorno debía quedar escribible: status=%s frozen_at=%v", instance.Status, instance.FrozenAt)
	}
	if stored.FrozenAt != nil || stored.Status != domain.PracticeEnvironmentStatusRunning {
		t.Fatalf("la fila debía quedar limpia de la pausa: status=%s frozen_at=%v", stored.Status, stored.FrozenAt)
	}
	if orch.lastConfig == nil || orch.lastConfig.ReadOnly {
		t.Fatal("el contenedor debía reconstruirse con el volumen en escritura")
	}
}
