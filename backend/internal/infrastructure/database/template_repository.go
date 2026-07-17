package database

import (
	"context"
	"fmt"

	"solv-backend/internal/domain"
)

func (db *Database) CreateTemplate(ctx context.Context, dto domain.CreateTemplateDTO) (string, error) {
	query := `
		INSERT INTO lab_templates (name, docker_image, base_ram_mb)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id string
	err := db.db.QueryRowContext(ctx, query, dto.Name, dto.DockerImage, dto.BaseRamMB).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to create lab template: %w", err)
	}

	return id, nil
}
func (db *Database) GetAllTemplates(ctx context.Context) ([]domain.TemplateResponseDTO, error) {
	query := `SELECT id, name, docker_image, base_ram_mb FROM lab_templates`

	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query lab templates: %w", err)
	}
	defer rows.Close()

	var templates []domain.TemplateResponseDTO
	for rows.Next() {
		var t domain.TemplateResponseDTO
		if err := rows.Scan(&t.ID, &t.Name, &t.DockerImage, &t.BaseRamMB); err != nil {
			return nil, fmt.Errorf("failed to scan lab template row: %w", err)
		}
		templates = append(templates, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating lab templates rows: %w", err)
	}
	if templates == nil {
		templates = []domain.TemplateResponseDTO{}
	}

	return templates, nil
}
