package services

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

// purposeOnlyExerciseRepo responde la única consulta que usa el registro de
// tecleo: el resto de la interfaz no participa de estas pruebas.
type purposeOnlyExerciseRepo struct {
	domain.ExerciseRepository
	purpose string
}

func (r purposeOnlyExerciseRepo) GetByIDAndTenant(context.Context, string, string) (*domain.Exercise, error) {
	return &domain.Exercise{ID: "ex-1", Purpose: r.purpose}, nil
}

func keystrokeSubmissionContext() context.Context {
	ctx := context.WithValue(context.Background(), domain.UserIDKey, "student-1")
	return context.WithValue(ctx, domain.UserRoleKey, "student")
}

func newKeystrokeProfileService(t *testing.T, purpose string) (*SubmissionService, *memoryKeystrokeRepo) {
	t.Helper()
	subs := newStubSubmissionRepo()
	if err := subs.Create(context.Background(), &domain.Submission{
		ID: "sub-1", TenantID: "tenant-1", StudentID: "student-1", ExerciseID: "ex-1",
	}); err != nil {
		t.Fatalf("no se pudo preparar la entrega: %v", err)
	}
	keystrokes := newMemoryKeystrokeRepo()
	service := NewSubmissionService(subs)
	service.SetKeystrokeRepository(keystrokes)
	service.SetExerciseRepository(purposeOnlyExerciseRepo{purpose: purpose})
	return service, keystrokes
}

func TestSaveKeystrokeEvents_ElExamenRegistraElTecleo(t *testing.T) {
	service, keystrokes := newKeystrokeProfileService(t, string(domain.ExercisePurposeExam))

	count, err := service.SaveKeystrokeEvents(keystrokeSubmissionContext(), "tenant-1", "sub-1",
		[]domain.SubmissionKeystrokeEvent{{EventType: domain.KeystrokeEventInsert, Content: "ab"}})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if count != 1 || len(keystrokes.events["sub-1"]) != 1 {
		t.Fatalf("el examen debe guardar el evento: devueltos %d, guardados %d", count, len(keystrokes.events["sub-1"]))
	}
}

func TestSaveKeystrokeEvents_LaPracticaNoRegistraElTecleo(t *testing.T) {
	service, keystrokes := newKeystrokeProfileService(t, string(domain.ExercisePurposeClass))

	_, err := service.SaveKeystrokeEvents(keystrokeSubmissionContext(), "tenant-1", "sub-1",
		[]domain.SubmissionKeystrokeEvent{{EventType: domain.KeystrokeEventInsert, Content: "ab"}})
	if !errors.Is(err, ErrKeystrokeCaptureDisabled) {
		t.Fatalf("esperado ErrKeystrokeCaptureDisabled, obtenido: %v", err)
	}
	if len(keystrokes.events["sub-1"]) != 0 {
		t.Fatalf("la práctica no debe guardar eventos: %d", len(keystrokes.events["sub-1"]))
	}
}

func TestSaveKeystrokeEvents_SinCatalogoConservaElRegistro(t *testing.T) {
	subs := newStubSubmissionRepo()
	if err := subs.Create(context.Background(), &domain.Submission{
		ID: "sub-1", TenantID: "tenant-1", StudentID: "student-1", ExerciseID: "ex-1",
	}); err != nil {
		t.Fatalf("no se pudo preparar la entrega: %v", err)
	}
	keystrokes := newMemoryKeystrokeRepo()
	service := NewSubmissionService(subs)
	service.SetKeystrokeRepository(keystrokes)

	if _, err := service.SaveKeystrokeEvents(keystrokeSubmissionContext(), "tenant-1", "sub-1",
		[]domain.SubmissionKeystrokeEvent{{EventType: domain.KeystrokeEventInsert}}); err != nil {
		t.Fatalf("sin catálogo no se bloquea el registro: %v", err)
	}
	if len(keystrokes.events["sub-1"]) != 1 {
		t.Fatalf("esperado un evento guardado, obtenidos %d", len(keystrokes.events["sub-1"]))
	}
}
