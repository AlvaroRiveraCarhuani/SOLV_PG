package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// fakeSessionPause registra las llamadas al sistema de archivos de los alumnos.
type fakeSessionPause struct {
	pauseCalls  int
	resumeCalls int
	report      services.FreezeReport
	err         error
}

func (f *fakeSessionPause) PauseSession(context.Context, string, string, string, string) (services.FreezeReport, error) {
	f.pauseCalls++
	return f.report, f.err
}

func (f *fakeSessionPause) ResumeSession(context.Context, string, string, string, string) (services.FreezeReport, error) {
	f.resumeCalls++
	return f.report, f.err
}

func examSessionForPause() *domain.ClassSession {
	session := newOpenSession()
	session.Purpose = string(domain.ExercisePurposeExam)
	return session
}

func TestPauseExam_SendsTheSessionToTheVolumeFreezer(t *testing.T) {
	session := examSessionForPause()
	pause := &fakeSessionPause{report: services.FreezeReport{Frozen: 2}}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, &fakeAdvanceSessions{session: session}).
		WithPauseStore(pause)

	if _, err := svc.Freeze(context.Background(), advanceTenantID, advanceTeacherID, "teacher", session.ID); err != nil {
		t.Fatalf("pausar un examen no debía fallar: %v", err)
	}
	if pause.pauseCalls != 1 || pause.resumeCalls != 0 {
		t.Fatalf("se esperaba una pausa y ninguna reanudación: %d/%d", pause.pauseCalls, pause.resumeCalls)
	}
}

// La pausa dejó de ser exclusiva del examen: una clase también puede dejar a
// los alumnos en solo lectura sin cerrar la sesión (ADR-005 §5.2).
func TestPauseClass_FreezesStudentVolumes(t *testing.T) {
	session := newOpenSession()
	session.Purpose = string(domain.ExercisePurposeClass)
	pause := &fakeSessionPause{report: services.FreezeReport{Frozen: 1}}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, &fakeAdvanceSessions{session: session}).
		WithPauseStore(pause)

	if _, err := svc.Freeze(context.Background(), advanceTenantID, advanceTeacherID, "teacher", session.ID); err != nil {
		t.Fatalf("pausar una clase no debía fallar: %v", err)
	}
	if pause.pauseCalls != 1 {
		t.Fatalf("la pausa debía llegar al sistema de archivos: %d", pause.pauseCalls)
	}
}

func TestPauseExam_KeepsTheSessionEditableWhenNoVolumeCouldBeFrozen(t *testing.T) {
	session := examSessionForPause()
	pause := &fakeSessionPause{report: services.FreezeReport{Failed: 3}}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, &fakeAdvanceSessions{session: session}).
		WithPauseStore(pause)

	_, err := svc.Freeze(context.Background(), advanceTenantID, advanceTeacherID, "teacher", session.ID)
	if !errors.Is(err, domain.ErrExamPauseFailed) {
		t.Fatalf("sin volúmenes congelados la sesión no debía marcarse: %v", err)
	}
}

func TestResumeClass_ReturnsTheVolumesToWritable(t *testing.T) {
	session := newOpenSession()
	session.Purpose = string(domain.ExercisePurposeClass)
	pause := &fakeSessionPause{}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, &fakeAdvanceSessions{session: session}).
		WithPauseStore(pause)

	if _, err := svc.Unfreeze(context.Background(), advanceTenantID, advanceTeacherID, "teacher", session.ID); err != nil {
		t.Fatalf("reanudar una clase no debía fallar: %v", err)
	}
	if pause.resumeCalls != 1 {
		t.Fatalf("se esperaba una reanudación: %d", pause.resumeCalls)
	}
}

func TestResumeExam_ReturnsTheVolumesToWritable(t *testing.T) {
	session := examSessionForPause()
	pause := &fakeSessionPause{}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, &fakeAdvanceSessions{session: session}).
		WithPauseStore(pause)

	if _, err := svc.Unfreeze(context.Background(), advanceTenantID, advanceTeacherID, "teacher", session.ID); err != nil {
		t.Fatalf("reanudar un examen no debía fallar: %v", err)
	}
	if pause.resumeCalls != 1 {
		t.Fatalf("se esperaba una reanudación: %d", pause.resumeCalls)
	}
}
