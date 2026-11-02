package services

import (
	"context"
	"fmt"
	"strings"
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

// GetPlagiarismPair arma el documento del visor para dos entregas de la misma
// práctica. El acceso se resuelve en la consulta: si una entrega no pertenece a
// una materia del docente, el par no existe para él.
func (s *TeacherService) GetPlagiarismPair(ctx context.Context, tenantID, teacherID, submissionA, submissionB string) (*domain.PlagiarismPair, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if strings.TrimSpace(submissionA) == "" || strings.TrimSpace(submissionB) == "" {
		return nil, domain.ErrNotFound
	}
	if strings.EqualFold(strings.TrimSpace(submissionA), strings.TrimSpace(submissionB)) {
		return nil, domain.ErrPlagiarismPairSameSubmission
	}

	rows, err := s.repo.GetPlagiarismPairSubmissions(ctx, tenantID, teacherID, submissionA, submissionB)
	if err != nil {
		return nil, err
	}
	if len(rows) != 2 {
		return nil, domain.ErrPlagiarismPairForbidden
	}

	var sideA, sideB *domain.PlagiarismPairSubmission
	for _, row := range rows {
		switch {
		case strings.EqualFold(row.SubmissionID, submissionA):
			sideA = row
		case strings.EqualFold(row.SubmissionID, submissionB):
			sideB = row
		}
	}
	if sideA == nil || sideB == nil {
		return nil, domain.ErrPlagiarismPairForbidden
	}
	if sideA.ExerciseID != sideB.ExerciseID {
		return nil, domain.ErrPlagiarismPairDifferentExercise
	}

	engine := NewASTPlagiarismEngine()
	similarity, _, ranges, _, _ := engine.CompareSubmissions(
		plagiarismSource(sideA),
		plagiarismSource(sideB),
	)

	exercise := domain.PlagiarismPairExercise{
		ID:                 sideA.ExerciseID,
		Title:              sideA.ExerciseTitle,
		SubjectID:          sideA.SubjectID,
		SubjectName:        sideA.SubjectName,
		EnvironmentType:    sideA.EnvironmentType,
		ExpectedComplexity: sideA.ExpectedComplexity,
	}
	if strings.EqualFold(sideA.EnvironmentType, string(domain.EnvironmentTypeIDEPersistente)) {
		if baseFiles, ok := domain.ParseProjectSnapshot(sideA.Boilerplate, sideA.Language); ok {
			exercise.BaseFiles = baseFiles
			exercise.BaseFilesAvailable = true
		}
	}

	return &domain.PlagiarismPair{
		Exercise:   exercise,
		A:          s.buildPlagiarismSide(ctx, sideA),
		B:          s.buildPlagiarismSide(ctx, sideB),
		Similarity: similarity,
		RiskLevel:  plagiarismRiskLevel(similarity),
		Ranges:     ranges,
	}, nil
}

func plagiarismSource(sub *domain.PlagiarismPairSubmission) *domain.SubmissionForPlagiarism {
	return &domain.SubmissionForPlagiarism{
		SubmissionID:    sub.SubmissionID,
		ExerciseID:      sub.ExerciseID,
		ExerciseTitle:   sub.ExerciseTitle,
		StudentID:       sub.StudentID,
		StudentName:     sub.StudentName,
		StudentEmail:    sub.StudentEmail,
		Language:        sub.Language,
		Code:            sub.Code,
		Boilerplate:     sub.Boilerplate,
		EnvironmentType: sub.EnvironmentType,
		Verdict:         sub.Verdict,
		SubmittedAt:     sub.SubmittedAt,
	}
}

func (s *TeacherService) buildPlagiarismSide(ctx context.Context, sub *domain.PlagiarismPairSubmission) domain.PlagiarismPairSide {
	side := domain.PlagiarismPairSide{
		SubmissionID:    sub.SubmissionID,
		StudentID:       sub.StudentID,
		StudentName:     sub.StudentName,
		StudentEmail:    sub.StudentEmail,
		Verdict:         sub.Verdict,
		Language:        sub.Language,
		ExecutionTimeMS: sub.ExecutionTimeMS,
		MemoryUsedMB:    sub.MemoryUsedMB,
		Metrics:         s.plagiarismPairMetrics(ctx, sub.SubmissionID),
	}
	if sub.ComplexityAnalysis != nil {
		side.Complexity = sub.ComplexityAnalysis.TimeComplexity
	}

	if strings.EqualFold(sub.EnvironmentType, string(domain.EnvironmentTypeIDEPersistente)) {
		if files, ok := domain.ParseProjectSnapshot(sub.Code, sub.Language); ok {
			side.Files = files
			side.FilesAvailable = true
			return side
		}
	}

	side.Code = sub.Code
	return side
}

// plagiarismPairMetrics resume la telemetría de escritura. Sin eventos
// disponibles devuelve ceros y events_available=false: la vista lo dice, no
// inventa un porcentaje de pegado.
func (s *TeacherService) plagiarismPairMetrics(ctx context.Context, submissionID string) domain.PlagiarismPairMetrics {
	metrics := domain.PlagiarismPairMetrics{}
	if s.keystrokeRepo == nil {
		return metrics
	}

	events, err := s.keystrokeRepo.GetBySubmission(ctx, submissionID)
	if err != nil {
		return metrics
	}

	var maxTimeMS int64
	totalTypedChars, totalPastedChars := 0, 0
	for _, ev := range events {
		if ev.TimestampMS > maxTimeMS {
			maxTimeMS = ev.TimestampMS
		}
		if ev.EventType == domain.KeystrokeEventPaste || ev.PasteSourceDetected {
			metrics.PasteEvents++
			totalPastedChars += len(ev.Content)
			continue
		}
		if ev.EventType == domain.KeystrokeEventInsert {
			metrics.TotalKeystrokes++
			totalTypedChars += len(ev.Content)
		}
	}

	totalChars := totalTypedChars + totalPastedChars
	if totalChars > 0 {
		metrics.PastePercentage = float64(totalPastedChars) / float64(totalChars) * 100.0
	}
	metrics.TotalTimeMS = maxTimeMS
	metrics.EventsAvailable = len(events) > 0
	return metrics
}

func plagiarismRiskLevel(similarity float64) string {
	switch {
	case similarity >= 85.0:
		return "critical"
	case similarity >= 70.0:
		return "warning"
	default:
		return "info"
	}
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
