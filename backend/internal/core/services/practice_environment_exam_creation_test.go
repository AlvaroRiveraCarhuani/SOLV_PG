package services_test

import (
	"context"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

// El arranque guarda el perfil y el intento que salen del ejercicio, y usa el
// volumen del intento cuando es examen. Es lo que impide que un examen herede
// los archivos que quedaron en el volumen de práctica.
func TestStartPracticeEnvironment_GuardaPerfilIntentoYVolumen(t *testing.T) {
	const (
		studentID = "student-1"
		subjectID = "subj-101"
	)

	cases := []struct {
		nombre         string
		purpose        domain.ExercisePurpose
		wantProfile    domain.EnvironmentProfile
		wantAttempt    int
		wantVolumeName string
	}{
		{
			nombre:         "examen estrena volumen del intento",
			purpose:        domain.ExercisePurposeExam,
			wantProfile:    domain.EnvironmentProfileExam,
			wantAttempt:    1,
			wantVolumeName: "solv_workspace_student-1_subj-101_practice-1_exam_1",
		},
		{
			nombre:         "práctica conserva su volumen",
			purpose:        domain.ExercisePurposeClass,
			wantProfile:    domain.EnvironmentProfilePractice,
			wantAttempt:    0,
			wantVolumeName: "solv_workspace_student-1_subj-101_practice-1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.nombre, func(t *testing.T) {
			wsRepo := newMockPracticeEnvironmentRepo()
			orch := &mockOrchestrator{}
			exerciseSubject := subjectID
			exerciseRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
				testPracticeID: {
					ID:              testPracticeID,
					SubjectID:       &exerciseSubject,
					Purpose:         string(tc.purpose),
					EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
				},
			}}

			svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
				WithProvisioning(
					&mockSubjectRepo{subjects: map[string]*domain.Subject{subjectID: {ID: subjectID}}},
					&mockTemplateRepo{templates: map[string]*domain.Template{}},
				).
				WithExerciseSubmission(exerciseRepo, &fakeSubmissionRepo{})

			ws, err := svc.StartPracticeEnvironmentForExercise(context.Background(), studentID, subjectID, testPracticeID)
			if err != nil {
				t.Fatalf("error inesperado al abrir el entorno: %v", err)
			}

			if ws.EnvironmentProfile != tc.wantProfile {
				t.Errorf("perfil guardado: esperado %q, obtenido %q", tc.wantProfile, ws.EnvironmentProfile)
			}
			if ws.ExamAttempt != tc.wantAttempt {
				t.Errorf("intento guardado: esperado %d, obtenido %d", tc.wantAttempt, ws.ExamAttempt)
			}
			if orch.lastConfig == nil {
				t.Fatal("no se envió configuración de contenedor al orquestador")
			}
			if orch.lastConfig.VolumeName != tc.wantVolumeName {
				t.Errorf("volumen del contenedor: esperado %q, obtenido %q", tc.wantVolumeName, orch.lastConfig.VolumeName)
			}
			if ws.VolumeName() != tc.wantVolumeName {
				t.Errorf("volumen del entorno: esperado %q, obtenido %q", tc.wantVolumeName, ws.VolumeName())
			}

			// El registro que queda en la base es el mismo: perfil, intento y
			// volumen no viven solo en la respuesta.
			stored, ok := wsRepo.environments[studentID+":"+subjectID+":"+testPracticeID]
			if !ok || stored == nil {
				t.Fatal("el entorno no quedó registrado en el repositorio")
			}
			if stored.EnvironmentProfile != tc.wantProfile || stored.ExamAttempt != tc.wantAttempt {
				t.Errorf("registro guardado: perfil %q intento %d", stored.EnvironmentProfile, stored.ExamAttempt)
			}
		})
	}
}
