package httpdelivery_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	httpdelivery "solv-backend/internal/delivery/http"
)

type mockExerciseRepoForRecs struct {
	recs *domain.StudentRecommendations
	err  error
}

func (m *mockExerciseRepoForRecs) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	return nil, nil
}
func (m *mockExerciseRepoForRecs) GetByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	return nil, nil
}
func (m *mockExerciseRepoForRecs) Create(ctx context.Context, exercise *domain.Exercise) error {
	return nil
}
func (m *mockExerciseRepoForRecs) Update(ctx context.Context, exercise *domain.Exercise) error {
	return nil
}
func (m *mockExerciseRepoForRecs) UpdateStatus(ctx context.Context, id, tenantID, status string) error {
	return nil
}
func (m *mockExerciseRepoForRecs) UpdateConfig(ctx context.Context, id, tenantID string, config domain.ExerciseConfig) error {
	return nil
}
func (m *mockExerciseRepoForRecs) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error {
	return nil
}
func (m *mockExerciseRepoForRecs) MarkExerciseStale(ctx context.Context, exerciseID, tenantID string, stale bool) error {
	return nil
}
func (m *mockExerciseRepoForRecs) UpdateExerciseLastValidDryRun(ctx context.Context, exerciseID, tenantID string, dryRunAt time.Time) error {
	return nil
}
func (m *mockExerciseRepoForRecs) CreateDryRunJob(ctx context.Context, job *domain.DryRunJob) error {
	return nil
}
func (m *mockExerciseRepoForRecs) GetDryRunJob(ctx context.Context, jobID string) (*domain.DryRunJob, error) {
	return nil, nil
}
func (m *mockExerciseRepoForRecs) UpdateDryRunJobProgress(ctx context.Context, jobID string, status domain.DryRunJobStatus, current, total int, result *domain.EvaluationResult, errMsg string) error {
	return nil
}
func (m *mockExerciseRepoForRecs) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) {
	return []*domain.DueAssignment{}, nil
}
func (m *mockExerciseRepoForRecs) ListBySubject(ctx context.Context, tenantID, subjectID string) ([]*domain.Exercise, error) {
	return []*domain.Exercise{}, nil
}
func (m *mockExerciseRepoForRecs) GetStudentRecommendations(ctx context.Context, tenantID, subjectID, studentID string) (*domain.StudentRecommendations, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.recs, nil
}

func TestGetStudentRecommendationsEndpoint(t *testing.T) {
	mockRecs := &domain.StudentRecommendations{
		HasEnoughData: true,
		WeakTags: []domain.WeakTag{
			{Tag: "recursion", SuccessRate: 0.33, Attempts: 9},
		},
		Recommendations: []domain.RecommendationItem{
			{
				ExerciseID: "ex-101",
				Title:      "Torres de Hanoi básico",
				Difficulty: "easy",
				MatchedTag: "recursion",
				Reason:     "Practica recursión con un ejercicio más sencillo",
			},
		},
	}

	mockRepo := &mockExerciseRepoForRecs{recs: mockRecs}
	handler := httpdelivery.NewStudentHandler(nil, nil, nil, mockRepo)

	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/student/courses/{id}/recommendations", handler.GetRecommendations)

	t.Run("Student receives recommendations when weak tags exist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/student/courses/subj-123/recommendations", nil)
		req.Header.Set("X-Tenant-Id", "tenant-test")
		req.Header.Set("X-User-Id", "student-user-1")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var resp struct {
			Success bool                          `json:"success"`
			Data    domain.StudentRecommendations `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Data.HasEnoughData {
			t.Errorf("expected has_enough_data to be true")
		}
		if len(resp.Data.WeakTags) != 1 || resp.Data.WeakTags[0].Tag != "recursion" {
			t.Errorf("expected weak tag 'recursion', got %+v", resp.Data.WeakTags)
		}
		if len(resp.Data.Recommendations) != 1 || resp.Data.Recommendations[0].ExerciseID != "ex-101" {
			t.Errorf("expected recommendation for ex-101, got %+v", resp.Data.Recommendations)
		}
	})

	t.Run("Missing auth headers returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/student/courses/subj-123/recommendations", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rr.Code)
		}
	})
}
