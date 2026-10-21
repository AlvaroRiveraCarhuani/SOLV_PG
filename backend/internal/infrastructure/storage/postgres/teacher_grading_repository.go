package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"solv-backend/internal/core/domain"
	"time"
)

func (r *PostgresTeacherRepository) ListCourseSubmissions(ctx context.Context, tenantID, teacherID, subjectID, exerciseID, verdict string) ([]*domain.SubmissionQueueItem, error) {
	// Verificar existencia de la materia y aislamiento docente
	var exists bool
	var err error
	if teacherID != "" {
		checkQuery := `SELECT EXISTS (SELECT 1 FROM subjects WHERE id = $1 AND tenant_id = $2 AND teacher_id = $3)`
		err = r.db.GetContext(ctx, &exists, checkQuery, subjectID, tenantID, teacherID)
	} else {
		checkQuery := `SELECT EXISTS (SELECT 1 FROM subjects WHERE id = $1 AND tenant_id = $2)`
		err = r.db.GetContext(ctx, &exists, checkQuery, subjectID, tenantID)
	}
	if err != nil || !exists {
		return nil, domain.ErrNotFound
	}

	query := `
		SELECT sub.id, sub.exercise_id, ex.title AS exercise_title, sub.student_id,
		       COALESCE(u.first_name || ' ' || u.last_name, 'Estudiante') AS student_name,
		       COALESCE(u.email, '') AS student_email,
		       sub.verdict, sub.score, COALESCE(sub.manual_override, FALSE) AS manual_override,
		       sub.execution_time_ms, sub.memory_used_mb, sub.submitted_at,
		       (SELECT COUNT(*) FROM submission_comments sc WHERE sc.submission_id = sub.id) AS comments_count
		FROM submissions sub
		JOIN exercises ex ON sub.exercise_id = ex.id
		JOIN subjects s ON ex.subject_id = s.id
		LEFT JOIN users u ON sub.student_id = u.id
		WHERE s.id = $1 AND s.tenant_id = $2
		  AND ($3 = '' OR sub.exercise_id = $3::uuid)
		  AND ($4 = '' OR sub.verdict = $4)
		ORDER BY sub.submitted_at DESC
	`
	type itemRow struct {
		ID              string        `db:"id"`
		ExerciseID      string        `db:"exercise_id"`
		ExerciseTitle   string        `db:"exercise_title"`
		StudentID       string        `db:"student_id"`
		StudentName     string        `db:"student_name"`
		StudentEmail    string        `db:"student_email"`
		Verdict         string        `db:"verdict"`
		Score           sql.NullInt64 `db:"score"`
		ManualOverride  bool          `db:"manual_override"`
		ExecutionTimeMS int           `db:"execution_time_ms"`
		MemoryUsedMB    int           `db:"memory_used_mb"`
		SubmittedAt     time.Time     `db:"submitted_at"`
		CommentsCount   int           `db:"comments_count"`
	}

	var rows []itemRow
	if err := r.db.SelectContext(ctx, &rows, query, subjectID, tenantID, exerciseID, verdict); err != nil {
		return nil, fmt.Errorf("failed to list submissions queue: %w", err)
	}

	result := make([]*domain.SubmissionQueueItem, 0, len(rows))
	for _, row := range rows {
		item := &domain.SubmissionQueueItem{
			ID:              row.ID,
			ExerciseID:      row.ExerciseID,
			ExerciseTitle:   row.ExerciseTitle,
			StudentID:       row.StudentID,
			StudentName:     row.StudentName,
			StudentEmail:    row.StudentEmail,
			Verdict:         row.Verdict,
			ManualOverride:  row.ManualOverride,
			ExecutionTimeMS: row.ExecutionTimeMS,
			MemoryUsedMB:    row.MemoryUsedMB,
			SubmittedAt:     row.SubmittedAt,
			CommentsCount:   row.CommentsCount,
		}
		if row.Score.Valid {
			s := int(row.Score.Int64)
			item.Score = &s
		}
		result = append(result, item)
	}

	return result, nil
}

func (r *PostgresTeacherRepository) GetTeacherSubmissionReview(ctx context.Context, tenantID, teacherID, submissionID string) (*domain.TeacherSubmissionReviewDTO, error) {
	type subDetailRow struct {
		ID                 string         `db:"id"`
		ExerciseID         string         `db:"exercise_id"`
		ExerciseTitle      string         `db:"exercise_title"`
		ExerciseConfig     []byte         `db:"exercise_config"`
		SubjectID          string         `db:"subject_id"`
		SubjectName        string         `db:"subject_name"`
		StudentID          string         `db:"student_id"`
		StudentName        string         `db:"student_name"`
		StudentEmail       string         `db:"student_email"`
		Code               string         `db:"code"`
		Verdict            string         `db:"verdict"`
		Score              sql.NullInt64  `db:"score"`
		ManualOverride     sql.NullBool   `db:"manual_override"`
		OverrideReason     sql.NullString `db:"override_reason"`
		GradedBy           sql.NullString `db:"graded_by"`
		GradedByName       sql.NullString `db:"graded_by_name"`
		ExecutionTimeMS    int            `db:"execution_time_ms"`
		MemoryUsedMB       int            `db:"memory_used_mb"`
		ASTResult          []byte         `db:"ast_result"`
		GeneratedCases     []byte         `db:"generated_cases"`
		ExpectedComplexity sql.NullString `db:"expected_complexity"`
		ComplexityAnalysis []byte         `db:"complexity_analysis"`
		SubmittedAt        time.Time      `db:"submitted_at"`
	}

	query := `
		SELECT sub.id, sub.exercise_id, ex.title AS exercise_title, ex.config AS exercise_config,
		       s.id AS subject_id, s.name AS subject_name,
		       sub.student_id, COALESCE(u.first_name || ' ' || u.last_name, 'Estudiante') AS student_name,
		       COALESCE(u.email, '') AS student_email,
		       sub.code, sub.verdict, sub.score, sub.manual_override, sub.override_reason,
		       sub.graded_by, COALESCE(g.first_name || ' ' || g.last_name, '') AS graded_by_name,
		       sub.execution_time_ms, sub.memory_used_mb, sub.ast_result, sub.generated_cases,
		       ex.expected_complexity, sub.complexity_analysis, sub.submitted_at
		FROM submissions sub
		JOIN exercises ex ON sub.exercise_id = ex.id
		JOIN subjects s ON ex.subject_id = s.id
		LEFT JOIN users u ON sub.student_id = u.id
		LEFT JOIN users g ON sub.graded_by = g.id
		WHERE sub.id = $1 AND sub.tenant_id = $2
	`
	var row subDetailRow
	if err := r.db.GetContext(ctx, &row, query, submissionID, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get submission review: %w", err)
	}

	var ca *domain.ComplexityAnalysis
	if len(row.ComplexityAnalysis) > 0 && string(row.ComplexityAnalysis) != "null" {
		var parsed domain.ComplexityAnalysis
		if err := json.Unmarshal(row.ComplexityAnalysis, &parsed); err == nil {
			ca = &parsed
		}
	}

	expComp := ""
	if row.ExpectedComplexity.Valid {
		expComp = row.ExpectedComplexity.String
	}

	review := &domain.TeacherSubmissionReviewDTO{
		ID:                 row.ID,
		ExerciseID:         row.ExerciseID,
		ExerciseTitle:      row.ExerciseTitle,
		SubjectID:          row.SubjectID,
		SubjectName:        row.SubjectName,
		StudentID:          row.StudentID,
		StudentName:        row.StudentName,
		StudentEmail:       row.StudentEmail,
		Code:               row.Code,
		Verdict:            row.Verdict,
		ManualOverride:     row.ManualOverride.Valid && row.ManualOverride.Bool,
		ExecutionTimeMS:    row.ExecutionTimeMS,
		MemoryUsedMB:       row.MemoryUsedMB,
		ASTResult:          row.ASTResult,
		GeneratedCases:     row.GeneratedCases,
		ExpectedComplexity: expComp,
		ComplexityAnalysis: ca,
		SubmittedAt:        row.SubmittedAt,
		TestCases:          make([]domain.TestCaseReview, 0),
		Comments:           make([]domain.SubmissionComment, 0),
	}

	if row.Score.Valid {
		s := int(row.Score.Int64)
		review.Score = &s
	}
	if row.OverrideReason.Valid {
		review.OverrideReason = row.OverrideReason.String
	}
	if row.GradedBy.Valid {
		gb := row.GradedBy.String
		review.GradedBy = &gb
	}
	if row.GradedByName.Valid {
		review.GradedByName = row.GradedByName.String
	}

	// Si existen casos generados snapshot (examen con semilla), usarlos prioritariamente
	if len(row.GeneratedCases) > 0 {
		var genCases []domain.TestCase
		if err := json.Unmarshal(row.GeneratedCases, &genCases); err == nil && len(genCases) > 0 {
			for _, gc := range genCases {
				review.TestCases = append(review.TestCases, domain.TestCaseReview{
					Input:          gc.Input,
					ExpectedOutput: gc.ExpectedOutput,
					IsHidden:       gc.Visibility == domain.TestCaseVisibilityHidden || gc.IsHidden,
					Passed:         (review.Verdict == "AC"),
				})
			}
		}
	}

	// Si no hubo casos generados o test_cases sigue vacío, desenmascarar desde exercise_config
	if len(review.TestCases) == 0 && len(row.ExerciseConfig) > 0 {
		var cfg map[string]interface{}
		if err := json.Unmarshal(row.ExerciseConfig, &cfg); err == nil {
			var tcRaw []interface{}
			if algo, ok := cfg["algorithm"].(map[string]interface{}); ok {
				if cases, ok := algo["test_cases"].([]interface{}); ok {
					tcRaw = cases
				}
			} else if cases, ok := cfg["test_cases"].([]interface{}); ok {
				tcRaw = cases
			}

			for _, c := range tcRaw {
				if cMap, ok := c.(map[string]interface{}); ok {
					tc := domain.TestCaseReview{
						Input:          fmt.Sprintf("%v", cMap["input"]),
						ExpectedOutput: fmt.Sprintf("%v", cMap["expected_output"]),
						Passed:         (review.Verdict == "AC"),
					}
					if isHidden, ok := cMap["is_hidden"].(bool); ok {
						tc.IsHidden = isHidden
					}
					review.TestCases = append(review.TestCases, tc)
				}
			}
		}
	}

	// SpeedGrader navigation: Buscar la entrega anterior y siguiente del mismo ejercicio
	var prevID, nextID string
	_ = r.db.GetContext(ctx, &prevID, `
		SELECT id FROM submissions
		WHERE exercise_id = $1 AND tenant_id = $2 AND submitted_at < $3
		ORDER BY submitted_at DESC
		LIMIT 1
	`, row.ExerciseID, tenantID, row.SubmittedAt)

	_ = r.db.GetContext(ctx, &nextID, `
		SELECT id FROM submissions
		WHERE exercise_id = $1 AND tenant_id = $2 AND submitted_at > $3
		ORDER BY submitted_at ASC
		LIMIT 1
	`, row.ExerciseID, tenantID, row.SubmittedAt)

	if prevID != "" {
		review.PrevSubmissionID = &prevID
	}
	if nextID != "" {
		review.NextSubmissionID = &nextID
	}

	// Obtener comentarios in-line
	comments, err := r.GetCommentsBySubmission(ctx, tenantID, submissionID)
	if err == nil && comments != nil {
		review.Comments = make([]domain.SubmissionComment, len(comments))
		for i, c := range comments {
			review.Comments[i] = *c
		}
	}

	return review, nil
}

func (r *PostgresTeacherRepository) AddComment(ctx context.Context, comment *domain.SubmissionComment) error {
	if comment.ID == "" {
		comment.ID = uuid.NewString()
	}
	query := `
		INSERT INTO submission_comments (id, tenant_id, submission_id, author_id, line_number, comment, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		comment.ID,
		comment.TenantID,
		comment.SubmissionID,
		comment.AuthorID,
		comment.LineNumber,
		comment.Comment,
	)
	if err != nil {
		return fmt.Errorf("failed to insert submission comment: %w", err)
	}
	return nil
}

func (r *PostgresTeacherRepository) GetCommentsBySubmission(ctx context.Context, tenantID, submissionID string) ([]*domain.SubmissionComment, error) {
	query := `
		SELECT sc.id, sc.tenant_id, sc.submission_id, sc.author_id,
		       COALESCE(u.first_name || ' ' || u.last_name, 'Docente') AS author_name,
		       sc.line_number, sc.comment, sc.created_at
		FROM submission_comments sc
		LEFT JOIN users u ON sc.author_id = u.id
		WHERE sc.submission_id = $1 AND sc.tenant_id = $2
		ORDER BY sc.line_number ASC, sc.created_at ASC
	`
	type cRow struct {
		ID           string    `db:"id"`
		TenantID     string    `db:"tenant_id"`
		SubmissionID string    `db:"submission_id"`
		AuthorID     string    `db:"author_id"`
		AuthorName   string    `db:"author_name"`
		LineNumber   int       `db:"line_number"`
		Comment      string    `db:"comment"`
		CreatedAt    time.Time `db:"created_at"`
	}

	var rows []cRow
	if err := r.db.SelectContext(ctx, &rows, query, submissionID, tenantID); err != nil {
		return nil, fmt.Errorf("failed to list submission comments: %w", err)
	}

	result := make([]*domain.SubmissionComment, 0, len(rows))
	for _, r := range rows {
		result = append(result, &domain.SubmissionComment{
			ID:           r.ID,
			TenantID:     r.TenantID,
			SubmissionID: r.SubmissionID,
			AuthorID:     r.AuthorID,
			AuthorName:   r.AuthorName,
			LineNumber:   r.LineNumber,
			Comment:      r.Comment,
			CreatedAt:    r.CreatedAt,
		})
	}
	return result, nil
}
