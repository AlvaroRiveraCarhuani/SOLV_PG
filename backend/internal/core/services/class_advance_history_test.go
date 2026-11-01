package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// recordingAudit guarda cada entrada para revisar acción, actor y metadata.
type recordingAudit struct {
	logs []*domain.AuditLog
}

func (r *recordingAudit) Create(_ context.Context, log *domain.AuditLog) error {
	r.logs = append(r.logs, log)
	return nil
}

func (r *recordingAudit) action(action string) *domain.AuditLog {
	for _, log := range r.logs {
		if log.Action == action {
			return log
		}
	}
	return nil
}

// failingAudit simula un registro caído: la acción del aula no debe fallar por eso.
type failingAudit struct{}

func (failingAudit) Create(context.Context, *domain.AuditLog) error {
	return errors.New("auditoría no disponible")
}

type fakeHistoryStore struct {
	events []domain.ClassHistoryEvent
	calls  int
}

func (f *fakeHistoryStore) SessionHistory(context.Context, string, string, int) ([]domain.ClassHistoryEvent, error) {
	f.calls++
	return f.events, nil
}

func auditMeta(t *testing.T, log *domain.AuditLog) map[string]any {
	t.Helper()
	if log == nil {
		t.Fatal("faltaba la entrada de auditoría")
	}
	var meta map[string]any
	if err := json.Unmarshal(log.Metadata, &meta); err != nil {
		t.Fatalf("metadata inválida: %v", err)
	}
	return meta
}

func TestClassAdvanceAudit_RecordsTeacherActionsWithPreviousAndNewWindow(t *testing.T) {
	session := newOpenSession()
	store := &fakeGraceStore{fakeAdvanceStore: fakeAdvanceStore{
		participants: []string{advanceStudentID},
		purpose:      "class",
	}}
	sessions := &fakeAdvanceSessions{session: session}
	audit := &recordingAudit{}
	svc := services.NewClassAdvanceService(store, sessions)
	svc.SetAuditRepository(audit)

	ctx := context.Background()
	_, recipients, err := svc.Issue(ctx, services.IssueAdvanceInput{
		TenantID:       advanceTenantID,
		TeacherID:      advanceTeacherID,
		Role:           "teacher",
		SessionID:      session.ID,
		ExerciseID:     "11111111-1111-1111-1111-111111111111",
		Scope:          domain.AdvanceScopeClass,
		SnapshotKey:    "class-advances/prueba.tar.gz",
		SnapshotSHA256: strings.Repeat("a", 64),
		SnapshotSize:   128,
		FileCount:      2,
	})
	if err != nil {
		t.Fatalf("Issue falló: %v", err)
	}
	if len(recipients) != 1 {
		t.Fatalf("se esperaba un destinatario, hubo %d", len(recipients))
	}
	if _, err := svc.Freeze(ctx, advanceTenantID, advanceTeacherID, "teacher", session.ID); err != nil {
		t.Fatalf("Freeze falló: %v", err)
	}
	if _, err := svc.Unfreeze(ctx, advanceTenantID, advanceTeacherID, "teacher", session.ID); err != nil {
		t.Fatalf("Unfreeze falló: %v", err)
	}

	closedAt := time.Now()
	session.Status = domain.ClassSessionStatusClosed
	session.ClosesAt = &closedAt
	if _, err := svc.AddGrace(ctx, advanceTenantID, advanceTeacherID, "teacher", session.ID, 15); err != nil {
		t.Fatalf("AddGrace falló: %v", err)
	}
	deadline := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	if _, err := svc.SetGraceDeadline(ctx, advanceTenantID, advanceTeacherID, "teacher", session.ID, deadline); err != nil {
		t.Fatalf("SetGraceDeadline falló: %v", err)
	}
	if _, err := svc.CloseGrace(ctx, advanceTenantID, advanceTeacherID, "teacher", session.ID); err != nil {
		t.Fatalf("CloseGrace falló: %v", err)
	}

	wantActions := []string{
		domain.AuditClassAdvanceEmitted,
		domain.AuditClassFrozen,
		domain.AuditClassUnfrozen,
		domain.AuditClassAdvanceGraceExtended,
		domain.AuditClassAdvanceGraceDeadline,
		domain.AuditClassAdvanceGraceClosed,
	}
	if len(audit.logs) != len(wantActions) {
		t.Fatalf("se esperaban %d entradas, hubo %d", len(wantActions), len(audit.logs))
	}
	for i, want := range wantActions {
		log := audit.logs[i]
		if log.Action != want {
			t.Errorf("entrada %d: acción %q, se esperaba %q", i, log.Action, want)
		}
		if log.TenantID != advanceTenantID || log.ActorID != advanceTeacherID {
			t.Errorf("entrada %d: tenant/actor inesperados: %+v", i, log)
		}
		if log.ResourceType != "class_session" || log.ResourceID == nil || *log.ResourceID != session.ID {
			t.Errorf("entrada %d: recurso inesperado: %+v", i, log)
		}
	}

	emitted := auditMeta(t, audit.action(domain.AuditClassAdvanceEmitted))
	if emitted["recipients"] != float64(1) || emitted["file_count"] != float64(2) {
		t.Errorf("metadata de emisión incompleta: %v", emitted)
	}
	if emitted["advance_id"] == "" {
		t.Error("la emisión debía guardar el id del avance")
	}

	extended := auditMeta(t, audit.action(domain.AuditClassAdvanceGraceExtended))
	if extended["minutes"] != float64(15) {
		t.Errorf("la suma debía guardar 15 minutos: %v", extended)
	}
	if extended["until"] == nil {
		t.Error("la suma debía guardar la nueva ventana")
	}
	if extended["previous_until"] != nil {
		t.Errorf("la primera suma no tenía ventana anterior: %v", extended["previous_until"])
	}

	deadlineMeta := auditMeta(t, audit.action(domain.AuditClassAdvanceGraceDeadline))
	if deadlineMeta["until"] == nil {
		t.Error("la fecha límite debía quedar en la metadata")
	}

	closed := auditMeta(t, audit.action(domain.AuditClassAdvanceGraceClosed))
	if closed["until"] == nil {
		t.Error("el cierre debía guardar la hora en que se cerró")
	}
}

func TestClassAdvanceAudit_RecordsStudentApplyAndDismiss(t *testing.T) {
	store := &fakeAdvanceStore{delivery: newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusOpen, nil)}
	volumes := &fakeAdvanceVolumes{}
	audit := &recordingAudit{}
	svc := services.NewClassAdvanceService(store, nil).
		WithApplier(&fakeAdvanceSnapshotsFiles{files: []domain.WorkspaceVolumeFile{
			{Path: "main.py", Content: []byte("nuevo\n")},
		}}, newRunningWorkspace(t), volumes)
	svc.SetAuditRepository(audit)

	ctx := context.Background()
	if _, err := svc.Apply(ctx, advanceTenantID, advanceStudentID, advanceID); err != nil {
		t.Fatalf("Apply falló: %v", err)
	}
	store.delivery = newAdvanceDelivery(domain.AdvanceRecipientPending, "class", domain.ClassSessionStatusOpen, nil)
	if err := svc.Dismiss(ctx, advanceTenantID, advanceStudentID, advanceID); err != nil {
		t.Fatalf("Dismiss falló: %v", err)
	}

	if len(audit.logs) != 2 {
		t.Fatalf("se esperaban dos entradas, hubo %d", len(audit.logs))
	}
	applied := audit.action(domain.AuditClassAdvanceApplied)
	if applied == nil || applied.ActorID != advanceStudentID {
		t.Fatalf("la aplicación debía registrarse con el alumno: %+v", applied)
	}
	appliedMeta := auditMeta(t, applied)
	if appliedMeta["advance_id"] != advanceID || appliedMeta["files_written"] != float64(1) {
		t.Errorf("metadata de aplicación incompleta: %v", appliedMeta)
	}

	dismissed := audit.action(domain.AuditClassAdvanceDismissed)
	if dismissed == nil || dismissed.ActorID != advanceStudentID {
		t.Fatalf("el descarte debía registrarse con el alumno: %+v", dismissed)
	}
	if meta := auditMeta(t, dismissed); meta["advance_id"] != advanceID {
		t.Errorf("metadata de descarte incompleta: %v", meta)
	}
}

func TestClassAdvanceAudit_FailureDoesNotBreakTheAction(t *testing.T) {
	session := newOpenSession()
	store := &fakeGraceStore{fakeAdvanceStore: fakeAdvanceStore{participants: []string{advanceStudentID}, purpose: "class"}}
	svc := services.NewClassAdvanceService(store, &fakeAdvanceSessions{session: session})
	svc.SetAuditRepository(failingAudit{})

	if _, err := svc.Freeze(context.Background(), advanceTenantID, advanceTeacherID, "teacher", session.ID); err != nil {
		t.Fatalf("un fallo de auditoría no debía romper la congelación: %v", err)
	}
}

func TestSessionHistory_ReadsEventsForTheOwningSession(t *testing.T) {
	session := newOpenSession()
	history := &fakeHistoryStore{events: []domain.ClassHistoryEvent{
		{ID: "evento-1", Action: domain.AuditClassFrozen, ActorName: "Docente"},
	}}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, &fakeAdvanceSessions{session: session})
	svc.SetHistoryStore(history)

	events, err := svc.SessionHistory(context.Background(), advanceTenantID, advanceTeacherID, "teacher", session.ID)
	if err != nil {
		t.Fatalf("SessionHistory falló: %v", err)
	}
	if len(events) != 1 || events[0].Action != domain.AuditClassFrozen {
		t.Fatalf("historial inesperado: %+v", events)
	}
	if history.calls != 1 {
		t.Errorf("se esperaba una lectura del historial, hubo %d", history.calls)
	}
}

func TestSessionHistory_WithoutStoreReturnsAnEmptyList(t *testing.T) {
	session := newOpenSession()
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, &fakeAdvanceSessions{session: session})

	events, err := svc.SessionHistory(context.Background(), advanceTenantID, advanceTeacherID, "teacher", session.ID)
	if err != nil {
		t.Fatalf("SessionHistory falló: %v", err)
	}
	if events == nil || len(events) != 0 {
		t.Fatalf("sin almacén se esperaba una lista vacía, no nil: %+v", events)
	}
}

func TestSessionHistory_RejectsAnInvalidSessionID(t *testing.T) {
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, &fakeAdvanceSessions{session: newOpenSession()})

	_, err := svc.SessionHistory(context.Background(), advanceTenantID, advanceTeacherID, "teacher", "no-es-un-uuid")
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("se esperaba ErrSessionNotFound, fue: %v", err)
	}
}
