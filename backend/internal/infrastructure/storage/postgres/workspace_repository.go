package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"solv-backend/internal/core/domain"
)

type PostgresWorkspaceRepository struct {
	db *sqlx.DB
}

func NewPostgresWorkspaceRepository(db *sqlx.DB) domain.WorkspaceRepository {
	return &PostgresWorkspaceRepository{db: db}
}

func (r *PostgresWorkspaceRepository) GetByStudentAndSubject(ctx context.Context, studentID string, subjectID string) (*domain.WorkspaceInstance, error) {
	query := `
		SELECT id, student_id, subject_id, container_id, status, access_url, created_at, updated_at
		FROM workspaces
		WHERE student_id = $1 AND subject_id = $2
	`
	var ws domain.WorkspaceInstance
	err := r.db.GetContext(ctx, &ws, query, studentID, subjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace for student %s and subject %s: %w", studentID, subjectID, err)
	}
	return &ws, nil
}

func (r *PostgresWorkspaceRepository) Create(ctx context.Context, workspace *domain.WorkspaceInstance) error {
	query := `
		INSERT INTO workspaces (id, student_id, subject_id, container_id, status, access_url, created_at, updated_at)
		VALUES (:id, :student_id, :subject_id, :container_id, :status, :access_url, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, workspace)
	if err != nil {
		return fmt.Errorf("failed to create workspace record via sqlx: %w", err)
	}
	return nil
}

func (r *PostgresWorkspaceRepository) UpdateContainerID(ctx context.Context, id string, containerID string) error {
	query := `
		UPDATE workspaces
		SET container_id = $2, updated_at = $3
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, containerID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update container_id for workspace %s: %w", id, err)
	}
	return nil
}

func (r *PostgresWorkspaceRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE workspaces
		SET status = $2, updated_at = $3
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update status for workspace %s: %w", id, err)
	}
	return nil
}
