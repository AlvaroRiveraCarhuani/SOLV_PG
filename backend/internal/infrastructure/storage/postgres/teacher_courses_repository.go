package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"solv-backend/internal/core/domain"
	"time"
)

func (r *PostgresTeacherRepository) GetCoursesSummary(ctx context.Context, tenantID, teacherID, periodID string) ([]*domain.TeacherCourseSummary, error) {
	type subjectRow struct {
		ID   string `db:"id"`
		Name string `db:"name"`
		Code string `db:"code"`
	}

	var rows []subjectRow
	var err error

	// Construir query con filtro opcional de período académico
	if periodID != "" {
		if teacherID != "" {
			query := `
				SELECT s.id, s.name, s.code
				FROM subjects s
				WHERE s.tenant_id = $1
				  AND s.teacher_id = $2
				  AND s.academic_period_id = $3::uuid
				ORDER BY s.name ASC
			`
			err = r.db.SelectContext(ctx, &rows, query, tenantID, teacherID, periodID)
		} else {
			query := `
				SELECT s.id, s.name, s.code
				FROM subjects s
				WHERE s.tenant_id = $1
				  AND s.academic_period_id = $2::uuid
				ORDER BY s.name ASC
			`
			err = r.db.SelectContext(ctx, &rows, query, tenantID, periodID)
		}
	} else {
		if teacherID != "" {
			query := `
				SELECT s.id, s.name, s.code
				FROM subjects s
				WHERE s.tenant_id = $1
				  AND s.teacher_id = $2
				ORDER BY s.name ASC
			`
			err = r.db.SelectContext(ctx, &rows, query, tenantID, teacherID)
		} else {
			query := `
				SELECT s.id, s.name, s.code
				FROM subjects s
				WHERE s.tenant_id = $1
				ORDER BY s.name ASC
			`
			err = r.db.SelectContext(ctx, &rows, query, tenantID)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list teacher subjects: %w", err)
	}

	result := make([]*domain.TeacherCourseSummary, 0, len(rows))
	for _, s := range rows {
		summary := &domain.TeacherCourseSummary{
			ID:   s.ID,
			Name: s.Name,
			Code: s.Code,
		}

		// Students count
		_ = r.db.GetContext(ctx, &summary.StudentsCount, `
			SELECT COUNT(DISTINCT student_id)
			FROM enrollments
			WHERE subject_id = $1 AND tenant_id = $2
		`, s.ID, tenantID)

		// Active now (heartbeat < 2 min)
		_ = r.db.GetContext(ctx, &summary.ActiveNow, `
			SELECT COUNT(DISTINCT student_id)
			FROM environments
			WHERE subject_id = $1 AND tenant_id = $2
			  AND last_heartbeat_at >= NOW() - INTERVAL '2 minutes'
		`, s.ID, tenantID)

		// Pendientes de corrección: estado de revisión declarado, no veredicto.
		_ = r.db.GetContext(ctx, &summary.PendingReview, `
			SELECT COUNT(DISTINCT sub.id)
			FROM submissions sub
			JOIN exercises ex ON sub.exercise_id = ex.id
			WHERE ex.subject_id = $1 AND sub.tenant_id = $2
			  AND sub.review_status IN ('pending', 'in_review')
		`, s.ID, tenantID)

		// At risk: Estudiantes matriculados sin submission para ejercicios con entrega en <24h Y sin heartbeat en <24h
		_ = r.db.GetContext(ctx, &summary.AtRisk, `
			SELECT COUNT(DISTINCT e.student_id)
			FROM enrollments e
			WHERE e.subject_id = $1 AND e.tenant_id = $2
			  AND EXISTS (
			      SELECT 1 FROM exercises ex
			      WHERE ex.subject_id = $1 AND ex.tenant_id = $2
			        AND ex.due_date > NOW() AND ex.due_date <= NOW() + INTERVAL '24 hours'
			        AND NOT EXISTS (
			            SELECT 1 FROM submissions sub
			            WHERE sub.exercise_id = ex.id AND sub.student_id = e.student_id AND sub.tenant_id = $2
			        )
			  )
			  AND NOT EXISTS (
			      SELECT 1 FROM environments w
			      WHERE w.subject_id = $1 AND w.student_id = e.student_id AND w.tenant_id = $2
			        AND w.last_heartbeat_at >= NOW() - INTERVAL '24 hours'
			  )
		`, s.ID, tenantID)

		result = append(result, summary)
	}

	return result, nil
}

// GetTeacherExerciseBank consolida el read model del banco de ejercicios del
// docente a través de todas sus materias. ReviewPending usa la semántica de cada
// entorno: juez -> veredictos WA/RE/TLE sin override; IDE -> entregas PENDING
// esperando corrección manual en SpeedGrader.
func (r *PostgresTeacherRepository) GetTeacherExerciseBank(ctx context.Context, tenantID, teacherID string) ([]*domain.TeacherExerciseBankItem, error) {
	type bankRow struct {
		ID               string       `db:"id"`
		Title            string       `db:"title"`
		Status           string       `db:"status"`
		Type             string       `db:"type"`
		EnvironmentType  string       `db:"environment_type"`
		Language         string       `db:"language"`
		MemoryLimitMB    int          `db:"memory_limit_mb"`
		TimeLimitMS      int          `db:"time_limit_ms"`
		DueDate          sql.NullTime `db:"due_date"`
		SubjectID        string       `db:"subject_id"`
		SubjectCode      string       `db:"subject_code"`
		SubjectName      string       `db:"subject_name"`
		SubmissionsCount int          `db:"submissions_count"`
		StudentsCount    int          `db:"students_count"`
		ReviewPending    int          `db:"review_pending"`
	}

	query := `
		SELECT
			e.id,
			e.title,
			COALESCE(e.status, 'draft') AS status,
			e.type,
			e.environment_type,
			e.language,
			e.memory_limit_mb,
			e.time_limit_ms,
			e.due_date,
			s.id AS subject_id,
			s.code AS subject_code,
			s.name AS subject_name,
			(SELECT COUNT(*) FROM submissions su WHERE su.exercise_id = e.id AND su.tenant_id = $1) AS submissions_count,
			(SELECT COUNT(DISTINCT en.student_id) FROM enrollments en WHERE en.subject_id = e.subject_id AND en.tenant_id = $1) AS students_count,
			CASE
				WHEN e.environment_type = 'IDE_PERSISTENTE' THEN
					(SELECT COUNT(*) FROM submissions su
					 WHERE su.exercise_id = e.id AND su.tenant_id = $1 AND su.verdict = 'PENDING')
				ELSE
					(SELECT COUNT(*) FROM submissions su
					 WHERE su.exercise_id = e.id AND su.tenant_id = $1
					   AND su.verdict IN ('WA', 'RE', 'TLE')
					   AND (su.manual_override IS FALSE OR su.manual_override IS NULL))
			END AS review_pending
		FROM exercises e
		JOIN subjects s ON s.id = e.subject_id
		WHERE e.tenant_id = $1
		  AND s.tenant_id = $1
	`
	args := []any{tenantID}
	if teacherID != "" {
		query += ` AND s.teacher_id = $2`
		args = append(args, teacherID)
	}
	query += `
		ORDER BY s.code, e.created_at ASC
	`

	var rows []bankRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list teacher exercise bank: %w", err)
	}

	items := make([]*domain.TeacherExerciseBankItem, 0, len(rows))
	for _, row := range rows {
		item := &domain.TeacherExerciseBankItem{
			ID:               row.ID,
			Title:            row.Title,
			Status:           row.Status,
			Type:             row.Type,
			EnvironmentType:  row.EnvironmentType,
			Language:         row.Language,
			MemoryLimitMB:    row.MemoryLimitMB,
			TimeLimitMS:      row.TimeLimitMS,
			SubjectID:        row.SubjectID,
			SubjectCode:      row.SubjectCode,
			SubjectName:      row.SubjectName,
			SubmissionsCount: row.SubmissionsCount,
			StudentsCount:    row.StudentsCount,
			ReviewPending:    row.ReviewPending,
		}
		if row.DueDate.Valid {
			item.DueDate = &row.DueDate.Time
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *PostgresTeacherRepository) GetCourseLabsStats(ctx context.Context, tenantID, teacherID, subjectID string) ([]*domain.TeacherLabStats, error) {
	// Verificar existencia de la materia y aislamiento por tenant
	var exists bool
	var err error

	if teacherID != "" {
		checkQuery := `
			SELECT EXISTS (
				SELECT 1 FROM subjects
				WHERE id = $1 AND tenant_id = $2
				  AND teacher_id = $3
			)
		`
		err = r.db.GetContext(ctx, &exists, checkQuery, subjectID, tenantID, teacherID)
	} else {
		checkQuery := `
			SELECT EXISTS (
				SELECT 1 FROM subjects
				WHERE id = $1 AND tenant_id = $2
			)
		`
		err = r.db.GetContext(ctx, &exists, checkQuery, subjectID, tenantID)
	}

	if err != nil || !exists {
		return nil, domain.ErrNotFound
	}

	// Total estudiantes inscritos en el curso
	var totalStudents int
	_ = r.db.GetContext(ctx, &totalStudents, `
		SELECT COUNT(DISTINCT student_id)
		FROM enrollments
		WHERE subject_id = $1 AND tenant_id = $2
	`, subjectID, tenantID)

	// Listar ejercicios del curso
	type exRow struct {
		ID              string       `db:"id"`
		Title           string       `db:"title"`
		Status          string       `db:"status"`
		EnvironmentType string       `db:"environment_type"`
		DueDate         sql.NullTime `db:"due_date"`
	}
	var exRows []exRow
	exercisesQuery := `
		SELECT id, title, COALESCE(status, 'draft') AS status, COALESCE(environment_type, 'JUEZ_EFIMERO') AS environment_type, due_date
		FROM exercises
		WHERE subject_id = $1 AND tenant_id = $2
		ORDER BY created_at ASC
	`
	if err := r.db.SelectContext(ctx, &exRows, exercisesQuery, subjectID, tenantID); err != nil {
		return nil, fmt.Errorf("failed to list exercises for subject: %w", err)
	}

	result := make([]*domain.TeacherLabStats, 0, len(exRows))
	for _, ex := range exRows {
		stat := &domain.TeacherLabStats{
			ID:              ex.ID,
			Title:           ex.Title,
			Status:          ex.Status,
			EnvironmentType: ex.EnvironmentType,
			StudentsCount:   totalStudents,
			VerdictsSummary: map[string]int{
				"AC":          0,
				"WA":          0,
				"TLE":         0,
				"RE":          0,
				"AST_BLOCKED": 0,
			},
		}
		if ex.DueDate.Valid {
			stat.DueDate = &ex.DueDate.Time
		}

		// Conteo total de entregas
		_ = r.db.GetContext(ctx, &stat.SubmissionsCount, `
			SELECT COUNT(*) FROM submissions
			WHERE exercise_id = $1 AND tenant_id = $2
		`, ex.ID, tenantID)

		// Auto graded (AC o con manual override)
		_ = r.db.GetContext(ctx, &stat.AutoGraded, `
			SELECT COUNT(*) FROM submissions
			WHERE exercise_id = $1 AND tenant_id = $2
			  AND (verdict = 'AC' OR manual_override = TRUE)
		`, ex.ID, tenantID)

		// Pendientes de corrección: estado de revisión declarado (ADR-053), no veredicto.
		_ = r.db.GetContext(ctx, &stat.PendingReview, `
			SELECT COUNT(*) FROM submissions
			WHERE exercise_id = $1 AND tenant_id = $2
			  AND review_status IN ('pending', 'in_review')
		`, ex.ID, tenantID)

		// At risk para este laboratorio específico (sin submission y due_date < 24h sin heartbeat)
		if stat.DueDate != nil && stat.DueDate.After(time.Now()) && stat.DueDate.Before(time.Now().Add(24*time.Hour)) {
			_ = r.db.GetContext(ctx, &stat.AtRisk, `
				SELECT COUNT(DISTINCT e.student_id)
				FROM enrollments e
				WHERE e.subject_id = $1 AND e.tenant_id = $2
				  AND NOT EXISTS (
				      SELECT 1 FROM submissions sub
				      WHERE sub.exercise_id = $3 AND sub.student_id = e.student_id AND sub.tenant_id = $2
				  )
				  AND NOT EXISTS (
				      SELECT 1 FROM environments w
				      WHERE w.subject_id = $1 AND w.student_id = e.student_id AND w.tenant_id = $2
				        AND w.last_heartbeat_at >= NOW() - INTERVAL '24 hours'
				  )
			`, subjectID, tenantID, ex.ID)
		}

		// Resumen de veredictos
		type vRow struct {
			Verdict string `db:"verdict"`
			Count   int    `db:"count"`
		}
		var vRows []vRow
		_ = r.db.SelectContext(ctx, &vRows, `
			SELECT verdict, COUNT(*) as count
			FROM submissions
			WHERE exercise_id = $1 AND tenant_id = $2
			GROUP BY verdict
		`, ex.ID, tenantID)

		for _, vr := range vRows {
			stat.VerdictsSummary[vr.Verdict] = vr.Count
		}

		result = append(result, stat)
	}

	return result, nil
}
