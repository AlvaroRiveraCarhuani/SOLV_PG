package services_test

import (
	"context"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// pausedAulaLinker responde la pausa del aula y registra el enlace del entorno.
type pausedAulaLinker struct {
	paused   bool
	err      error
	attached int
}

func (l *pausedAulaLinker) AttachPracticeEnvironmentToOpenSession(context.Context, string, string, string, string) error {
	l.attached++
	return nil
}

func (l *pausedAulaLinker) StudentSessionPaused(context.Context, string, string, string, string) (bool, error) {
	return l.paused, l.err
}

func pausedAulaService(repo *mockPracticeEnvironmentRepo, orch *mockOrchestrator, linker *pausedAulaLinker) *services.PracticeEnvironmentService {
	return services.NewPracticeEnvironmentService(repo, orch, &mockHostMonitor{}).
		WithProvisioning(
			&mockSubjectRepo{subjects: map[string]*domain.Subject{}},
			&mockTemplateRepo{templates: map[string]*domain.Template{}},
		).
		WithClassSessions(linker)
}

// Un entorno que nace con la clase pausada monta el volumen en solo lectura y
// queda marcado como congelado: el alumno entra a leer, no a escribir.
func TestStartPracticeEnvironment_StartsWithPausedAulaInReadOnly(t *testing.T) {
	repo := newMockPracticeEnvironmentRepo()
	orch := &mockOrchestrator{}
	linker := &pausedAulaLinker{paused: true}
	svc := pausedAulaService(repo, orch, linker)

	instance, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-paused", "subj-paused", testPracticeID)
	if err != nil {
		t.Fatalf("abrir el entorno con el aula pausada no debía fallar: %v", err)
	}
	if orch.lastConfig == nil {
		t.Fatal("no se envió configuración de contenedor al orquestador")
	}
	if !orch.lastConfig.ReadOnly {
		t.Fatal("el volumen debía montarse en solo lectura")
	}
	if instance.FrozenAt == nil || instance.Status != domain.PracticeEnvironmentStatusFrozen {
		t.Fatalf("el entorno debía quedar congelado: status=%s frozen_at=%v", instance.Status, instance.FrozenAt)
	}
	stored := repo.environments["student-paused:subj-paused:"+testPracticeID]
	if stored == nil || stored.FrozenAt == nil || stored.Status != domain.PracticeEnvironmentStatusFrozen {
		t.Fatalf("la fila del entorno debía quedar congelada: %+v", stored)
	}
	if linker.attached != 1 {
		t.Fatalf("el entorno debía enlazarse al aula para que la reanudación lo encuentre: %d", linker.attached)
	}
}

// Un entorno hibernado que vuelve con la clase pausada se reconstruye en solo
// lectura; no se despausa el contenedor en escritura.
func TestReactivatePracticeEnvironment_StartsWithPausedAulaInReadOnly(t *testing.T) {
	repo := newMockPracticeEnvironmentRepo()
	orch := &mockOrchestrator{}
	linker := &pausedAulaLinker{paused: true}
	exerciseID := testPracticeID
	repo.environments["student-hibernated:subj-paused:"+testPracticeID] = &domain.PracticeEnvironment{
		ID:         "ws-hibernated",
		TenantID:   domain.DefaultTenantID,
		StudentID:  "student-hibernated",
		SubjectID:  "subj-paused",
		ExerciseID: &exerciseID,
		Status:     domain.PracticeEnvironmentStatusHibernated,
		AccessURL:  "http://127.0.0.1:3999",
	}
	repo.environments["ws-hibernated"] = repo.environments["student-hibernated:subj-paused:"+testPracticeID]
	svc := pausedAulaService(repo, orch, linker)

	instance, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-hibernated", "subj-paused", testPracticeID)
	if err != nil {
		t.Fatalf("reactivar el entorno con el aula pausada no debía fallar: %v", err)
	}
	if orch.lastConfig == nil {
		t.Fatal("no se envió configuración de contenedor al orquestador")
	}
	if !orch.lastConfig.ReadOnly {
		t.Fatal("la reactivación debía montar el volumen en solo lectura")
	}
	if instance.FrozenAt == nil || instance.Status != domain.PracticeEnvironmentStatusFrozen {
		t.Fatalf("el entorno reactivado debía quedar congelado: status=%s frozen_at=%v", instance.Status, instance.FrozenAt)
	}
}

// Sin pausa, el arranque conserva la escritura: la lectura solo depende del
// estado del aula, no del ciclo de vida del entorno.
func TestStartPracticeEnvironment_WithoutPausedAulaKeepsWritable(t *testing.T) {
	repo := newMockPracticeEnvironmentRepo()
	orch := &mockOrchestrator{}
	svc := pausedAulaService(repo, orch, &pausedAulaLinker{paused: false})

	instance, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-free", "subj-free", testPracticeID)
	if err != nil {
		t.Fatalf("abrir el entorno sin pausa no debía fallar: %v", err)
	}
	if orch.lastConfig == nil {
		t.Fatal("no se envió configuración de contenedor al orquestador")
	}
	if orch.lastConfig.ReadOnly {
		t.Fatal("sin pausa el volumen debía montarse en escritura")
	}
	if instance.FrozenAt != nil || instance.Status != domain.PracticeEnvironmentStatusRunning {
		t.Fatalf("sin pausa el entorno debía quedar escribible: status=%s frozen_at=%v", instance.Status, instance.FrozenAt)
	}
}
