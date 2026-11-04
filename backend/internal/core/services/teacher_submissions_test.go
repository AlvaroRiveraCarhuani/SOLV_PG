package services_test

import (
	"context"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type mockTeacherSubmissionsRepo struct {
	domain.TeacherRepository
	reviewResult *domain.TeacherSubmissionReviewDTO
	reviewErr    error
	added        *domain.SubmissionComment
}

func (m *mockTeacherSubmissionsRepo) AddComment(_ context.Context, comment *domain.SubmissionComment) error {
	m.added = comment
	return nil
}

func (m *mockTeacherSubmissionsRepo) GetTeacherSubmissionReview(ctx context.Context, tenantID, teacherID, submissionID string) (*domain.TeacherSubmissionReviewDTO, error) {
	return m.reviewResult, m.reviewErr
}

type mockSubRepo struct {
	domain.SubmissionRepository
	updateOverrideCalled bool
	lastTenantID         string
	lastSubmissionID     string
	lastVerdict          string
	lastReason           string
	lastScore            *int
	lastGradedBy         *string
	updateOverrideErr    error
}

func (m *mockSubRepo) UpdateOverride(ctx context.Context, tenantID, id, verdict, reason string, score *int, gradedBy *string) error {
	m.updateOverrideCalled = true
	m.lastTenantID = tenantID
	m.lastSubmissionID = id
	m.lastVerdict = verdict
	m.lastReason = reason
	m.lastScore = score
	m.lastGradedBy = gradedBy
	return m.updateOverrideErr
}

type mockMailer struct {
	sentMessages []*domain.EmailMessage
}

func (m *mockMailer) Send(ctx context.Context, msg *domain.EmailMessage) error {
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}

func (m *mockMailer) SendAsync(msg *domain.EmailMessage) {
	m.sentMessages = append(m.sentMessages, msg)
}

func (m *mockMailer) Close() error {
	return nil
}

type mockTenantRepoForTeacher struct {
	domain.TenantRepository
	tenant *domain.Tenant
}

func (m *mockTenantRepoForTeacher) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	return m.tenant, nil
}

func ideReviewFixture() *domain.TeacherSubmissionReviewDTO {
	return &domain.TeacherSubmissionReviewDTO{
		ID:              "sub-1",
		EnvironmentType: domain.PracticeEnvironmentTypeIDEPersistente,
		FilesAvailable:  true,
		Code:            "# === FILE: src/app.js ===\nline1\nline2\nline3\nline4\n",
		Files: []domain.ProjectFile{
			{Path: "src/app.js", StartLine: 2, LineCount: 4},
		},
	}
}

func TestTeacherService_AddComment_ResolvesFileAnchor(t *testing.T) {
	repo := &mockTeacherSubmissionsRepo{reviewResult: ideReviewFixture()}
	svc := services.NewTeacherService(repo, nil)

	path := "src/app.js"
	local := 3
	localEnd := 4
	comment := &domain.SubmissionComment{
		TenantID:           "t-1",
		SubmissionID:       "sub-1",
		AuthorID:           "teacher-1",
		FilePath:           &path,
		LocalLineNumber:    &local,
		LocalEndLineNumber: &localEnd,
		Comment:            "Revisar este bloque",
	}
	if err := svc.AddComment(context.Background(), comment); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if comment.LineNumber != 4 {
		t.Fatalf("expected global line 4, got %d", comment.LineNumber)
	}
	if comment.EndLineNumber == nil || *comment.EndLineNumber != 5 {
		t.Fatalf("expected global end 5, got %v", comment.EndLineNumber)
	}
	if repo.added == nil || repo.added.FilePath == nil || *repo.added.FilePath != path {
		t.Fatalf("expected persisted file anchor, got %+v", repo.added)
	}
}

func TestTeacherService_AddComment_DerivesFileFromGlobalLine(t *testing.T) {
	repo := &mockTeacherSubmissionsRepo{reviewResult: ideReviewFixture()}
	svc := services.NewTeacherService(repo, nil)

	comment := &domain.SubmissionComment{
		TenantID:     "t-1",
		SubmissionID: "sub-1",
		AuthorID:     "teacher-1",
		LineNumber:   4,
		Comment:      "Comentario del cliente anterior",
	}
	if err := svc.AddComment(context.Background(), comment); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if comment.FilePath == nil || *comment.FilePath != "src/app.js" {
		t.Fatalf("expected derived file path, got %v", comment.FilePath)
	}
	if comment.LocalLineNumber == nil || *comment.LocalLineNumber != 3 {
		t.Fatalf("expected derived local line 3, got %v", comment.LocalLineNumber)
	}
}

func TestTeacherService_AddComment_RejectsUnknownFile(t *testing.T) {
	repo := &mockTeacherSubmissionsRepo{reviewResult: ideReviewFixture()}
	svc := services.NewTeacherService(repo, nil)

	path := "src/missing.js"
	local := 1
	err := svc.AddComment(context.Background(), &domain.SubmissionComment{
		TenantID:        "t-1",
		SubmissionID:    "sub-1",
		AuthorID:        "teacher-1",
		FilePath:        &path,
		LocalLineNumber: &local,
		Comment:         "Comentario",
	})
	if err != domain.ErrCommentFileUnknown {
		t.Fatalf("expected ErrCommentFileUnknown, got %v", err)
	}
}

func TestTeacherService_AddComment_RejectsMismatchedAnchor(t *testing.T) {
	repo := &mockTeacherSubmissionsRepo{reviewResult: ideReviewFixture()}
	svc := services.NewTeacherService(repo, nil)

	path := "src/app.js"
	local := 3
	err := svc.AddComment(context.Background(), &domain.SubmissionComment{
		TenantID:        "t-1",
		SubmissionID:    "sub-1",
		AuthorID:        "teacher-1",
		LineNumber:      99,
		FilePath:        &path,
		LocalLineNumber: &local,
		Comment:         "Comentario",
	})
	if err != domain.ErrLineRangeInvalid {
		t.Fatalf("expected ErrLineRangeInvalid, got %v", err)
	}
}

func TestTeacherService_AddComment_JudgeKeepsGlobalAnchor(t *testing.T) {
	repo := &mockTeacherSubmissionsRepo{reviewResult: &domain.TeacherSubmissionReviewDTO{
		ID:              "sub-2",
		EnvironmentType: domain.PracticeEnvironmentTypeJuezEfimero,
		FilesAvailable:  false,
		Code:            "print(1)\nprint(2)\n",
	}}
	svc := services.NewTeacherService(repo, nil)

	comment := &domain.SubmissionComment{
		TenantID:     "t-1",
		SubmissionID: "sub-2",
		AuthorID:     "teacher-1",
		LineNumber:   2,
		Comment:      "Comentario de juez",
	}
	if err := svc.AddComment(context.Background(), comment); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if comment.FilePath != nil || comment.LocalLineNumber != nil {
		t.Fatalf("expected no file anchor in judge mode, got %v/%v", comment.FilePath, comment.LocalLineNumber)
	}
}

func TestTeacherService_OverrideSubmission_Validations(t *testing.T) {
	svc := services.NewTeacherService(nil, nil)

	// Tenant faltante
	err := svc.OverrideSubmission(context.Background(), "", "sub-1", "AC", "Razon valida de correccion", nil, nil)
	if err != domain.ErrInvalidTenant {
		t.Fatalf("expected ErrInvalidTenant, got %v", err)
	}

	// SubmissionID faltante
	err = svc.OverrideSubmission(context.Background(), "t-1", "", "AC", "Razon valida de correccion", nil, nil)
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// Razon demasiado corta
	err = svc.OverrideSubmission(context.Background(), "t-1", "sub-1", "AC", "corta", nil, nil)
	if err != domain.ErrInvalidOverrideReason {
		t.Fatalf("expected ErrInvalidOverrideReason, got %v", err)
	}
}

func TestTeacherService_OverrideSubmission_NotifiesStudentViaMailer(t *testing.T) {
	subRepo := &mockSubRepo{}
	teacherRepo := &mockTeacherSubmissionsRepo{
		reviewResult: &domain.TeacherSubmissionReviewDTO{
			ID:            "sub-123",
			ExerciseID:    "ex-456",
			ExerciseTitle: "Algoritmos Greedy",
			SubjectID:     "cs-101",
			SubjectName:   "Estructuras de Datos",
			StudentID:     "stu-789",
			StudentName:   "Carlos Perez",
			StudentEmail:  "carlos.perez@uab.edu.bo",
		},
	}
	mailerMock := &mockMailer{}
	tenantRepo := &mockTenantRepoForTeacher{
		tenant: &domain.Tenant{
			ID:   "t-1",
			Name: "Universidad Adventista de Bolivia",
			Slug: "uab",
			Config: []byte(`{
				"institution_name": "Universidad Adventista de Bolivia",
				"logo_url": "/assets/uab-logo.png",
				"base_domain": "solv.uab.edu.bo"
			}`),
		},
	}

	svc := services.NewTeacherService(teacherRepo, subRepo)
	svc.SetMailer(mailerMock)
	svc.SetTenantRepository(tenantRepo)

	score := 90
	teacherID := "teacher-1"
	err := svc.OverrideSubmission(
		context.Background(),
		"t-1",
		"sub-123",
		"AC",
		"Excelente implementacion con analisis asintotico completo.",
		&score,
		&teacherID,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !subRepo.updateOverrideCalled {
		t.Fatalf("expected subRepo.UpdateOverride to be called")
	}

	if len(mailerMock.sentMessages) != 1 {
		t.Fatalf("expected 1 email message sent, got %d", len(mailerMock.sentMessages))
	}

	msg := mailerMock.sentMessages[0]
	if msg.To != "carlos.perez@uab.edu.bo" {
		t.Errorf("expected recipient carlos.perez@uab.edu.bo, got %s", msg.To)
	}
	if !strings.Contains(msg.Subject, "Universidad Adventista de Bolivia") {
		t.Errorf("subject should contain institution name: %s", msg.Subject)
	}
	if !strings.Contains(msg.Subject, "Algoritmos Greedy") {
		t.Errorf("subject should contain exercise title: %s", msg.Subject)
	}
	if strings.Contains(msg.Subject, "SOLV") || strings.Contains(msg.HTMLBody, "SOLV") {
		t.Errorf("email should not leak SOLV internal brand")
	}
	if !strings.Contains(msg.HTMLBody, "Carlos Perez") {
		t.Errorf("HTML body missing student name")
	}
	if !strings.Contains(msg.HTMLBody, "https://solv.uab.edu.bo/student/courses/cs-101/assignments/ex-456") {
		t.Errorf("HTML body missing canonical review URL")
	}
}

func TestTeacherService_OverrideSubmission_DecoupledWhenMailerNil(t *testing.T) {
	subRepo := &mockSubRepo{}
	teacherRepo := &mockTeacherSubmissionsRepo{}

	// Mailer es nil
	svc := services.NewTeacherService(teacherRepo, subRepo)

	score := 85
	err := svc.OverrideSubmission(
		context.Background(),
		"t-1",
		"sub-123",
		"AC",
		"Aprobado tras verificacion de casos borde.",
		&score,
		nil,
	)

	if err != nil {
		t.Fatalf("expected success even if mailer is nil, got %v", err)
	}
	if !subRepo.updateOverrideCalled {
		t.Fatalf("expected subRepo.UpdateOverride to be called")
	}
}
