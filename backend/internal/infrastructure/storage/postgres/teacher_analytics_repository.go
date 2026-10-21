package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"solv-backend/internal/core/domain"
)

func (r *PostgresTeacherRepository) GetCourseGradesMatrix(ctx context.Context, tenantID, teacherID, subjectID string) (*domain.CourseGradesMatrix, error) {
	// 1. Obtener datos de la materia y validar aislamiento
	type subRow struct {
		ID   string `db:"id"`
		Name string `db:"name"`
		Code string `db:"code"`
	}
	var subject subRow
	var err error
	if teacherID != "" {
		query := `SELECT id, name, code FROM subjects WHERE id = $1 AND tenant_id = $2 AND teacher_id = $3`
		err = r.db.GetContext(ctx, &subject, query, subjectID, tenantID, teacherID)
	} else {
		query := `SELECT id, name, code FROM subjects WHERE id = $1 AND tenant_id = $2`
		err = r.db.GetContext(ctx, &subject, query, subjectID, tenantID)
	}
	if err != nil {
		return nil, domain.ErrNotFound
	}

	matrix := &domain.CourseGradesMatrix{
		SubjectID:   subject.ID,
		SubjectName: subject.Name,
		SubjectCode: subject.Code,
		Exercises:   make([]domain.CourseExerciseHeader, 0),
		Students:    make([]domain.StudentGradesRow, 0),
	}

	// 2. Listar ejercicios de la materia
	type exHeaderRow struct {
		ID    string `db:"id"`
		Title string `db:"title"`
	}
	var exercises []exHeaderRow
	exQuery := `SELECT id, title FROM exercises WHERE subject_id = $1 AND tenant_id = $2 ORDER BY created_at ASC`
	if err := r.db.SelectContext(ctx, &exercises, exQuery, subjectID, tenantID); err != nil {
		return nil, fmt.Errorf("failed to list exercises: %w", err)
	}
	for _, ex := range exercises {
		matrix.Exercises = append(matrix.Exercises, domain.CourseExerciseHeader{
			ID:    ex.ID,
			Title: ex.Title,
		})
	}

	// 3. Listar estudiantes inscritos
	type stuRow struct {
		StudentID    string `db:"student_id"`
		StudentName  string `db:"student_name"`
		StudentEmail string `db:"student_email"`
	}
	var students []stuRow
	stuQuery := `
		SELECT e.student_id,
		       COALESCE(u.first_name || ' ' || u.last_name, 'Estudiante') AS student_name,
		       COALESCE(u.email, '') AS student_email
		FROM enrollments e
		JOIN users u ON e.student_id = u.id
		WHERE e.subject_id = $1 AND e.tenant_id = $2
		ORDER BY student_name ASC
	`
	if err := r.db.SelectContext(ctx, &students, stuQuery, subjectID, tenantID); err != nil {
		return nil, fmt.Errorf("failed to list students: %w", err)
	}

	// 4. Obtener entregas
	type gradeRecord struct {
		StudentID      string        `db:"student_id"`
		ExerciseID     string        `db:"exercise_id"`
		Verdict        string        `db:"verdict"`
		Score          sql.NullInt64 `db:"score"`
		ManualOverride sql.NullBool  `db:"manual_override"`
	}
	var submissions []gradeRecord
	subQuery := `
		SELECT sub.student_id, sub.exercise_id, sub.verdict, sub.score, sub.manual_override
		FROM submissions sub
		JOIN exercises ex ON sub.exercise_id = ex.id
		WHERE ex.subject_id = $1 AND sub.tenant_id = $2
		ORDER BY sub.submitted_at ASC
	`
	_ = r.db.SelectContext(ctx, &submissions, subQuery, subjectID, tenantID)

	// Mapear mejor nota por estudiante y ejercicio
	gradesMap := make(map[string]map[string]int) // studentID -> exerciseID -> score
	for _, sub := range submissions {
		if _, ok := gradesMap[sub.StudentID]; !ok {
			gradesMap[sub.StudentID] = make(map[string]int)
		}
		var finalScore int
		if sub.Score.Valid {
			finalScore = int(sub.Score.Int64)
		} else if sub.Verdict == "AC" || (sub.ManualOverride.Valid && sub.ManualOverride.Bool) {
			finalScore = 100
		} else {
			finalScore = 0
		}

		// Mantener la mejor nota obtenida
		if curr, ok := gradesMap[sub.StudentID][sub.ExerciseID]; !ok || finalScore > curr {
			gradesMap[sub.StudentID][sub.ExerciseID] = finalScore
		}
	}

	// 5. Construir filas de estudiantes con cálculo de promedio
	for _, stu := range students {
		row := domain.StudentGradesRow{
			StudentID:    stu.StudentID,
			StudentName:  stu.StudentName,
			StudentEmail: stu.StudentEmail,
			Grades:       make(map[string]int),
		}

		var totalScore float64
		for _, ex := range matrix.Exercises {
			score := 0
			if sGrades, ok := gradesMap[stu.StudentID]; ok {
				if s, ok := sGrades[ex.ID]; ok {
					score = s
				}
			}
			row.Grades[ex.ID] = score
			totalScore += float64(score)
		}

		if len(matrix.Exercises) > 0 {
			row.Average = totalScore / float64(len(matrix.Exercises))
		}

		matrix.Students = append(matrix.Students, row)
	}

	return matrix, nil
}

func (r *PostgresTeacherRepository) GetExerciseSubmissionsForPlagiarism(ctx context.Context, tenantID, teacherID, subjectID, exerciseID string) ([]*domain.SubmissionForPlagiarism, error) {
	query := `
		SELECT DISTINCT ON (sub.student_id, sub.exercise_id)
			sub.id AS submission_id,
			sub.exercise_id,
			ex.title AS exercise_title,
			sub.student_id,
			COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email) AS student_name,
			u.email AS student_email,
			COALESCE(ex.language, 'python') AS language,
			sub.code AS code,
			sub.verdict,
			sub.submitted_at
		FROM submissions sub
		JOIN exercises ex ON sub.exercise_id = ex.id
		JOIN users u ON sub.student_id = u.id
		WHERE ex.subject_id = $1 AND sub.tenant_id = $2
		  AND ($3 = '' OR sub.exercise_id::text = $3)
		  AND LENGTH(TRIM(sub.code)) > 0
		ORDER BY sub.student_id, sub.exercise_id, (CASE WHEN sub.verdict = 'AC' THEN 1 ELSE 2 END), sub.submitted_at DESC
	`

	var list []*domain.SubmissionForPlagiarism
	err := r.db.SelectContext(ctx, &list, query, subjectID, tenantID, exerciseID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch submissions for plagiarism: %w", err)
	}

	if list == nil {
		list = make([]*domain.SubmissionForPlagiarism, 0)
	}

	return list, nil
}

func (r *PostgresTeacherRepository) ListLiveWorkspaceSessions(ctx context.Context, tenantID, teacherID string) ([]*domain.LiveWorkspaceSession, error) {
	query := `
		SELECT 
			w.id AS workspace_id,
			COALESCE(w.container_id, '') AS container_id,
			w.student_id,
			COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email) AS student_name,
			u.email AS student_email,
			w.subject_id,
			s.name AS subject_name,
			w.status,
			w.memory_limit_mb,
			w.oom_strike_count AS oom_strikes,
			COALESCE(w.last_heartbeat_at, w.created_at) AS last_heartbeat
		FROM workspaces w
		JOIN users u ON w.student_id = u.id
		JOIN subjects s ON w.subject_id = s.id
		WHERE w.tenant_id = $1
		  AND ($2 = '' OR s.teacher_id = $2::uuid)
		  AND w.status IN ('running', 'active', 'pending')
		ORDER BY w.last_heartbeat_at DESC NULLS LAST
	`

	var sessions []*domain.LiveWorkspaceSession
	err := r.db.SelectContext(ctx, &sessions, query, tenantID, teacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to query live workspace sessions: %w", err)
	}

	if sessions == nil {
		sessions = make([]*domain.LiveWorkspaceSession, 0)
	}

	for _, s := range sessions {
		if s.ContainerID != "" && s.Status == "running" {
			s.IsAttached = true
		}
	}

	return sessions, nil
}

func (r *PostgresTeacherRepository) GetCourseAnalytics(ctx context.Context, tenantID, teacherID, subjectID string) (*domain.CourseAnalytics, error) {
	analytics := &domain.CourseAnalytics{
		DifficultyDistribution:        make(map[string]domain.DifficultyMetric),
		TopTags:                       make([]domain.TagMetric, 0),
		MostFailedCases:               make([]domain.FailedCaseMetric, 0),
		AvgResolutionTimeByDifficulty: make(map[string]int),
		SubmissionsTimeline:           make([]domain.TimelineMetric, 0),
	}

	// 1. Distribución por dificultad
	type diffRow struct {
		Difficulty  string  `db:"difficulty"`
		Count       int     `db:"count"`
		SuccessRate float64 `db:"success_rate"`
	}
	var diffRows []diffRow
	diffQuery := `
		SELECT 
			e.difficulty,
			COUNT(DISTINCT e.id) as count,
			COALESCE(AVG(CASE WHEN s.score = 100 OR s.verdict = 'AC' THEN 1.0 ELSE 0.0 END), 0.0) as success_rate
		FROM exercises e
		LEFT JOIN submissions s ON s.exercise_id = e.id
		WHERE e.tenant_id = $1 AND e.subject_id = $2 AND e.difficulty IS NOT NULL AND e.difficulty != ''
		GROUP BY e.difficulty
	`
	if err := r.db.SelectContext(ctx, &diffRows, diffQuery, tenantID, subjectID); err == nil {
		for _, row := range diffRows {
			analytics.DifficultyDistribution[row.Difficulty] = domain.DifficultyMetric{
				Count:       row.Count,
				SuccessRate: row.SuccessRate,
			}
		}
	}

	// 2. Top Tags
	type tagRow struct {
		Tag         string  `db:"tag"`
		Count       int     `db:"exercise_count"`
		SuccessRate float64 `db:"success_rate"`
	}
	var tagRows []tagRow
	tagQuery := `
		SELECT 
			t.tag,
			COUNT(DISTINCT e.id) as exercise_count,
			COALESCE(AVG(CASE WHEN s.score = 100 OR s.verdict = 'AC' THEN 1.0 ELSE 0.0 END), 0.0) as success_rate
		FROM exercises e
		CROSS JOIN LATERAL unnest(e.tags) AS t(tag)
		LEFT JOIN submissions s ON s.exercise_id = e.id
		WHERE e.tenant_id = $1 AND e.subject_id = $2 AND t.tag != ''
		GROUP BY t.tag
		ORDER BY exercise_count DESC, success_rate ASC
		LIMIT 10
	`
	if err := r.db.SelectContext(ctx, &tagRows, tagQuery, tenantID, subjectID); err == nil {
		for _, row := range tagRows {
			analytics.TopTags = append(analytics.TopTags, domain.TagMetric{
				Tag:         row.Tag,
				Count:       row.Count,
				SuccessRate: row.SuccessRate,
			})
		}
	}

	// 3. Casos más fallados (run_metrics con fallback a submissions)
	type failRow struct {
		ExerciseTitle string `db:"exercise_title"`
		CaseIndex     int    `db:"case_index"`
		FailCount     int    `db:"fail_count"`
	}
	var failRows []failRow
	failQuery := `
		SELECT 
			e.title as exercise_title,
			rm.case_index as case_index,
			COUNT(*) as fail_count
		FROM run_metrics rm
		JOIN exercises e ON rm.exercise_id = e.id
		WHERE e.tenant_id = $1 AND e.subject_id = $2 AND rm.verdict != 'AC'
		GROUP BY e.id, e.title, rm.case_index
		ORDER BY fail_count DESC
		LIMIT 5
	`
	if err := r.db.SelectContext(ctx, &failRows, failQuery, tenantID, subjectID); err == nil && len(failRows) > 0 {
		for _, row := range failRows {
			analytics.MostFailedCases = append(analytics.MostFailedCases, domain.FailedCaseMetric{
				ExerciseTitle: row.ExerciseTitle,
				CaseIndex:     row.CaseIndex,
				FailCount:     row.FailCount,
			})
		}
	} else {
		// Fallback desde tabla submissions
		fallbackQuery := `
			SELECT 
				e.title as exercise_title,
				1 as case_index,
				COUNT(*) as fail_count
			FROM submissions s
			JOIN exercises e ON s.exercise_id = e.id
			WHERE e.tenant_id = $1 AND e.subject_id = $2 AND s.verdict != 'AC'
			GROUP BY e.id, e.title
			ORDER BY fail_count DESC
			LIMIT 5
		`
		var fallbackRows []failRow
		if err := r.db.SelectContext(ctx, &fallbackRows, fallbackQuery, tenantID, subjectID); err == nil {
			for _, row := range fallbackRows {
				analytics.MostFailedCases = append(analytics.MostFailedCases, domain.FailedCaseMetric{
					ExerciseTitle: row.ExerciseTitle,
					CaseIndex:     row.CaseIndex,
					FailCount:     row.FailCount,
				})
			}
		}
	}

	// 4. Tiempo promedio de resolución por dificultad
	type timeRow struct {
		Difficulty string `db:"difficulty"`
		AvgSeconds int    `db:"avg_seconds"`
	}
	var timeRows []timeRow
	timeQuery := `
		WITH first_seen AS (
			SELECT student_id, exercise_id, MIN(submitted_at) as first_time
			FROM submissions
			GROUP BY student_id, exercise_id
		),
		first_ac AS (
			SELECT student_id, exercise_id, MIN(submitted_at) as ac_time
			FROM submissions
			WHERE score = 100 OR verdict = 'AC'
			GROUP BY student_id, exercise_id
		),
		durations AS (
			SELECT 
				e.difficulty,
				EXTRACT(EPOCH FROM (fa.ac_time - fs.first_time)) as duration_sec
			FROM first_seen fs
			JOIN first_ac fa ON fs.student_id = fa.student_id AND fs.exercise_id = fa.exercise_id
			JOIN exercises e ON e.id = fs.exercise_id
			WHERE e.tenant_id = $1 AND e.subject_id = $2 AND e.difficulty IS NOT NULL AND e.difficulty != ''
			  AND fa.ac_time >= fs.first_time
		)
		SELECT 
			difficulty,
			COALESCE(AVG(duration_sec), 0)::int as avg_seconds
		FROM durations
		GROUP BY difficulty
	`
	if err := r.db.SelectContext(ctx, &timeRows, timeQuery, tenantID, subjectID); err == nil {
		for _, row := range timeRows {
			analytics.AvgResolutionTimeByDifficulty[row.Difficulty] = row.AvgSeconds
		}
	}

	// 5. Timeline de envíos (últimos 30 días)
	type timelineRow struct {
		DateStr string `db:"date_str"`
		Count   int    `db:"count"`
	}
	var timelineRows []timelineRow
	timelineQuery := `
		SELECT 
			TO_CHAR(s.submitted_at, 'YYYY-MM-DD') as date_str,
			COUNT(*) as count
		FROM submissions s
		JOIN exercises e ON s.exercise_id = e.id
		WHERE e.tenant_id = $1 AND e.subject_id = $2
		  AND s.submitted_at >= NOW() - INTERVAL '30 days'
		GROUP BY date_str
		ORDER BY date_str ASC
	`
	if err := r.db.SelectContext(ctx, &timelineRows, timelineQuery, tenantID, subjectID); err == nil {
		for _, row := range timelineRows {
			analytics.SubmissionsTimeline = append(analytics.SubmissionsTimeline, domain.TimelineMetric{
				Date:  row.DateStr,
				Count: row.Count,
			})
		}
	}

	return analytics, nil
}
