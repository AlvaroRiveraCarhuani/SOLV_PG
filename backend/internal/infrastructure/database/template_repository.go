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
	query := `
		SELECT 
			lt.id, 
			lt.name, 
			COALESCE(lt.description, ''),
			COALESCE(tc.name, 'General'),
			lt.docker_image, 
			lt.base_ram_mb, 
			COALESCE(lt.target_environment, 'IDE_PERSISTENTE'), 
			COALESCE(lt.status, 'approved'),
			(lt.requested_by IS NULL) AS is_official,
			COALESCE(lt.entrypoint, ''), 
			COALESCE(lt.services_config, '{"services": []}'::jsonb) 
		FROM lab_templates lt
		LEFT JOIN template_categories tc ON tc.id = lt.category_id
	`

	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query lab templates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var templates []domain.TemplateResponseDTO
	for rows.Next() {
		var t domain.TemplateResponseDTO
		if err := rows.Scan(
			&t.ID, 
			&t.Name, 
			&t.Description, 
			&t.Category, 
			&t.DockerImage, 
			&t.BaseRamMB, 
			&t.TargetEnvironment, 
			&t.Status, 
			&t.IsOfficial,
			&t.Entrypoint, 
			&t.ServicesConfig,
		); err != nil {
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

// GetTemplateByID retrieves a template by UUID.
func (db *Database) GetTemplateByID(ctx context.Context, id string) (domain.TemplateResponseDTO, error) {
	query := `
		SELECT 
			lt.id, 
			lt.name, 
			COALESCE(lt.description, ''),
			COALESCE(tc.name, 'General'),
			lt.docker_image, 
			lt.base_ram_mb, 
			COALESCE(lt.target_environment, 'IDE_PERSISTENTE'), 
			COALESCE(lt.status, 'approved'),
			(lt.requested_by IS NULL) AS is_official,
			COALESCE(lt.entrypoint, ''), 
			COALESCE(lt.services_config, '{"services": []}'::jsonb) 
		FROM lab_templates lt
		LEFT JOIN template_categories tc ON tc.id = lt.category_id
		WHERE lt.id = $1
	`
	var t domain.TemplateResponseDTO
	err := db.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, 
		&t.Name, 
		&t.Description, 
		&t.Category, 
		&t.DockerImage, 
		&t.BaseRamMB, 
		&t.TargetEnvironment, 
		&t.Status, 
		&t.IsOfficial,
		&t.Entrypoint, 
		&t.ServicesConfig,
	)
	if err != nil {
		return domain.TemplateResponseDTO{}, fmt.Errorf("failed to get template by id: %w", err)
	}
	return t, nil
}
