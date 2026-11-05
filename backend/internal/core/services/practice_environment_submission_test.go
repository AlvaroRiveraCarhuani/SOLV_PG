package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type fakeVolumeOrchestrator struct {
	*mockOrchestrator
	volume         string
	writtenFiles   map[string]string
	snapshotFiles  []domain.PracticeEnvironmentVolumeFile
	snapshotCalled bool
}

func (f *fakeVolumeOrchestrator) WritePracticeEnvironmentVolumeFiles(ctx context.Context, volumeName string, files map[string]string) error {
	f.volume = volumeName
	f.writtenFiles = files
	return nil
}

func (f *fakeVolumeOrchestrator) SnapshotPracticeEnvironmentVolume(ctx context.Context, volumeName string, maxBytes int64) ([]domain.PracticeEnvironmentVolumeFile, error) {
	f.volume = volumeName
	f.snapshotCalled = true
	return f.snapshotFiles, nil
}

type fakeSubmissionRepo struct {
	created *domain.Submission
	stored  map[string]*domain.Submission
	creates int
}

func (f *fakeSubmissionRepo) Create(ctx context.Context, submission *domain.Submission) error {
	f.creates++
	if f.stored != nil {
		if _, exists := f.stored[submission.ID]; exists {
			return domain.ErrDuplicateSubmission
		}
		f.stored[submission.ID] = submission
	}
	f.created = submission
	return nil
}

func (f *fakeSubmissionRepo) GetByID(ctx context.Context, tenantID, id string) (*domain.Submission, error) {
	if f.stored != nil {
		if sub, ok := f.stored[id]; ok {
			return sub, nil
		}
	}
	return nil, nil
}

func (f *fakeSubmissionRepo) ListByExerciseAndStudent(ctx context.Context, tenantID, exerciseID, studentID string) ([]*domain.Submission, error) {
	return nil, nil
}

func (f *fakeSubmissionRepo) ListByExercise(ctx context.Context, tenantID, exerciseID string) ([]*domain.Submission, error) {
	return nil, nil
}

func (f *fakeSubmissionRepo) ListByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.Submission, error) {
	return nil, nil
}

func (f *fakeSubmissionRepo) UpdateOverride(ctx context.Context, tenantID, id, verdict, reason string, score *int, gradedBy *string) error {
	return nil
}

const testExerciseID = "11111111-1111-1111-1111-111111111111"

func TestSubmitPracticeEnvironmentProject_CreatesPendingSubmissionWithSnapshot(t *testing.T) {
	wsRepo := newMockPracticeEnvironmentRepo()
	orch := &fakeVolumeOrchestrator{
		mockOrchestrator: &mockOrchestrator{},
		snapshotFiles: []domain.PracticeEnvironmentVolumeFile{
			{Path: "utils.py", Content: []byte("x = 2\n")},
			{Path: "main.py", Content: []byte("print(1)\n")},
		},
	}
	subRepo := &fakeSubmissionRepo{}
	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithExerciseSubmission(nil, subRepo)

	practice := testExerciseID
	environment := &domain.PracticeEnvironment{
		ID:         "22222222-2222-2222-2222-222222222222",
		TenantID:   domain.DefaultTenantID,
		StudentID:  "student-1",
		SubjectID:  "subject-1",
		ExerciseID: &practice,
		Status:     domain.PracticeEnvironmentStatusRunning,
	}
	if err := wsRepo.Create(context.Background(), environment); err != nil {
		t.Fatalf("no se pudo preparar el environment: %v", err)
	}

	_, err := svc.SubmitPracticeEnvironmentProject(context.Background(), environment.ID, "student-1", testExerciseID, map[string]string{
		"helpers/extra.py": "print(3)\n",
	})
	if err != nil {
		t.Fatalf("error inesperado al entregar el proyecto: %v", err)
	}

	if !orch.snapshotCalled {
		t.Error("esperado snapshot del volumen")
	}
	if orch.volume != "solv_workspace_student-1_subject-1_"+testExerciseID {
		t.Errorf("volumen inesperado: %s", orch.volume)
	}
	if _, ok := orch.writtenFiles["helpers/extra.py"]; !ok {
		t.Error("esperado que los archivos enviados se escriban en el volumen")
	}
	if subRepo.created == nil {
		t.Fatal("esperada submission persistida")
	}
	if subRepo.created.Verdict != "" {
		t.Errorf("una entrega de IDE no lleva veredicto del juez, obtenido: %q", subRepo.created.Verdict)
	}
	if subRepo.created.ReviewStatus != string(domain.ReviewPending) {
		t.Errorf("esperado review_status pending para una entrega IDE, obtenido: %q", subRepo.created.ReviewStatus)
	}
	if subRepo.created.Score != nil {
		t.Errorf("una entrega sin corregir no debe llevar nota, obtenido: %v", *subRepo.created.Score)
	}
	if subRepo.created.EnvironmentID == nil || *subRepo.created.EnvironmentID != environment.ID {
		t.Errorf("esperado environment_id %s, obtenido: %v", environment.ID, subRepo.created.EnvironmentID)
	}
	if subRepo.created.ExerciseID != testExerciseID {
		t.Errorf("esperado exercise_id %s, obtenido: %s", testExerciseID, subRepo.created.ExerciseID)
	}
	if !strings.HasPrefix(subRepo.created.Code, "# === FILE: main.py ===\n") {
		t.Errorf("snapshot sin cabecera determinista: %q", subRepo.created.Code)
	}
	if strings.Index(subRepo.created.Code, "main.py") > strings.Index(subRepo.created.Code, "utils.py") {
		t.Errorf("el snapshot no quedó ordenado alfabéticamente: %q", subRepo.created.Code)
	}
}

type fakeExerciseRepo struct {
	domain.ExerciseRepository
	exercise *domain.Exercise
	err      error
}

func (f *fakeExerciseRepo) GetByIDAndTenant(_ context.Context, _, _ string) (*domain.Exercise, error) {
	return f.exercise, f.err
}

type environmentFreezeCall struct {
	tenantID    string
	actorID     string
	environmentID string
}

type fakePracticeEnvironmentExamFreezer struct {
	calls []environmentFreezeCall
	err   error
}

func (f *fakePracticeEnvironmentExamFreezer) FreezePracticeEnvironment(_ context.Context, tenantID, actorID, environmentID string) error {
	f.calls = append(f.calls, environmentFreezeCall{tenantID: tenantID, actorID: actorID, environmentID: environmentID})
	return f.err
}

func newProjectSubmissionService(t *testing.T, purpose string) (*services.PracticeEnvironmentService, *fakeVolumeOrchestrator, *fakeSubmissionRepo, *fakePracticeEnvironmentExamFreezer, *domain.PracticeEnvironment) {
	t.Helper()
	wsRepo := newMockPracticeEnvironmentRepo()
	orch := &fakeVolumeOrchestrator{
		mockOrchestrator: &mockOrchestrator{},
		snapshotFiles: []domain.PracticeEnvironmentVolumeFile{
			{Path: "main.py", Content: []byte("print(1)\n")},
		},
	}
	subRepo := &fakeSubmissionRepo{}
	examRepo := &fakeExerciseRepo{exercise: &domain.Exercise{ID: testExerciseID, Purpose: purpose}}
	freezer := &fakePracticeEnvironmentExamFreezer{}
	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithExerciseSubmission(examRepo, subRepo).
		SetExamSubmissionFreezer(freezer)

	environment := &domain.PracticeEnvironment{
		ID:        "44444444-4444-4444-8444-444444444444",
		TenantID:  domain.DefaultTenantID,
		StudentID: "student-1",
		SubjectID: "subject-1",
		Status:    domain.PracticeEnvironmentStatusRunning,
	}
	if err := wsRepo.Create(context.Background(), environment); err != nil {
		t.Fatalf("no se pudo preparar el environment: %v", err)
	}
	return svc, orch, subRepo, freezer, environment
}

// Entregar un examen cierra solo el volumen de quien entrega: el aula sigue
// abierta para el resto (ADR-005 §5.2).
func TestSubmitPracticeEnvironmentProject_FreezesExamPracticeEnvironment(t *testing.T) {
	svc, _, subRepo, freezer, environment := newProjectSubmissionService(t, string(domain.ExercisePurposeExam))

	if _, err := svc.SubmitPracticeEnvironmentProject(context.Background(), environment.ID, "student-1", testExerciseID, nil); err != nil {
		t.Fatalf("la entrega del examen no debía fallar: %v", err)
	}
	if subRepo.created == nil {
		t.Fatal("la entrega debía quedar registrada")
	}
	if len(freezer.calls) != 1 {
		t.Fatalf("el volumen del alumno debía congelarse una vez: %+v", freezer.calls)
	}
	call := freezer.calls[0]
	if call.environmentID != environment.ID || call.actorID != environment.StudentID || call.tenantID != domain.DefaultTenantID {
		t.Fatalf("el cierre debía apuntar al alumno y su institución: %+v", call)
	}
}

// Una práctica de clase no congela nada: entregar no termina el laboratorio.
func TestSubmitPracticeEnvironmentProject_DoesNotFreezeClassPracticeEnvironment(t *testing.T) {
	svc, _, subRepo, freezer, environment := newProjectSubmissionService(t, string(domain.ExercisePurposeClass))

	if _, err := svc.SubmitPracticeEnvironmentProject(context.Background(), environment.ID, "student-1", testExerciseID, nil); err != nil {
		t.Fatalf("la entrega de clase no debía fallar: %v", err)
	}
	if subRepo.created == nil {
		t.Fatal("la entrega debía quedar registrada")
	}
	if len(freezer.calls) != 0 {
		t.Fatalf("un aula de clase no debía congelar volúmenes: %+v", freezer.calls)
	}
}

// Un volumen congelado no acepta entregas: escribir por esta vía sería seguir
// editando después del cierre.
// Un examen de otra materia no cierra el laboratorio de esta: el cierre apunta
// al entorno donde se trabajó, no a cualquier entrega con purpose exam.
func TestSubmitPracticeEnvironmentProject_DoesNotFreezeExamOfAnotherSubject(t *testing.T) {
	wsRepo := newMockPracticeEnvironmentRepo()
	orch := &fakeVolumeOrchestrator{
		mockOrchestrator: &mockOrchestrator{},
		snapshotFiles:    []domain.PracticeEnvironmentVolumeFile{{Path: "main.py", Content: []byte("print(1)\n")}},
	}
	subRepo := &fakeSubmissionRepo{}
	otherSubject := "subject-2"
	examRepo := &fakeExerciseRepo{exercise: &domain.Exercise{
		ID:        testExerciseID,
		Purpose:   string(domain.ExercisePurposeExam),
		SubjectID: &otherSubject,
	}}
	freezer := &fakePracticeEnvironmentExamFreezer{}
	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithExerciseSubmission(examRepo, subRepo).
		SetExamSubmissionFreezer(freezer)

	environment := &domain.PracticeEnvironment{
		ID:        "55555555-5555-4555-8555-555555555555",
		TenantID:  domain.DefaultTenantID,
		StudentID: "student-1",
		SubjectID: "subject-1",
		Status:    domain.PracticeEnvironmentStatusRunning,
	}
	if err := wsRepo.Create(context.Background(), environment); err != nil {
		t.Fatalf("no se pudo preparar el environment: %v", err)
	}

	if _, err := svc.SubmitPracticeEnvironmentProject(context.Background(), environment.ID, "student-1", testExerciseID, nil); err != nil {
		t.Fatalf("la entrega no debía fallar: %v", err)
	}
	if len(freezer.calls) != 0 {
		t.Fatalf("un examen de otra materia no debía congelar este volumen: %+v", freezer.calls)
	}
}

func TestSubmitPracticeEnvironmentProject_RejectsFrozenPracticeEnvironment(t *testing.T) {
	svc, orch, subRepo, freezer, environment := newProjectSubmissionService(t, string(domain.ExercisePurposeExam))
	frozenAt := time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)
	environment.FrozenAt = &frozenAt
	environment.Status = domain.PracticeEnvironmentStatusFrozen

	_, err := svc.SubmitPracticeEnvironmentProject(context.Background(), environment.ID, "student-1", testExerciseID, nil)
	if !errors.Is(err, domain.ErrPracticeEnvironmentFrozen) {
		t.Fatalf("esperado ErrPracticeEnvironmentFrozen, obtenido: %v", err)
	}
	if orch.snapshotCalled {
		t.Fatal("un volumen congelado no debía leerse para entregar")
	}
	if subRepo.created != nil {
		t.Fatal("un volumen congelado no debía registrar la entrega")
	}
	if len(freezer.calls) != 0 {
		t.Fatalf("una entrega rechazada no debía volver a congelar: %+v", freezer.calls)
	}
}

func TestSubmitPracticeEnvironmentProject_RejectsForeignPracticeEnvironment(t *testing.T) {
	wsRepo := newMockPracticeEnvironmentRepo()
	orch := &fakeVolumeOrchestrator{mockOrchestrator: &mockOrchestrator{}}
	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithExerciseSubmission(nil, &fakeSubmissionRepo{})

	environment := &domain.PracticeEnvironment{
		ID:        "33333333-3333-3333-3333-333333333333",
		StudentID: "student-owner",
		SubjectID: "subject-1",
	}
	if err := wsRepo.Create(context.Background(), environment); err != nil {
		t.Fatalf("no se pudo preparar el environment: %v", err)
	}

	_, err := svc.SubmitPracticeEnvironmentProject(context.Background(), environment.ID, "student-impostor", testExerciseID, nil)
	if !errors.Is(err, services.ErrPracticeEnvironmentForbidden) {
		t.Fatalf("esperado ErrPracticeEnvironmentForbidden, obtenido: %v", err)
	}
}

// Un entorno ausente no es un fallo interno: el servicio devuelve ErrNotFound
// (404 en el transporte) y no persiste ninguna entrega.
func TestSubmitPracticeEnvironmentProject_UnknownPracticeEnvironmentIsNotFound(t *testing.T) {
	wsRepo := newMockPracticeEnvironmentRepo()
	subRepo := &fakeSubmissionRepo{}
	svc := services.NewPracticeEnvironmentService(wsRepo, &fakeVolumeOrchestrator{mockOrchestrator: &mockOrchestrator{}}, &mockHostMonitor{}).
		WithExerciseSubmission(nil, subRepo)

	_, err := svc.SubmitPracticeEnvironmentProject(context.Background(), "99999999-9999-4999-8999-999999999999", "student-1", testExerciseID, nil)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("esperado domain.ErrNotFound, obtenido: %v", err)
	}
	if subRepo.created != nil {
		t.Error("un entorno inexistente no debe generar una entrega")
	}
}
