package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"solv-backend/internal/core/domain"

	"github.com/jmoiron/sqlx"
)

type PostgresTeacherInvitationRepository struct {
	db *sqlx.DB
}

func NewPostgresTeacherInvitationRepository(db *sqlx.DB) *PostgresTeacherInvitationRepository {
	return &PostgresTeacherInvitationRepository{db: db}
}

func (r *PostgresTeacherInvitationRepository) Create(ctx context.Context, inv *domain.TeacherInvitation) error {
	if inv.Origin == "" {
		inv.Origin = "manual"
	}
	if inv.RoleType == "" {
		inv.RoleType = "titular"
	}
	query := `
		INSERT INTO teacher_invitations (id, tenant_id, token, email, origin, role_type, used, expires_at, created_at)
		VALUES (:id, :tenant_id, :token, :email, :origin, :role_type, :used, :expires_at, NOW())
	`
	_, err := r.db.NamedExecContext(ctx, query, inv)
	if err != nil {
		return fmt.Errorf("failed to create teacher invitation: %w", err)
	}
	return nil
}

func (r *PostgresTeacherInvitationRepository) GetByToken(ctx context.Context, tenantID, token string) (*domain.TeacherInvitation, error) {
	var inv domain.TeacherInvitation
	query := `
		SELECT id, tenant_id, token, email, COALESCE(origin, 'manual') as origin,
		       COALESCE(role_type, 'titular') as role_type, used, expires_at, created_at
		FROM teacher_invitations
		WHERE tenant_id = $1 AND token = $2
	`
	err := r.db.GetContext(ctx, &inv, query, tenantID, token)
	if err != nil {
		return nil, fmt.Errorf("teacher invitation not found: %w", err)
	}
	return &inv, nil
}

func (r *PostgresTeacherInvitationRepository) GetByID(ctx context.Context, tenantID, id string) (*domain.TeacherInvitation, error) {
	var inv domain.TeacherInvitation
	query := `
		SELECT id, tenant_id, token, email, COALESCE(origin, 'manual') as origin,
		       COALESCE(role_type, 'titular') as role_type, used, expires_at, created_at
		FROM teacher_invitations
		WHERE tenant_id = $1 AND id = $2
	`
	err := r.db.GetContext(ctx, &inv, query, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("teacher invitation not found: %w", err)
	}
	return &inv, nil
}

func (r *PostgresTeacherInvitationRepository) Update(ctx context.Context, inv *domain.TeacherInvitation) error {
	query := `
		UPDATE teacher_invitations
		SET token = :token, email = :email, origin = :origin, role_type = :role_type, used = :used, expires_at = :expires_at
		WHERE tenant_id = :tenant_id AND id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, inv)
	if err != nil {
		return fmt.Errorf("failed to update teacher invitation: %w", err)
	}
	return nil
}

func (r *PostgresTeacherInvitationRepository) AcceptInvitationTx(ctx context.Context, tenantID, token, userID, userEmail string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var inv domain.TeacherInvitation
	getInvQuery := `
		SELECT id, tenant_id, token, email, used, expires_at, created_at
		FROM teacher_invitations
		WHERE tenant_id = $1 AND token = $2
		FOR UPDATE
	`
	if err := tx.GetContext(ctx, &inv, getInvQuery, tenantID, token); err != nil {
		return fmt.Errorf("invitation not found: %w", err)
	}

	if inv.Used {
		return errors.New("invitation token has already been used")
	}

	if time.Now().After(inv.ExpiresAt) {
		return errors.New("invitation token has expired")
	}

	if inv.Email != userEmail {
		return fmt.Errorf("email mismatch: invitation issued for %s, but logged in as %s", inv.Email, userEmail)
	}

	updateUserQuery := `UPDATE users SET role = 'teacher' WHERE tenant_id = $1 AND id = $2`
	res, err := tx.ExecContext(ctx, updateUserQuery, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("user not found for role update")
	}

	markUsedQuery := `UPDATE teacher_invitations SET used = TRUE WHERE id = $1`
	if _, err := tx.ExecContext(ctx, markUsedQuery, inv.ID); err != nil {
		return fmt.Errorf("failed to mark invitation as used: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresTeacherInvitationRepository) ListTeachers(ctx context.Context, tenantID, search, status, origin string) ([]*domain.TeacherListItem, error) {
	// 1. Docentes activos registrados en users
	type activeRow struct {
		ID            string         `db:"id"`
		FirstName     string         `db:"first_name"`
		LastName      string         `db:"last_name"`
		Email         string         `db:"email"`
		Origin        sql.NullString `db:"origin"`
		CreatedAt     time.Time      `db:"created_at"`
		LastLogin     sql.NullTime   `db:"last_login_at"`
		ActiveCourses int            `db:"active_courses"`
	}

	activeQuery := `
		SELECT u.id, u.first_name, u.last_name, u.email, u.origin, u.created_at, u.last_login_at,
		       (SELECT COUNT(*) FROM subjects s WHERE s.teacher_id = u.id AND s.tenant_id = u.tenant_id) as active_courses
		FROM users u
		WHERE u.tenant_id = $1 AND u.role = 'teacher'
		ORDER BY u.first_name ASC, u.last_name ASC
	`
	var activeRows []activeRow
	_ = r.db.SelectContext(ctx, &activeRows, activeQuery, tenantID)

	// 2. Invitaciones pendientes o expiradas no canjeadas
	type invRow struct {
		ID        string    `db:"id"`
		Email     string    `db:"email"`
		Origin    string    `db:"origin"`
		RoleType  string    `db:"role_type"`
		Token     string    `db:"token"`
		Used      bool      `db:"used"`
		ExpiresAt time.Time `db:"expires_at"`
		CreatedAt time.Time `db:"created_at"`
	}

	invQuery := `
		SELECT id, email, COALESCE(origin, 'manual') as origin, COALESCE(role_type, 'titular') as role_type,
		       token, used, expires_at, created_at
		FROM teacher_invitations
		WHERE tenant_id = $1 AND used = FALSE
		ORDER BY created_at DESC
	`
	var invRows []invRow
	_ = r.db.SelectContext(ctx, &invRows, invQuery, tenantID)

	var list []*domain.TeacherListItem

	for _, u := range activeRows {
		fullName := strings.TrimSpace(u.FirstName + " " + u.LastName)
		if fullName == "" {
			fullName = strings.Split(u.Email, "@")[0]
		}
		itemOrigin := "manual"
		if u.Origin.Valid && u.Origin.String != "" {
			itemOrigin = u.Origin.String
		}

		var lastLoginStr *string
		if u.LastLogin.Valid {
			formatted := u.LastLogin.Time.Format("02-Jan 15:04")
			lastLoginStr = &formatted
		}

		item := &domain.TeacherListItem{
			ID:            u.ID,
			FullName:      fullName,
			Email:         u.Email,
			Origin:        itemOrigin,
			Status:        "active",
			RoleType:      "titular",
			InvitedAt:     u.CreatedAt.Format("02-Jan-2006"),
			LastLogin:     lastLoginStr,
			ActiveCourses: u.ActiveCourses,
		}
		list = append(list, item)
	}

	activeEmails := make(map[string]bool)
	for _, u := range activeRows {
		activeEmails[strings.ToLower(u.Email)] = true
	}

	for _, inv := range invRows {
		if activeEmails[strings.ToLower(inv.Email)] {
			continue
		}
		invStatus := "pending"
		if time.Now().After(inv.ExpiresAt) {
			invStatus = "expired"
		}

		diff := time.Since(inv.CreatedAt)
		var invitedText string
		if diff < time.Hour {
			invitedText = "Hace unos minutos"
		} else if diff < 24*time.Hour {
			invitedText = fmt.Sprintf("Hace %dh", int(diff.Hours()))
		} else if diff < 7*24*time.Hour {
			invitedText = fmt.Sprintf("Hace %dd", int(diff.Hours()/24))
		} else {
			invitedText = inv.CreatedAt.Format("02-Jan-2006")
		}

		rawName := strings.Split(inv.Email, "@")[0]
		parts := strings.Split(strings.ReplaceAll(rawName, ".", " "), " ")
		for i, p := range parts {
			if len(p) > 0 {
				parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
			}
		}
		formattedName := strings.Join(parts, " ")

		item := &domain.TeacherListItem{
			ID:            inv.ID,
			FullName:      formattedName,
			Email:         inv.Email,
			Origin:        inv.Origin,
			Status:        invStatus,
			RoleType:      inv.RoleType,
			InvitedAt:     invitedText,
			ActiveCourses: 0,
			Token:         inv.Token,
		}
		list = append(list, item)
	}

	search = strings.ToLower(strings.TrimSpace(search))
	status = strings.ToLower(strings.TrimSpace(status))
	origin = strings.ToLower(strings.TrimSpace(origin))

	filtered := make([]*domain.TeacherListItem, 0, len(list))
	for _, t := range list {
		if status != "" && status != "all" && t.Status != status {
			continue
		}
		if origin != "" && origin != "all" && t.Origin != origin {
			continue
		}
		if search != "" {
			matchName := strings.Contains(strings.ToLower(t.FullName), search)
			matchEmail := strings.Contains(strings.ToLower(t.Email), search)
			if !matchName && !matchEmail {
				continue
			}
		}
		filtered = append(filtered, t)
	}

	return filtered, nil
}

func (r *PostgresTeacherInvitationRepository) GetTeacherCourses(ctx context.Context, tenantID, teacherID string) ([]*domain.TeacherCourseItem, error) {
	type row struct {
		ID            string `db:"id"`
		Name          string `db:"name"`
		Code          string `db:"code"`
		StudentsCount int    `db:"students_count"`
	}
	query := `
		SELECT s.id, s.name, s.code,
		       (SELECT COUNT(DISTINCT student_id) FROM enrollments e WHERE e.subject_id = s.id AND e.tenant_id = s.tenant_id) as students_count
		FROM subjects s
		WHERE s.teacher_id = $1 AND s.tenant_id = $2
		ORDER BY s.name ASC
	`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, teacherID, tenantID); err != nil {
		return nil, fmt.Errorf("failed to list courses for teacher: %w", err)
	}

	result := make([]*domain.TeacherCourseItem, 0, len(rows))
	for _, r := range rows {
		result = append(result, &domain.TeacherCourseItem{
			ID:            r.ID,
			Name:          r.Name,
			Code:          r.Code,
			StudentsCount: r.StudentsCount,
		})
	}
	return result, nil
}

func (r *PostgresTeacherInvitationRepository) DeleteInvitation(ctx context.Context, tenantID, id string) error {
	query := `DELETE FROM teacher_invitations WHERE id = $1 AND tenant_id = $2 AND used = FALSE`
	res, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete teacher invitation: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("invitation not found or already used")
	}
	return nil
}

