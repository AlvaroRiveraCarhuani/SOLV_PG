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
	if comment.FilePath != nil && strings.TrimSpace(*comment.FilePath) == "" {
		return domain.ErrCommentFileUnknown
	}
	// El archivo y la línea local van juntos o no van: un ancla a medias no se
	// puede resolver y ocultaría un error del cliente.
	if (comment.FilePath == nil) != (comment.LocalLineNumber == nil) {
		return domain.ErrLineRangeInvalid
	}
	if comment.LocalLineNumber != nil && *comment.LocalLineNumber < 1 {
		return domain.ErrLineRangeInvalid
	}

	review, err := s.repo.GetTeacherSubmissionReview(ctx, comment.TenantID, comment.AuthorID, comment.SubmissionID)
	if err != nil {
		return err
	}
	if review != nil {
		totalLines := strings.Count(review.Code, "\n") + 1

		if comment.FilePath != nil {
			// Ancla por archivo: el backend es el único que conoce `start_line`.
			if !review.FilesAvailable || len(review.Files) == 0 {
				return domain.ErrCommentFileUnknown
			}
			file := domain.ProjectFileByPath(review.Files, *comment.FilePath)
			if file == nil {
				return domain.ErrCommentFileUnknown
			}
			local := *comment.LocalLineNumber
			if local > file.LineCount {
				return domain.ErrLineOutOfRange
			}
			global := file.StartLine + local - 1
			if comment.LineNumber != 0 && comment.LineNumber != global {
				return domain.ErrLineRangeInvalid
			}
			comment.LineNumber = global

			if comment.LocalEndLineNumber != nil {
				localEnd := *comment.LocalEndLineNumber
				if localEnd < local || localEnd > file.LineCount {
					return domain.ErrLineRangeInvalid
				}
				globalEnd := file.StartLine + localEnd - 1
				if comment.EndLineNumber != nil && *comment.EndLineNumber != globalEnd {
					return domain.ErrLineRangeInvalid
				}
				comment.EndLineNumber = &globalEnd
			}
		} else if review.FilesAvailable {
			// Cliente con coordenadas globales: se deriva el ancla por archivo
			// cuando la línea cae dentro de uno. Una cabecera del snapshot queda
			// sin archivo, como los comentarios generales de la rúbrica.
			if file := domain.ProjectFileForGlobalLine(review.Files, comment.LineNumber); file != nil {
				path := file.Path
				local := comment.LineNumber - file.StartLine + 1
				comment.FilePath = &path
				comment.LocalLineNumber = &local
			}
		}

		endLine := comment.LineNumber
		if comment.EndLineNumber != nil {
			endLine = *comment.EndLineNumber
		}
		if comment.LineNumber < 1 || endLine < comment.LineNumber {
			return domain.ErrLineRangeInvalid
		}
		if endLine > totalLines {
			return domain.ErrLineOutOfRange
		}
		// Un bloque IDE no puede cruzar cabeceras ni archivos: el comentario se
		// ancla a una fila del árbol, no a dos. Los comentarios de una sola línea
		// siguen pudiendo caer en una cabecera (nota general de la rúbrica).
		if review.FilesAvailable && endLine > comment.LineNumber {
			startFile := domain.ProjectFileForGlobalLine(review.Files, comment.LineNumber)
			endFile := domain.ProjectFileForGlobalLine(review.Files, endLine)
			if startFile == nil || endFile == nil || startFile.Path != endFile.Path {
				return domain.ErrLineRangeSpansFiles
			}
		}
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

	s.syncReviewAfterOverride(ctx, tenantID, submissionID, reason, score, gradedBy)

	// Despacho asíncrono de notificación al estudiante (desacoplado de la persistencia de la nota)
	s.notifyGradePublished(ctx, tenantID, submissionID, gradedBy)

	return nil
}

// syncReviewAfterOverride alinea el estado de revisión y la libreta con una
// calificación hecha en SpeedGrader: corregir la entrega es haberla revisado.
func (s *TeacherService) syncReviewAfterOverride(ctx context.Context, tenantID, submissionID, reason string, score *int, gradedBy *string) {
	if s.reviewStore == nil {
		return
	}
	actorID := ""
	if gradedBy != nil {
		actorID = *gradedBy
	}

	if current, err := s.reviewStore.GetSubmissionReviewState(ctx, tenantID, actorID, submissionID); err == nil {
		if current == domain.ReviewPending || current == domain.ReviewInReview {
			_ = s.reviewStore.UpdateSubmissionReview(
				ctx, tenantID, actorID, submissionID,
				current, domain.ReviewGraded, reason, score, actorID,
			)
		}
	}

	_ = s.reviewStore.RecordGradeFromSubmission(ctx, tenantID, submissionID, actorID, reason)

	s.auditReviewAction(ctx, tenantID, actorID, "SUBMISSION_GRADE_OVERRIDDEN", submissionID, map[string]interface{}{
		"reason": reason,
		"score":  score,
	})
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
