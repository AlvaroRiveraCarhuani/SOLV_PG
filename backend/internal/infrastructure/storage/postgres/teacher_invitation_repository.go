package postgres

import (
	"context"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"
	"time"

	"github.com/jmoiron/sqlx"
)

type PostgresTeacherInvitationRepository struct {
	db *sqlx.DB
}

func NewPostgresTeacherInvitationRepository(db *sqlx.DB) *PostgresTeacherInvitationRepository {
	return &PostgresTeacherInvitationRepository{db: db}
}

func (r *PostgresTeacherInvitationRepository) Create(ctx context.Context, inv *domain.TeacherInvitation) error {
	query := `
		INSERT INTO teacher_invitations (id, tenant_id, token, email, used, expires_at, created_at)
		VALUES (:id, :tenant_id, :token, :email, :used, :expires_at, NOW())
	`
	_, err := r.db.NamedExecContext(ctx, query, inv)
	if err != nil {
		return fmt.Errorf("failed to create teacher invitation: %w", err)
	}
	return nil
}

func (r *PostgresTeacherInvitationRepository) GetByToken(ctx context.Context, tenantID, token string) (*domain.TeacherInvitation, error) {
	var inv domain.TeacherInvitation
	query := `SELECT id, tenant_id, token, email, used, expires_at, created_at FROM teacher_invitations WHERE tenant_id = $1 AND token = $2`
	err := r.db.GetContext(ctx, &inv, query, tenantID, token)
	if err != nil {
		return nil, fmt.Errorf("teacher invitation not found: %w", err)
	}
	return &inv, nil
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
