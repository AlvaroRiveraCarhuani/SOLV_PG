package database

import (
	"context"
	"fmt"

	"github.com/lib/pq"
	"solv-backend/internal/domain"
)

func (db *Database) CreateTemplate(ctx context.Context, tenantID string, dto domain.CreateTemplateDTO) (string, error) {
	query := `
		INSERT INTO lab_templates (tenant_id, name, docker_image, base_ram_mb)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var id string
	err := db.db.QueryRowContext(ctx, query, tenantID, dto.Name, dto.DockerImage, dto.BaseRamMB).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to create lab template: %w", err)
	}

	return id, nil
}

func (db *Database) GetAllTemplates(ctx context.Context, tenantID string) ([]domain.TemplateResponseDTO, error) {
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
			COALESCE(lt.services_config, '{"services": []}'::jsonb),
			COALESCE(lt.stack_config, '{}'::jsonb)
		FROM lab_templates lt
		LEFT JOIN template_categories tc ON tc.id = lt.category_id
		WHERE lt.tenant_id = $1
	`

	rows, err := db.db.QueryContext(ctx, query, tenantID)
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
			&t.StackConfig,
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

	if len(templates) > 0 {
		ids := make([]string, len(templates))
		idMap := make(map[string]int, len(templates))
		for i, t := range templates {
			ids[i] = t.ID
			idMap[t.ID] = i
		}
		runnerQuery := `
			SELECT template_id::text, language, COALESCE(entrypoint, ''), COALESCE(docker_image, '')
			FROM template_language_runners
			WHERE template_id = ANY($1)
			ORDER BY language
		`
		runnerRows, err := db.db.QueryContext(ctx, runnerQuery, pq.Array(ids))
		if err == nil {
			defer func() { _ = runnerRows.Close() }()
			for runnerRows.Next() {
				var tID, lang, entrypoint, dockerImg string
				if err := runnerRows.Scan(&tID, &lang, &entrypoint, &dockerImg); err == nil {
					if idx, ok := idMap[tID]; ok {
						templates[idx].LanguageRunners = append(templates[idx].LanguageRunners, domain.TemplateLanguageRunnerDTO{
							Language:    lang,
							Entrypoint:  entrypoint,
							DockerImage: dockerImg,
						})
					}
				}
			}
		}
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
			COALESCE(lt.services_config, '{"services": []}'::jsonb),
			COALESCE(lt.stack_config, '{}'::jsonb)
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
		&t.StackConfig,
	)
	if err != nil {
		return domain.TemplateResponseDTO{}, fmt.Errorf("failed to get template by id: %w", err)
	}

	runnerRows, err := db.db.QueryContext(ctx, `
		SELECT language, COALESCE(entrypoint, ''), COALESCE(docker_image, '')
		FROM template_language_runners
		WHERE template_id = $1
		ORDER BY language
	`, id)
	if err == nil {
		defer func() { _ = runnerRows.Close() }()
		for runnerRows.Next() {
			var lang, entrypoint, dockerImg string
			if err := runnerRows.Scan(&lang, &entrypoint, &dockerImg); err == nil {
				t.LanguageRunners = append(t.LanguageRunners, domain.TemplateLanguageRunnerDTO{
					Language:    lang,
					Entrypoint:  entrypoint,
					DockerImage: dockerImg,
				})
			}
		}
	}

	return t, nil
}
