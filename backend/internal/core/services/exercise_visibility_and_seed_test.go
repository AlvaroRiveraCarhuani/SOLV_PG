package services_test

import (
	"context"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	"solv-backend/internal/delivery/http/dto"
)

type mockExerciseRepo struct {
	exercises map[string]*domain.Exercise
}

func (m *mockExerciseRepo) GetByID(_ context.Context, id string) (*domain.Exercise, error) {
	ex, ok := m.exercises[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return ex, nil
}

func (m *mockExerciseRepo) GetByIDAndTenant(_ context.Context, id, _ string) (*domain.Exercise, error) {
	return m.GetByID(context.Background(), id)
}

func (m *mockExerciseRepo) Create(_ context.Context, ex *domain.Exercise) error {
	m.exercises[ex.ID] = ex
	return nil
}

func (m *mockExerciseRepo) Update(_ context.Context, ex *domain.Exercise) error {
	m.exercises[ex.ID] = ex
	return nil
}

func (m *mockExerciseRepo) UpdateStatus(_ context.Context, id, _ string, status string) error {
	if ex, ok := m.exercises[id]; ok {
		ex.Status = status
		return nil
	}
	return domain.ErrNotFound
}

func (m *mockExerciseRepo) UpdateConfig(_ context.Context, id, _ string, cfg domain.ExerciseConfig) error {
	if ex, ok := m.exercises[id]; ok {
		ex.Config = cfg
		return nil
	}
	return domain.ErrNotFound
}

func (m *mockExerciseRepo) UpdateExpectedJSON(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockExerciseRepo) MarkExerciseStale(_ context.Context, _, _ string, _ bool) error {
	return nil
}

func (m *mockExerciseRepo) UpdateExerciseLastValidDryRun(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}

func (m *mockExerciseRepo) CreateDryRunJob(_ context.Context, _ *domain.DryRunJob) error {
	return nil
}

func (m *mockExerciseRepo) GetDryRunJob(_ context.Context, _ string) (*domain.DryRunJob, error) {
	return nil, nil
}

func (m *mockExerciseRepo) UpdateDryRunJobProgress(_ context.Context, _ string, _ domain.DryRunJobStatus, _, _ int, _ *domain.EvaluationResult, _ string) error {
	return nil
}

func (m *mockExerciseRepo) ListDueByStudent(_ context.Context, _, _ string) ([]*domain.DueAssignment, error) {
	return nil, nil
}

func (m *mockExerciseRepo) ListBySubject(_ context.Context, _, _ string) ([]*domain.Exercise, error) {
	return nil, nil
}

func (m *mockExerciseRepo) GetStudentRecommendations(_ context.Context, _, _, _ string) (*domain.StudentRecommendations, error) {
	return nil, nil
}

func TestExerciseValidation_PerStudentSeedRequiresExam(t *testing.T) {
	repo := &mockExerciseRepo{exercises: make(map[string]*domain.Exercise)}
	svc := services.NewEvaluationService(repo, nil, nil, nil)

	t.Run("per_student_seed with class purpose must fail", func(t *testing.T) {
		ex := &domain.Exercise{
			Title:          "Exam Seed Violation",
			Type:           domain.ExerciseTypeAlgorithm,
			Purpose:        string(domain.ExercisePurposeClass),
			PerStudentSeed: true,
		}
		err := svc.CreateExercise(context.Background(), ex)
		if err == nil {
			t.Fatalf("expected error when per_student_seed is true with purpose class, got nil")
		}
		if err != domain.ErrSeedRequiresExamPurpose {
			t.Errorf("expected ErrSeedRequiresExamPurpose, got %v", err)
		}
	})

	t.Run("per_student_seed with exam purpose must succeed", func(t *testing.T) {
		ex := &domain.Exercise{
			Title:          "Valid Exam Exercise",
			Type:           domain.ExerciseTypeAlgorithm,
			Purpose:        string(domain.ExercisePurposeExam),
			PerStudentSeed: true,
		}
		err := svc.CreateExercise(context.Background(), ex)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestTestCase_VisibilityNormalizationAndDualWrite(t *testing.T) {
	tests := []struct {
		name           string
		tc             domain.TestCase
		wantVisibility domain.TestCaseVisibility
		wantIsHidden   bool
		wantIsSample   bool
		wantWeight     float64
	}{
		{
			name: "legacy is_hidden = true",
			tc: domain.TestCase{
				Input:          "1 2",
				ExpectedOutput: "3",
				IsHidden:       true,
			},
			wantVisibility: domain.TestCaseVisibilityHidden,
			wantIsHidden:   true,
			wantIsSample:   false,
			wantWeight:     1.0,
		},
		{
			name: "legacy is_sample = true",
			tc: domain.TestCase{
				Input:          "1 2",
				ExpectedOutput: "3",
				IsSample:       true,
			},
			wantVisibility: domain.TestCaseVisibilityExample,
			wantIsHidden:   false,
			wantIsSample:   true,
			wantWeight:     1.0,
		},
		{
			name: "legacy public (both false)",
			tc: domain.TestCase{
				Input:          "1 2",
				ExpectedOutput: "3",
				IsHidden:       false,
				IsSample:       false,
			},
			wantVisibility: domain.TestCaseVisibilityPublic,
			wantIsHidden:   false,
			wantIsSample:   false,
			wantWeight:     1.0,
		},
		{
			name: "explicit visibility = example with custom weight",
			tc: domain.TestCase{
				Input:          "10",
				ExpectedOutput: "20",
				Visibility:     domain.TestCaseVisibilityExample,
				Weight:         2.5,
			},
			wantVisibility: domain.TestCaseVisibilityExample,
			wantIsHidden:   false,
			wantIsSample:   true,
			wantWeight:     2.5,
		},
		{
			name: "explicit visibility = hidden",
			tc: domain.TestCase{
				Input:          "secret",
				ExpectedOutput: "pass",
				Visibility:     domain.TestCaseVisibilityHidden,
			},
			wantVisibility: domain.TestCaseVisibilityHidden,
			wantIsHidden:   true,
			wantIsSample:   false,
			wantWeight:     1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := tt.tc
			tc.Normalize()
			if tc.Visibility != tt.wantVisibility {
				t.Errorf("Visibility = %q, want %q", tc.Visibility, tt.wantVisibility)
			}
			if tc.IsHidden != tt.wantIsHidden {
				t.Errorf("IsHidden = %v, want %v", tc.IsHidden, tt.wantIsHidden)
			}
			if tc.IsSample != tt.wantIsSample {
				t.Errorf("IsSample = %v, want %v", tc.IsSample, tt.wantIsSample)
			}
			if tc.Weight != tt.wantWeight {
				t.Errorf("Weight = %v, want %v", tc.Weight, tt.wantWeight)
			}
			if err := tc.Validate(); err != nil {
				t.Errorf("unexpected Validate error: %v", err)
			}
		})
	}
}

func TestToExercisePublicResponse_ExposesOnlyExampleCases(t *testing.T) {
	diff := "medium"
	ex := &domain.Exercise{
		ID:          "ex-123",
		Title:       "Algoritmo de Prueba",
		Description: "Descripción",
		Type:        domain.ExerciseTypeAlgorithm,
		Difficulty:  &diff,
		Tags:        []string{"dp", "math"},
		Purpose:     "class",
		Config: domain.ExerciseConfig{
			Algorithm: &domain.AlgorithmConfig{
				TimeLimitMS:   1000,
				MemoryLimitMB: 128,
				TestCases: []domain.TestCase{
					{
						Input:          "in-example",
						ExpectedOutput: "out-example",
						Visibility:     domain.TestCaseVisibilityExample,
					},
					{
						Input:          "in-public",
						ExpectedOutput: "out-public",
						Visibility:     domain.TestCaseVisibilityPublic,
					},
					{
						Input:          "in-hidden",
						ExpectedOutput: "out-hidden",
						Visibility:     domain.TestCaseVisibilityHidden,
					},
				},
			},
		},
	}

	for i := range ex.Config.Algorithm.TestCases {
		ex.Config.Algorithm.TestCases[i].Normalize()
	}

	pub := dto.ToExercisePublicResponse(ex)
	if pub == nil {
		t.Fatalf("expected non-nil public response")
	}

	if len(pub.PublicTestCases) != 1 {
		t.Fatalf("expected exactly 1 public test case (the example), got %d", len(pub.PublicTestCases))
	}

	if pub.PublicTestCases[0].Input != "in-example" || pub.PublicTestCases[0].ExpectedOutput != "out-example" {
		t.Errorf("unexpected exposed case: %+v", pub.PublicTestCases[0])
	}
	if pub.Difficulty == nil || *pub.Difficulty != "medium" {
		t.Errorf("expected difficulty 'medium', got %v", pub.Difficulty)
	}
	if len(pub.Tags) != 2 || pub.Tags[0] != "dp" || pub.Tags[1] != "math" {
		t.Errorf("unexpected tags: %v", pub.Tags)
	}
}
