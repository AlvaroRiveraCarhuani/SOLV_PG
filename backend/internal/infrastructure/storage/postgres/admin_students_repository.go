package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"solv-backend/internal/core/domain"
	"strings"
	"time"
)

func (r *PostgresAdminGovernanceRepository) ListStudentsDirectory(
	ctx context.Context,
	tenantID, search, subjectID, status, periodID string,
) ([]*domain.AdminStudentDirectoryItem, error) {
	// Determinar el periodo objetivo sólo cuando se envía explícitamente y no es "all"
	var targetPeriodID string
	if periodID != "" && periodID != "all" {
		targetPeriodID = periodID
	}

	args := []interface{}{tenantID}
	argIdx := 2

	eCountQuery := `
		SELECT e.student_id, COUNT(DISTINCT e.subject_id) AS total_enrolled
		FROM enrollments e
		WHERE e.tenant_id = $1
		GROUP BY e.student_id
	`
	wStatsQuery := `
		SELECT 
			w.student_id,
			COUNT(*) FILTER (WHERE w.status = 'running') AS active_count,
			COALESCE(MAX(w.oom_strike_count), 0) AS total_strikes,
			MAX(w.last_oom_killed_at) AS last_oom_killed
		FROM workspaces w
		WHERE w.tenant_id = $1
		GROUP BY w.student_id
	`

	if targetPeriodID != "" {
		eCountQuery = fmt.Sprintf(`
			SELECT e.student_id, COUNT(DISTINCT e.subject_id) AS total_enrolled
			FROM enrollments e
			JOIN subjects s ON s.id = e.subject_id
			WHERE e.tenant_id = $1 AND s.academic_period_id = $%d::uuid
			GROUP BY e.student_id
		`, argIdx)
		wStatsQuery = fmt.Sprintf(`
			SELECT 
				w.student_id,
				COUNT(*) FILTER (WHERE w.status = 'running') AS active_count,
				COALESCE(MAX(w.oom_strike_count), 0) AS total_strikes,
				MAX(w.last_oom_killed_at) AS last_oom_killed
			FROM workspaces w
			LEFT JOIN subjects s ON s.id = w.subject_id
			WHERE w.tenant_id = $1 AND (s.academic_period_id = $%d::uuid OR w.subject_id IS NULL)
			GROUP BY w.student_id
		`, argIdx)
		args = append(args, targetPeriodID)
		argIdx++
	}

	baseQuery := fmt.Sprintf(`
		SELECT 
			u.id,
			u.first_name,
			u.last_name,
			u.email,
			u.role,
			COALESCE(u.status, 'active') AS status,
			u.suspension_reason,
			COALESCE(e_count.total_enrolled, 0) AS enrolled_courses_count,
			COALESCE(w_stats.active_count, 0) AS active_workspaces_count,
			COALESCE(w_stats.total_strikes, 0) AS oom_strike_count,
			w_stats.last_oom_killed
		FROM users u
		LEFT JOIN (%s) e_count ON e_count.student_id = u.id
		LEFT JOIN (%s) w_stats ON w_stats.student_id = u.id
		WHERE u.tenant_id = $1 AND u.role = 'student'
	`, eCountQuery, wStatsQuery)

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		baseQuery += fmt.Sprintf(` AND (LOWER(u.first_name || ' ' || u.last_name) LIKE $%d OR LOWER(u.email) LIKE $%d)`, argIdx, argIdx)
		args = append(args, searchPattern)
		argIdx++
	}

	if subjectID != "" {
		baseQuery += fmt.Sprintf(` AND u.id IN (SELECT student_id FROM enrollments WHERE tenant_id = $1 AND subject_id = $%d)`, argIdx)
		args = append(args, subjectID)
	}

	if status != "" {
		switch strings.ToLower(status) {
		case "suspended":
			baseQuery += ` AND u.status = 'suspended'`
		case "enrolled", "active_academic":
			baseQuery += ` AND u.status != 'suspended' AND COALESCE(e_count.total_enrolled, 0) > 0`
		case "inactive", "historical":
			baseQuery += ` AND u.status != 'suspended' AND COALESCE(e_count.total_enrolled, 0) = 0`
		case "oom_killed", "strikes", "penalized":
			baseQuery += ` AND COALESCE(w_stats.total_strikes, 0) > 0`
		case "active", "running":
			baseQuery += ` AND COALESCE(w_stats.active_count, 0) > 0`
		case "idle":
			baseQuery += ` AND COALESCE(w_stats.active_count, 0) = 0`
		}
	}

	baseQuery += ` ORDER BY u.last_name ASC, u.first_name ASC`

	type rowStruct struct {
		ID                    string         `db:"id"`
		FirstName             string         `db:"first_name"`
		LastName              string         `db:"last_name"`
		Email                 string         `db:"email"`
		Role                  string         `db:"role"`
		Status                string         `db:"status"`
		SuspensionReason      sql.NullString `db:"suspension_reason"`
		EnrolledCoursesCount  int            `db:"enrolled_courses_count"`
		ActiveWorkspacesCount int            `db:"active_workspaces_count"`
		OOMStrikeCount        int            `db:"oom_strike_count"`
		LastOOMKilledAt       sql.NullTime   `db:"last_oom_killed"`
	}

	var rows []rowStruct
	err := r.db.SelectContext(ctx, &rows, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error listing students directory: %w", err)
	}

	results := make([]*domain.AdminStudentDirectoryItem, len(rows))
	for i, row := range rows {
		var lastOOM *time.Time
		if row.LastOOMKilledAt.Valid {
			t := row.LastOOMKilledAt.Time
			lastOOM = &t
		}

		var suspReason *string
		if row.SuspensionReason.Valid && row.SuspensionReason.String != "" {
			rStr := row.SuspensionReason.String
			suspReason = &rStr
		}

		acadStatus := "inactive"
		if row.Status == "suspended" {
			acadStatus = "suspended"
		} else if row.EnrolledCoursesCount > 0 {
			acadStatus = "enrolled"
		}

		results[i] = &domain.AdminStudentDirectoryItem{
			ID:                    row.ID,
			FirstName:             row.FirstName,
			LastName:              row.LastName,
			Email:                 row.Email,
			Role:                  row.Role,
			Status:                row.Status,
			SuspensionReason:      suspReason,
			AcademicStatus:        acadStatus,
			EnrolledCoursesCount:  row.EnrolledCoursesCount,
			ActiveWorkspacesCount: row.ActiveWorkspacesCount,
			OOMStrikeCount:        row.OOMStrikeCount,
			LastOOMKilledAt:       lastOOM,
		}
	}

	return results, nil
}

func (r *PostgresAdminGovernanceRepository) CreateStudent(
	ctx context.Context,
	tenantID, email, firstName, lastName string,
) (*domain.AdminStudentDirectoryItem, error) {
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE tenant_id = $1 AND email = $2)`
	if err := r.db.GetContext(ctx, &exists, checkQuery, tenantID, email); err != nil {
		return nil, fmt.Errorf("error checking student existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("student_already_exists: ya existe un usuario con este correo")
	}

	insertQuery := `
		INSERT INTO users (id, tenant_id, email, first_name, last_name, role, status, origin, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, 'student', 'active', 'manual', NOW())
		RETURNING id
	`
	var newID string
	if err := r.db.GetContext(ctx, &newID, insertQuery, tenantID, email, firstName, lastName); err != nil {
		return nil, fmt.Errorf("error creating student: %w", err)
	}

	return &domain.AdminStudentDirectoryItem{
		ID:                    newID,
		FirstName:             firstName,
		LastName:              lastName,
		Email:                 email,
		Role:                  "student",
		Status:                "active",
		EnrolledCoursesCount:  0,
		ActiveWorkspacesCount: 0,
		OOMStrikeCount:        0,
	}, nil
}

func (r *PostgresAdminGovernanceRepository) UpdateStudentStatus(
	ctx context.Context,
	tenantID, studentID, status, reason string,
) error {
	var nullReason *string
	if strings.TrimSpace(reason) != "" {
		nullReason = &reason
	}

	query := `
		UPDATE users
		SET status = $1, suspension_reason = $2
		WHERE tenant_id = $3 AND id = $4 AND role = 'student'
	`
	res, err := r.db.ExecContext(ctx, query, status, nullReason, tenantID, studentID)
	if err != nil {
		return fmt.Errorf("error updating student status: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("student not found")
	}
	return nil
}

func (r *PostgresAdminGovernanceRepository) GetStudentCourses(
	ctx context.Context,
	tenantID, studentID string,
) ([]*domain.AdminStudentCourseItem, error) {
	query := `
		SELECT 
			s.id AS subject_id,
			s.code AS subject_code,
			s.name AS subject_name,
			COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), 'Sin asignar') AS teacher_name,
			e.enrolled_at,
			w.id AS workspace_id,
			w.status AS workspace_status,
			w.memory_limit_mb,
			w.oom_strike_count,
			w.last_oom_killed_at
		FROM enrollments e
		JOIN subjects s ON s.id = e.subject_id AND s.tenant_id = e.tenant_id
		LEFT JOIN users u ON u.id = s.teacher_id
		LEFT JOIN workspaces w ON w.student_id = e.student_id AND w.subject_id = e.subject_id AND w.tenant_id = e.tenant_id
		WHERE e.tenant_id = $1 AND e.student_id = $2
		ORDER BY s.name ASC
	`

	type courseRow struct {
		SubjectID       string         `db:"subject_id"`
		SubjectCode     string         `db:"subject_code"`
		SubjectName     string         `db:"subject_name"`
		TeacherName     string         `db:"teacher_name"`
		EnrolledAt      time.Time      `db:"enrolled_at"`
		WorkspaceID     sql.NullString `db:"workspace_id"`
		WorkspaceStatus sql.NullString `db:"workspace_status"`
		MemoryLimitMB   sql.NullInt64  `db:"memory_limit_mb"`
		OOMStrikeCount  sql.NullInt64  `db:"oom_strike_count"`
		LastOOMKilledAt sql.NullTime   `db:"last_oom_killed_at"`
	}

	var rows []courseRow
	err := r.db.SelectContext(ctx, &rows, query, tenantID, studentID)
	if err != nil {
		return nil, fmt.Errorf("error getting student courses: %w", err)
	}

	results := make([]*domain.AdminStudentCourseItem, len(rows))
	for i, row := range rows {
		var wsID, wsStatus *string
		var memLimit, oomStrikes *int
		var lastOOM *time.Time

		if row.WorkspaceID.Valid {
			wsID = &row.WorkspaceID.String
		}
		if row.WorkspaceStatus.Valid {
			wsStatus = &row.WorkspaceStatus.String
		}
		if row.MemoryLimitMB.Valid {
			v := int(row.MemoryLimitMB.Int64)
			memLimit = &v
		}
		if row.OOMStrikeCount.Valid {
			v := int(row.OOMStrikeCount.Int64)
			oomStrikes = &v
		}
		if row.LastOOMKilledAt.Valid {
			t := row.LastOOMKilledAt.Time
			lastOOM = &t
		}

		results[i] = &domain.AdminStudentCourseItem{
			SubjectID:       row.SubjectID,
			SubjectCode:     row.SubjectCode,
			SubjectName:     row.SubjectName,
			TeacherName:     row.TeacherName,
			EnrolledAt:      row.EnrolledAt,
			WorkspaceID:     wsID,
			WorkspaceStatus: wsStatus,
			MemoryLimitMB:   memLimit,
			OOMStrikeCount:  oomStrikes,
			LastOOMKilledAt: lastOOM,
		}
	}

	return results, nil
}

func (r *PostgresAdminGovernanceRepository) ResetStudentOOM(ctx context.Context, tenantID, studentID string) (int64, error) {
	// 1. Verificar que el estudiante existe en el tenant
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE tenant_id = $1 AND id = $2 AND role = 'student')`
	err := r.db.GetContext(ctx, &exists, checkQuery, tenantID, studentID)
	if err != nil {
		return 0, fmt.Errorf("error checking student existence: %w", err)
	}
	if !exists {
		return 0, fmt.Errorf("student not found")
	}

	// 2. Resetear strikes y fecha de OOM en todos los workspaces del estudiante
	resetQuery := `
		UPDATE workspaces
		SET oom_strike_count = 0, last_oom_killed_at = NULL, updated_at = NOW()
		WHERE tenant_id = $1 AND student_id = $2
	`
	res, err := r.db.ExecContext(ctx, resetQuery, tenantID, studentID)
	if err != nil {
		return 0, fmt.Errorf("error resetting student OOM strikes: %w", err)
	}

	rows, _ := res.RowsAffected()
	return rows, nil
}

func (r *PostgresAdminGovernanceRepository) ValidateTeacherRole(ctx context.Context, tenantID, userID string) (bool, error) {
	var isValid bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users 
			WHERE tenant_id = $1 AND id = $2 AND (role = 'teacher' OR role = 'admin')
		)
	`
	err := r.db.GetContext(ctx, &isValid, query, tenantID, userID)
	if err != nil {
		return false, fmt.Errorf("error validating teacher role: %w", err)
	}
	return isValid, nil
}
