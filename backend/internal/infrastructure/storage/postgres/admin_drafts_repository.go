package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"
	"strings"
)

func (r *PostgresAdminGovernanceRepository) SaveDraft(ctx context.Context, tenantID, userID string, formData json.RawMessage, templateID *string) (*domain.TemplateDraft, error) {
	if len(formData) == 0 {
		formData = json.RawMessage("{}")
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	query := `
		INSERT INTO template_drafts (tenant_id, user_id, form_data, template_id, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (tenant_id, user_id)
		DO UPDATE SET
			form_data = EXCLUDED.form_data,
			template_id = EXCLUDED.template_id,
			updated_at = NOW()
		RETURNING id, tenant_id, user_id, form_data, template_id, updated_at;
	`

	var draft domain.TemplateDraft
	var rawForm []byte
	var tID *string

	err = tx.QueryRowxContext(ctx, query, tenantID, userID, []byte(formData), templateID).Scan(
		&draft.ID,
		&draft.TenantID,
		&draft.UserID,
		&rawForm,
		&tID,
		&draft.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert template draft: %w", err)
	}

	draft.FormData = json.RawMessage(rawForm)
	draft.TemplateID = tID

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &draft, nil
}

func (r *PostgresAdminGovernanceRepository) GetDraftByUser(ctx context.Context, tenantID, userID string) (*domain.TemplateDraft, error) {
	query := `
		SELECT id, tenant_id, user_id, form_data, template_id, updated_at
		FROM template_drafts
		WHERE tenant_id = $1 AND user_id = $2
	`

	var draft domain.TemplateDraft
	var rawForm []byte
	var tID *string

	err := r.db.QueryRowxContext(ctx, query, tenantID, userID).Scan(
		&draft.ID,
		&draft.TenantID,
		&draft.UserID,
		&rawForm,
		&tID,
		&draft.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query template draft: %w", err)
	}

	draft.FormData = json.RawMessage(rawForm)
	draft.TemplateID = tID

	return &draft, nil
}

func (r *PostgresAdminGovernanceRepository) DeleteDraft(ctx context.Context, tenantID, userID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	query := `DELETE FROM template_drafts WHERE tenant_id = $1 AND user_id = $2`
	if _, err := tx.ExecContext(ctx, query, tenantID, userID); err != nil {
		return fmt.Errorf("delete template draft: %w", err)
	}

	return tx.Commit()
}

func (r *PostgresAdminGovernanceRepository) GetImageUsageCounts(ctx context.Context, tenantID string) (map[string]int, error) {
	query := `
		SELECT docker_image, COUNT(*) AS count
		FROM lab_templates
		WHERE (tenant_id = $1 OR tenant_id IS NULL)
		  AND status != 'RECHAZADA'
		GROUP BY docker_image
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error querying image usage counts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	usageMap := make(map[string]int)
	for rows.Next() {
		var img string
		var count int
		if err := rows.Scan(&img, &count); err != nil {
			return nil, fmt.Errorf("error scanning image usage row: %w", err)
		}
		usageMap[img] = count
	}
	return usageMap, rows.Err()
}

func (r *PostgresAdminGovernanceRepository) GetTemplateByID(ctx context.Context, id string) (*domain.AdminTemplateReviewItem, error) {
	query := `
		SELECT lt.id, lt.tenant_id, lt.name, lt.docker_image, lt.base_ram_mb, lt.status, 
		       COALESCE(lt.rejection_reason, '') AS rejection_reason, 
		       lt.reviewed_by, lt.reviewed_at, lt.requested_by,
		       COALESCE(u.first_name || ' ' || u.last_name, 'Docente') AS requested_by_name,
		       COALESCE(lt.description, '') AS description,
		       COALESCE(lt.target_environment, 'IDE_PERSISTENTE') AS target_environment,
		       COALESCE(lt.entrypoint, '') AS entrypoint,
		       COALESCE(lt.timeout_ms, 5000) AS timeout_ms,
		       COALESCE(lt.sample_input, '') AS sample_input,
		       lt.category_id,
		       COALESCE(c.name, '') AS category_name,
		       lt.model_id,
		       COALESCE(lt.services_config, '{"services":[]}'::jsonb) AS services_config,
		       COALESCE(lt.resource_profile, '{"min_mb":256,"high_mb":768,"max_mb":1024}'::jsonb) AS resource_profile,
		       COALESCE(lt.setup_script, '') AS setup_script,
		       COALESCE(lt.stack_config, '{}'::jsonb) AS stack_config,
		       COALESCE(lt.smoke_test_status, 'pending') AS smoke_test_status,
		       COALESCE(lt.smoke_test_output, '') AS smoke_test_output,
		       COALESCE(lt.security_audit_status, 'pending') AS security_audit_status,
		       COALESCE(lt.cve_critical_count, 0) AS cve_critical_count,
		       COALESCE(lt.cve_high_count, 0) AS cve_high_count,
		       lt.security_audited_at,
		       COALESCE(lt.eol_status, 'supported') AS eol_status,
		       COALESCE(lt.eol_date, '') AS eol_date,
		       COALESCE(lt.eol_message, '') AS eol_message,
		       lt.eol_checked_at,
		       lt.created_at
		FROM lab_templates lt
		LEFT JOIN users u ON u.id = lt.requested_by
		LEFT JOIN template_categories c ON c.id = lt.category_id
		WHERE lt.id = $1
	`
	var t domain.AdminTemplateReviewItem
	err := r.db.GetContext(ctx, &t, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("template not found")
		}
		return nil, fmt.Errorf("failed to get template by id %s: %w", id, err)
	}
	return &t, nil
}

func (r *PostgresAdminGovernanceRepository) GetTemplateByIDAndTenant(ctx context.Context, tenantID, id string) (*domain.AdminTemplateReviewItem, error) {
	template, err := r.GetTemplateByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if template.TenantID != nil && *template.TenantID != tenantID {
		return nil, errors.New("template not found")
	}
	return template, nil
}

func (r *PostgresAdminGovernanceRepository) GetTemplateForLanguage(ctx context.Context, environmentType, language string) (*domain.AdminTemplateReviewItem, error) {
	langLower := strings.ToLower(strings.TrimSpace(language))
	query := `
		SELECT lt.id, lt.tenant_id, lt.name, lt.docker_image, lt.base_ram_mb, lt.status, 
		       COALESCE(lt.rejection_reason, '') AS rejection_reason, 
		       lt.reviewed_by, lt.reviewed_at, lt.requested_by,
		       COALESCE(u.first_name || ' ' || u.last_name, 'Docente') AS requested_by_name,
		       COALESCE(lt.description, '') AS description,
		       COALESCE(lt.target_environment, 'IDE_PERSISTENTE') AS target_environment,
		       COALESCE(lt.entrypoint, '') AS entrypoint,
		       COALESCE(lt.timeout_ms, 5000) AS timeout_ms,
		       COALESCE(lt.sample_input, '') AS sample_input,
		       lt.category_id,
		       COALESCE(c.name, '') AS category_name,
		       lt.model_id,
		       COALESCE(lt.services_config, '{"services":[]}'::jsonb) AS services_config,
		       COALESCE(lt.resource_profile, '{"min_mb":256,"high_mb":768,"max_mb":1024}'::jsonb) AS resource_profile,
		       COALESCE(lt.setup_script, '') AS setup_script,
		       COALESCE(lt.stack_config, '{}'::jsonb) AS stack_config,
		       COALESCE(lt.smoke_test_status, 'pending') AS smoke_test_status,
		       COALESCE(lt.smoke_test_output, '') AS smoke_test_output,
		       COALESCE(lt.security_audit_status, 'pending') AS security_audit_status,
		       COALESCE(lt.cve_critical_count, 0) AS cve_critical_count,
		       COALESCE(lt.cve_high_count, 0) AS cve_high_count,
		       lt.security_audited_at,
		       COALESCE(lt.eol_status, 'supported') AS eol_status,
		       COALESCE(lt.eol_date, '') AS eol_date,
		       COALESCE(lt.eol_message, '') AS eol_message,
		       lt.eol_checked_at,
		       lt.created_at
		FROM lab_templates lt
		LEFT JOIN users u ON u.id = lt.requested_by
		LEFT JOIN template_categories c ON c.id = lt.category_id
		WHERE (lt.target_environment = $1 OR lt.target_environment = 'AMBOS') AND LOWER(lt.status) IN ('approved', 'aprobada')
		  AND (
		    ( $2 IN ('python', 'python3') AND (lt.docker_image LIKE 'python%' OR LOWER(lt.name) LIKE '%python%') )
		    OR ( $2 IN ('javascript', 'js', 'node') AND (lt.docker_image LIKE 'node%' OR LOWER(lt.name) LIKE '%node%' OR LOWER(lt.name) LIKE '%javascript%') )
		    OR ( $2 IN ('cpp', 'c++', 'c') AND (lt.docker_image LIKE 'gcc%' OR LOWER(lt.name) LIKE '%c++%' OR LOWER(lt.name) LIKE '%gcc%') )
		    OR ( $2 = 'java' AND (lt.docker_image LIKE 'eclipse-temurin%' OR lt.docker_image LIKE 'java%' OR LOWER(lt.name) LIKE '%java%') )
		    OR ( $2 IN ('csharp', 'c#', 'cs') AND (lt.docker_image LIKE 'mono%' OR LOWER(lt.name) LIKE '%csharp%' OR LOWER(lt.name) LIKE '%mono%') )
		    OR ( $2 IN ('go', 'golang') AND (lt.docker_image LIKE 'golang%' OR LOWER(lt.name) LIKE '%go%') )
		    OR ( $2 IN ('rust', 'rs') AND (lt.docker_image LIKE 'rust%' OR LOWER(lt.name) LIKE '%rust%') )
		    OR ( LOWER(lt.docker_image) LIKE '%' || $2 || '%' OR LOWER(lt.name) LIKE '%' || $2 || '%' )
		  )
		ORDER BY (lt.description = 'SYSTEM_SEED_RUNNER') DESC, lt.created_at ASC
		LIMIT 1
	`
	var t domain.AdminTemplateReviewItem
	err := r.db.GetContext(ctx, &t, query, environmentType, langLower)
	if err != nil {
		fallbackQuery := `
			SELECT lt.id, lt.tenant_id, lt.name, lt.docker_image, lt.base_ram_mb, lt.status, 
			       COALESCE(lt.rejection_reason, '') AS rejection_reason, 
			       lt.reviewed_by, lt.reviewed_at, lt.requested_by,
			       COALESCE(u.first_name || ' ' || u.last_name, 'Docente') AS requested_by_name,
			       COALESCE(lt.description, '') AS description,
			       COALESCE(lt.target_environment, 'IDE_PERSISTENTE') AS target_environment,
			       COALESCE(lt.entrypoint, '') AS entrypoint,
			       COALESCE(lt.timeout_ms, 5000) AS timeout_ms,
			       COALESCE(lt.sample_input, '') AS sample_input,
			       lt.category_id,
			       COALESCE(c.name, '') AS category_name,
			       lt.model_id,
			       COALESCE(lt.services_config, '{"services":[]}'::jsonb) AS services_config,
			       COALESCE(lt.resource_profile, '{"min_mb":256,"high_mb":768,"max_mb":1024}'::jsonb) AS resource_profile,
			       COALESCE(lt.setup_script, '') AS setup_script,
			       COALESCE(lt.smoke_test_status, 'pending') AS smoke_test_status,
			       COALESCE(lt.smoke_test_output, '') AS smoke_test_output,
			       COALESCE(lt.security_audit_status, 'pending') AS security_audit_status,
			       COALESCE(lt.cve_critical_count, 0) AS cve_critical_count,
			       COALESCE(lt.cve_high_count, 0) AS cve_high_count,
			       lt.security_audited_at,
			       COALESCE(lt.eol_status, 'supported') AS eol_status,
			       COALESCE(lt.eol_date, '') AS eol_date,
			       COALESCE(lt.eol_message, '') AS eol_message,
			       lt.eol_checked_at,
			       lt.created_at
			FROM lab_templates lt
			LEFT JOIN users u ON u.id = lt.requested_by
			LEFT JOIN template_categories c ON c.id = lt.category_id
			WHERE (lt.target_environment = $1 OR lt.target_environment = 'AMBOS') AND LOWER(lt.status) IN ('approved', 'aprobada')
			ORDER BY (lt.description = 'SYSTEM_SEED_RUNNER') DESC, lt.created_at ASC
			LIMIT 1
		`
		if fbErr := r.db.GetContext(ctx, &t, fallbackQuery, environmentType); fbErr != nil {
			return nil, fmt.Errorf("failed to find approved template for environment %s and language %s: %w", environmentType, language, err)
		}
	}
	return &t, nil
}

// GetServerMajorVersion consulta SHOW server_version en el motor PostgreSQL activo y extrae la versión mayor.
func (r *PostgresAdminGovernanceRepository) GetServerMajorVersion(ctx context.Context) (string, error) {
	if r == nil || r.db == nil {
		return "", errors.New("conexión a base de datos no inicializada")
	}
	var rawVersion string
	if err := r.db.QueryRowContext(ctx, `SHOW server_version;`).Scan(&rawVersion); err != nil {
		return "", fmt.Errorf("error consultando versión de PostgreSQL: %w", err)
	}
	rawVersion = strings.TrimSpace(rawVersion)
	if rawVersion == "" {
		return "", nil
	}
	fields := strings.Fields(rawVersion)
	versionToken := fields[0]
	major := strings.Split(versionToken, ".")[0]
	return strings.TrimSpace(major), nil
}
