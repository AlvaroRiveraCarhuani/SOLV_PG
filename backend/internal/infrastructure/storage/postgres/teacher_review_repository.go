package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"solv-backend/internal/core/domain"
)

// ListPendingReviewGroups devuelve la cola real de corrección agrupada por
// práctica. Fuente de verdad: submissions.review_status, no el veredicto.
func (r *PostgresTeacherRepository) ListPendingReviewGroups(ctx context.Context, tenantID, teacherID, subjectID string) ([]*domain.ReviewQueueGroup, error) {
	type groupRow struct {
		ExerciseID        string       `db:"exercise_id"`
		ExerciseTitle     string       `db:"exercise_title"`
		SubjectID         string       `db:"subject_id"`
		SubjectName       string       `db:"subject_name"`
		EnvironmentType   string       `db:"environment_type"`
		Pending           int          `db:"pending"`
		Total             int          `db:"total"`
		OldestSubmittedAt sql.NullTime `db:"oldest_submitted_at"`
	}

	query := `
		SELECT ex.id AS exercise_id,
		       ex.title AS exercise_title,
		       s.id AS subject_id,
		       s.name AS subject_name,
		       COALESCE(ex.environment_type, '') AS environment_type,
		       COUNT(sub.id)::int AS pending,
		       COALESCE((
		           SELECT COUNT(*) FROM enrollments e
		           WHERE e.subject_id = s.id AND e.tenant_id = s.tenant_id
		       ), 0)::int AS total,
		       MIN(sub.submitted_at) AS oldest_submitted_at
		FROM submissions sub
		JOIN exercises ex ON ex.id = sub.exercise_id
		JOIN subjects s ON s.id = ex.subject_id
		WHERE sub.tenant_id = $1
		  AND s.tenant_id = $1
		  AND sub.review_status IN ('pending', 'in_review')
		  AND ($2 = '' OR s.teacher_id::text = $2)
		  AND ($3 = '' OR s.id::text = $3)
		GROUP BY ex.id, ex.title, s.id, s.name, ex.environment_type
		ORDER BY MIN(sub.submitted_at) ASC
	`

	var rows []groupRow
	if err := r.db.SelectContext(ctx, &rows, query, tenantID, teacherID, subjectID); err != nil {
		return nil, fmt.Errorf("failed to list pending review groups: %w", err)
	}

	groups := make([]*domain.ReviewQueueGroup, 0, len(rows))
	for _, row := range rows {
		group := &domain.ReviewQueueGroup{
			ExerciseID:      row.ExerciseID,
			ExerciseTitle:   row.ExerciseTitle,
			SubjectID:       row.SubjectID,
			SubjectName:     row.SubjectName,
			EnvironmentType: row.EnvironmentType,
			Pending:         row.Pending,
			Total:           row.Total,
		}
		if row.OldestSubmittedAt.Valid {
			group.OldestSubmittedAt = row.OldestSubmittedAt.Time
		}
		groups = append(groups, group)
	}
	return groups, nil
}

// GetSubmissionReviewState lee el estado actual de revisión comprobando que la
// entrega pertenece a una materia del docente.
func (r *PostgresTeacherRepository) GetSubmissionReviewState(ctx context.Context, tenantID, teacherID, submissionID string) (domain.ReviewStatus, error) {
	const query = `
		SELECT sub.review_status
		FROM submissions sub
		JOIN exercises ex ON ex.id = sub.exercise_id
		JOIN subjects s ON s.id = ex.subject_id
		WHERE sub.id = $1 AND sub.tenant_id = $2
		  AND ($3 = '' OR s.teacher_id::text = $3)
	`
	var status string
	if err := r.db.GetContext(ctx, &status, query, submissionID, tenantID, teacherID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("failed to load submission review state: %w", err)
	}
	return domain.ReviewStatus(status), nil
}

// UpdateSubmissionReview persiste el nuevo estado de revisión. La transición se
// valida antes en el servicio; aquí se exige que el estado no haya cambiado
// entre la lectura y la escritura para que dos docentes no pisen la corrección.
func (r *PostgresTeacherRepository) UpdateSubmissionReview(
	ctx context.Context,
	tenantID, teacherID, submissionID string,
	current domain.ReviewStatus,
	next domain.ReviewStatus,
	feedback string,
	score *int,
	actorID string,
) error {
	const query = `
		UPDATE submissions sub
		SET review_status = $4,
		    reviewed_at = NOW(),
		    reviewed_by = NULLIF($5, '')::uuid,
		    feedback_summary = $6,
		    returned_at = CASE WHEN $4 = 'returned' THEN NOW() ELSE sub.returned_at END,
		    score = COALESCE($7, sub.score)
		FROM exercises ex
		JOIN subjects s ON s.id = ex.subject_id
		WHERE sub.exercise_id = ex.id
		  AND sub.id = $1
		  AND sub.tenant_id = $2
		  AND ($3 = '' OR s.teacher_id::text = $3)
		  AND sub.review_status = $8
		RETURNING sub.id
	`
	var updatedID string
	err := r.db.GetContext(ctx, &updatedID, query, submissionID, tenantID, teacherID, string(next), actorID, feedback, score, string(current))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrInvalidReviewTransition
		}
		return fmt.Errorf("failed to update submission review: %w", err)
	}
	return nil
}

// RecordGradeFromSubmission refleja en la celda de nota la entrega recién
// calificada, sin pisar una selección manual del docente.
func (r *PostgresTeacherRepository) RecordGradeFromSubmission(ctx context.Context, tenantID, submissionID string) error {
	const query = `
		INSERT INTO exercise_grades (
		    student_id, exercise_id, subject_id, tenant_id,
		    selected_submission_id, score, source, updated_at
		)
		SELECT sub.student_id,
		       sub.exercise_id,
		       ex.subject_id,
		       sub.tenant_id,
		       sub.id,
		       COALESCE(sub.score, CASE WHEN sub.verdict = 'AC' OR COALESCE(sub.manual_override, FALSE) THEN 100 ELSE NULL END),
		       'auto',
		       NOW()
		FROM submissions sub
		JOIN exercises ex ON ex.id = sub.exercise_id
		WHERE sub.id = $1 AND sub.tenant_id = $2 AND ex.subject_id IS NOT NULL
		ON CONFLICT (student_id, exercise_id) DO UPDATE
		SET selected_submission_id = EXCLUDED.selected_submission_id,
		    score = EXCLUDED.score,
		    updated_at = NOW()
		WHERE exercise_grades.source <> 'manual'
	`
	if _, err := r.db.ExecContext(ctx, query, submissionID, tenantID); err != nil {
		return fmt.Errorf("failed to record grade from submission: %w", err)
	}
	return nil
}

// SetExerciseGrade fija la entrega que cuenta para la nota de un alumno en una
// práctica. Devuelve la nota resultante para que la API responda la celda.
func (r *PostgresTeacherRepository) SetExerciseGrade(
	ctx context.Context,
	tenantID, teacherID, studentID, exerciseID, selectedSubmissionID string,
	requestedScore *int,
	reason string,
	actorID string,
) (*int, error) {
	const subjectQuery = `
		SELECT ex.subject_id
		FROM exercises ex
		JOIN subjects s ON s.id = ex.subject_id
		WHERE ex.id = $1 AND ex.tenant_id = $2
		  AND ($3 = '' OR s.teacher_id::text = $3)
	`
	var subjectID string
	if err := r.db.GetContext(ctx, &subjectID, subjectQuery, exerciseID, tenantID, teacherID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to load exercise subject: %w", err)
	}

	const submissionQuery = `
		SELECT sub.verdict, sub.score, COALESCE(sub.manual_override, FALSE) AS manual_override
		FROM submissions sub
		WHERE sub.id = $1
		  AND sub.student_id = $2
		  AND sub.exercise_id = $3
		  AND sub.tenant_id = $4
	`
	type submissionRow struct {
		Verdict        string        `db:"verdict"`
		Score          sql.NullInt64 `db:"score"`
		ManualOverride bool          `db:"manual_override"`
	}
	var selected submissionRow
	if err := r.db.GetContext(ctx, &selected, submissionQuery, selectedSubmissionID, studentID, exerciseID, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubmissionNotInSubject
		}
		return nil, fmt.Errorf("failed to load selected submission: %w", err)
	}

	var storedScore *int
	if selected.Score.Valid {
		score := int(selected.Score.Int64)
		storedScore = &score
	}
	baseScore := domain.ResolvedSubmissionScore(storedScore, selected.Verdict, selected.ManualOverride)

	finalScore := requestedScore
	if finalScore == nil {
		finalScore = baseScore
	}
	if domain.NeedsGradeReason(baseScore, requestedScore) && len(strings.TrimSpace(reason)) < 10 {
		return nil, domain.ErrGradeReasonRequired
	}

	source := "auto"
	overriddenBy := ""
	if strings.TrimSpace(reason) != "" {
		source = "manual"
		overriddenBy = actorID
	}

	const upsertQuery = `
		INSERT INTO exercise_grades (
		    student_id, exercise_id, subject_id, tenant_id,
		    selected_submission_id, score, source, overridden_by, reason, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, '')::uuid, $9, NOW())
		ON CONFLICT (student_id, exercise_id) DO UPDATE
		SET selected_submission_id = EXCLUDED.selected_submission_id,
		    score = EXCLUDED.score,
		    source = EXCLUDED.source,
		    overridden_by = EXCLUDED.overridden_by,
		    reason = EXCLUDED.reason,
		    updated_at = NOW()
	`
	if _, err := r.db.ExecContext(ctx, upsertQuery,
		studentID, exerciseID, subjectID, tenantID, selectedSubmissionID,
		finalScore, source, overriddenBy, strings.TrimSpace(reason),
	); err != nil {
		return nil, fmt.Errorf("failed to upsert exercise grade: %w", err)
	}

	return finalScore, nil
}

// GetExerciseGradeCell devuelve la celda actual, para responder al docente qué
// entrega quedó seleccionada y con qué nota.
func (r *PostgresTeacherRepository) GetExerciseGradeCell(ctx context.Context, tenantID, studentID, exerciseID string) (*domain.ExerciseGradeCell, error) {
	type cellRow struct {
		SelectedSubmissionID sql.NullString `db:"selected_submission_id"`
		Score                sql.NullInt64  `db:"score"`
		Source               string         `db:"source"`
		Reason               string         `db:"reason"`
	}
	const query = `
		SELECT selected_submission_id, score, source, reason
		FROM exercise_grades
		WHERE student_id = $1 AND exercise_id = $2 AND tenant_id = $3
	`
	var row cellRow
	if err := r.db.GetContext(ctx, &row, query, studentID, exerciseID, tenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrGradeCellNotFound
		}
		return nil, fmt.Errorf("failed to load grade cell: %w", err)
	}

	cell := &domain.ExerciseGradeCell{Source: row.Source, Reason: row.Reason}
	if row.SelectedSubmissionID.Valid {
		submissionID := row.SelectedSubmissionID.String
		cell.SelectedSubmissionID = &submissionID
	}
	if row.Score.Valid {
		score := int(row.Score.Int64)
		cell.Score = &score
	}
	return cell, nil
}
