package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"solv-backend/internal/core/domain"
)

type PostgresAdminGovernanceRepository struct {
	db *sqlx.DB
}

func NewPostgresAdminGovernanceRepository(db *sqlx.DB) *PostgresAdminGovernanceRepository {
	return &PostgresAdminGovernanceRepository{db: db}
}

func (r *PostgresAdminGovernanceRepository) ListStudentsDirectory(
	ctx context.Context,
	tenantID, search, subjectID, status string,
) ([]*domain.AdminStudentDirectoryItem, error) {
	baseQuery := `
		SELECT 
			u.id,
			u.first_name,
			u.last_name,
			u.email,
			u.role,
			COALESCE(e_count.total_enrolled, 0) AS enrolled_courses_count,
			COALESCE(w_stats.active_count, 0) AS active_workspaces_count,
			COALESCE(w_stats.total_strikes, 0) AS oom_strike_count,
			w_stats.last_oom_killed
		FROM users u
		LEFT JOIN (
			SELECT student_id, COUNT(*) AS total_enrolled
			FROM enrollments
			WHERE tenant_id = $1
			GROUP BY student_id
		) e_count ON e_count.student_id = u.id
		LEFT JOIN (
			SELECT 
				student_id,
				COUNT(*) FILTER (WHERE status = 'running') AS active_count,
				COALESCE(MAX(oom_strike_count), 0) AS total_strikes,
				MAX(last_oom_killed_at) AS last_oom_killed
			FROM workspaces
			WHERE tenant_id = $1
			GROUP BY student_id
		) w_stats ON w_stats.student_id = u.id
		WHERE u.tenant_id = $1 AND u.role = 'student'
	`

	args := []interface{}{tenantID}
	argIdx := 2

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		baseQuery += fmt.Sprintf(` AND (LOWER(u.first_name || ' ' || u.last_name) LIKE $%d OR LOWER(u.email) LIKE $%d)`, argIdx, argIdx)
		args = append(args, searchPattern)
		argIdx++
	}

	if subjectID != "" {
		baseQuery += fmt.Sprintf(` AND u.id IN (SELECT student_id FROM enrollments WHERE tenant_id = $1 AND subject_id = $%d)`, argIdx)
		args = append(args, subjectID)
		argIdx++
	}

	if status != "" {
		switch strings.ToLower(status) {
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

		results[i] = &domain.AdminStudentDirectoryItem{
			ID:                    row.ID,
			FirstName:             row.FirstName,
			LastName:              row.LastName,
			Email:                 row.Email,
			Role:                  row.Role,
			EnrolledCoursesCount:  row.EnrolledCoursesCount,
			ActiveWorkspacesCount: row.ActiveWorkspacesCount,
			OOMStrikeCount:        row.OOMStrikeCount,
			LastOOMKilledAt:       lastOOM,
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
