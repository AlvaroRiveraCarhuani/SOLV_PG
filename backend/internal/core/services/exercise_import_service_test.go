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

type mockImportExerciseRepo struct {
	exercises map[string]*domain.Exercise
}

func newMockImportRepo() *mockImportExerciseRepo {
	return &mockImportExerciseRepo{
		exercises: make(map[string]*domain.Exercise),
	}
}

func (m *mockImportExerciseRepo) Create(ctx context.Context, exercise *domain.Exercise) error {
	m.exercises[exercise.ID] = exercise
	return nil
}

func (m *mockImportExerciseRepo) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	if ex, ok := m.exercises[id]; ok {
		return ex, nil
	}
	return nil, errors.New("exercise not found")
}

func (m *mockImportExerciseRepo) GetByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	return m.GetByID(ctx, id)
}

func (m *mockImportExerciseRepo) Update(ctx context.Context, exercise *domain.Exercise) error {
	m.exercises[exercise.ID] = exercise
	return nil
}

func (m *mockImportExerciseRepo) UpdateStatus(ctx context.Context, id, tenantID, status string) error {
	if ex, ok := m.exercises[id]; ok {
		ex.Status = status
	}
	return nil
}

func (m *mockImportExerciseRepo) UpdateConfig(ctx context.Context, id, tenantID string, config domain.ExerciseConfig) error {
	if ex, ok := m.exercises[id]; ok {
		ex.Config = config
	}
	return nil
}

func (m *mockImportExerciseRepo) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error {
	return nil
}

func (m *mockImportExerciseRepo) MarkExerciseStale(ctx context.Context, exerciseID, tenantID string, stale bool) error {
	if ex, ok := m.exercises[exerciseID]; ok {
		ex.Stale = stale
	}
	return nil
}

func (m *mockImportExerciseRepo) UpdateExerciseLastValidDryRun(ctx context.Context, exerciseID, tenantID string, dryRunAt time.Time) error {
	if ex, ok := m.exercises[exerciseID]; ok {
		ex.LastValidDryRunAt = &dryRunAt
	}
	return nil
}

func (m *mockImportExerciseRepo) CreateDryRunJob(ctx context.Context, job *domain.DryRunJob) error {
	return nil
}

func (m *mockImportExerciseRepo) GetDryRunJob(ctx context.Context, jobID string) (*domain.DryRunJob, error) {
	return nil, nil
}

func (m *mockImportExerciseRepo) UpdateDryRunJobProgress(ctx context.Context, jobID string, status domain.DryRunJobStatus, current, total int, result *domain.EvaluationResult, errMsg string) error {
	return nil
}

func (m *mockImportExerciseRepo) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) {
	return nil, nil
}

func (m *mockImportExerciseRepo) ListBySubject(ctx context.Context, tenantID, subjectID string) ([]*domain.Exercise, error) {
	var res []*domain.Exercise
	for _, ex := range m.exercises {
		if ex.SubjectID != nil && *ex.SubjectID == subjectID {
			res = append(res, ex)
		}
	}
	return res, nil
}

func (m *mockImportExerciseRepo) GetStudentRecommendations(ctx context.Context, tenantID, subjectID, studentID string) (*domain.StudentRecommendations, error) {
	return nil, nil
}

func TestImportExercises_JSONAndYAML(t *testing.T) {
	ctx := context.Background()

	t.Run("valid JSON array import in dry-run mode", func(t *testing.T) {
		repo := newMockImportRepo()
		svc := services.NewEvaluationService(repo, nil, nil, nil)

		jsonPayload := `[
			{
				"title": "Suma de dos enteros",
				"statement": "Calcular la suma de dos números a y b.",
				"language": "python",
				"difficulty": "easy",
				"tags": ["math"],
				"config": {
					"algorithm": {
						"test_cases": [
							{"input": "2 3\n", "expected_output": "5\n", "is_hidden": false}
						]
					}
				}
			}
		]`

		res, err := svc.ImportExercises(ctx, "course-123", []byte(jsonPayload), "exercises.json", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Mode != "dry_run" {
			t.Errorf("expected mode 'dry_run', got %s", res.Mode)
		}
		if !res.CanImport {
			t.Errorf("expected CanImport=true, got false")
		}
		if len(res.Exercises) != 1 {
			t.Fatalf("expected 1 exercise item, got %d", len(res.Exercises))
		}
		if !res.Exercises[0].Valid {
			t.Errorf("expected exercise to be valid, got errors: %v", res.Exercises[0].Errors)
		}
		if len(repo.exercises) != 0 {
			t.Errorf("expected 0 exercises stored in dry-run, got %d", len(repo.exercises))
		}
	})

	t.Run("valid YAML hierarchical import in actual mode", func(t *testing.T) {
		repo := newMockImportRepo()
		svc := services.NewEvaluationService(repo, nil, nil, nil)

		yamlPayload := `
metadata:
  title: "Factorial Iterativo"
  difficulty: "medium"
  tags:
    - "loops"
    - "math"
statement: "Dado N, imprimir N!"
contract:
  language: "python"
cases:
  - input: "5\n"
    expected_output: "120\n"
    is_hidden: false
`

		res, err := svc.ImportExercises(ctx, "course-123", []byte(yamlPayload), "exercise.yaml", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Mode != "import" {
			t.Errorf("expected mode 'import', got %s", res.Mode)
		}
		if !res.CanImport {
			t.Errorf("expected CanImport=true, got false")
		}
		if res.ImportedCount != 1 {
			t.Errorf("expected ImportedCount=1, got %d", res.ImportedCount)
		}
		if len(repo.exercises) != 1 {
			t.Fatalf("expected 1 exercise in DB, got %d", len(repo.exercises))
		}

		for _, ex := range repo.exercises {
			if ex.Title != "Factorial Iterativo" {
				t.Errorf("expected title 'Factorial Iterativo', got %s", ex.Title)
			}
			if ex.SubjectID == nil || *ex.SubjectID != "course-123" {
				t.Errorf("expected SubjectID 'course-123', got %v", ex.SubjectID)
			}
		}
	})

	t.Run("title collision appends importado suffix", func(t *testing.T) {
		repo := newMockImportRepo()
		courseID := "course-123"
		repo.exercises["ex-existing"] = &domain.Exercise{
			ID:        "ex-existing",
			SubjectID: &courseID,
			Title:     "Suma de dos enteros",
		}
		svc := services.NewEvaluationService(repo, nil, nil, nil)

		jsonPayload := `[
			{
				"title": "Suma de dos enteros",
				"statement": "Calcular suma",
				"language": "python",
				"config": {
					"algorithm": {
						"test_cases": [{"input": "1 1\n", "expected_output": "2\n"}]
					}
				}
			}
		]`

		res, err := svc.ImportExercises(ctx, "course-123", []byte(jsonPayload), "test.json", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.CanImport {
			t.Fatalf("expected CanImport=true, got false")
		}

		var imported *domain.Exercise
		for _, ex := range repo.exercises {
			if ex.ID != "ex-existing" {
				imported = ex
				break
			}
		}

		if imported == nil {
			t.Fatalf("expected imported exercise in DB")
		}
		if imported.Title != "Suma de dos enteros (importado)" {
			t.Errorf("expected title 'Suma de dos enteros (importado)', got '%s'", imported.Title)
		}
	})

	t.Run("validation failure rejects entire import fail-closed", func(t *testing.T) {
		repo := newMockImportRepo()
		svc := services.NewEvaluationService(repo, nil, nil, nil)

		jsonPayload := `[
			{
				"title": "Ejercicio Válido",
				"statement": "OK",
				"language": "python",
				"config": {
					"algorithm": {
						"test_cases": [{"input": "1", "expected_output": "1"}]
					}
				}
			},
			{
				"title": "",
				"statement": "",
				"language": "unsupported_lang",
				"config": {
					"algorithm": {
						"test_cases": []
					}
				}
			}
		]`

		res, err := svc.ImportExercises(ctx, "course-123", []byte(jsonPayload), "test.json", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.CanImport {
			t.Errorf("expected CanImport=false when any exercise is invalid, got true")
		}
		if res.ImportedCount != 0 {
			t.Errorf("expected 0 imported exercises on validation failure, got %d", res.ImportedCount)
		}
		if len(repo.exercises) != 0 {
			t.Errorf("expected DB to remain empty, got %d stored exercises", len(repo.exercises))
		}
	})

	t.Run("exceeding 50 exercises returns limit error", func(t *testing.T) {
		repo := newMockImportRepo()
		svc := services.NewEvaluationService(repo, nil, nil, nil)

		var items []string
		for i := 0; i < 51; i++ {
			items = append(items, `{"title": "Ex", "statement": "Stmt", "language": "python"}`)
		}
		payload := "[" + strings.Join(items, ",") + "]"

		_, err := svc.ImportExercises(ctx, "course-123", []byte(payload), "test.json", false)
		if err == nil || !strings.Contains(err.Error(), "50 ejercicios") {
			t.Errorf("expected too many exercises error, got: %v", err)
		}
	})
}
