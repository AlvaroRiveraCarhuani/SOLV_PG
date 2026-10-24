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

type mockSubjectRepoForStudent struct {
	subjects []*domain.Subject
}

func (m *mockSubjectRepoForStudent) Create(ctx context.Context, subject *domain.Subject) error {
	return nil
}
func (m *mockSubjectRepoForStudent) GetByID(ctx context.Context, tenantID, id string) (*domain.Subject, error) {
	return nil, nil
}
func (m *mockSubjectRepoForStudent) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Subject, error) {
	return m.subjects, nil
}
func (m *mockSubjectRepoForStudent) EnrollStudent(ctx context.Context, enrollment *domain.Enrollment) error {
	return nil
}
func (m *mockSubjectRepoForStudent) ListStudentsBySubject(ctx context.Context, tenantID, subjectID string) ([]string, error) {
	return []string{}, nil
}
func (m *mockSubjectRepoForStudent) ListByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.Subject, error) {
	return m.subjects, nil
}
func (m *mockSubjectRepoForStudent) ReassignTeacher(ctx context.Context, tenantID, subjectID, newTeacherID string) error {
	return nil
}
func (m *mockSubjectRepoForStudent) ArchiveSubject(ctx context.Context, tenantID, subjectID string, isArchived bool) error {
	return nil
}
func (m *mockSubjectRepoForStudent) Update(ctx context.Context, tenantID, subjectID, name, code string) error {
	return nil
}
func (m *mockSubjectRepoForStudent) GetTemplateStatus(ctx context.Context, templateID string) (string, error) {
	return "", nil
}

type mockSubmissionRepoForStudent struct {
	submissions []*domain.Submission
}

func (m *mockSubmissionRepoForStudent) Create(ctx context.Context, submission *domain.Submission) error {
	return nil
}
func (m *mockSubmissionRepoForStudent) GetByID(ctx context.Context, tenantID, id string) (*domain.Submission, error) {
	return nil, nil
}
func (m *mockSubmissionRepoForStudent) ListByExerciseAndStudent(ctx context.Context, tenantID, exerciseID, studentID string) ([]*domain.Submission, error) {
	return []*domain.Submission{}, nil
}
func (m *mockSubmissionRepoForStudent) ListByExercise(ctx context.Context, tenantID, exerciseID string) ([]*domain.Submission, error) {
	return []*domain.Submission{}, nil
}
func (m *mockSubmissionRepoForStudent) ListByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.Submission, error) {
	return m.submissions, nil
}
func (m *mockSubmissionRepoForStudent) UpdateOverride(ctx context.Context, tenantID, id, verdict, reason string, score *int, gradedBy *string) error {
	return nil
}

type mockExerciseRepoForStudent struct {
	bySubjectID map[string][]*domain.Exercise
	byID        map[string]*domain.Exercise
}

func (m *mockExerciseRepoForStudent) GetByID(ctx context.Context, id string) (*domain.Exercise, error) {
	return m.byID[id], nil
}
func (m *mockExerciseRepoForStudent) GetByIDAndTenant(ctx context.Context, id, tenantID string) (*domain.Exercise, error) {
	return m.byID[id], nil
}
func (m *mockExerciseRepoForStudent) Create(ctx context.Context, exercise *domain.Exercise) error {
	return nil
}
func (m *mockExerciseRepoForStudent) Update(ctx context.Context, exercise *domain.Exercise) error {
	return nil
}
func (m *mockExerciseRepoForStudent) UpdateStatus(ctx context.Context, id, tenantID, status string) error {
	return nil
}
func (m *mockExerciseRepoForStudent) UpdateConfig(ctx context.Context, id, tenantID string, config domain.ExerciseConfig) error {
	return nil
}
func (m *mockExerciseRepoForStudent) UpdateExpectedJSON(ctx context.Context, id string, expectedJSON string) error {
	return nil
}
func (m *mockExerciseRepoForStudent) MarkExerciseStale(ctx context.Context, exerciseID, tenantID string, stale bool) error {
	return nil
}
func (m *mockExerciseRepoForStudent) UpdateExerciseLastValidDryRun(ctx context.Context, exerciseID, tenantID string, dryRunAt time.Time) error {
	return nil
}
func (m *mockExerciseRepoForStudent) CreateDryRunJob(ctx context.Context, job *domain.DryRunJob) error {
	return nil
}
func (m *mockExerciseRepoForStudent) GetDryRunJob(ctx context.Context, jobID string) (*domain.DryRunJob, error) {
	return nil, nil
}
func (m *mockExerciseRepoForStudent) UpdateDryRunJobProgress(ctx context.Context, jobID string, status domain.DryRunJobStatus, current, total int, result *domain.EvaluationResult, errMsg string) error {
	return nil
}
func (m *mockExerciseRepoForStudent) ListDueByStudent(ctx context.Context, tenantID, studentID string) ([]*domain.DueAssignment, error) {
	return []*domain.DueAssignment{}, nil
}
func (m *mockExerciseRepoForStudent) ListBySubject(ctx context.Context, tenantID, subjectID string) ([]*domain.Exercise, error) {
	return m.bySubjectID[subjectID], nil
}
func (m *mockExerciseRepoForStudent) GetStudentRecommendations(ctx context.Context, tenantID, subjectID, studentID string) (*domain.StudentRecommendations, error) {
	return &domain.StudentRecommendations{}, nil
}

func decodeStudentData(t *testing.T, body []byte, target any) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("failed to decode response envelope: %v", err)
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		t.Fatalf("failed to decode response data: %v", err)
	}
}

func TestGetStudentEvaluationsClassifiesAcademicStatus(t *testing.T) {
	pastDue := time.Now().Add(-48 * time.Hour)
	subjectID := "subj-1"
	subject := &domain.Subject{ID: subjectID, Name: "Algoritmos", Code: "ALG-101"}

	exercises := []*domain.Exercise{
		{
			ID:        "ex-completed",
			SubjectID: &subjectID,
			Title:     "Suma de enteros",
			Type:      domain.ExerciseType("code"),
			Status:    "published",
			DueDate:   &pastDue,
		},
		{
			ID:        "ex-overdue",
			SubjectID: &subjectID,
			Title:     "Recorrido de listas",
			Type:      domain.ExerciseType("code"),
			Status:    "published",
			DueDate:   &pastDue,
		},
		{
			ID:        "ex-in-progress",
			SubjectID: &subjectID,
			Title:     "Ordenamiento",
			Type:      domain.ExerciseType("code"),
			Status:    "published",
		},
		{
			ID:        "ex-pending",
			SubjectID: &subjectID,
			Title:     "Búsqueda binaria",
			Type:      domain.ExerciseType("code"),
			Status:    "",
		},
		{
			ID:        "ex-draft",
			SubjectID: &subjectID,
			Title:     "Borrador del docente",
			Type:      domain.ExerciseType("code"),
			Status:    "draft",
		},
	}

	now := time.Now()
	submissions := []*domain.Submission{
		{ID: "sub-1", ExerciseID: "ex-completed", Verdict: "AC", SubmittedAt: now.Add(-time.Hour)},
		{ID: "sub-2", ExerciseID: "ex-in-progress", Verdict: "WA", SubmittedAt: now.Add(-30 * time.Minute)},
	}

	handler := httpdelivery.NewStudentHandler(
		&mockSubjectRepoForStudent{subjects: []*domain.Subject{subject}},
		nil,
		&mockSubmissionRepoForStudent{submissions: submissions},
		&mockExerciseRepoForStudent{
			bySubjectID: map[string][]*domain.Exercise{subjectID: exercises},
			byID: map[string]*domain.Exercise{
				"ex-completed":   exercises[0],
				"ex-overdue":     exercises[1],
				"ex-in-progress": exercises[2],
				"ex-pending":     exercises[3],
				"ex-draft":       exercises[4],
			},
		},
	)

	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/student/evaluations", handler.GetEvaluations)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/student/evaluations", nil)
	req.Header.Set("X-Tenant-Id", "tenant-test")
	req.Header.Set("X-User-Id", "student-1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var items []httpdelivery.StudentEvaluationItem
	decodeStudentData(t, rr.Body.Bytes(), &items)

	if len(items) != 4 {
		t.Fatalf("expected 4 published exercises, got %d: %+v", len(items), items)
	}

	statusByExercise := make(map[string]httpdelivery.StudentEvaluationItem, len(items))
	for _, item := range items {
		statusByExercise[item.ExerciseID] = item
	}

	if _, found := statusByExercise["ex-draft"]; found {
		t.Errorf("draft exercise must not be listed")
	}

	cases := []struct {
		exerciseID     string
		wantStatus     string
		wantAttempts   int
		wantBestScore  *int
		wantLastResult string
	}{
		{exerciseID: "ex-completed", wantStatus: "completed", wantAttempts: 1, wantBestScore: intPtr(100), wantLastResult: "AC"},
		{exerciseID: "ex-overdue", wantStatus: "overdue", wantAttempts: 0},
		{exerciseID: "ex-in-progress", wantStatus: "in_progress", wantAttempts: 1, wantLastResult: "WA"},
		{exerciseID: "ex-pending", wantStatus: "pending", wantAttempts: 0},
	}

	for _, tc := range cases {
		item, found := statusByExercise[tc.exerciseID]
		if !found {
			t.Fatalf("missing evaluation for exercise %s", tc.exerciseID)
		}
		if item.AcademicStatus != tc.wantStatus {
			t.Errorf("exercise %s: expected status %q, got %q", tc.exerciseID, tc.wantStatus, item.AcademicStatus)
		}
		if item.AttemptsCount != tc.wantAttempts {
			t.Errorf("exercise %s: expected %d attempts, got %d", tc.exerciseID, tc.wantAttempts, item.AttemptsCount)
		}
		if tc.wantLastResult != "" && item.LastVerdict != tc.wantLastResult {
			t.Errorf("exercise %s: expected last verdict %q, got %q", tc.exerciseID, tc.wantLastResult, item.LastVerdict)
		}
		if tc.wantBestScore == nil {
			if item.BestScore != nil {
				t.Errorf("exercise %s: expected no best score, got %d", tc.exerciseID, *item.BestScore)
			}
		} else if item.BestScore == nil || *item.BestScore != *tc.wantBestScore {
			t.Errorf("exercise %s: expected best score %d, got %v", tc.exerciseID, *tc.wantBestScore, item.BestScore)
		}
		if item.SubjectCode != "ALG-101" || item.SubjectName != "Algoritmos" {
			t.Errorf("exercise %s: unexpected subject enrichment %q %q", tc.exerciseID, item.SubjectCode, item.SubjectName)
		}
	}
}

func TestGetStudentSubmissionsReturnsFullHistory(t *testing.T) {
	subjectID := "subj-1"
	subject := &domain.Subject{ID: subjectID, Name: "Algoritmos", Code: "ALG-101"}
	exercise := &domain.Exercise{
		ID:        "ex-1",
		SubjectID: &subjectID,
		Title:     "Suma de enteros",
		Type:      domain.ExerciseType("code"),
		Purpose:   "class",
	}

	now := time.Now()
	submissions := make([]*domain.Submission, 0, 7)
	for i := 0; i < 7; i++ {
		submissions = append(submissions, &domain.Submission{
			ID:          "sub-" + string(rune('a'+i)),
			ExerciseID:  "ex-1",
			Verdict:     "WA",
			Code:        "print(1)",
			SubmittedAt: now.Add(-time.Duration(i) * time.Minute),
		})
	}

	handler := httpdelivery.NewStudentHandler(
		&mockSubjectRepoForStudent{subjects: []*domain.Subject{subject}},
		nil,
		&mockSubmissionRepoForStudent{submissions: submissions},
		&mockExerciseRepoForStudent{
			byID: map[string]*domain.Exercise{"ex-1": exercise},
		},
	)

	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/student/submissions", handler.GetSubmissions)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/student/submissions", nil)
	req.Header.Set("X-Tenant-Id", "tenant-test")
	req.Header.Set("X-User-Id", "student-1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var items []httpdelivery.StudentSubmissionHistoryItem
	decodeStudentData(t, rr.Body.Bytes(), &items)

	if len(items) != 7 {
		t.Fatalf("expected all 7 submissions without truncation, got %d", len(items))
	}

	for _, item := range items {
		if item.ExerciseTitle != "Suma de enteros" {
			t.Errorf("expected exercise_title enrichment, got %q", item.ExerciseTitle)
		}
		if item.SubjectCode != "ALG-101" {
			t.Errorf("expected subject_code enrichment, got %q", item.SubjectCode)
		}
		if item.SubjectName != "Algoritmos" {
			t.Errorf("expected subject_name enrichment, got %q", item.SubjectName)
		}
		if item.ExerciseType != "code" || item.ExercisePurpose != "class" {
			t.Errorf("unexpected exercise enrichment %q %q", item.ExerciseType, item.ExercisePurpose)
		}
	}
}

func TestGetStudentEvaluationsRequiresIdentity(t *testing.T) {
	handler := httpdelivery.NewStudentHandler(nil, nil, nil, nil)
	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/student/evaluations", handler.GetEvaluations)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/student/evaluations", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func intPtr(v int) *int {
	return &v
}
