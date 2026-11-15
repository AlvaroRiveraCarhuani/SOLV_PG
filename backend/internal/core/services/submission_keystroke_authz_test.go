package services

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

type keystrokeAuthzSubmissionRepo struct {
	domain.SubmissionRepository
	submission *domain.Submission
}

func (r *keystrokeAuthzSubmissionRepo) GetByID(context.Context, string, string) (*domain.Submission, error) {
	return r.submission, nil
}

type keystrokeAuthzStore struct {
	domain.KeystrokeRepository
	saved int
}

func (s *keystrokeAuthzStore) SaveBatch(_ context.Context, _ string, events []domain.SubmissionKeystrokeEvent) error {
	s.saved = len(events)
	return nil
}

func keystrokeAuthzContext(role, userID string) context.Context {
	ctx := context.WithValue(context.Background(), domain.TenantIDKey, "tenant-1")
	ctx = context.WithValue(ctx, domain.UserIDKey, userID)
	return context.WithValue(ctx, domain.UserRoleKey, role)
}

func newKeystrokeAuthzService() (*SubmissionService, *keystrokeAuthzStore) {
	repo := &keystrokeAuthzSubmissionRepo{submission: &domain.Submission{
		ID:        "submission-1",
		TenantID:  "tenant-1",
		StudentID: "student-owner",
	}}
	store := &keystrokeAuthzStore{}
	service := NewSubmissionService(repo)
	service.SetKeystrokeRepository(store)
	return service, store
}

func keystrokeEvents() []domain.SubmissionKeystrokeEvent {
	return []domain.SubmissionKeystrokeEvent{{}}
}

// Los eventos de teclado son evidencia de integridad: un alumno no puede
// escribirlos sobre la entrega de otro.
func TestSaveKeystrokeEvents_OtroAlumnoNoEscribe(t *testing.T) {
	service, store := newKeystrokeAuthzService()

	count, err := service.SaveKeystrokeEvents(keystrokeAuthzContext("student", "student-intruder"),
		"tenant-1", "submission-1", keystrokeEvents())

	if !errors.Is(err, ErrSubmissionForbidden) {
		t.Fatalf("se esperaba ErrSubmissionForbidden, se obtuvo %v", err)
	}
	if count != 0 || store.saved != 0 {
		t.Fatalf("no debía guardarse nada: count=%d saved=%d", count, store.saved)
	}
}

func TestSaveKeystrokeEvents_DuenoYDocenteSiEscriben(t *testing.T) {
	for _, tc := range []struct{ nombre, rol, usuario string }{
		{nombre: "alumno dueño", rol: "student", usuario: "student-owner"},
		{nombre: "docente", rol: "teacher", usuario: "teacher-1"},
		{nombre: "administrador", rol: "admin", usuario: "admin-1"},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			service, store := newKeystrokeAuthzService()

			count, err := service.SaveKeystrokeEvents(keystrokeAuthzContext(tc.rol, tc.usuario),
				"tenant-1", "submission-1", keystrokeEvents())

			if err != nil {
				t.Fatalf("debía guardar: %v", err)
			}
			if count != 1 || store.saved != 1 {
				t.Fatalf("no se guardaron los eventos: count=%d saved=%d", count, store.saved)
			}
		})
	}
}

// Sin repositorio de eventos no se persistió nada: responder con la cantidad de
// eventos era un éxito falso.
func TestSaveKeystrokeEvents_SinRepositorioNoFinjeExito(t *testing.T) {
	service := NewSubmissionService(&keystrokeAuthzSubmissionRepo{submission: &domain.Submission{
		ID:        "submission-1",
		TenantID:  "tenant-1",
		StudentID: "student-owner",
	}})

	count, err := service.SaveKeystrokeEvents(keystrokeAuthzContext("student", "student-owner"),
		"tenant-1", "submission-1", keystrokeEvents())

	if !errors.Is(err, ErrSubmissionNotConfigured) {
		t.Fatalf("se esperaba ErrSubmissionNotConfigured, se obtuvo %v", err)
	}
	if count != 0 {
		t.Fatalf("no debe informarse una cantidad guardada: %d", count)
	}
}
