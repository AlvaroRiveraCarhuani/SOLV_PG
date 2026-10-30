package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"
)

// teacherScopeClause agrega los filtros opcionales de docente y período sobre el
// alias `s` (subjects). startIdx es el número del primer placeholder libre.
func teacherScopeClause(teacherID, periodID string, startIdx int) (string, []any) {
	clause := ""
	args := make([]any, 0, 2)
	idx := startIdx
	if teacherID != "" {
		clause += fmt.Sprintf(" AND s.teacher_id = $%d", idx)
		args = append(args, teacherID)
		idx++
	}
	if periodID != "" {
		clause += fmt.Sprintf(" AND s.academic_period_id = $%d::uuid", idx)
		args = append(args, periodID)
	}
	return clause, args
}

// GetTeacherOverview resuelve el panel docente en un número fijo de consultas,
// sin recorrer las materias en un bucle. Los conteos se calculan aquí y no en el
// cliente.
func (r *PostgresTeacherRepository) GetTeacherOverview(ctx context.Context, tenantID, teacherID, periodID string) (*domain.TeacherOverview, error) {
	overview := &domain.TeacherOverview{
		Subjects: make([]domain.TeacherOverviewSubject, 0),
		Signals:  make([]domain.TeacherOverviewSignal, 0),
	}

	// 1. Período académico que enmarca la pantalla.
	overview.Period = r.getOverviewPeriod(ctx, tenantID, periodID)

	// 2. Materias con sus conteos.
	type subjectRow struct {
		ID                string         `db:"id"`
		Code              string         `db:"code"`
		Name              string         `db:"name"`
		Enrolled          int            `db:"enrolled"`
		OpenEnvironments  int            `db:"open_environments"`
		PendingReview     int            `db:"pending_review"`
		WithoutSubmission sql.NullInt64  `db:"without_submission"`
		NextDueAt         sql.NullTime   `db:"next_due_at"`
		NextDueTitle      sql.NullString `db:"next_due_title"`
	}

	scope, scopeArgs := teacherScopeClause(teacherID, periodID, 2)
	subjectsQuery := `
		SELECT
			s.id,
			s.code,
			s.name,
			(SELECT COUNT(DISTINCT en.student_id) FROM enrollments en
			   WHERE en.subject_id = s.id AND en.tenant_id = $1) AS enrolled,
			(SELECT COUNT(*) FROM workspaces w
			   WHERE w.subject_id = s.id AND w.tenant_id = $1
			     AND w.status IN ('running', 'active', 'pending')) AS open_environments,
			(SELECT COUNT(*) FROM submissions sub
			   JOIN exercises ex ON sub.exercise_id = ex.id
			   WHERE ex.subject_id = s.id AND sub.tenant_id = $1
			     AND sub.review_status IN ('pending', 'in_review')) AS pending_review,
			nex.due_date AS next_due_at,
			nex.title AS next_due_title,
			CASE WHEN nex.next_exercise_id IS NULL THEN NULL ELSE
				(SELECT COUNT(DISTINCT en2.student_id) FROM enrollments en2
				   WHERE en2.subject_id = s.id AND en2.tenant_id = $1
				     AND NOT EXISTS (
				         SELECT 1 FROM submissions sub2
				         WHERE sub2.exercise_id = nex.next_exercise_id
				           AND sub2.student_id = en2.student_id
				           AND sub2.tenant_id = $1
				     ))
			END AS without_submission
		FROM subjects s
		LEFT JOIN LATERAL (
			SELECT ex.id AS next_exercise_id, ex.title, ex.due_date
			FROM exercises ex
			WHERE ex.subject_id = s.id AND ex.tenant_id = $1
			  AND ex.due_date > NOW()
			  AND COALESCE(ex.status, 'draft') = 'published'
			ORDER BY ex.due_date ASC
			LIMIT 1
		) nex ON TRUE
		WHERE s.tenant_id = $1` + scope + `
		ORDER BY s.code ASC
	`

	subjectArgs := append([]any{tenantID}, scopeArgs...)
	var rows []subjectRow
	if err := r.db.SelectContext(ctx, &rows, subjectsQuery, subjectArgs...); err != nil {
		return nil, fmt.Errorf("failed to build teacher overview subjects: %w", err)
	}
	for _, row := range rows {
		item := domain.TeacherOverviewSubject{
			ID:               row.ID,
			Code:             row.Code,
			Name:             row.Name,
			Enrolled:         row.Enrolled,
			OpenEnvironments: row.OpenEnvironments,
			PendingReview:    row.PendingReview,
		}
		if row.WithoutSubmission.Valid {
			value := int(row.WithoutSubmission.Int64)
			item.WithoutSubmission = &value
		}
		if row.NextDueAt.Valid {
			due := row.NextDueAt.Time
			item.NextDueAt = &due
			// El vencimiento más cercano se resuelve aquí, no en la vista.
			if overview.NextDue == nil || due.Before(overview.NextDue.DueAt) {
				overview.NextDue = &domain.TeacherOverviewNextDue{
					SubjectID:     row.ID,
					SubjectName:   row.Name,
					ExerciseTitle: row.NextDueTitle.String,
					DueAt:         due,
				}
			}
		}
		overview.Subjects = append(overview.Subjects, item)
	}

	// 3. Cola de corrección: una sola consulta agregada.
	queueScope, queueScopeArgs := teacherScopeClause(teacherID, periodID, 2)
	queueQuery := `
		SELECT
			COUNT(*) AS pending,
			COUNT(DISTINCT sub.exercise_id) AS practices,
			MIN(sub.submitted_at) AS oldest
		FROM submissions sub
		JOIN exercises ex ON sub.exercise_id = ex.id
		JOIN subjects s ON ex.subject_id = s.id
		WHERE sub.tenant_id = $1
		  AND sub.review_status IN ('pending', 'in_review')` + queueScope
	queueArgs := append([]any{tenantID}, queueScopeArgs...)

	var queueRow struct {
		Pending   int          `db:"pending"`
		Practices int          `db:"practices"`
		Oldest    sql.NullTime `db:"oldest"`
	}
	if err := r.db.GetContext(ctx, &queueRow, queueQuery, queueArgs...); err != nil {
		return nil, fmt.Errorf("failed to build teacher overview queue: %w", err)
	}
	overview.Queue = domain.TeacherOverviewQueue{
		Pending:   queueRow.Pending,
		Practices: queueRow.Practices,
	}
	if queueRow.Oldest.Valid {
		oldest := queueRow.Oldest.Time
		overview.Queue.OldestSubmittedAt = &oldest
	}

	// 4. Entornos abiertos, con la misma definición que usa la tabla de materias:
	// contenedor en estado running, active o pending.
	envScope, envScopeArgs := teacherScopeClause(teacherID, periodID, 2)
	envQuery := `
		SELECT COUNT(*)
		FROM workspaces w
		JOIN subjects s ON w.subject_id = s.id
		WHERE w.tenant_id = $1
		  AND w.status IN ('running', 'active', 'pending')` + envScope
	envArgs := append([]any{tenantID}, envScopeArgs...)
	if err := r.db.GetContext(ctx, &overview.Environments.OpenNow, envQuery, envArgs...); err != nil {
		return nil, fmt.Errorf("failed to count open environments: %w", err)
	}

	return overview, nil
}

// getOverviewPeriod devuelve el período pedido o, si no se pide ninguno, el
// activo. Un período inexistente deja el campo nulo en lugar de fallar la carga.
func (r *PostgresTeacherRepository) getOverviewPeriod(ctx context.Context, tenantID, periodID string) *domain.TeacherOverviewPeriod {
	period := &domain.TeacherOverviewPeriod{}
	var err error
	if periodID != "" {
		err = r.db.GetContext(ctx, period, `
			SELECT id, name, code, is_active
			FROM academic_periods
			WHERE id = $1::uuid AND tenant_id = $2
		`, periodID, tenantID)
	} else {
		err = r.db.GetContext(ctx, period, `
			SELECT id, name, code, is_active
			FROM academic_periods
			WHERE tenant_id = $1 AND is_active = TRUE
			ORDER BY start_date DESC
			LIMIT 1
		`, tenantID)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return nil
	}
	return period
}
