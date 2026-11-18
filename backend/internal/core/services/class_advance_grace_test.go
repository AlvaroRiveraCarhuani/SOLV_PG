package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

const graceSessionID = "55555555-5555-5555-5555-555555555555"

// fakeGraceStore hereda el resto del contrato del almacén de avances y registra
// cada ventana que el servicio decide escribir.
type fakeGraceStore struct {
	fakeAdvanceStore
	graceWrites []time.Time
}

func (f *fakeGraceStore) SetGrace(ctx context.Context, tenantID, teacherID, sessionID string, until time.Time) (*domain.ClassSession, error) {
	f.graceWrites = append(f.graceWrites, until)
	return &domain.ClassSession{
		ID:         sessionID,
		TenantID:   tenantID,
		Status:     domain.ClassSessionStatusClosed,
		GraceUntil: &until,
	}, nil
}

// closedSessionForGrace arma un aula cerrada con su hora de cierre y la ventana
// vigente, que es lo que mira el servicio antes de fijar la hora final.
func closedSessionForGrace(closesAt time.Time, graceUntil *time.Time) *domain.ClassSession {
	return &domain.ClassSession{
		ID:         graceSessionID,
		TenantID:   advanceTenantID,
		SubjectID:  advanceSubjectID,
		Status:     domain.ClassSessionStatusClosed,
		OpensAt:    closesAt.Add(-90 * time.Minute),
		EndsAt:     closesAt,
		ClosesAt:   &closesAt,
		GraceUntil: graceUntil,
	}
}

func newGraceService(session *domain.ClassSession) (*services.ClassAdvanceService, *fakeGraceStore) {
	store := &fakeGraceStore{}
	return services.NewClassAdvanceService(store, &fakeAdvanceSessions{session: session}), store
}

func TestSetGraceUntil_WritesTheExactEndOfTheWindow(t *testing.T) {
	now := time.Now()
	service, store := newGraceService(closedSessionForGrace(now, nil))
	until := now.Add(45 * time.Minute).UTC().Truncate(time.Second)

	session, err := service.SetGraceUntil(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, until)
	if err != nil {
		t.Fatalf("SetGraceUntil falló: %v", err)
	}
	if len(store.graceWrites) != 1 || !store.graceWrites[0].Equal(until) {
		t.Fatalf("el almacén debía recibir la hora final %v: %v", until, store.graceWrites)
	}
	if session == nil || session.GraceUntil == nil || !session.GraceUntil.Equal(until) {
		t.Fatal("la sesión devuelta debía traer la hora final")
	}
}

// El doble clic histórico: el aula ya mandó la nueva hora y la petición se
// repite. Con la misma hora final no se vuelve a escribir ni se audita.
func TestSetGraceUntil_RepeatingTheSameEndDoesNotMoveTheWindow(t *testing.T) {
	now := time.Now()
	base := now.Add(10 * time.Minute)
	session := closedSessionForGrace(now, &base)
	service, store := newGraceService(session)
	until := base.Add(30 * time.Minute).UTC().Truncate(time.Second)

	if _, err := service.SetGraceUntil(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, until); err != nil {
		t.Fatalf("el primer ajuste falló: %v", err)
	}
	// El servidor ya guardó la ventana: el segundo clic pide la misma hora.
	saved := until
	session.GraceUntil = &saved
	again, err := service.SetGraceUntil(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, until)
	if err != nil {
		t.Fatalf("el ajuste repetido falló: %v", err)
	}
	if len(store.graceWrites) != 1 {
		t.Fatalf("un clic repetido no debía volver a escribir la ventana: %v", store.graceWrites)
	}
	if again == nil || again.GraceUntil == nil || !again.GraceUntil.Equal(until) {
		t.Fatal("el ajuste repetido debía devolver la ventana vigente")
	}
}

func TestSetGraceUntil_RejectsPastAndBeyondMaximum(t *testing.T) {
	now := time.Now()
	service, store := newGraceService(closedSessionForGrace(now, nil))

	for _, until := range []time.Time{now.Add(-time.Minute), now.Add(8 * 24 * time.Hour)} {
		if _, err := service.SetGraceUntil(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, until); !errors.Is(err, domain.ErrAdvanceGraceDeadline) {
			t.Errorf("%v: se esperaba ErrAdvanceGraceDeadline, fue: %v", until, err)
		}
	}
	if len(store.graceWrites) != 0 {
		t.Fatalf("una hora inválida no debía escribirse: %v", store.graceWrites)
	}
}

func TestSetGraceUntil_RejectsAnOpenSession(t *testing.T) {
	service, store := newGraceService(newOpenSession())
	until := time.Now().Add(15 * time.Minute).UTC().Truncate(time.Second)

	_, err := service.SetGraceUntil(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, until)
	if !errors.Is(err, domain.ErrAdvanceNotCloseable) {
		t.Fatalf("se esperaba ErrAdvanceNotCloseable, fue: %v", err)
	}
	if len(store.graceWrites) != 0 {
		t.Fatalf("con el aula abierta no se escribía la ventana: %v", store.graceWrites)
	}
}

func TestCloseGrace_SetsTheWindowToNow(t *testing.T) {
	now := time.Now()
	service, store := newGraceService(closedSessionForGrace(now, nil))

	before := time.Now()
	if _, err := service.CloseGrace(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID); err != nil {
		t.Fatalf("CloseGrace falló: %v", err)
	}
	if len(store.graceWrites) != 1 {
		t.Fatal("cerrar la ventana debía escribir su hora")
	}
	closedAt := store.graceWrites[0]
	if closedAt.Before(before.Add(-time.Second)) || closedAt.After(time.Now().Add(time.Second)) {
		t.Fatalf("la ventana debía cerrarse ahora, quedó en %v", closedAt)
	}
}
