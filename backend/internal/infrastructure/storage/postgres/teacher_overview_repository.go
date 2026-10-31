package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

// overviewTimeZone devuelve la zona que define "hoy" y "mañana" para el panel.
// Se configura con APP_TIMEZONE; por defecto, la zona de Bolivia.
func overviewTimeZone() string {
	if tz := strings.TrimSpace(os.Getenv("APP_TIMEZONE")); tz != "" {
		if _, err := time.LoadLocation(tz); err == nil {
			return tz
		}
	}
	return "America/La_Paz"
}

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
		DueSoon:  make([]domain.TeacherOverviewDue, 0),
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
			ID:            row.ID,
			Code:          row.Code,
			Name:          row.Name,
			Enrolled:      row.Enrolled,
			PendingReview: row.PendingReview,
		}
		if row.WithoutSubmission.Valid {
			value := int(row.WithoutSubmission.Int64)
			item.WithoutSubmission = &value
		}
		if row.NextDueAt.Valid {
			due := row.NextDueAt.Time
			item.NextDueAt = &due
			// El título viaja con su fecha: la tarjeta nombra la práctica que vence.
			item.NextDueTitle = row.NextDueTitle.String
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

	// 4. Vencimientos próximos: todas las prácticas publicadas aún no vencidas,
	// igual que la próxima práctica de cada materia en el canvas. El total viaja
	// antes del límite, para que la vista declare el corte sin contar filas.
	dueScope, dueScopeArgs := teacherScopeClause(teacherID, periodID, 2)
	// La zona define "hoy" y "mañana" para el panel, no el reloj de la base.
	tzParam := fmt.Sprintf("$%d", 2+len(dueScopeArgs))
	dueQuery := `
		SELECT d.* FROM (
		SELECT
			ex.subject_id,
			s.code AS subject_code,
			s.name AS subject_name,
			ex.id AS exercise_id,
			ex.title AS exercise_title,
			ex.due_date AS due_at,
			COUNT(*) OVER () AS total,
			CASE
			  WHEN ex.due_date IS NULL THEN 'none'
			  WHEN ex.due_date <= NOW() THEN 'overdue'
			  WHEN (ex.due_date AT TIME ZONE ` + tzParam + `)::date = (NOW() AT TIME ZONE ` + tzParam + `)::date THEN 'today'
			  WHEN ex.due_date <= NOW() + INTERVAL '7 days' THEN 'soon'
			  ELSE 'later'
			END AS urgency,
			(SELECT COUNT(*) FROM submissions sub
			   WHERE sub.exercise_id = ex.id AND sub.tenant_id = $1
			     AND sub.review_status IN ('pending', 'in_review')) AS pending_review
		FROM exercises ex
		JOIN subjects s ON s.id = ex.subject_id
		WHERE ex.tenant_id = $1
		  AND s.tenant_id = $1
		  AND COALESCE(ex.status, 'draft') = 'published'
		  AND (
		    ex.due_date > NOW()
		    OR EXISTS (
		      SELECT 1 FROM submissions sub
		      WHERE sub.exercise_id = ex.id AND sub.tenant_id = $1
		        AND sub.review_status IN ('pending', 'in_review')
		    )
		  )` + dueScope + `
		) d
		ORDER BY
		  CASE d.urgency WHEN 'overdue' THEN 0 WHEN 'today' THEN 1 WHEN 'soon' THEN 2 WHEN 'later' THEN 3 ELSE 4 END ASC,
		  d.due_at ASC NULLS LAST,
		  d.subject_code ASC
		LIMIT 50
	`
	dueArgs := append(append([]any{tenantID}, dueScopeArgs...), overviewTimeZone())
	var dueRows []struct {
		SubjectID     string       `db:"subject_id"`
		SubjectCode   string       `db:"subject_code"`
		SubjectName   string       `db:"subject_name"`
		ExerciseID    string       `db:"exercise_id"`
		ExerciseTitle string       `db:"exercise_title"`
		DueAt         sql.NullTime `db:"due_at"`
		Total         int          `db:"total"`
		Urgency       string       `db:"urgency"`
		PendingReview int          `db:"pending_review"`
	}
	if err := r.db.SelectContext(ctx, &dueRows, dueQuery, dueArgs...); err != nil {
		return nil, fmt.Errorf("failed to build teacher overview due list: %w", err)
	}
	for _, row := range dueRows {
		due := domain.TeacherOverviewDue{
			SubjectID:     row.SubjectID,
			SubjectCode:   row.SubjectCode,
			SubjectName:   row.SubjectName,
			ExerciseID:    row.ExerciseID,
			ExerciseTitle: row.ExerciseTitle,
			Urgency:       row.Urgency,
			PendingReview: row.PendingReview,
		}
		if row.DueAt.Valid {
			due.DueAt = row.DueAt.Time
		}
		overview.DueSoon = append(overview.DueSoon, due)
		overview.DueSoonTotal = row.Total
	}
	// El corte también viaja resuelto: la columna dice "y N más" sin restar filas.
	overview.DueSoonHidden = overview.DueSoonTotal - len(overview.DueSoon)

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
