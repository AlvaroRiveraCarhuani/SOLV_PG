package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"
	"strings"
	"time"
)

func (r *PostgresAdminGovernanceRepository) ListTemplateCategories(ctx context.Context, tenantID string) ([]*domain.TemplateCategory, error) {
	query := `
		SELECT id, tenant_id, name, COALESCE(description, '') AS description, COALESCE(is_active, true) AS is_active, COALESCE(sort_order, 0) AS sort_order, created_at, updated_at
		FROM template_categories
		WHERE tenant_id = $1 OR tenant_id IS NULL
		ORDER BY sort_order ASC, name ASC
	`
	var list []*domain.TemplateCategory
	err := r.db.SelectContext(ctx, &list, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error listing template categories: %w", err)
	}
	return list, nil
}

func (r *PostgresAdminGovernanceRepository) CreateTemplateCategory(ctx context.Context, tenantID string, dto domain.CreateCategoryDTO) (*domain.TemplateCategory, error) {
	query := `
		INSERT INTO template_categories (id, tenant_id, name, description, is_active, sort_order)
		VALUES (gen_random_uuid(), $1, $2, $3, true, $4)
		ON CONFLICT (tenant_id, name) DO NOTHING
		RETURNING id, tenant_id, name, COALESCE(description, '') AS description, COALESCE(is_active, true) AS is_active, COALESCE(sort_order, 0) AS sort_order, created_at, updated_at
	`
	var cat domain.TemplateCategory
	err := r.db.QueryRowContext(ctx, query, tenantID, strings.TrimSpace(dto.Name), strings.TrimSpace(dto.Description), dto.SortOrder).Scan(
		&cat.ID, &cat.TenantID, &cat.Name, &cat.Description, &cat.IsActive, &cat.SortOrder, &cat.CreatedAt, &cat.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTemplateNameConflict
		}
		return nil, fmt.Errorf("error creating template category: %w", err)
	}
	return &cat, nil
}

func (r *PostgresAdminGovernanceRepository) UpdateTemplateCategory(ctx context.Context, tenantID, categoryID string, dto domain.UpdateCategoryDTO) (*domain.TemplateCategory, error) {
	query := `
		UPDATE template_categories
		SET name = $1, description = $2,
		    is_active = COALESCE($3, is_active),
		    sort_order = COALESCE($4, sort_order),
		    updated_at = NOW()
		WHERE id = $5 AND (tenant_id = $6 OR tenant_id IS NULL)
		RETURNING id, tenant_id, name, COALESCE(description, '') AS description, COALESCE(is_active, true) AS is_active, COALESCE(sort_order, 0) AS sort_order, created_at, updated_at
	`
	var cat domain.TemplateCategory
	err := r.db.QueryRowContext(ctx, query, strings.TrimSpace(dto.Name), strings.TrimSpace(dto.Description), dto.IsActive, dto.SortOrder, categoryID, tenantID).Scan(
		&cat.ID, &cat.TenantID, &cat.Name, &cat.Description, &cat.IsActive, &cat.SortOrder, &cat.CreatedAt, &cat.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("error updating template category: %w", err)
	}
	return &cat, nil
}

func (r *PostgresAdminGovernanceRepository) ReorderTemplateCategories(ctx context.Context, tenantID string, items []domain.ReorderCategoryItemDTO) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	query := `UPDATE template_categories SET sort_order = $1, updated_at = NOW() WHERE id = $2 AND (tenant_id = $3 OR tenant_id IS NULL)`
	for _, item := range items {
		if _, err := tx.ExecContext(ctx, query, item.SortOrder, item.ID, tenantID); err != nil {
			return fmt.Errorf("reorder category %s: %w", item.ID, err)
		}
	}
	return tx.Commit()
}

func (r *PostgresAdminGovernanceRepository) DeleteTemplateCategory(ctx context.Context, tenantID, categoryID string) error {
	checkQuery := `
		SELECT 
			(SELECT COUNT(*) FROM lab_templates WHERE category_id = $1 AND (tenant_id = $2 OR tenant_id IS NULL)) +
			(SELECT COUNT(*) FROM template_models WHERE category_id = $1 AND (tenant_id = $2 OR tenant_id IS NULL))
	`
	var count int
	if err := r.db.QueryRowContext(ctx, checkQuery, categoryID, tenantID).Scan(&count); err != nil {
		return fmt.Errorf("error comprobando uso de categoría: %w", err)
	}
	if count > 0 {
		return domain.ErrCategoryInUse
	}

	deleteQuery := `DELETE FROM template_categories WHERE id = $1 AND (tenant_id = $2 OR tenant_id IS NULL)`
	res, err := r.db.ExecContext(ctx, deleteQuery, categoryID, tenantID)
	if err != nil {
		return fmt.Errorf("error deleting template category: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

type templateModelRow struct {
	ID                string    `db:"id"`
	TenantID          string    `db:"tenant_id"`
	CategoryID        string    `db:"category_id"`
	CategoryName      string    `db:"category_name"`
	SourceTemplateID  *string   `db:"source_template_id"`
	Title             string    `db:"title"`
	Description       string    `db:"description"`
	DockerImage       string    `db:"docker_image"`
	BaseRamMB         int       `db:"base_ram_mb"`
	Tools             []byte    `db:"tools"`
	TargetEnvironment string    `db:"target_environment"`
	Entrypoint        string    `db:"entrypoint"`
	TimeoutMS         int       `db:"timeout_ms"`
	SampleInput       string    `db:"sample_input"`
	UsageCount        int       `db:"usage_count"`
	IsActive          bool      `db:"is_active"`
	SortOrder         int       `db:"sort_order"`
	CreatedAt         time.Time `db:"created_at"`
}

func (r *PostgresAdminGovernanceRepository) ListTemplateModels(ctx context.Context, tenantID, targetEnv, categoryID string, includeInactive bool) ([]*domain.TemplateModelItemDTO, error) {
	query := `
		SELECT 
			tm.id,
			COALESCE(tm.tenant_id::text, '') AS tenant_id,
			COALESCE(tm.category_id::text, '') AS category_id,
			COALESCE(tc.name, '') AS category_name,
			tm.source_template_id::text AS source_template_id,
			tm.title,
			COALESCE(tm.description, '') AS description,
			tm.docker_image,
			tm.base_ram_mb,
			COALESCE(tm.tools, '[]'::jsonb) AS tools,
			COALESCE(tm.target_environment, 'IDE_PERSISTENTE') AS target_environment,
			COALESCE(tm.entrypoint, '') AS entrypoint,
			COALESCE(tm.timeout_ms, 5000) AS timeout_ms,
			COALESCE(tm.sample_input, '') AS sample_input,
			COALESCE(tm.is_active, true) AS is_active,
			COALESCE(tm.sort_order, 0) AS sort_order,
			(
				SELECT COUNT(*)
				FROM lab_templates lt
				WHERE lt.model_id = tm.id
				  AND (lt.tenant_id = tm.tenant_id OR (lt.tenant_id IS NULL AND tm.tenant_id IS NULL))
				  AND ` + approvedTemplateStatusClause("lt") + `
			) AS usage_count,
			tm.created_at
		FROM template_models tm
		LEFT JOIN template_categories tc ON tc.id = tm.category_id
		WHERE (tm.tenant_id = $1 OR tm.tenant_id IS NULL)
		  AND ($2 = '' OR tm.target_environment = $2 OR tm.target_environment = 'AMBOS')
		  AND ($3 = '' OR tm.category_id::text = $3)
		  AND ($4 = true OR COALESCE(tm.is_active, true) = true)
		ORDER BY COALESCE(tc.sort_order, 0) ASC, tc.name ASC, COALESCE(tm.sort_order, 0) ASC, tm.title ASC
	`
	var rows []templateModelRow
	err := r.db.SelectContext(ctx, &rows, query, tenantID, targetEnv, categoryID, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("error listing template models: %w", err)
	}

	result := make([]*domain.TemplateModelItemDTO, len(rows))
	for i, row := range rows {
		var toolsList []string
		if len(row.Tools) > 0 {
			_ = json.Unmarshal(row.Tools, &toolsList)
		}
		result[i] = &domain.TemplateModelItemDTO{
			ID:                row.ID,
			TenantID:          row.TenantID,
			CategoryID:        row.CategoryID,
			CategoryName:      row.CategoryName,
			SourceTemplateID:  row.SourceTemplateID,
			Title:             row.Title,
			Description:       row.Description,
			DockerImage:       row.DockerImage,
			BaseRamMB:         row.BaseRamMB,
			Tools:             toolsList,
			TargetEnvironment: row.TargetEnvironment,
			Entrypoint:        row.Entrypoint,
			TimeoutMS:         row.TimeoutMS,
			SampleInput:       row.SampleInput,
			UsageCount:        row.UsageCount,
			IsActive:          row.IsActive,
			SortOrder:         row.SortOrder,
			CreatedAt:         row.CreatedAt,
		}
	}
	return result, nil
}

func (r *PostgresAdminGovernanceRepository) UpdateTemplateModel(ctx context.Context, tenantID, modelID string, dto domain.UpdateTemplateModelDTO) (*domain.TemplateModelItemDTO, error) {
	var catName string
	catCheckQuery := `SELECT name FROM template_categories WHERE id = $1 AND (tenant_id = $2 OR tenant_id IS NULL)`
	if err := r.db.QueryRowContext(ctx, catCheckQuery, dto.CategoryID, tenantID).Scan(&catName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("check category: %w", err)
	}

	query := `
		UPDATE template_models
		SET title = $1, description = $2, category_id = $3, updated_at = NOW()
		WHERE id = $4 AND (tenant_id = $5 OR tenant_id IS NULL)
	`
	res, err := r.db.ExecContext(ctx, query, strings.TrimSpace(dto.Title), strings.TrimSpace(dto.Description), dto.CategoryID, modelID, tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "uk_template_models_tenant_title") || strings.Contains(err.Error(), "duplicate key") {
			return nil, domain.ErrTemplateNameConflict
		}
		return nil, fmt.Errorf("update template model: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, errors.New("template model not found")
	}

	models, err := r.ListTemplateModels(ctx, tenantID, "", "", true)
	if err != nil {
		return nil, err
	}
	for _, m := range models {
		if m.ID == modelID {
			return m, nil
		}
	}
	return nil, errors.New("template model not found after update")
}

func (r *PostgresAdminGovernanceRepository) SetTemplateModelActive(ctx context.Context, tenantID, modelID string, isActive bool) error {
	query := `
		UPDATE template_models
		SET is_active = $1, updated_at = NOW()
		WHERE id = $2 AND (tenant_id = $3 OR tenant_id IS NULL)
	`
	res, err := r.db.ExecContext(ctx, query, isActive, modelID, tenantID)
	if err != nil {
		return fmt.Errorf("set template model active: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("template model not found")
	}
	return nil
}

func (r *PostgresAdminGovernanceRepository) PromoteTemplateToModel(
	ctx context.Context,
	tenantID, templateID, adminID string,
	dto domain.PromoteTemplateToModelDTO,
) (*domain.TemplateModelItemDTO, error) {
	// 1. Obtener la plantilla original
	queryTpl := `
		SELECT id, tenant_id, name, docker_image, base_ram_mb, status, COALESCE(description, ''),
		       COALESCE(target_environment, 'IDE_PERSISTENTE'), COALESCE(entrypoint, ''),
		       COALESCE(timeout_ms, 5000), COALESCE(sample_input, ''),
		       COALESCE(tools_declared, '[]'::jsonb)
		FROM lab_templates
		WHERE id = $1 AND (tenant_id = $2 OR tenant_id IS NULL)
	`
	var (
		id, tID, name, dockerImg, status, desc, targetEnv, entrypoint, sampleInput string
		baseRam, timeoutMS                                                         int
		toolsJSON                                                                  []byte
	)
	err := r.db.QueryRowContext(ctx, queryTpl, templateID, tenantID).Scan(
		&id, &tID, &name, &dockerImg, &baseRam, &status, &desc,
		&targetEnv, &entrypoint, &timeoutMS, &sampleInput, &toolsJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("plantilla no encontrada")
		}
		return nil, fmt.Errorf("error obteniendo plantilla para promover: %w", err)
	}

	// 2. Verificar guarda de aprobación (helper único de estado aprobado)
	if !isTemplateApproved(status) {
		return nil, domain.ErrTemplateNotApproved
	}

	// 3. Verificar que la categoría exista
	var catName string
	err = r.db.QueryRowContext(ctx, `SELECT name FROM template_categories WHERE id = $1 AND (tenant_id = $2 OR tenant_id IS NULL)`, dto.CategoryID, tenantID).Scan(&catName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("error verificando categoría: %w", err)
	}

	title := strings.TrimSpace(dto.Title)
	if title == "" {
		title = name
	}
	modelDesc := strings.TrimSpace(dto.Description)
	if modelDesc == "" {
		modelDesc = desc
	}

	// 4. Insertar en template_models
	insertModelQuery := `
		INSERT INTO template_models (
			id, tenant_id, category_id, source_template_id, title, description, docker_image,
			base_ram_mb, tools, target_environment, entrypoint, timeout_ms, sample_input
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12
		)
		ON CONFLICT (tenant_id, title) DO NOTHING
		RETURNING id, created_at
	`
	var newID string
	var createdAt time.Time
	err = r.db.QueryRowContext(
		ctx, insertModelQuery,
		tenantID, dto.CategoryID, templateID, title, modelDesc, dockerImg,
		baseRam, toolsJSON, targetEnv, entrypoint, timeoutMS, sampleInput,
	).Scan(&newID, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTemplateNameConflict
		}
		return nil, fmt.Errorf("error guardando nuevo modelo: %w", err)
	}

	// 5. Vincular la plantilla al modelo recién creado
	_, _ = r.db.ExecContext(ctx, `UPDATE lab_templates SET model_id = $1, category_id = $2 WHERE id = $3`, newID, dto.CategoryID, templateID)

	var toolsList []string
	if len(toolsJSON) > 0 {
		_ = json.Unmarshal(toolsJSON, &toolsList)
	}

	return &domain.TemplateModelItemDTO{
		ID:                newID,
		TenantID:          tenantID,
		CategoryID:        dto.CategoryID,
		CategoryName:      catName,
		SourceTemplateID:  &templateID,
		Title:             title,
		Description:       modelDesc,
		DockerImage:       dockerImg,
		BaseRamMB:         baseRam,
		Tools:             toolsList,
		TargetEnvironment: targetEnv,
		Entrypoint:        entrypoint,
		TimeoutMS:         timeoutMS,
		SampleInput:       sampleInput,
		UsageCount:        1,
		CreatedAt:         createdAt,
	}, nil
}
