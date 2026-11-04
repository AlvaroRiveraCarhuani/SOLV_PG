package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type fakeExamFreezeStore struct {
	isExam       bool
	participants []domain.ExamFreezePracticeEnvironment
	closed       []domain.ExamSessionRef
	frozenMarks  map[string]bool
	markErr      error
}

func (f *fakeExamFreezeStore) SessionIsExam(context.Context, string, string) (bool, error) {
	return f.isExam, nil
}

func (f *fakeExamFreezeStore) ParticipantPracticeEnvironments(context.Context, string, string) ([]domain.ExamFreezePracticeEnvironment, error) {
	return f.participants, nil
}

func (f *fakeExamFreezeStore) ClosedExamSessions(context.Context, int) ([]domain.ExamSessionRef, error) {
	return f.closed, nil
}

func (f *fakeExamFreezeStore) SetPracticeEnvironmentFrozen(_ context.Context, _, environmentID string, frozen bool, _ time.Time) error {
	if f.markErr != nil {
		return f.markErr
	}
	if f.frozenMarks == nil {
		f.frozenMarks = map[string]bool{}
	}
	f.frozenMarks[environmentID] = frozen
	return nil
}

type fakeExamSessions struct {
	session *domain.ClassSession
	err     error
}

func (f *fakeExamSessions) GetSession(context.Context, string, string, string) (*domain.ClassSession, error) {
	return f.session, f.err
}

type fakeExamPracticeEnvironments struct {
	byID map[string]*domain.PracticeEnvironment
}

func (f *fakeExamPracticeEnvironments) GetByID(_ context.Context, id string) (*domain.PracticeEnvironment, error) {
	instance, ok := f.byID[id]
	if !ok {
		// Igual que el repositorio: una fila que no existe se informa como
		// dominio, no como un fallo de lectura.
		return nil, domain.ErrNotFound
	}
	return instance, nil
}

type mountCall struct {
	environmentID string
	readOnly    bool
}

type fakeMounts struct {
	calls []mountCall
	err   error
}

func (f *fakeMounts) SetPracticeEnvironmentMount(_ context.Context, instance *domain.PracticeEnvironment, readOnly bool) error {
	if f.err != nil {
		return f.err
	}
	f.calls = append(f.calls, mountCall{environmentID: instance.ID, readOnly: readOnly})
	return nil
}

type fakeExamAudit struct {
	logs []*domain.AuditLog
}

func (f *fakeExamAudit) Create(_ context.Context, log *domain.AuditLog) error {
	f.logs = append(f.logs, log)
	return nil
}

func newExamFreezeFixture(t *testing.T) (*ExamFreezeService, *fakeExamFreezeStore, *fakeMounts, *fakeExamAudit) {
	t.Helper()
	store := &fakeExamFreezeStore{
		isExam: true,
		participants: []domain.ExamFreezePracticeEnvironment{
			{EnvironmentID: "ws-a", StudentID: "student-a"},
			{EnvironmentID: "ws-b", StudentID: "student-b"},
		},
	}
	sessions := &fakeExamSessions{session: &domain.ClassSession{
		ID:        "11111111-1111-1111-1111-111111111111",
		Status:    domain.ClassSessionStatusClosed,
		CreatedBy: "teacher-1",
	}}
	environments := &fakeExamPracticeEnvironments{byID: map[string]*domain.PracticeEnvironment{
		"ws-a": {ID: "ws-a", StudentID: "student-a", SubjectID: "subject-1"},
		"ws-b": {ID: "ws-b", StudentID: "student-b", SubjectID: "subject-1"},
	}}
	mounts := &fakeMounts{}
	audit := &fakeExamAudit{}

	service := NewExamFreezeService(store, sessions, environments, mounts).
		SetAuditRepository(audit).
		WithClock(func() time.Time { return time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC) })
	return service, store, mounts, audit
}

func TestFreezeSession_MountsReadOnlyAndSkipsFrozen(t *testing.T) {
	service, store, mounts, audit := newExamFreezeFixture(t)
	alreadyFrozen := time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC)
	store.participants[1].FrozenAt = &alreadyFrozen

	report, err := service.FreezeSession(context.Background(), "tenant-1", "teacher-1", "teacher", "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("FreezeSession falló: %v", err)
	}
	if report.Frozen != 1 || report.Skipped != 1 || report.Failed != 0 {
		t.Fatalf("conteo inesperado: %+v", report)
	}
	if len(mounts.calls) != 1 || mounts.calls[0].environmentID != "ws-a" || !mounts.calls[0].readOnly {
		t.Fatalf("se esperaba un solo montaje en solo lectura de ws-a: %+v", mounts.calls)
	}
	if !store.frozenMarks["ws-a"] {
		t.Fatalf("ws-a debía quedar marcado como congelado: %v", store.frozenMarks)
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != domain.AuditExamPracticeEnvironmentFrozen {
		t.Fatalf("se esperaba una entrada de auditoría de congelamiento: %+v", audit.logs)
	}
}

func TestFreezeSession_RejectsOpenExam(t *testing.T) {
	service, store, mounts, _ := newExamFreezeFixture(t)
	service.sessions.(*fakeExamSessions).session.Status = domain.ClassSessionStatusOpen

	_, err := service.FreezeSession(context.Background(), "tenant-1", "teacher-1", "teacher", "11111111-1111-1111-1111-111111111111")
	if !errors.Is(err, domain.ErrExamSessionOpen) {
		t.Fatalf("se esperaba ErrExamSessionOpen, fue: %v", err)
	}
	if len(mounts.calls) != 0 || len(store.frozenMarks) != 0 {
		t.Fatal("un examen abierto no debía montar ni marcar nada")
	}
}

func TestFreezeSession_RejectsClassSession(t *testing.T) {
	service, store, mounts, _ := newExamFreezeFixture(t)
	store.isExam = false

	_, err := service.FreezeSession(context.Background(), "tenant-1", "teacher-1", "teacher", "11111111-1111-1111-1111-111111111111")
	if !errors.Is(err, domain.ErrExamFreezeNotAllowed) {
		t.Fatalf("se esperaba ErrExamFreezeNotAllowed, fue: %v", err)
	}
	if len(mounts.calls) != 0 {
		t.Fatal("una clase no debía congelar volúmenes")
	}
}

// Un participante cuyo laboratorio ya no existe no es un fallo del cierre: no
// hay volumen que congelar y el barrido no tiene que reintentarlo.
func TestFreezeSession_SkipsPracticeEnvironmentThatNoLongerExists(t *testing.T) {
	service, store, mounts, _ := newExamFreezeFixture(t)
	delete(service.environments.(*fakeExamPracticeEnvironments).byID, "ws-b")

	report, err := service.FreezeSession(context.Background(), "tenant-1", "teacher-1", "teacher", "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("FreezeSession falló: %v", err)
	}
	if report.Frozen != 1 || report.Skipped != 1 || report.Failed != 0 {
		t.Fatalf("conteo inesperado: %+v", report)
	}
	if len(mounts.calls) != 1 || mounts.calls[0].environmentID != "ws-a" {
		t.Fatalf("solo el laboratorio existente debía montarse: %+v", mounts.calls)
	}
	if len(store.frozenMarks) != 1 || !store.frozenMarks["ws-a"] {
		t.Fatalf("solo el laboratorio existente debía marcarse: %v", store.frozenMarks)
	}
}

func TestFreezeSession_CountsMountFailuresWithoutStopping(t *testing.T) {
	service, store, mounts, audit := newExamFreezeFixture(t)
	mounts.err = errors.New("docker no responde")

	report, err := service.FreezeSession(context.Background(), "tenant-1", "teacher-1", "teacher", "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("un fallo de montaje no debía cortar el cierre: %v", err)
	}
	if report.Failed != 2 || report.Frozen != 0 {
		t.Fatalf("conteo inesperado: %+v", report)
	}
	if len(store.frozenMarks) != 0 {
		t.Fatal("sin montaje no se podía marcar el laboratorio")
	}
	if len(audit.logs) != 0 {
		t.Fatal("un fallo no debía escribir auditoría de congelamiento")
	}
}

func TestUnfreezeStudent_RequiresReason(t *testing.T) {
	service, _, mounts, _ := newExamFreezeFixture(t)

	_, err := service.UnfreezeStudent(context.Background(), "tenant-1", "teacher-1", "teacher", "11111111-1111-1111-1111-111111111111", "student-a", "   ")
	if !errors.Is(err, domain.ErrFreezeReasonRequired) {
		t.Fatalf("se esperaba ErrFreezeReasonRequired, fue: %v", err)
	}
	if len(mounts.calls) != 0 {
		t.Fatal("sin motivo no se debía tocar el contenedor")
	}
}

func TestUnfreezeStudent_RemountsReadWrite(t *testing.T) {
	service, store, mounts, audit := newExamFreezeFixture(t)
	frozenAt := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	store.participants[0].FrozenAt = &frozenAt
	store.participants[0].Status = domain.PracticeEnvironmentStatusFrozen

	report, err := service.UnfreezeStudent(context.Background(), "tenant-1", "teacher-1", "teacher", "11111111-1111-1111-1111-111111111111", "student-a", "se le cortó la luz")
	if err != nil {
		t.Fatalf("UnfreezeStudent falló: %v", err)
	}
	if report.Frozen != 1 || report.Failed != 0 {
		t.Fatalf("conteo inesperado: %+v", report)
	}
	if len(mounts.calls) != 1 || mounts.calls[0].readOnly {
		t.Fatalf("se esperaba un montaje en escritura: %+v", mounts.calls)
	}
	if store.frozenMarks["ws-a"] {
		t.Fatal("ws-a debía quedar sin marca de congelado")
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != domain.AuditExamPracticeEnvironmentUnfrozen {
		t.Fatalf("se esperaba la auditoría de reapertura: %+v", audit.logs)
	}
}

func TestUnfreezeStudent_UnknownStudent(t *testing.T) {
	service, _, _, _ := newExamFreezeFixture(t)

	_, err := service.UnfreezeStudent(context.Background(), "tenant-1", "teacher-1", "teacher", "11111111-1111-1111-1111-111111111111", "student-z", "motivo")
	if !errors.Is(err, domain.ErrFreezeStudentNotInSession) {
		t.Fatalf("se esperaba ErrFreezeStudentNotInSession, fue: %v", err)
	}
}

func TestFreezeClosedExams_SweepsEverySession(t *testing.T) {
	service, store, mounts, _ := newExamFreezeFixture(t)
	store.closed = []domain.ExamSessionRef{
		{TenantID: "tenant-1", SessionID: "11111111-1111-1111-1111-111111111111"},
		{TenantID: "tenant-1", SessionID: "22222222-2222-2222-2222-222222222222"},
	}

	report, err := service.FreezeClosedExams(context.Background(), 10)
	if err != nil {
		t.Fatalf("FreezeClosedExams falló: %v", err)
	}
	if report.Frozen != 4 || len(mounts.calls) != 4 {
		t.Fatalf("el barrido debía congelar los cuatro laboratorios: %+v / %+v", report, mounts.calls)
	}
}

func TestFreezeClosedExams_NoCandidates(t *testing.T) {
	service, _, mounts, _ := newExamFreezeFixture(t)

	report, err := service.FreezeClosedExams(context.Background(), 10)
	if err != nil {
		t.Fatalf("FreezeClosedExams falló: %v", err)
	}
	if report.Frozen != 0 || len(service.store.(*fakeExamFreezeStore).frozenMarks) != 0 || len(mounts.calls) != 0 {
		t.Fatalf("sin aulas cerradas no debía tocarse nada: %+v", report)
	}
}
