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

// fakeGraceStore hereda el resto del contrato del almacén de avances y solo
// registra la ventana que el servicio decide escribir.
type fakeGraceStore struct {
	fakeAdvanceStore
	graceUntil *time.Time
}

func (f *fakeGraceStore) SetGrace(ctx context.Context, tenantID, teacherID, sessionID string, until time.Time) (*domain.ClassSession, error) {
	f.graceUntil = &until
	return &domain.ClassSession{
		ID:         sessionID,
		TenantID:   tenantID,
		Status:     domain.ClassSessionStatusClosed,
		GraceUntil: &until,
	}, nil
}

// closedSessionForGrace arma un aula cerrada con su hora de cierre y la ventana
// vigente, que es lo que mira el servicio antes de sumar o fijar una fecha.
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

func TestAddGrace_AddsToCurrentWindow(t *testing.T) {
	now := time.Now()
	base := now.Add(10 * time.Minute)
	service, store := newGraceService(closedSessionForGrace(now, &base))

	before := time.Now()
	session, err := service.AddGrace(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, 15)
	if err != nil {
		t.Fatalf("AddGrace falló: %v", err)
	}
	if store.graceUntil == nil {
		t.Fatal("el almacén debía recibir la nueva ventana")
	}
	delta := store.graceUntil.Sub(before)
	if delta < 24*time.Minute || delta > 26*time.Minute {
		t.Fatalf("sumar 15 minutos a una ventana de 10 debía dejar 25, quedó a %v", delta)
	}
	if session == nil || session.GraceUntil == nil {
		t.Fatal("la sesión devuelta debía traer la ventana")
	}
}

func TestAddGrace_ReopensAnExpiredWindowFromNow(t *testing.T) {
	now := time.Now()
	expired := now.Add(-time.Hour)
	service, store := newGraceService(closedSessionForGrace(now, &expired))

	before := time.Now()
	if _, err := service.AddGrace(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, 20); err != nil {
		t.Fatalf("AddGrace sobre una ventana vencida falló: %v", err)
	}
	delta := store.graceUntil.Sub(before)
	if delta < 19*time.Minute || delta > 21*time.Minute {
		t.Fatalf("una ventana vencida debía contar desde ahora: %v", delta)
	}
}

func TestAddGrace_RejectsBeyondTheInstitutionalMaximum(t *testing.T) {
	closesAt := time.Now().Add(-8 * 24 * time.Hour)
	service, store := newGraceService(closedSessionForGrace(closesAt, nil))

	_, err := service.AddGrace(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, 60)
	if !errors.Is(err, domain.ErrAdvanceGraceExceeded) {
		t.Fatalf("se esperaba ErrAdvanceGraceExceeded, fue: %v", err)
	}
	if store.graceUntil != nil {
		t.Fatal("una ventana fuera del máximo no debía escribirse")
	}
}

func TestAddGrace_RejectsAnOpenSession(t *testing.T) {
	session := newOpenSession()
	service, store := newGraceService(session)

	_, err := service.AddGrace(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, 15)
	if !errors.Is(err, domain.ErrAdvanceNotCloseable) {
		t.Fatalf("se esperaba ErrAdvanceNotCloseable, fue: %v", err)
	}
	if store.graceUntil != nil {
		t.Fatal("con el aula abierta no se escribía la ventana")
	}
}

func TestAddGrace_RejectsMinutesOutOfRange(t *testing.T) {
	now := time.Now()
	service, _ := newGraceService(closedSessionForGrace(now, nil))

	for _, minutes := range []int{0, 481, -5} {
		if _, err := service.AddGrace(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, minutes); !errors.Is(err, domain.ErrAdvanceGraceInvalid) {
			t.Errorf("%d minutos: se esperaba ErrAdvanceGraceInvalid, fue: %v", minutes, err)
		}
	}
}

func TestSetGraceDeadline_AcceptsAFutureDateWithinTheMaximum(t *testing.T) {
	now := time.Now()
	service, store := newGraceService(closedSessionForGrace(now, nil))
	deadline := now.Add(24 * time.Hour).Truncate(time.Second)

	session, err := service.SetGraceDeadline(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, deadline)
	if err != nil {
		t.Fatalf("SetGraceDeadline falló: %v", err)
	}
	if store.graceUntil == nil || !store.graceUntil.Equal(deadline) {
		t.Fatalf("la fecha límite debía ser %v, fue %v", deadline, store.graceUntil)
	}
	if session == nil || session.GraceUntil == nil {
		t.Fatal("la sesión devuelta debía traer la ventana")
	}
}

func TestSetGraceDeadline_RejectsPastAndBeyondMaximum(t *testing.T) {
	now := time.Now()
	service, store := newGraceService(closedSessionForGrace(now, nil))

	for _, deadline := range []time.Time{now.Add(-time.Minute), now.Add(8 * 24 * time.Hour)} {
		if _, err := service.SetGraceDeadline(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID, deadline); !errors.Is(err, domain.ErrAdvanceGraceDeadline) {
			t.Errorf("%v: se esperaba ErrAdvanceGraceDeadline, fue: %v", deadline, err)
		}
	}
	if store.graceUntil != nil {
		t.Fatal("una fecha inválida no debía escribirse")
	}
}

func TestCloseGrace_SetsTheWindowToNow(t *testing.T) {
	now := time.Now()
	service, store := newGraceService(closedSessionForGrace(now, nil))

	before := time.Now()
	if _, err := service.CloseGrace(context.Background(), advanceTenantID, advanceTeacherID, "teacher", graceSessionID); err != nil {
		t.Fatalf("CloseGrace falló: %v", err)
	}
	if store.graceUntil == nil {
		t.Fatal("cerrar la ventana debía escribir su hora")
	}
	if store.graceUntil.Before(before.Add(-time.Second)) || store.graceUntil.After(time.Now().Add(time.Second)) {
		t.Fatalf("la ventana debía cerrarse ahora, quedó en %v", store.graceUntil)
	}
}
