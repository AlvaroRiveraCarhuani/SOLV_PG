package services

import (
	"context"
	"fmt"
	"strings"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/mailer"
)

func (s *TeacherService) ListCourseSubmissions(ctx context.Context, tenantID, teacherID, subjectID, exerciseID, verdict string) ([]*domain.SubmissionQueueItem, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if subjectID == "" {
		return nil, domain.ErrNotFound
	}
	items, err := s.repo.ListCourseSubmissions(ctx, tenantID, teacherID, subjectID, exerciseID, verdict)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]*domain.SubmissionQueueItem, 0)
	}
	return items, nil
}

func (s *TeacherService) GetTeacherSubmissionReview(ctx context.Context, tenantID, teacherID, submissionID string) (*domain.TeacherSubmissionReviewDTO, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if submissionID == "" {
		return nil, domain.ErrNotFound
	}
	review, err := s.repo.GetTeacherSubmissionReview(ctx, tenantID, teacherID, submissionID)
	if err != nil {
		return nil, err
	}
	return review, nil
}

func (s *TeacherService) AddComment(ctx context.Context, comment *domain.SubmissionComment) error {
	if comment.TenantID == "" {
		return domain.ErrInvalidTenant
	}
	if comment.SubmissionID == "" || strings.TrimSpace(comment.Comment) == "" {
		return fmt.Errorf("comentario y submission_id son requeridos")
	}
	return s.repo.AddComment(ctx, comment)
}

func (s *TeacherService) GetCommentsBySubmission(ctx context.Context, tenantID, submissionID string) ([]*domain.SubmissionComment, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if submissionID == "" {
		return nil, domain.ErrNotFound
	}
	comments, err := s.repo.GetCommentsBySubmission(ctx, tenantID, submissionID)
	if err != nil {
		return nil, err
	}
	if comments == nil {
		comments = make([]*domain.SubmissionComment, 0)
	}
	return comments, nil
}

func (s *TeacherService) OverrideSubmission(ctx context.Context, tenantID, submissionID, verdict, reason string, score *int, gradedBy *string) error {
	if tenantID == "" {
		return domain.ErrInvalidTenant
	}
	if submissionID == "" {
		return domain.ErrNotFound
	}
	if len(strings.TrimSpace(reason)) < 10 {
		return domain.ErrInvalidOverrideReason
	}
	if s.subRepo == nil {
		return fmt.Errorf("submission repository not configured")
	}
	if err := s.subRepo.UpdateOverride(ctx, tenantID, submissionID, verdict, reason, score, gradedBy); err != nil {
		return err
	}

	// Despacho asíncrono de notificación al estudiante (desacoplado de la persistencia de la nota)
	s.notifyGradePublished(ctx, tenantID, submissionID, gradedBy)

	return nil
}

func (s *TeacherService) notifyGradePublished(ctx context.Context, tenantID, submissionID string, gradedBy *string) {
	if s.mailer == nil || s.repo == nil {
		return
	}

	teacherID := ""
	if gradedBy != nil {
		teacherID = *gradedBy
	}

	review, err := s.repo.GetTeacherSubmissionReview(ctx, tenantID, teacherID, submissionID)
	if err != nil || review == nil || strings.TrimSpace(review.StudentEmail) == "" {
		return
	}

	branding := domain.EmailBranding{
		InstitutionName: "Plataforma Académica",
	}
	if s.tenantRepo != nil {
		if tenant, err := s.tenantRepo.GetByID(ctx, tenantID); err == nil && tenant != nil {
			branding = tenant.ResolveBranding()
		}
	}

	domainName := strings.TrimSpace(branding.BaseDomain)
	if domainName == "" {
		domainName = "localhost:4200"
	}
	reviewURL := fmt.Sprintf("https://%s/student/courses/%s/assignments/%s", domainName, review.SubjectID, review.ExerciseID)

	htmlBody, textBody := mailer.BuildGradeFeedbackEmail(
		branding,
		review.StudentName,
		review.ExerciseTitle,
		review.SubjectName,
		reviewURL,
	)

	s.mailer.SendAsync(&domain.EmailMessage{
		To:       review.StudentEmail,
		Subject:  fmt.Sprintf("[%s] Evaluación publicada: %s — %s", branding.InstitutionName, review.ExerciseTitle, review.SubjectName),
		HTMLBody: htmlBody,
		TextBody: textBody,
	})
}
