package httpdelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type memoryKeystrokeRepoForHTTP struct {
	events map[string][]domain.SubmissionKeystrokeEvent
}

func newMemoryKeystrokeRepoForHTTP() *memoryKeystrokeRepoForHTTP {
	return &memoryKeystrokeRepoForHTTP{
		events: make(map[string][]domain.SubmissionKeystrokeEvent),
	}
}

func (m *memoryKeystrokeRepoForHTTP) SaveBatch(_ context.Context, submissionID string, events []domain.SubmissionKeystrokeEvent) error {
	m.events[submissionID] = append(m.events[submissionID], events...)
	return nil
}

func (m *memoryKeystrokeRepoForHTTP) GetBySubmission(_ context.Context, submissionID string) ([]domain.SubmissionKeystrokeEvent, error) {
	return m.events[submissionID], nil
}

type stubSubmissionRepoForHTTP struct {
	subs map[string]*domain.Submission
}

func newStubSubmissionRepoForHTTP() *stubSubmissionRepoForHTTP {
	return &stubSubmissionRepoForHTTP{subs: make(map[string]*domain.Submission)}
}

func (s *stubSubmissionRepoForHTTP) Create(_ context.Context, sub *domain.Submission) error {
	s.subs[sub.ID] = sub
	return nil
}

func (s *stubSubmissionRepoForHTTP) GetByID(_ context.Context, _, id string) (*domain.Submission, error) {
	if sub, ok := s.subs[id]; ok {
		return sub, nil
	}
	return nil, domain.ErrNotFound
}

func (s *stubSubmissionRepoForHTTP) ListByExerciseAndStudent(_ context.Context, _, _, _ string) ([]*domain.Submission, error) {
	return nil, nil
}

func (s *stubSubmissionRepoForHTTP) ListByExercise(_ context.Context, _, _ string) ([]*domain.Submission, error) {
	return nil, nil
}

func (s *stubSubmissionRepoForHTTP) ListByStudent(_ context.Context, _, _ string) ([]*domain.Submission, error) {
	return nil, nil
}

func (s *stubSubmissionRepoForHTTP) UpdateOverride(_ context.Context, _, _, _, _ string, _ *int, _ *string) error {
	return nil
}

type stubTeacherRepoForHTTP struct {
	review *domain.TeacherSubmissionReviewDTO
}

func (r *stubTeacherRepoForHTTP) GetCoursesSummary(_ context.Context, _, _ string) ([]*domain.TeacherCourseSummary, error) {
	return nil, nil
}
func (r *stubTeacherRepoForHTTP) GetAttentionWidget(_ context.Context, _, _ string) (*domain.TeacherAttentionWidget, error) {
	return nil, nil
}
func (r *stubTeacherRepoForHTTP) GetCourseLabsStats(_ context.Context, _, _, _ string) ([]*domain.TeacherLabStats, error) {
	return nil, nil
}
func (r *stubTeacherRepoForHTTP) ListCourseSubmissions(_ context.Context, _, _, _, _, _ string) ([]*domain.SubmissionQueueItem, error) {
	return nil, nil
}
func (r *stubTeacherRepoForHTTP) GetTeacherSubmissionReview(_ context.Context, _, _, _ string) (*domain.TeacherSubmissionReviewDTO, error) {
	if r.review == nil {
		return nil, domain.ErrNotFound
	}
	return r.review, nil
}
func (r *stubTeacherRepoForHTTP) AddComment(_ context.Context, _ *domain.SubmissionComment) error {
	return nil
}
func (r *stubTeacherRepoForHTTP) GetCommentsBySubmission(_ context.Context, _, _ string) ([]*domain.SubmissionComment, error) {
	return nil, nil
}
func (r *stubTeacherRepoForHTTP) GetCourseGradesMatrix(_ context.Context, _, _, _ string) (*domain.CourseGradesMatrix, error) {
	return nil, nil
}
func (r *stubTeacherRepoForHTTP) GetExerciseSubmissionsForPlagiarism(_ context.Context, _, _, _, _ string) ([]*domain.SubmissionForPlagiarism, error) {
	return nil, nil
}
func (r *stubTeacherRepoForHTTP) ListLiveWorkspaceSessions(_ context.Context, _, _ string) ([]*domain.LiveWorkspaceSession, error) {
	return nil, nil
}
func (r *stubTeacherRepoForHTTP) GetCourseAnalytics(_ context.Context, _, _, _ string) (*domain.CourseAnalytics, error) {
	return nil, nil
}

func TestKeystrokeHandler_SaveAndGetEvents(t *testing.T) {
	subRepo := newStubSubmissionRepoForHTTP()
	keystrokeRepo := newMemoryKeystrokeRepoForHTTP()

	subService := services.NewSubmissionService(subRepo)
	subService.SetKeystrokeRepository(keystrokeRepo)
	subHandler := NewSubmissionHandler(subService)

	sub := &domain.Submission{
		ID:         "sub-http-1",
		TenantID:   domain.DefaultTenantID,
		ExerciseID: "ex-1",
		StudentID:  "stu-1",
		Verdict:    "AC",
	}
	_ = subRepo.Create(context.Background(), sub)

	teacherRepo := &stubTeacherRepoForHTTP{
		review: &domain.TeacherSubmissionReviewDTO{
			ID:          "sub-http-1",
			StudentID:   "stu-1",
			StudentName: "Estudiante 1",
			Verdict:     "AC",
			SubmittedAt: time.Now(),
		},
	}
	teacherService := services.NewTeacherService(teacherRepo, subRepo)
	teacherService.SetKeystrokeRepository(keystrokeRepo)
	teacherHandler := NewTeacherHandler(teacherService)

	mux := http.NewServeMux()
	SetupRoutes(mux, &Handlers{
		SubmissionHandler: subHandler,
		TeacherHandler:    teacherHandler,
		TenantMiddleware:  func(next http.Handler) http.Handler { return next },
	})

	// 1. Post keystroke events as student
	payload := map[string]interface{}{
		"events": []map[string]interface{}{
			{
				"timestamp_ms": 100,
				"event_type":   "insert",
				"position":     0,
				"content":      "def hello():",
			},
			{
				"timestamp_ms": 2500,
				"event_type":   "paste",
				"position":     12,
				"content":      "    print('world')\n    return True",
			},
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/sub-http-1/keystroke-events", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "student")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Query as student -> must be Forbidden (403)
	getReqStudent := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/submissions/sub-http-1/keystroke-events", nil)
	getReqStudent.Header.Set("X-User-Role", "student")
	wStudent := httptest.NewRecorder()

	mux.ServeHTTP(wStudent, getReqStudent)
	if wStudent.Code != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden for student role, got %d", wStudent.Code)
	}

	// 3. Query as teacher -> must be 200 OK with report
	getReqTeacher := httptest.NewRequest(http.MethodGet, "/api/v1/teacher/submissions/sub-http-1/keystroke-events", nil)
	getReqTeacher.Header.Set("X-User-Role", "teacher")
	getReqTeacher.Header.Set("X-User-Id", "teacher-1")
	wTeacher := httptest.NewRecorder()

	mux.ServeHTTP(wTeacher, getReqTeacher)
	if wTeacher.Code != http.StatusOK {
		t.Fatalf("expected status 200 for teacher role, got %d: %s", wTeacher.Code, wTeacher.Body.String())
	}

	var resp struct {
		Data domain.SubmissionKeystrokeReport `json:"data"`
	}
	if err := json.NewDecoder(wTeacher.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.SubmissionID != "sub-http-1" {
		t.Errorf("expected submission_id 'sub-http-1', got %q", resp.Data.SubmissionID)
	}
	if len(resp.Data.Events) != 2 {
		t.Errorf("expected 2 events in report, got %d", len(resp.Data.Events))
	}
	if resp.Data.PasteCount != 1 {
		t.Errorf("expected PasteCount 1, got %d", resp.Data.PasteCount)
	}
}

func TestSubmissionHandler_KeystrokesLimit_Returns413(t *testing.T) {
	mockSubmissionRepo := newStubSubmissionRepoForHTTP()
	subService := services.NewSubmissionService(mockSubmissionRepo)
	subHandler := NewSubmissionHandler(subService)

	mux := http.NewServeMux()
	SetupRoutes(mux, &Handlers{
		SubmissionHandler: subHandler,
		TenantMiddleware:  func(next http.Handler) http.Handler { return next },
	})

	// Prepare payload with 10,001 events
	events := make([]map[string]interface{}, 10001)
	for i := 0; i < 10001; i++ {
		events[i] = map[string]interface{}{
			"timestamp_ms": int64(i * 10),
			"event_type":   "insert",
			"position":     i,
			"content":      "a",
		}
	}

	payload := map[string]interface{}{
		"events": events,
	}
	payloadBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/sub-overflow/keystroke-events", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413 Payload Too Large, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Limit   int    `json:"limit"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != "too_many_keystroke_events" {
		t.Errorf("expected error 'too_many_keystroke_events', got %q", resp.Error)
	}
	if resp.Limit != 10000 {
		t.Errorf("expected limit 10000, got %d", resp.Limit)
	}
}

