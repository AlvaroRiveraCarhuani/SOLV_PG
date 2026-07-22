package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

type postgresLabInstanceRepo struct {
	db *sqlx.DB
}

func NewPostgresLabInstanceRepository(db *sqlx.DB) domain.LabInstanceRepository {
	return &postgresLabInstanceRepo{db: db}
}

func (r *postgresLabInstanceRepo) Create(ctx context.Context, instance *domain.LabInstance) error {
	query := `
		INSERT INTO lab_instances (id, template_id, user_id, container_id, status, ram_limit_mb, created_at, updated_at)
		VALUES (:id, :template_id, :user_id, :container_id, :status, :ram_limit_mb, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, instance)
	if err != nil {
		return fmt.Errorf("failed to create lab instance: %w", err)
	}
	return nil
}

func (r *postgresLabInstanceRepo) GetByUserAndTemplate(ctx context.Context, userID string, templateID string) (*domain.LabInstance, error) {
	query := `
		SELECT id, template_id, user_id, container_id, status, ram_limit_mb, created_at, updated_at 
		FROM lab_instances 
		WHERE user_id = $1 AND template_id = $2
	`
	var instance domain.LabInstance
	err := r.db.GetContext(ctx, &instance, query, userID, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get lab instance by user and template: %w", err)
	}
	return &instance, nil
}

func (r *postgresLabInstanceRepo) GetByID(ctx context.Context, id string) (*domain.LabInstance, error) {
	query := `
		SELECT id, template_id, user_id, container_id, status, ram_limit_mb, created_at, updated_at 
		FROM lab_instances 
		WHERE id = $1
	`
	var instance domain.LabInstance
	err := r.db.GetContext(ctx, &instance, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get lab instance by id: %w", err)
	}
	return &instance, nil
}

func (r *postgresLabInstanceRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE lab_instances 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update lab instance status: %w", err)
	}
	return nil
}

func (r *postgresLabInstanceRepo) UpdateContainerID(ctx context.Context, id string, containerID string) error {
	query := `
		UPDATE lab_instances 
		SET container_id = $1, updated_at = NOW() 
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, containerID, id)
	if err != nil {
		return fmt.Errorf("failed to update lab instance container id: %w", err)
	}
	return nil
}
