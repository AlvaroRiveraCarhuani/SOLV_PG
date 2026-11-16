package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func examEnvironmentForPause(id string) *domain.PracticeEnvironment {
	containerID := "c-" + id
	return &domain.PracticeEnvironment{
		ID:                 id,
		StudentID:          "student-1",
		SubjectID:          "subj-101",
		Status:             domain.PracticeEnvironmentStatusRunning,
		ContainerID:        &containerID,
		EnvironmentProfile: domain.EnvironmentProfileExam,
		ExamAttempt:        1,
	}
}

// Un examen no se apaga a mano: el docente que pausa a mitad deja al alumno sin
// su espacio, y el cierre del examen es otra operación, la del aula.
func TestPausePracticeEnvironment_ElExamenNoSePausa(t *testing.T) {
	repo := newMockPracticeEnvironmentRepo()
	exam := examEnvironmentForPause("env-exam")
	repo.environments[exam.ID] = exam
	svc := services.NewPracticeEnvironmentService(repo, &mockOrchestrator{}, &mockHostMonitor{})

	if _, err := svc.PausePracticeEnvironment(context.Background(), exam.ID); !errors.Is(err, services.ErrExamPauseForbidden) {
		t.Fatalf("se esperaba ErrExamPauseForbidden, se obtuvo: %v", err)
	}
	if exam.Status != domain.PracticeEnvironmentStatusRunning {
		t.Fatalf("el examen no debía cambiar de estado: %s", exam.Status)
	}
}

func TestHibernatePracticeEnvironment_ElExamenNoSeDuerme(t *testing.T) {
	repo := newMockPracticeEnvironmentRepo()
	exam := examEnvironmentForPause("env-exam-hibernate")
	repo.environments[exam.ID] = exam
	svc := services.NewPracticeEnvironmentService(repo, &mockOrchestrator{}, &mockHostMonitor{})

	if err := svc.HibernatePracticeEnvironment(context.Background(), exam.ID); !errors.Is(err, services.ErrExamPauseForbidden) {
		t.Fatalf("se esperaba ErrExamPauseForbidden, se obtuvo: %v", err)
	}
	if exam.Status != domain.PracticeEnvironmentStatusRunning {
		t.Fatalf("el examen no debía cambiar de estado: %s", exam.Status)
	}
}

// La práctica sí se pausa: ahí la inactividad es real y el alumno puede volver a
// despertarla.
func TestPausePracticeEnvironment_LaPracticaSiSePausa(t *testing.T) {
	repo := newMockPracticeEnvironmentRepo()
	practice := examEnvironmentForPause("env-practice")
	practice.EnvironmentProfile = domain.EnvironmentProfilePractice
	practice.ExamAttempt = 0
	repo.environments[practice.ID] = practice
	svc := services.NewPracticeEnvironmentService(repo, &mockOrchestrator{}, &mockHostMonitor{})

	updated, err := svc.PausePracticeEnvironment(context.Background(), practice.ID)
	if err != nil {
		t.Fatalf("la práctica debía pausarse: %v", err)
	}
	if updated.Status != domain.PracticeEnvironmentStatusHibernated {
		t.Fatalf("la práctica debía quedar hibernada: %s", updated.Status)
	}
}
