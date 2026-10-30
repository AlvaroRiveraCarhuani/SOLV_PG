package services

import (
	"context"
	"encoding/json"
	"strings"

	"solv-backend/internal/core/domain"
)

func (s *TeacherService) reviewStoreOrErr() (domain.TeacherReviewStore, error) {
	if s.reviewStore == nil {
		return nil, domain.ErrReviewStoreUnavailable
	}
	return s.reviewStore, nil
}

// ListPendingReviewGroups devuelve la cola de corrección agrupada por práctica.
func (s *TeacherService) ListPendingReviewGroups(ctx context.Context, tenantID, teacherID, subjectID string) ([]*domain.ReviewQueueGroup, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	store, err := s.reviewStoreOrErr()
	if err != nil {
		return nil, err
	}
	groups, err := store.ListPendingReviewGroups(ctx, tenantID, teacherID, subjectID)
	if err != nil {
		return nil, err
	}
	if groups == nil {
		groups = make([]*domain.ReviewQueueGroup, 0)
	}
	return groups, nil
}

// UpdateSubmissionReview valida la transición y persiste el estado de revisión.
func (s *TeacherService) UpdateSubmissionReview(ctx context.Context, tenantID, teacherID, submissionID string, req domain.UpdateReviewRequest) (*domain.ExerciseGradeCell, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if strings.TrimSpace(submissionID) == "" {
		return nil, domain.ErrNotFound
	}
	store, err := s.reviewStoreOrErr()
	if err != nil {
		return nil, err
	}

	next, err := domain.ParseReviewStatus(req.Status)
	if err != nil {
		return nil, err
	}
	if req.Score != nil && (*req.Score < 0 || *req.Score > 100) {
		return nil, domain.ErrInvalidScoreRange
	}

	current, err := store.GetSubmissionReviewState(ctx, tenantID, teacherID, submissionID)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidateReviewTransition(current, next); err != nil {
		return nil, err
	}

	feedback := strings.TrimSpace(req.FeedbackSummary)
	if err := store.UpdateSubmissionReview(ctx, tenantID, teacherID, submissionID, current, next, feedback, req.Score, teacherID); err != nil {
		return nil, err
	}

	if next == domain.ReviewGraded || next == domain.ReviewReturned {
		// Calificar alimenta la libreta; el error no invalida la corrección.
		_ = store.RecordGradeFromSubmission(ctx, tenantID, submissionID)
	}
	if next == domain.ReviewReturned {
		s.notifyGradePublished(ctx, tenantID, submissionID, &teacherID)
	}

	s.auditReviewAction(ctx, tenantID, teacherID, "SUBMISSION_REVIEW_UPDATED", submissionID, map[string]interface{}{
		"from":     string(current),
		"to":       string(next),
		"score":    req.Score,
		"feedback": feedback,
	})

	return s.loadCellSafe(ctx, tenantID, teacherID, submissionID), nil
}

// SetExerciseGrade fija qué entrega cuenta para la nota de la práctica.
func (s *TeacherService) SetExerciseGrade(ctx context.Context, tenantID, teacherID, studentID, exerciseID string, req domain.SetExerciseGradeRequest) (*domain.ExerciseGradeCell, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if strings.TrimSpace(studentID) == "" || strings.TrimSpace(exerciseID) == "" || strings.TrimSpace(req.SelectedSubmissionID) == "" {
		return nil, domain.ErrNotFound
	}
	if req.Score != nil && (*req.Score < 0 || *req.Score > 100) {
		return nil, domain.ErrInvalidScoreRange
	}
	store, err := s.reviewStoreOrErr()
	if err != nil {
		return nil, err
	}

	if _, err := store.SetExerciseGrade(ctx, tenantID, teacherID, studentID, exerciseID, req.SelectedSubmissionID, req.Score, req.Reason, teacherID); err != nil {
		return nil, err
	}

	cell, err := store.GetExerciseGradeCell(ctx, tenantID, studentID, exerciseID)
	if err != nil {
		return nil, err
	}

	s.auditReviewAction(ctx, tenantID, teacherID, "EXERCISE_GRADE_SELECTED", exerciseID, map[string]interface{}{
		"student_id":             studentID,
		"selected_submission_id": req.SelectedSubmissionID,
		"score":                  cell.Score,
		"reason":                 strings.TrimSpace(req.Reason),
	})

	return cell, nil
}

func (s *TeacherService) loadCellSafe(ctx context.Context, tenantID, teacherID, submissionID string) *domain.ExerciseGradeCell {
	if s.reviewStore == nil || s.repo == nil {
		return nil
	}
	// La celda se busca por la entrega recién revisada para devolver la nota
	// resultante; si aún no existe, la respuesta simplemente no la incluye.
	review, err := s.repo.GetTeacherSubmissionReview(ctx, tenantID, teacherID, submissionID)
	if err != nil || review == nil {
		return nil
	}
	cell, err := s.reviewStore.GetExerciseGradeCell(ctx, tenantID, review.StudentID, review.ExerciseID)
	if err != nil {
		return nil
	}
	return cell
}

func (s *TeacherService) auditReviewAction(ctx context.Context, tenantID, actorID, action, resourceID string, metadata map[string]interface{}) {
	if s.auditRepo == nil || tenantID == "" || actorID == "" {
		return
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return
	}
	id := resourceID
	_ = s.auditRepo.Create(ctx, &domain.AuditLog{
		TenantID:     tenantID,
		ActorID:      actorID,
		Action:       action,
		ResourceType: "submission",
		ResourceID:   &id,
		StatusCode:   200,
		Metadata:     raw,
	})
}
