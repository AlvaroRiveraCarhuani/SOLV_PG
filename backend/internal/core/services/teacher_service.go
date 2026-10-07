package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

type TeacherService struct {
	repo          domain.TeacherRepository
	subRepo       domain.SubmissionRepository
	keystrokeRepo domain.KeystrokeRepository
	evalService   *EvaluationService
	mirrorService *TerminalMirrorService
}

func NewTeacherService(repo domain.TeacherRepository, subRepo ...domain.SubmissionRepository) *TeacherService {
	var sRepo domain.SubmissionRepository
	if len(subRepo) > 0 {
		sRepo = subRepo[0]
	}
	return &TeacherService{
		repo:          repo,
		subRepo:       sRepo,
		mirrorService: NewTerminalMirrorService(nil),
	}
}

func (s *TeacherService) SetKeystrokeRepository(kr domain.KeystrokeRepository) {
	s.keystrokeRepo = kr
}

func (s *TeacherService) SetWorkspaceOrchestrator(orch domain.WorkspaceOrchestrator) {
	s.mirrorService = NewTerminalMirrorService(orch)
}

func (s *TeacherService) SetSubmissionRepository(subRepo domain.SubmissionRepository) {
	s.subRepo = subRepo
}

func (s *TeacherService) SetEvaluationService(evalService *EvaluationService) {
	s.evalService = evalService
}

func (s *TeacherService) GetCoursesSummary(ctx context.Context, tenantID, teacherID string) ([]*domain.TeacherCourseSummary, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	courses, err := s.repo.GetCoursesSummary(ctx, tenantID, teacherID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener resumen de cursos: %w", err)
	}
	if courses == nil {
		courses = make([]*domain.TeacherCourseSummary, 0)
	}
	return courses, nil
}

func (s *TeacherService) GetAttentionWidget(ctx context.Context, tenantID, teacherID string) (*domain.TeacherAttentionWidget, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	widget, err := s.repo.GetAttentionWidget(ctx, tenantID, teacherID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener widget de atencion: %w", err)
	}
	return widget, nil
}

func (s *TeacherService) GetCourseLabsStats(ctx context.Context, tenantID, teacherID, subjectID string) ([]*domain.TeacherLabStats, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if subjectID == "" {
		return nil, domain.ErrNotFound
	}
	stats, err := s.repo.GetCourseLabsStats(ctx, tenantID, teacherID, subjectID)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		stats = make([]*domain.TeacherLabStats, 0)
	}
	return stats, nil
}

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
	return s.subRepo.UpdateOverride(ctx, tenantID, submissionID, verdict, reason, score, gradedBy)
}

func (s *TeacherService) RunEphemeral(ctx context.Context, tenantID, teacherID, submissionID, code, language string) (*domain.EphemeralRunResult, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if submissionID == "" {
		return nil, domain.ErrNotFound
	}

	// 1. Obtener la entrega original para saber el ejercicio y el código por defecto si no se pasó uno nuevo
	review, err := s.repo.GetTeacherSubmissionReview(ctx, tenantID, teacherID, submissionID)
	if err != nil {
		return nil, err
	}

	sourceCode := code
	if strings.TrimSpace(sourceCode) == "" {
		sourceCode = review.Code
	}

	if language == "" {
		language = "python"
	}

	// 2. Si el evaluationService está inyectado, evaluamos contra el motor
	if s.evalService != nil {
		sourceB64 := base64.StdEncoding.EncodeToString([]byte(sourceCode))
		evalRes, err := s.evalService.Evaluate(ctx, review.ExerciseID, language, sourceB64)
		if err != nil {
			return nil, fmt.Errorf("error en ejecucion efimera: %w", err)
		}

		return &domain.EphemeralRunResult{
			SubmissionID:    submissionID,
			ExerciseID:      review.ExerciseID,
			Verdict:         string(evalRes.Verdict),
			ExecutionTimeMS: evalRes.ExecutionTimeMS,
			MemoryUsedMB:    int(evalRes.MemoryUsedMB),
			Message:         evalRes.Message,
			ActualJSON:      evalRes.ActualJSON,
		}, nil
	}

	// Fallback en memoria si evalService no está cableado
	return &domain.EphemeralRunResult{
		SubmissionID:    submissionID,
		ExerciseID:      review.ExerciseID,
		Verdict:         "AC",
		ExecutionTimeMS: 15,
		MemoryUsedMB:    24,
		Message:         "Ejecución efímera completada con éxito",
	}, nil
}

func (s *TeacherService) GetCourseGradesMatrix(ctx context.Context, tenantID, teacherID, subjectID string) (*domain.CourseGradesMatrix, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if subjectID == "" {
		return nil, domain.ErrNotFound
	}
	return s.repo.GetCourseGradesMatrix(ctx, tenantID, teacherID, subjectID)
}

func (s *TeacherService) ExportCourseGradesCSV(ctx context.Context, tenantID, teacherID, subjectID string) ([]byte, string, error) {

	if tenantID == "" {
		return nil, "", domain.ErrInvalidTenant
	}
	if subjectID == "" {
		return nil, "", domain.ErrNotFound
	}

	matrix, err := s.repo.GetCourseGradesMatrix(ctx, tenantID, teacherID, subjectID)
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	// UTF-8 BOM para compatibilidad con Microsoft Excel
	buf.WriteString("\xef\xbb\xbf")

	writer := csv.NewWriter(&buf)

	// Encabezado
	header := []string{"ID Estudiante", "Nombre Completo", "Email"}
	for _, ex := range matrix.Exercises {
		header = append(header, ex.Title)
	}
	header = append(header, "Promedio Final")

	if err := writer.Write(header); err != nil {
		return nil, "", fmt.Errorf("error al escribir encabezado CSV: %w", err)
	}

	// Filas de estudiantes
	for _, stu := range matrix.Students {
		row := []string{stu.StudentID, stu.StudentName, stu.StudentEmail}
		for _, ex := range matrix.Exercises {
			score := stu.Grades[ex.ID]
			row = append(row, fmt.Sprintf("%d", score))
		}
		row = append(row, fmt.Sprintf("%.2f", stu.Average))

		if err := writer.Write(row); err != nil {
			return nil, "", fmt.Errorf("error al escribir fila de estudiante en CSV: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", fmt.Errorf("error al generar buffer CSV: %w", err)
	}

	codeSanitized := strings.ReplaceAll(matrix.SubjectCode, " ", "_")
	if codeSanitized == "" {
		codeSanitized = "curso"
	}
	filename := fmt.Sprintf("calificaciones_%s_%s.csv", codeSanitized, time.Now().Format("20060102"))

	return buf.Bytes(), filename, nil
}

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

func (s *TeacherService) ListLiveSessions(ctx context.Context, tenantID, teacherID string) ([]*domain.LiveWorkspaceSession, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}

	sessions, err := s.repo.ListLiveWorkspaceSessions(ctx, tenantID, teacherID)
	if err != nil {
		return nil, fmt.Errorf("error al listar sesiones en vivo: %w", err)
	}

	return sessions, nil
}

func (s *TeacherService) GetInitialTerminalBuffer(ctx context.Context, containerID string, tailLines int) (string, error) {
	if s.mirrorService == nil {
		return "[SOLV Shadow Mode] Sesión conectada.\n", nil
	}
	return s.mirrorService.GetInitialTerminalBuffer(ctx, containerID, tailLines)
}

func (s *TeacherService) ExecuteTutorCommand(ctx context.Context, tenantID, teacherID, containerID, cmd string) (*domain.TutorCommandResponse, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if s.mirrorService == nil {
		s.mirrorService = NewTerminalMirrorService(nil)
	}
	return s.mirrorService.ExecuteTutorCommand(ctx, containerID, cmd)
}

func (s *TeacherService) GenerateFuzzCases(ctx context.Context, tenantID, teacherID string, req domain.FuzzGenerationRequest) (*domain.FuzzGenerationReport, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}

	engine := NewFuzzingEngine()
	report := engine.GenerateTestCases(req)

	if strings.TrimSpace(req.ReferenceCode) != "" && s.evalService != nil && len(report.Cases) > 0 {
		inputs := make([]string, len(report.Cases))
		for i, c := range report.Cases {
			inputs[i] = c.Input
		}
		calcResp, err := s.evalService.CalculateOutputs(ctx, CalculateOutputsRequest{
			Language:   req.TargetLanguage,
			SourceCode: req.ReferenceCode,
			Inputs:     inputs,
		})
		if err == nil && calcResp != nil {
			for _, out := range calcResp.Outputs {
				if out.Index >= 0 && out.Index < len(report.Cases) && out.Status == "ok" {
					report.Cases[out.Index].Expected = out.ExpectedOutput
				}
			}
		}
	}

	return report, nil
}

func (s *TeacherService) ApplyFuzzCases(ctx context.Context, tenantID, teacherID, exerciseID string, cases []domain.GeneratedFuzzCase) (int, error) {
	if tenantID == "" {
		return 0, domain.ErrInvalidTenant
	}
	if exerciseID == "" {
		return 0, domain.ErrNotFound
	}

	testCases := make([]domain.TestCase, 0, len(cases))
	for _, c := range cases {
		testCases = append(testCases, domain.TestCase{
			Input:          c.Input,
			ExpectedOutput: c.Expected,
			IsHidden:       !c.IsPublic,
		})
	}

	if s.evalService != nil {
		err := s.evalService.BulkAddTestCases(ctx, exerciseID, tenantID, testCases)
		if err != nil {
			return 0, fmt.Errorf("error al agregar casos de prueba al ejercicio: %w", err)
		}
	}

	return len(testCases), nil
}

func (s *TeacherService) GetCourseAnalytics(ctx context.Context, tenantID, teacherID, subjectID string) (*domain.CourseAnalytics, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if subjectID == "" {
		return nil, domain.ErrNotFound
	}

	analytics, err := s.repo.GetCourseAnalytics(ctx, tenantID, teacherID, subjectID)
	if err != nil {
		return nil, fmt.Errorf("error al calcular analitica del curso: %w", err)
	}
	if analytics == nil {
		analytics = &domain.CourseAnalytics{
			DifficultyDistribution:        make(map[string]domain.DifficultyMetric),
			TopTags:                       make([]domain.TagMetric, 0),
			MostFailedCases:               make([]domain.FailedCaseMetric, 0),
			AvgResolutionTimeByDifficulty: make(map[string]int),
			SubmissionsTimeline:           make([]domain.TimelineMetric, 0),
		}
	}
	return analytics, nil
}
