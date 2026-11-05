package services_test

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/core/services"
)

type fakeReviewStore struct {
	groups          []*domain.ReviewQueueGroup
	currentStatus   domain.ReviewStatus
	environmentType string
	cell            *domain.ExerciseGradeCell
	updated         bool
	recorded        bool
	gradeSet        bool
	lastNext        domain.ReviewStatus
	lastFeedback    string
	lastScore       *int
	lastReason      string
	err             error
}

func (f *fakeReviewStore) ListPendingReviewGroups(_ context.Context, _, _, _ string) ([]*domain.ReviewQueueGroup, error) {
	return f.groups, f.err
}

func (f *fakeReviewStore) GetSubmissionReviewState(_ context.Context, _, _, _ string) (domain.ReviewStatus, error) {
	return f.currentStatus, f.err
}

func (f *fakeReviewStore) GetSubmissionEnvironmentType(_ context.Context, _, _, _ string) (string, error) {
	return f.environmentType, f.err
}

func (f *fakeReviewStore) UpdateSubmissionReview(_ context.Context, _, _, _ string, _, next domain.ReviewStatus, feedback string, score *int, _ string) error {
	if f.err != nil {
		return f.err
	}
	f.updated = true
	f.lastNext = next
	f.lastFeedback = feedback
	f.lastScore = score
	return nil
}

func (f *fakeReviewStore) GetSubmissionGradeCell(_ context.Context, _, _ string) (*domain.ExerciseGradeCell, error) {
	if f.cell == nil {
		return nil, domain.ErrGradeCellNotFound
	}
	return f.cell, nil
}

func (f *fakeReviewStore) RecordGradeFromSubmission(_ context.Context, _, _, _, _ string) error {
	f.recorded = true
	return nil
}

func (f *fakeReviewStore) SetExerciseGrade(_ context.Context, _, _, _, _, _ string, _ *int, reason, _ string) (*int, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.gradeSet = true
	f.lastReason = reason
	return nil, nil
}

func (f *fakeReviewStore) GetExerciseGradeCell(_ context.Context, _, _, _ string) (*domain.ExerciseGradeCell, error) {
	if f.cell == nil {
		return nil, domain.ErrGradeCellNotFound
	}
	return f.cell, nil
}

type fakeAuditRepo struct {
	domain.AuditLogRepository
	logs []*domain.AuditLog
}

func (f *fakeAuditRepo) Create(_ context.Context, log *domain.AuditLog) error {
	f.logs = append(f.logs, log)
	return nil
}

func newReviewService(store domain.TeacherReviewStore) (*services.TeacherService, *fakeAuditRepo) {
	repo := &mockTeacherSubmissionsRepo{
		reviewResult: &domain.TeacherSubmissionReviewDTO{
			ID:         "sub-1",
			StudentID:  "stu-1",
			ExerciseID: "ex-1",
		},
	}
	svc := services.NewTeacherService(repo, nil)
	svc.SetReviewStore(store)
	audit := &fakeAuditRepo{}
	svc.SetAuditRepository(audit)
	return svc, audit
}

func TestListPendingReviewGroups_RequiresTenantAndStore(t *testing.T) {
	svc := services.NewTeacherService(nil, nil)

	if _, err := svc.ListPendingReviewGroups(context.Background(), "", "t-1", ""); !errors.Is(err, domain.ErrInvalidTenant) {
		t.Fatalf("expected ErrInvalidTenant, got %v", err)
	}
	if _, err := svc.ListPendingReviewGroups(context.Background(), "t-1", "t-1", ""); !errors.Is(err, domain.ErrReviewStoreUnavailable) {
		t.Fatalf("expected ErrReviewStoreUnavailable, got %v", err)
	}
}

func TestUpdateSubmissionReview_ValidatesTransitionAndAudits(t *testing.T) {
	store := &fakeReviewStore{currentStatus: domain.ReviewPending}
	svc, audit := newReviewService(store)

	score := 85
	cell, err := svc.UpdateSubmissionReview(context.Background(), "t-1", "teacher-1", "sub-1", domain.UpdateReviewRequest{
		Status:          "graded",
		FeedbackSummary: "  Buen trabajo  ",
		Score:           &score,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.updated || store.lastNext != domain.ReviewGraded {
		t.Fatalf("expected review to be graded, got %v", store.lastNext)
	}
	if store.lastFeedback != "Buen trabajo" {
		t.Errorf("feedback must be trimmed, got %q", store.lastFeedback)
	}
	if !store.recorded {
		t.Error("grading must update the grade cell")
	}
	if cell != nil {
		t.Errorf("expected no cell when store has none, got %+v", cell)
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != "SUBMISSION_REVIEW_UPDATED" {
		t.Fatalf("expected one audit entry, got %+v", audit.logs)
	}

	// Transición no permitida.
	store.currentStatus = domain.ReviewNotRequired
	if _, err := svc.UpdateSubmissionReview(context.Background(), "t-1", "teacher-1", "sub-1", domain.UpdateReviewRequest{Status: "graded"}); !errors.Is(err, domain.ErrInvalidReviewTransition) {
		t.Fatalf("expected ErrInvalidReviewTransition, got %v", err)
	}

	// Estado desconocido.
	if _, err := svc.UpdateSubmissionReview(context.Background(), "t-1", "teacher-1", "sub-1", domain.UpdateReviewRequest{Status: "aprobado"}); !errors.Is(err, domain.ErrInvalidReviewStatus) {
		t.Fatalf("expected ErrInvalidReviewStatus, got %v", err)
	}

	// Nota fuera de rango.
	bad := 101
	if _, err := svc.UpdateSubmissionReview(context.Background(), "t-1", "teacher-1", "sub-1", domain.UpdateReviewRequest{Status: "graded", Score: &bad}); !errors.Is(err, domain.ErrInvalidScoreRange) {
		t.Fatalf("expected ErrInvalidScoreRange, got %v", err)
	}
}

// Una entrega IDE anterior a la separación de modalidades pudo quedar en
// 'not_required'. El docente debe poder calificarla; en juez la transición
// sigue cerrada porque ese estado lo fija el propio juez.
func TestUpdateSubmissionReview_AllowsLegacyIDEProjectFromNotRequired(t *testing.T) {
	ideStore := &fakeReviewStore{
		currentStatus:   domain.ReviewNotRequired,
		environmentType: domain.PracticeEnvironmentTypeIDEPersistente,
	}
	ideService, _ := newReviewService(ideStore)

	score := 70
	if _, err := ideService.UpdateSubmissionReview(context.Background(), "t-1", "teacher-1", "sub-1", domain.UpdateReviewRequest{
		Status: "graded",
		Score:  &score,
	}); err != nil {
		t.Fatalf("un proyecto IDE en not_required debe poder calificarse: %v", err)
	}
	if !ideStore.updated || ideStore.lastNext != domain.ReviewGraded {
		t.Fatalf("esperado review graded en IDE, obtenido %v", ideStore.lastNext)
	}

	judgeStore := &fakeReviewStore{
		currentStatus:   domain.ReviewNotRequired,
		environmentType: domain.PracticeEnvironmentTypeJuezEfimero,
	}
	judgeService, _ := newReviewService(judgeStore)
	if _, err := judgeService.UpdateSubmissionReview(context.Background(), "t-1", "teacher-1", "sub-2", domain.UpdateReviewRequest{
		Status: "graded",
		Score:  &score,
	}); !errors.Is(err, domain.ErrInvalidReviewTransition) {
		t.Fatalf("en juez la transición not_required -> graded debe seguir cerrada, obtenido %v", err)
	}
}

// La nota manual de otro docente no se reemplaza sin motivo. Se valida antes
// de escribir: un intento rechazado no deja la revisión a medias, y el mismo
// docente conserva la posibilidad de ajustar su propia nota.
func TestUpdateSubmissionReview_ProtectsManualCellFromOtherTeachers(t *testing.T) {
	ctx := context.Background()
	score70 := 70
	score80 := 80
	manualCell := func(author string) *domain.ExerciseGradeCell {
		return &domain.ExerciseGradeCell{Score: &score70, Source: "manual", OverriddenBy: author}
	}

	rejected := &fakeReviewStore{currentStatus: domain.ReviewGraded, cell: manualCell("teacher-a")}
	rejectedService, _ := newReviewService(rejected)
	if _, err := rejectedService.UpdateSubmissionReview(ctx, "t-1", "teacher-b", "sub-1", domain.UpdateReviewRequest{
		Status: "graded",
		Score:  &score80,
	}); !errors.Is(err, domain.ErrGradeReasonRequired) {
		t.Fatalf("un cambio de nota ajeno sin motivo debe rechazarse, obtenido %v", err)
	}
	if rejected.updated {
		t.Fatal("el intento rechazado no debe escribir la revisión")
	}

	justified := &fakeReviewStore{currentStatus: domain.ReviewGraded, cell: manualCell("teacher-a")}
	justifiedService, _ := newReviewService(justified)
	if _, err := justifiedService.UpdateSubmissionReview(ctx, "t-1", "teacher-b", "sub-1", domain.UpdateReviewRequest{
		Status:          "graded",
		Score:           &score80,
		FeedbackSummary: "La entrega cumple los criterios acordados en la rúbrica.",
	}); err != nil {
		t.Fatalf("una recorrección justificada debe permitirse: %v", err)
	}
	if !justified.updated || justified.lastFeedback == "" {
		t.Fatalf("la recorrección justificada debe escribir la revisión con su motivo: %+v", justified)
	}

	own := &fakeReviewStore{currentStatus: domain.ReviewGraded, cell: manualCell("teacher-a")}
	ownService, _ := newReviewService(own)
	if _, err := ownService.UpdateSubmissionReview(ctx, "t-1", "teacher-a", "sub-1", domain.UpdateReviewRequest{
		Status: "graded",
		Score:  &score80,
	}); err != nil {
		t.Fatalf("el mismo docente puede ajustar su nota sin un motivo nuevo: %v", err)
	}
}

func TestSetExerciseGrade_DelegatesAndAudits(t *testing.T) {
	store := &fakeReviewStore{cell: &domain.ExerciseGradeCell{Source: "manual", Score: nil}}
	svc, audit := newReviewService(store)

	cell, err := svc.SetExerciseGrade(context.Background(), "t-1", "teacher-1", "stu-1", "ex-1", domain.SetExerciseGradeRequest{
		SelectedSubmissionID: "sub-1",
		Reason:               "Entregó una versión posterior corregida",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.gradeSet {
		t.Fatal("expected the store to persist the selection")
	}
	if cell == nil || cell.Source != "manual" {
		t.Fatalf("unexpected cell: %+v", cell)
	}
	if len(audit.logs) != 1 || audit.logs[0].Action != "EXERCISE_GRADE_SELECTED" {
		t.Fatalf("expected one audit entry, got %+v", audit.logs)
	}

	// Identificadores obligatorios.
	if _, err := svc.SetExerciseGrade(context.Background(), "t-1", "teacher-1", "", "ex-1", domain.SetExerciseGradeRequest{SelectedSubmissionID: "sub-1"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing student, got %v", err)
	}
}

func TestUpdateSubmissionReview_ScoreOutOfRangeRejected(t *testing.T) {
	store := &fakeReviewStore{currentStatus: domain.ReviewPending}
	svc, _ := newReviewService(store)

	negative := -1
	if _, err := svc.UpdateSubmissionReview(context.Background(), "t-1", "teacher-1", "sub-1", domain.UpdateReviewRequest{Status: "graded", Score: &negative}); !errors.Is(err, domain.ErrInvalidScoreRange) {
		t.Fatalf("expected ErrInvalidScoreRange, got %v", err)
	}
	if store.updated {
		t.Error("invalid score must not reach the store")
	}
}
