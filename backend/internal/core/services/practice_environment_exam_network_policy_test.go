package services_test

import (
	"context"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

func TestPracticeEnvironment_ExamNetworkPolicyAssignment(t *testing.T) {
	tests := []struct {
		name         string
		purpose      domain.ExercisePurpose
		examNetwork  string
		wantNoEgress bool
	}{
		{
			name:         "examen con red none aplica ExamNoEgress = true",
			purpose:      domain.ExercisePurposeExam,
			examNetwork:  domain.ExamNetworkNone,
			wantNoEgress: true,
		},
		{
			name:         "examen con red vacia normaliza a none y aplica ExamNoEgress = true",
			purpose:      domain.ExercisePurposeExam,
			examNetwork:  "",
			wantNoEgress: true,
		},
		{
			name:         "examen con red egress autorizada aplica ExamNoEgress = false",
			purpose:      domain.ExercisePurposeExam,
			examNetwork:  domain.ExamNetworkEgress,
			wantNoEgress: false,
		},
		{
			name:         "practica de clase no aplica aislamiento de red (ExamNoEgress = false)",
			purpose:      domain.ExercisePurposeClass,
			examNetwork:  domain.ExamNetworkNone,
			wantNoEgress: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wsRepo := newMockPracticeEnvironmentRepo()
			orch := &mockOrchestrator{}
			subjectID := "subj001"
			templateID := "tpl-python"

			subjRepo := &mockSubjectRepo{subjects: map[string]*domain.Subject{
				subjectID: {ID: subjectID, Name: "Algoritmos"},
			}}
			tplRepo := &mockTemplateRepo{templates: map[string]*domain.Template{
				templateID: {
					ID:          templateID,
					DockerImage: "solv/ide:python",
					BaseRamMB:   512,
				},
			}}
			exRepo := &mockExerciseRepo{exercises: map[string]*domain.Exercise{
				"ex-net": {
					ID:          "ex-net",
					TemplateID:  templateID,
					SubjectID:   &subjectID,
					Purpose:     string(tc.purpose),
					ExamNetwork: tc.examNetwork,
				},
			}}

			svc := services.NewPracticeEnvironmentService(wsRepo, orch, &mockHostMonitor{}).
				WithProvisioning(subjRepo, tplRepo).
				WithExerciseSubmission(exRepo, &fakeSubmissionRepo{})

			_, err := svc.StartPracticeEnvironmentForExercise(context.Background(), "std0001", subjectID, "ex-net")
			if err != nil {
				t.Fatalf("error al iniciar el entorno: %v", err)
			}

			if orch.lastConfig == nil {
				t.Fatal("no se envió configuración de contenedor")
			}

			if orch.lastConfig.ExamNoEgress != tc.wantNoEgress {
				t.Fatalf("ExamNoEgress = %v; esperaba %v", orch.lastConfig.ExamNoEgress, tc.wantNoEgress)
			}
		})
	}
}
