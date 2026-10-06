package httpdelivery_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
	httpdelivery "solv-backend/internal/delivery/http"
)

type mockTeacherAnalyticsRepo struct {
	analytics *domain.CourseAnalytics
	err       error
}

func (m *mockTeacherAnalyticsRepo) GetCoursesSummary(ctx context.Context, tenantID, teacherID string) ([]*domain.TeacherCourseSummary, error) {
	return nil, nil
}
func (m *mockTeacherAnalyticsRepo) GetAttentionWidget(ctx context.Context, tenantID, teacherID string) (*domain.TeacherAttentionWidget, error) {
	return nil, nil
}
func (m *mockTeacherAnalyticsRepo) GetCourseLabsStats(ctx context.Context, tenantID, teacherID, subjectID string) ([]*domain.TeacherLabStats, error) {
	return nil, nil
}
func (m *mockTeacherAnalyticsRepo) ListCourseSubmissions(ctx context.Context, tenantID, teacherID, subjectID, exerciseID, verdict string) ([]*domain.SubmissionQueueItem, error) {
	return nil, nil
}
func (m *mockTeacherAnalyticsRepo) GetTeacherSubmissionReview(ctx context.Context, tenantID, teacherID, submissionID string) (*domain.TeacherSubmissionReviewDTO, error) {
	return nil, nil
}
func (m *mockTeacherAnalyticsRepo) AddComment(ctx context.Context, comment *domain.SubmissionComment) error {
	return nil
}
func (m *mockTeacherAnalyticsRepo) GetCommentsBySubmission(ctx context.Context, tenantID, submissionID string) ([]*domain.SubmissionComment, error) {
	return nil, nil
}
func (m *mockTeacherAnalyticsRepo) GetCourseGradesMatrix(ctx context.Context, tenantID, teacherID, subjectID string) (*domain.CourseGradesMatrix, error) {
	return nil, nil
}
func (m *mockTeacherAnalyticsRepo) GetExerciseSubmissionsForPlagiarism(ctx context.Context, tenantID, teacherID, subjectID, exerciseID string) ([]*domain.SubmissionForPlagiarism, error) {
	return nil, nil
}
func (m *mockTeacherAnalyticsRepo) ListLiveWorkspaceSessions(ctx context.Context, tenantID, teacherID string) ([]*domain.LiveWorkspaceSession, error) {
	return nil, nil
}
func (m *mockTeacherAnalyticsRepo) GetCourseAnalytics(ctx context.Context, tenantID, teacherID, subjectID string) (*domain.CourseAnalytics, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.analytics, nil
}

func TestGetCourseAnalyticsEndpoint(t *testing.T) {
	mockAnalytics := &domain.CourseAnalytics{
		DifficultyDistribution: map[string]domain.DifficultyMetric{
			"easy":   {Count: 5, SuccessRate: 0.85},
			"medium": {Count: 8, SuccessRate: 0.62},
			"hard":   {Count: 3, SuccessRate: 0.40},
		},
		TopTags: []domain.TagMetric{
			{Tag: "recursion", Count: 12, SuccessRate: 0.30},
			{Tag: "arrays", Count: 15, SuccessRate: 0.75},
		},
		MostFailedCases: []domain.FailedCaseMetric{
			{ExerciseTitle: "Torres de Hanoi", CaseIndex: 3, FailCount: 45},
		},
		AvgResolutionTimeByDifficulty: map[string]int{
			"easy":   420,
			"medium": 1200,
			"hard":   2400,
		},
		SubmissionsTimeline: []domain.TimelineMetric{
			{Date: "2026-01-01", Count: 12},
			{Date: "2026-01-02", Count: 8},
		},
	}

	repo := &mockTeacherAnalyticsRepo{analytics: mockAnalytics}
	teacherService := services.NewTeacherService(repo)
	handler := httpdelivery.NewTeacherHandler(teacherService)

	t.Run("Teacher can get aggregated course analytics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/teacher/courses/course-123/analytics", nil)
		req.SetPathValue("id", "course-123")
		req.Header.Set("X-User-Role", "teacher")
		req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")

		w := httptest.NewRecorder()
		handler.GetCourseAnalytics(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data domain.CourseAnalytics `json:"data"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Data.TopTags) != 2 {
			t.Errorf("expected 2 top tags, got %d", len(resp.Data.TopTags))
		}
		if resp.Data.DifficultyDistribution["easy"].Count != 5 {
			t.Errorf("expected easy count 5, got %d", resp.Data.DifficultyDistribution["easy"].Count)
		}
		if len(resp.Data.MostFailedCases) != 1 {
			t.Errorf("expected 1 most failed case, got %d", len(resp.Data.MostFailedCases))
		}
		if resp.Data.MostFailedCases[0].FailCount != 45 {
			t.Errorf("expected 45 fail count, got %d", resp.Data.MostFailedCases[0].FailCount)
		}
	})

	t.Run("Student is forbidden from viewing course analytics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/teacher/courses/course-123/analytics", nil)
		req.SetPathValue("id", "course-123")
		req.Header.Set("X-User-Role", "student")
		req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")

		w := httptest.NewRecorder()
		handler.GetCourseAnalytics(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", w.Code)
		}
	})
}
