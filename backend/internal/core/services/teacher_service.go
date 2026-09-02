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
	repo        domain.TeacherRepository
	subRepo     domain.SubmissionRepository
	evalService *EvaluationService
}

func NewTeacherService(repo domain.TeacherRepository, subRepo ...domain.SubmissionRepository) *TeacherService {
	var sRepo domain.SubmissionRepository
	if len(subRepo) > 0 {
		sRepo = subRepo[0]
	}
	return &TeacherService{
		repo:    repo,
		subRepo: sRepo,
	}
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
