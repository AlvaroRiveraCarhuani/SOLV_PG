package database

import (
	"context"
	"fmt"
	"github.com/lib/pq"

	"solv-backend/internal/domain"
)

// CreateInstance inserts a new lab instance orchestration record.
func (db *Database) CreateInstance(ctx context.Context, dto domain.CreateInstanceDTO) (string, error) {
	query := `
		INSERT INTO lab_instances (user_id, template_id, container_name, traefik_url, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id string
	err := db.db.QueryRowContext(ctx, query, dto.UserID, dto.TemplateID, dto.ContainerName, dto.TraefikURL, dto.Status).Scan(&id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23503" { // foreign_key_violation
				return "", fmt.Errorf("invalid user_id or template_id provided")
			}
			if pqErr.Code == "23505" { // unique_violation
				return "", fmt.Errorf("container name or traefik URL already exists")
			}
		}
		return "", fmt.Errorf("failed to create lab instance: %w", err)
	}

	return id, nil
}

// UpdateInstanceStatus updates the status of a lab instance.
func (db *Database) UpdateInstanceStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE lab_instances SET status = $1 WHERE id = $2`
	_, err := db.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}
	return nil
}
