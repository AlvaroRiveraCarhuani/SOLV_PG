package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

const (
	advanceTeacherID  = "77777777-7777-7777-7777-777777777777"
	advanceExerciseID = "11111111-1111-1111-1111-111111111111"
)

// fakeAdvanceSessions devuelve siempre la sesión preparada por la prueba.
type fakeAdvanceSessions struct {
	session *domain.ClassSession
}

func (f *fakeAdvanceSessions) OpenSession(ctx context.Context, tenantID, teacherID, subjectID string, exerciseID *string, durationMinutes int) (*domain.ClassSession, error) {
	return f.session, nil
}

func (f *fakeAdvanceSessions) CloseSession(ctx context.Context, tenantID, teacherID, sessionID string) (*domain.ClassSession, error) {
	return f.session, nil
}

func (f *fakeAdvanceSessions) GetOpenSessionForTeacher(ctx context.Context, tenantID, teacherID string) (*domain.ClassSession, error) {
	return f.session, nil
}

func (f *fakeAdvanceSessions) GetLastClosedSessionForTeacher(ctx context.Context, tenantID, teacherID string) (*domain.ClassSession, error) {
	return f.session, nil
}

func (f *fakeAdvanceSessions) GetSession(ctx context.Context, tenantID, teacherID, sessionID string) (*domain.ClassSession, error) {
	if f.session == nil {
		return nil, domain.ErrSessionNotFound
	}
	return f.session, nil
}

func (f *fakeAdvanceSessions) Board(ctx context.Context, tenantID, teacherID, sessionID string) (*domain.ClassSessionBoard, error) {
	return nil, nil
}

func (f *fakeAdvanceSessions) OpenSessionsForStudent(ctx context.Context, tenantID, studentID string) ([]domain.ClassSession, error) {
	return nil, nil
}

func (f *fakeAdvanceSessions) TouchParticipant(ctx context.Context, tenantID, studentID, sessionID string) error {
	return nil
}

func (f *fakeAdvanceSessions) AttachPracticeEnvironmentToOpenSession(ctx context.Context, tenantID, studentID, subjectID, environmentID string) error {
	return nil
}

// newOpenSession crea un aula atada a una práctica: el avance se emite desde el
// entorno de esa práctica y no desde uno de materia.
func newOpenSession() *domain.ClassSession {
	exerciseID := advanceExerciseID
	return &domain.ClassSession{
		ID:         "55555555-5555-5555-5555-555555555555",
		TenantID:   advanceTenantID,
		SubjectID:  advanceSubjectID,
		ExerciseID: &exerciseID,
		Status:     domain.ClassSessionStatusOpen,
		OpensAt:    time.Now(),
	}
}

func TestEmitAdvance_StoresSnapshotAndIssuesTheAdvance(t *testing.T) {
	store := &fakeAdvanceStore{
		participants: []string{advanceStudentID, "88888888-8888-8888-8888-888888888888"},
		purpose:      "class",
	}
	volumes := &fakeAdvanceVolumes{existing: []domain.PracticeEnvironmentVolumeFile{
		{Path: "solucion.py", Content: []byte("print(1)\n")},
		{Path: "helpers.py", Content: []byte("x = 1\n")},
	}}
	snapshots := &fakeAdvanceSnapshotsFiles{}
	environments := newMockPracticeEnvironmentRepo()
	teacherPractice := advanceExerciseID
	if err := environments.Create(context.Background(), &domain.PracticeEnvironment{
		ID:         "99999999-9999-9999-9999-999999999999",
		StudentID:  advanceTeacherID,
		SubjectID:  advanceSubjectID,
		ExerciseID: &teacherPractice,
		Status:     domain.PracticeEnvironmentStatusRunning,
	}); err != nil {
		t.Fatalf("no se pudo preparar el entorno del docente: %v", err)
	}

	svc := services.NewClassAdvanceService(store, &fakeAdvanceSessions{session: newOpenSession()}).
		WithApplier(snapshots, environments, volumes)

	advance, recipients, err := svc.Emit(context.Background(), services.EmitAdvanceInput{
		TenantID:   advanceTenantID,
		TeacherID:  advanceTeacherID,
		Role:       "teacher",
		SessionID: newOpenSession().ID,
		Scope:     domain.AdvanceScopeClass,
	})
	if err != nil {
		t.Fatalf("error inesperado al emitir: %v", err)
	}

	wantKey := "class-advances/" + advanceTenantID + "/" + advance.ID + ".tar.gz"
	if snapshots.putKey != wantKey {
		t.Errorf("clave del snapshot = %q, se esperaba %q", snapshots.putKey, wantKey)
	}
	if _, ok := snapshots.putFiles["solucion.py"]; !ok {
		t.Errorf("el snapshot debía incluir el código del docente: %v", snapshots.putFiles)
	}
	if advance.SnapshotKey != wantKey || advance.FileCount != 2 {
		t.Errorf("el avance debía guardar clave y cantidad de archivos: %+v", advance)
	}
	if len(recipients) != 2 {
		t.Errorf("esperados 2 destinatarios, hubo %d", len(recipients))
	}
	if store.created == nil || store.created.ID != advance.ID {
		t.Errorf("el avance debía persistirse con el mismo id: %+v", store.created)
	}
	if len(snapshots.deleted) != 0 {
		t.Errorf("una emisión correcta no borra el snapshot: %v", snapshots.deleted)
	}
}

func TestEmitAdvance_RejectsClosedSessionBeforePackaging(t *testing.T) {
	session := newOpenSession()
	session.Status = domain.ClassSessionStatusClosed
	snapshots := &fakeAdvanceSnapshotsFiles{}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, &fakeAdvanceSessions{session: session}).
		WithApplier(snapshots, newRunningPracticeEnvironment(t), &fakeAdvanceVolumes{})

	_, _, err := svc.Emit(context.Background(), services.EmitAdvanceInput{
		TenantID:   advanceTenantID,
		TeacherID:  advanceTeacherID,
		Role:       "teacher",
		SessionID:  session.ID,
		Scope:      domain.AdvanceScopeClass,
	})
	if !errors.Is(err, domain.ErrAdvanceSessionClosed) {
		t.Fatalf("esperado aula cerrada, obtenido: %v", err)
	}
	if snapshots.putKey != "" {
		t.Error("no se debía empaquetar nada con el aula cerrada")
	}
}

func TestEmitAdvance_RequiresTheTeacherEnvironment(t *testing.T) {
	snapshots := &fakeAdvanceSnapshotsFiles{}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, &fakeAdvanceSessions{session: newOpenSession()}).
		WithApplier(snapshots, newMockPracticeEnvironmentRepo(), &fakeAdvanceVolumes{})

	_, _, err := svc.Emit(context.Background(), services.EmitAdvanceInput{
		TenantID:   advanceTenantID,
		TeacherID:  advanceTeacherID,
		Role:       "teacher",
		SessionID: newOpenSession().ID,
		Scope:     domain.AdvanceScopeClass,
	})
	if !errors.Is(err, domain.ErrAdvanceSourceMissing) {
		t.Fatalf("esperado entorno del docente ausente, obtenido: %v", err)
	}
	if snapshots.putKey != "" {
		t.Error("sin entorno no hay snapshot que guardar")
	}
}

// El avance vive en la práctica del aula: sin práctica no hay entorno que
// empaquetar y el motivo es de dominio, no un fallo de almacenamiento.
func TestEmitAdvance_RequiresPracticeInSession(t *testing.T) {
	session := newOpenSession()
	session.ExerciseID = nil
	snapshots := &fakeAdvanceSnapshotsFiles{}
	svc := services.NewClassAdvanceService(&fakeAdvanceStore{}, &fakeAdvanceSessions{session: session}).
		WithApplier(snapshots, newRunningPracticeEnvironment(t), &fakeAdvanceVolumes{})

	_, _, err := svc.Emit(context.Background(), services.EmitAdvanceInput{
		TenantID:  advanceTenantID,
		TeacherID: advanceTeacherID,
		Role:      "teacher",
		SessionID: session.ID,
		Scope:     domain.AdvanceScopeClass,
	})
	if !errors.Is(err, domain.ErrAdvancePracticeRequired) {
		t.Fatalf("esperado aula sin práctica, obtenido: %v", err)
	}
	if snapshots.putKey != "" {
		t.Error("sin práctica no se empaqueta nada")
	}
}

func TestEmitAdvance_DeletesSnapshotWhenRecipientsAreRejected(t *testing.T) {
	store := &fakeAdvanceStore{
		participants: []string{advanceStudentID},
		purpose:      "class",
	}
	volumes := &fakeAdvanceVolumes{existing: []domain.PracticeEnvironmentVolumeFile{
		{Path: "solucion.py", Content: []byte("print(1)\n")},
	}}
	snapshots := &fakeAdvanceSnapshotsFiles{}
	environments := newMockPracticeEnvironmentRepo()
	teacherPractice := advanceExerciseID
	if err := environments.Create(context.Background(), &domain.PracticeEnvironment{
		ID:         "99999999-9999-9999-9999-999999999999",
		StudentID:  advanceTeacherID,
		SubjectID:  advanceSubjectID,
		ExerciseID: &teacherPractice,
		Status:     domain.PracticeEnvironmentStatusRunning,
	}); err != nil {
		t.Fatalf("no se pudo preparar el entorno del docente: %v", err)
	}

	svc := services.NewClassAdvanceService(store, &fakeAdvanceSessions{session: newOpenSession()}).
		WithApplier(snapshots, environments, volumes)

	_, _, err := svc.Emit(context.Background(), services.EmitAdvanceInput{
		TenantID:   advanceTenantID,
		TeacherID:  advanceTeacherID,
		Role:       "teacher",
		SessionID:  newOpenSession().ID,
		Scope:      domain.AdvanceScopeSelected,
		StudentIDs: []string{"88888888-8888-8888-8888-888888888888"},
	})
	if !errors.Is(err, domain.ErrAdvanceRecipientNotInSession) {
		t.Fatalf("esperado destinatario fuera del aula, obtenido: %v", err)
	}
	if len(snapshots.deleted) != 1 || snapshots.deleted[0] != snapshots.putKey {
		t.Errorf("el snapshot debía borrarse al fallar la emisión: %v", snapshots.deleted)
	}
}

// La sesión con práctica se lee del entorno de esa práctica: desde ADR-059 el
// IDE del docente se acota al ejercicio y el entorno por materia ya no se crea.
func TestEmitAdvance_ReadsThePracticeEnvironment(t *testing.T) {
	exerciseID := "11111111-1111-1111-1111-111111111111"
	session := newOpenSession()
	session.ExerciseID = &exerciseID

	store := &fakeAdvanceStore{
		participants: []string{advanceStudentID},
		purpose:      "class",
	}
	volumes := &fakeAdvanceVolumes{existing: []domain.PracticeEnvironmentVolumeFile{
		{Path: "practica.py", Content: []byte("print(2)\n")},
	}}
	snapshots := &fakeAdvanceSnapshotsFiles{}

	practice := &domain.PracticeEnvironment{
		ID:         "99999999-9999-9999-9999-999999999998",
		StudentID:  advanceTeacherID,
		SubjectID:  advanceSubjectID,
		ExerciseID: &exerciseID,
		Status:     domain.PracticeEnvironmentStatusRunning,
	}
	repo := newPracticePracticeEnvironmentRepo()
	if err := repo.Create(context.Background(), practice); err != nil {
		t.Fatalf("no se pudo preparar el entorno de la práctica: %v", err)
	}

	svc := services.NewClassAdvanceService(store, &fakeAdvanceSessions{session: session}).
		WithApplier(snapshots, repo, volumes)

	_, _, err := svc.Emit(context.Background(), services.EmitAdvanceInput{
		TenantID:   advanceTenantID,
		TeacherID:  advanceTeacherID,
		Role:       "teacher",
		SessionID: session.ID,
		Scope:     domain.AdvanceScopeClass,
	})
	if err != nil {
		t.Fatalf("error inesperado al emitir: %v", err)
	}
	if len(volumes.snapshots) != 1 || volumes.snapshots[0] != practice.VolumeName() {
		t.Fatalf("se esperaba leer el volumen de la práctica %q, se leyó %v", practice.VolumeName(), volumes.snapshots)
	}
	if _, ok := snapshots.putFiles["practica.py"]; !ok {
		t.Errorf("el snapshot debía salir del entorno de la práctica: %v", snapshots.putFiles)
	}
}
