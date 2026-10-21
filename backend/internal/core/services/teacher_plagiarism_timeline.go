package services

import (
	"context"
	"fmt"
	"time"

	"solv-backend/internal/core/domain"
)

func (s *TeacherService) AnalyzePlagiarism(ctx context.Context, tenantID, teacherID, subjectID, exerciseID string) (*domain.PlagiarismReport, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if subjectID == "" {
		return nil, domain.ErrNotFound
	}

	submissions, err := s.repo.GetExerciseSubmissionsForPlagiarism(ctx, tenantID, teacherID, subjectID, exerciseID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener entregas para analisis anti-plagio: %w", err)
	}

	engine := NewASTPlagiarismEngine()
	matches := engine.AnalyzeSubmissions(submissions)

	exerciseTitle := ""
	if len(submissions) > 0 && exerciseID != "" {
		exerciseTitle = submissions[0].ExerciseTitle
	}

	report := &domain.PlagiarismReport{
		SubjectID:         subjectID,
		ExerciseID:        exerciseID,
		ExerciseTitle:     exerciseTitle,
		AnalyzedAt:        time.Now(),
		TotalSubmissions:  len(submissions),
		SuspectPairsCount: len(matches),
		Matches:           matches,
	}

	return report, nil
}

func (s *TeacherService) GetKeystrokeEvents(ctx context.Context, tenantID, teacherID, submissionID string) (*domain.SubmissionKeystrokeReport, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if submissionID == "" {
		return nil, domain.ErrNotFound
	}

	_, err := s.repo.GetTeacherSubmissionReview(ctx, tenantID, teacherID, submissionID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener entrega para auditoría de eventos: %w", err)
	}

	var events []domain.SubmissionKeystrokeEvent
	if s.keystrokeRepo != nil {
		events, err = s.keystrokeRepo.GetBySubmission(ctx, submissionID)
		if err != nil {
			return nil, fmt.Errorf("error al consultar eventos de escritura: %w", err)
		}
	}

	report := &domain.SubmissionKeystrokeReport{
		SubmissionID: submissionID,
		Events:       events,
	}

	totalTypedChars := 0
	totalPastedChars := 0
	pasteCount := 0
	var maxTimeMS int64 = 0

	for _, ev := range events {
		if ev.TimestampMS > maxTimeMS {
			maxTimeMS = ev.TimestampMS
		}
		if ev.EventType == domain.KeystrokeEventPaste || ev.PasteSourceDetected {
			pasteCount++
			totalPastedChars += len(ev.Content)
		} else if ev.EventType == domain.KeystrokeEventInsert {
			totalTypedChars += len(ev.Content)
		}
	}

	totalChars := totalTypedChars + totalPastedChars
	var pastePct float64 = 0
	if totalChars > 0 {
		pastePct = float64(totalPastedChars) / float64(totalChars)
	}

	report.TotalTimeMS = maxTimeMS
	report.PasteCount = pasteCount
	report.PastePercentage = pastePct
	report.TotalCharsTyped = totalTypedChars
	report.TotalCharsPasted = totalPastedChars

	return report, nil
}

func (s *TeacherService) GetSubmissionTimeline(ctx context.Context, tenantID, teacherID, submissionID string) (*domain.SubmissionTimeline, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if submissionID == "" {
		return nil, domain.ErrNotFound
	}

	review, err := s.repo.GetTeacherSubmissionReview(ctx, tenantID, teacherID, submissionID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener entrega para timeline: %w", err)
	}

	if s.keystrokeRepo != nil {
		events, err := s.keystrokeRepo.GetBySubmission(ctx, submissionID)
		if err == nil && len(events) > 0 {
			keyframes := make([]domain.TimelineKeyframe, 0, len(events))
			totalKeystrokes := 0
			pasteEventsCount := 0
			totalPastedChars := 0
			totalChars := 0

			for _, ev := range events {
				isPaste := ev.EventType == domain.KeystrokeEventPaste || ev.PasteSourceDetected
				if isPaste {
					pasteEventsCount++
					totalPastedChars += len(ev.Content)
				} else if ev.EventType == domain.KeystrokeEventInsert {
					totalKeystrokes++
				}
				totalChars += len(ev.Content)

				keyframes = append(keyframes, domain.TimelineKeyframe{
					OffsetMS:   int(ev.TimestampMS),
					Action:     string(ev.EventType),
					Content:    ev.Content,
					CursorLine: ev.Position,
					IsPaste:    isPaste,
					CharCount:  len(ev.Content),
				})
			}

			pastePercentage := 0.0
			if totalChars > 0 {
				pastePercentage = float64(totalPastedChars) / float64(totalChars) * 100.0
			}

			totalDurationSecs := 0
			if len(events) > 0 {
				totalDurationSecs = int(events[len(events)-1].TimestampMS / 1000)
			}

			return &domain.SubmissionTimeline{
				SubmissionID:         submissionID,
				StudentID:            review.StudentID,
				StudentName:          review.StudentName,
				TotalDurationSeconds: totalDurationSecs,
				TotalKeystrokes:      totalKeystrokes,
				PasteEventsCount:     pasteEventsCount,
				PastePercentage:      pastePercentage,
				SuspiciousPasteFlag:  pasteEventsCount > 0,
				Keyframes:            keyframes,
			}, nil
		}
	}

	engine := NewTimelineEngine()
	timeline := engine.GenerateTimeline(submissionID, review.StudentID, review.StudentName, review.Code, "")

	return timeline, nil
}
