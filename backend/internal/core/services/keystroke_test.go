package services

import (
	"context"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

type memoryKeystrokeRepo struct {
	events map[string][]domain.SubmissionKeystrokeEvent
}

func newMemoryKeystrokeRepo() *memoryKeystrokeRepo {
	return &memoryKeystrokeRepo{
		events: make(map[string][]domain.SubmissionKeystrokeEvent),
	}
}

func (m *memoryKeystrokeRepo) SaveBatch(_ context.Context, submissionID string, events []domain.SubmissionKeystrokeEvent) error {
	m.events[submissionID] = append(m.events[submissionID], events...)
	return nil
}

func (m *memoryKeystrokeRepo) GetBySubmission(_ context.Context, submissionID string) ([]domain.SubmissionKeystrokeEvent, error) {
	return m.events[submissionID], nil
}

type stubSubmissionRepo struct {
	subs map[string]*domain.Submission
}

func newStubSubmissionRepo() *stubSubmissionRepo {
	return &stubSubmissionRepo{subs: make(map[string]*domain.Submission)}
}

func (s *stubSubmissionRepo) Create(_ context.Context, sub *domain.Submission) error {
	s.subs[sub.ID] = sub
	return nil
}

func (s *stubSubmissionRepo) GetByID(_ context.Context, _, id string) (*domain.Submission, error) {
	if sub, ok := s.subs[id]; ok {
		return sub, nil
	}
	return nil, domain.ErrNotFound
}

func (s *stubSubmissionRepo) ListByExerciseAndStudent(_ context.Context, _, _, _ string) ([]*domain.Submission, error) {
	return nil, nil
}

func (s *stubSubmissionRepo) ListByExercise(_ context.Context, _, _ string) ([]*domain.Submission, error) {
	return nil, nil
}

func (s *stubSubmissionRepo) ListByStudent(_ context.Context, _, _ string) ([]*domain.Submission, error) {
	return nil, nil
}

func (s *stubSubmissionRepo) UpdateOverride(_ context.Context, _, _, _, _ string, _ *int, _ *string) error {
	return nil
}

type stubTeacherRepoForKeystroke struct {
	review *domain.TeacherSubmissionReviewDTO
}

func (r *stubTeacherRepoForKeystroke) GetCoursesSummary(_ context.Context, _, _ string) ([]*domain.TeacherCourseSummary, error) {
	return nil, nil
}
func (r *stubTeacherRepoForKeystroke) GetAttentionWidget(_ context.Context, _, _ string) (*domain.TeacherAttentionWidget, error) {
	return nil, nil
}
func (r *stubTeacherRepoForKeystroke) GetCourseLabsStats(_ context.Context, _, _, _ string) ([]*domain.TeacherLabStats, error) {
	return nil, nil
}
func (r *stubTeacherRepoForKeystroke) ListCourseSubmissions(_ context.Context, _, _, _, _, _ string) ([]*domain.SubmissionQueueItem, error) {
	return nil, nil
}
func (r *stubTeacherRepoForKeystroke) GetTeacherSubmissionReview(_ context.Context, _, _, _ string) (*domain.TeacherSubmissionReviewDTO, error) {
	return r.review, nil
}
func (r *stubTeacherRepoForKeystroke) AddComment(_ context.Context, _ *domain.SubmissionComment) error {
	return nil
}
func (r *stubTeacherRepoForKeystroke) GetCommentsBySubmission(_ context.Context, _, _ string) ([]*domain.SubmissionComment, error) {
	return nil, nil
}
func (r *stubTeacherRepoForKeystroke) GetCourseGradesMatrix(_ context.Context, _, _, _ string) (*domain.CourseGradesMatrix, error) {
	return nil, nil
}
func (r *stubTeacherRepoForKeystroke) GetExerciseSubmissionsForPlagiarism(_ context.Context, _, _, _, _ string) ([]*domain.SubmissionForPlagiarism, error) {
	return nil, nil
}
func (r *stubTeacherRepoForKeystroke) ListLiveWorkspaceSessions(_ context.Context, _, _ string) ([]*domain.LiveWorkspaceSession, error) {
	return nil, nil
}
func (r *stubTeacherRepoForKeystroke) GetCourseAnalytics(_ context.Context, _, _, _ string) (*domain.CourseAnalytics, error) {
	return nil, nil
}

func TestSubmissionService_SaveKeystrokeEvents_LimitEnforcement(t *testing.T) {
	subRepo := newStubSubmissionRepo()
	keystrokeRepo := newMemoryKeystrokeRepo()
	subService := NewSubmissionService(subRepo)
	subService.SetKeystrokeRepository(keystrokeRepo)

	sub := &domain.Submission{
		ID:         "sub-123",
		TenantID:   "tenant-1",
		ExerciseID: "ex-1",
		StudentID:  "stu-1",
		Verdict:    "AC",
	}
	_ = subRepo.Create(context.Background(), sub)

	// Test exceeding max limit > 10,000
	tooManyEvents := make([]domain.SubmissionKeystrokeEvent, 10001)
	_, err := subService.SaveKeystrokeEvents(context.Background(), "tenant-1", "sub-123", tooManyEvents)
	if err == nil {
		t.Fatal("expected error for exceeding 10,000 events limit, got nil")
	}

	// Valid batch
	validEvents := []domain.SubmissionKeystrokeEvent{
		{TimestampMS: 100, EventType: domain.KeystrokeEventInsert, Position: 0, Content: "def "},
		{TimestampMS: 500, EventType: domain.KeystrokeEventInsert, Position: 4, Content: "solve():"},
		{TimestampMS: 1500, EventType: domain.KeystrokeEventPaste, Position: 12, Content: "    return 42", PasteSourceDetected: true},
	}
	count, err := subService.SaveKeystrokeEvents(context.Background(), "tenant-1", "sub-123", validEvents)
	if err != nil {
		t.Fatalf("unexpected error saving keystroke events: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestTeacherService_GetKeystrokeEvents_MetricsCalculation(t *testing.T) {
	keystrokeRepo := newMemoryKeystrokeRepo()
	review := &domain.TeacherSubmissionReviewDTO{
		ID:          "sub-456",
		StudentID:   "stu-1",
		StudentName: "Juan Perez",
		Code:        "def solve():\n    return 42",
		Verdict:     "AC",
		SubmittedAt: time.Now(),
	}
	tRepo := &stubTeacherRepoForKeystroke{review: review}
	tService := NewTeacherService(tRepo)
	tService.SetKeystrokeRepository(keystrokeRepo)

	// Save events: 10 typed chars, 20 pasted chars
	events := []domain.SubmissionKeystrokeEvent{
		{TimestampMS: 1000, EventType: domain.KeystrokeEventInsert, Position: 0, Content: "0123456789"},
		{TimestampMS: 3000, EventType: domain.KeystrokeEventPaste, Position: 10, Content: "01234567890123456789", PasteSourceDetected: true},
	}
	_ = keystrokeRepo.SaveBatch(context.Background(), "sub-456", events)

	report, err := tService.GetKeystrokeEvents(context.Background(), "tenant-1", "teacher-1", "sub-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalTimeMS != 3000 {
		t.Errorf("expected TotalTimeMS 3000, got %d", report.TotalTimeMS)
	}
	if report.PasteCount != 1 {
		t.Errorf("expected PasteCount 1, got %d", report.PasteCount)
	}
	if report.TotalCharsTyped != 10 {
		t.Errorf("expected TotalCharsTyped 10, got %d", report.TotalCharsTyped)
	}
	if report.TotalCharsPasted != 20 {
		t.Errorf("expected TotalCharsPasted 20, got %d", report.TotalCharsPasted)
	}
	expectedPct := 20.0 / 30.0
	if report.PastePercentage < expectedPct-0.01 || report.PastePercentage > expectedPct+0.01 {
		t.Errorf("expected PastePercentage ~%.2f, got %.2f", expectedPct, report.PastePercentage)
	}
}

func TestTeacherService_GetSubmissionTimeline_WithRealKeystrokeEvents(t *testing.T) {
	keystrokeRepo := newMemoryKeystrokeRepo()
	review := &domain.TeacherSubmissionReviewDTO{
		ID:          "sub-789",
		StudentID:   "stu-1",
		StudentName: "Ana Gomez",
		Code:        "x = 10",
		Verdict:     "AC",
		SubmittedAt: time.Now(),
	}
	tRepo := &stubTeacherRepoForKeystroke{review: review}
	tService := NewTeacherService(tRepo)
	tService.SetKeystrokeRepository(keystrokeRepo)

	events := []domain.SubmissionKeystrokeEvent{
		{TimestampMS: 100, EventType: domain.KeystrokeEventInsert, Position: 0, Content: "x = "},
		{TimestampMS: 2500, EventType: domain.KeystrokeEventPaste, Position: 4, Content: "10000000000000000000000000000000000000000000000000000", PasteSourceDetected: true},
	}
	_ = keystrokeRepo.SaveBatch(context.Background(), "sub-789", events)

	timeline, err := tService.GetSubmissionTimeline(context.Background(), "tenant-1", "teacher-1", "sub-789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(timeline.Keyframes) != 2 {
		t.Fatalf("expected 2 keyframes, got %d", len(timeline.Keyframes))
	}
	if !timeline.SuspiciousPasteFlag {
		t.Error("expected SuspiciousPasteFlag to be true")
	}
	if timeline.PasteEventsCount != 1 {
		t.Errorf("expected PasteEventsCount 1, got %d", timeline.PasteEventsCount)
	}
}
