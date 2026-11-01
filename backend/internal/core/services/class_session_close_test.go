package services

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

// closeStore solo responde al cierre: el resto del contrato del aula no se usa
// en esta prueba y queda sin implementar en la interfaz embebida.
type closeStore struct {
	domain.ClassSessionStore
	session   *domain.ClassSession
	err       error
	gotTenant string
	gotOwner  string
	calls     int
}

func (s *closeStore) CloseSession(_ context.Context, tenantID, teacherID, sessionID string) (*domain.ClassSession, error) {
	s.calls++
	s.gotTenant = tenantID
	s.gotOwner = teacherID
	if s.session != nil && s.session.ID == "" {
		s.session.ID = sessionID
	}
	return s.session, s.err
}

type fakeExamFreezer struct {
	report     FreezeReport
	err        error
	freezeCall int
	gotRole    string
	gotSession string
}

func (f *fakeExamFreezer) FreezeSession(_ context.Context, _, _, role, sessionID string) (FreezeReport, error) {
	f.freezeCall++
	f.gotRole = role
	f.gotSession = sessionID
	return f.report, f.err
}

func (f *fakeExamFreezer) UnfreezeStudent(context.Context, string, string, string, string, string, string) (FreezeReport, error) {
	return FreezeReport{}, nil
}

const closeTestSessionID = "0a4f0b52-3d1c-4f8e-9a4a-3c1d2b5e6f70"

func TestClassSessionService_CloseAulaReportsFreeze(t *testing.T) {
	store := &closeStore{session: &domain.ClassSession{ID: closeTestSessionID, Status: domain.ClassSessionStatusClosed}}
	freezer := &fakeExamFreezer{report: FreezeReport{Frozen: 3, Skipped: 1, Failed: 1}}
	svc := NewClassSessionService(store).WithExamFreezer(freezer)

	session, report, err := svc.CloseAula(context.Background(), domain.DefaultTenantID, "docente-1", "teacher", closeTestSessionID)
	if err != nil {
		t.Fatalf("cierre inesperado: %v", err)
	}
	if session == nil || session.Status != domain.ClassSessionStatusClosed {
		t.Fatalf("el aula debía quedar cerrada: %+v", session)
	}
	if report.Frozen != 3 || report.Skipped != 1 || report.Failed != 1 {
		t.Fatalf("el conteo del congelamiento debía volver al cierre: %+v", report)
	}
	if freezer.freezeCall != 1 || freezer.gotSession != closeTestSessionID {
		t.Fatalf("el cierre debía congelar el aula una vez: calls=%d session=%q", freezer.freezeCall, freezer.gotSession)
	}
	if store.gotOwner != "docente-1" || store.gotTenant != domain.DefaultTenantID {
		t.Fatalf("el cierre debía acotar por docente e institución: %+v", store)
	}
}

func TestClassSessionService_CloseAulaWithoutExamKeepsZeroReport(t *testing.T) {
	store := &closeStore{session: &domain.ClassSession{ID: closeTestSessionID, Status: domain.ClassSessionStatusClosed}}
	// Un aula de clase no es un examen: el congelador responde el error de
	// negocio y el cierre no cambia su resultado.
	freezer := &fakeExamFreezer{err: domain.ErrExamFreezeNotAllowed}
	svc := NewClassSessionService(store).WithExamFreezer(freezer)

	session, report, err := svc.CloseAula(context.Background(), domain.DefaultTenantID, "docente-1", "teacher", closeTestSessionID)
	if err != nil {
		t.Fatalf("una clase sin examen no debía fallar al cerrar: %v", err)
	}
	if session == nil {
		t.Fatal("el aula debía devolverse cerrada")
	}
	if report.Frozen != 0 || report.Skipped != 0 || report.Failed != 0 {
		t.Fatalf("una clase sin examen no debía reportar volúmenes: %+v", report)
	}
}

func TestClassSessionService_CloseAulaSkipsFreezeWhenStoreFails(t *testing.T) {
	store := &closeStore{err: errors.New("sin conexión")}
	freezer := &fakeExamFreezer{report: FreezeReport{Frozen: 1}}
	svc := NewClassSessionService(store).WithExamFreezer(freezer)

	_, report, err := svc.CloseAula(context.Background(), domain.DefaultTenantID, "docente-1", "teacher", closeTestSessionID)
	if err == nil {
		t.Fatal("el fallo del almacén debía propagarse")
	}
	if freezer.freezeCall != 0 {
		t.Fatal("sin aula cerrada no se debía congelar nada")
	}
	if report.Frozen != 0 {
		t.Fatalf("un cierre fallido no debía reportar congelados: %+v", report)
	}
}
