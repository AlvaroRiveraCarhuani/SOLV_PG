package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// fakeExamSessionLinker cumple el enlace con el aula y la lectura de la última
// convocatoria del ejercicio: es la capacidad que decide si hay recuperatorio.
type fakeExamSessionLinker struct {
	openedAt time.Time
	found    bool
}

func (f *fakeExamSessionLinker) AttachPracticeEnvironmentToOpenSession(context.Context, string, string, string, string) error {
	return nil
}

func (f *fakeExamSessionLinker) LatestExamSessionOpen(context.Context, string, string, string) (time.Time, bool, error) {
	return f.openedAt, f.found, nil
}

func newRetakeService(t *testing.T, purpose domain.ExercisePurpose, closed *domain.PracticeEnvironment, linker *fakeExamSessionLinker) (*services.PracticeEnvironmentService, *mockPracticeEnvironmentRepo, *mockOrchestrator) {
	t.Helper()
	const (
		studentID = "student-1"
		subjectID = "subj-101"
	)
	wsRepo := newMockPracticeEnvironmentRepo()
	wsRepo.environments[studentID+":"+subjectID+":"+testPracticeID] = closed
	wsRepo.environments[closed.ID] = closed

	orch := &mockOrchestrator{}
	exerciseSubject := subjectID
	exerciseRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
		testPracticeID: {
			ID:              testPracticeID,
			SubjectID:       &exerciseSubject,
			Purpose:         string(purpose),
			EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		},
	}}

	svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
		WithProvisioning(
			&mockSubjectRepo{subjects: map[string]*domain.Subject{subjectID: {ID: subjectID}}},
			&mockTemplateRepo{templates: map[string]*domain.Template{}},
		).
		WithExerciseSubmission(exerciseRepo, &fakeSubmissionRepo{}).
		WithClassSessions(linker)
	return svc, wsRepo, orch
}

func closedExamEnvironment(frozenAt time.Time) *domain.PracticeEnvironment {
	exerciseID := testPracticeID
	return &domain.PracticeEnvironment{
		ID:                 "env-attempt-1",
		StudentID:          "student-1",
		SubjectID:          "subj-101",
		ExerciseID:         &exerciseID,
		Status:             domain.PracticeEnvironmentStatusFrozen,
		EnvironmentProfile: domain.EnvironmentProfileExam,
		ExamAttempt:        1,
		FrozenAt:           &frozenAt,
	}
}

// Una convocatoria abierta después del cierre es un recuperatorio: el alumno
// estrena intento y volumen, y no ve los archivos del examen anterior.
func TestStartPracticeEnvironment_RecuperatorioEstrenaIntentoYVolumen(t *testing.T) {
	frozenAt := time.Now().Add(-24 * time.Hour)
	linker := &fakeExamSessionLinker{openedAt: time.Now().Add(-time.Hour), found: true}
	svc, wsRepo, orch := newRetakeService(t, domain.ExercisePurposeExam, closedExamEnvironment(frozenAt), linker)

	ws, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-1", "subj-101", testPracticeID)
	if err != nil {
		t.Fatalf("el recuperatorio debía abrir un intento nuevo: %v", err)
	}
	if ws.ExamAttempt != 2 {
		t.Errorf("intento esperado 2, obtenido %d", ws.ExamAttempt)
	}
	if ws.FrozenAt != nil {
		t.Error("el intento nuevo no puede nacer congelado")
	}
	if ws.EnvironmentProfile != domain.EnvironmentProfileExam {
		t.Errorf("perfil esperado %q, obtenido %q", domain.EnvironmentProfileExam, ws.EnvironmentProfile)
	}
	wantVolume := "solv_workspace_student-1_subj-101_practice-1_exam_2"
	if orch.lastConfig == nil || orch.lastConfig.VolumeName != wantVolume {
		t.Errorf("volumen del contenedor: esperado %q, obtenido %+v", wantVolume, orch.lastConfig)
	}
	if ws.VolumeName() != wantVolume {
		t.Errorf("volumen del entorno: esperado %q, obtenido %q", wantVolume, ws.VolumeName())
	}
	if stored := wsRepo.environments["student-1:subj-101:"+testPracticeID]; stored == nil || stored.ExamAttempt != 2 {
		t.Errorf("el registro vigente debía quedar en el intento 2: %+v", stored)
	}
}

// Sin convocatoria nueva el examen sigue cerrado: recargar la página no reabre.
func TestStartPracticeEnvironment_RecargaNoAbreRecuperatorio(t *testing.T) {
	frozenAt := time.Now().Add(-24 * time.Hour)
	cases := []struct {
		nombre string
		linker *fakeExamSessionLinker
	}{
		{"sin convocatoria registrada", &fakeExamSessionLinker{found: false}},
		{"convocatoria anterior al cierre", &fakeExamSessionLinker{openedAt: frozenAt.Add(-time.Hour), found: true}},
		{"convocatoria que todavía no empezó", &fakeExamSessionLinker{openedAt: time.Now().Add(time.Hour), found: true}},
	}

	for _, tc := range cases {
		t.Run(tc.nombre, func(t *testing.T) {
			svc, _, _ := newRetakeService(t, domain.ExercisePurposeExam, closedExamEnvironment(frozenAt), tc.linker)

			ws, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-1", "subj-101", testPracticeID)
			if !errors.Is(err, domain.ErrPracticeEnvironmentFrozen) {
				t.Fatalf("se esperaba el entorno cerrado, se obtuvo ws=%+v err=%v", ws, err)
			}
		})
	}
}

// La práctica no tiene intentos: una convocatoria nueva no la reabre.
func TestStartPracticeEnvironment_PracticaCerradaNoEstrenaIntento(t *testing.T) {
	frozenAt := time.Now().Add(-24 * time.Hour)
	linker := &fakeExamSessionLinker{openedAt: time.Now().Add(-time.Hour), found: true}
	closed := closedExamEnvironment(frozenAt)
	closed.EnvironmentProfile = domain.EnvironmentProfilePractice
	closed.ExamAttempt = 0

	svc, _, _ := newRetakeService(t, domain.ExercisePurposeClass, closed, linker)

	if _, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "student-1", "subj-101", testPracticeID); !errors.Is(err, domain.ErrPracticeEnvironmentFrozen) {
		t.Fatalf("la práctica cerrada debía seguir cerrada, error: %v", err)
	}
}
