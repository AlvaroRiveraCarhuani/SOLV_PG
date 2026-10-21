package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

var nonAlnumSlugRegex = regexp.MustCompile(`[^a-z0-9]+`)

func slugifyComponentName(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	slug := strings.Trim(nonAlnumSlugRegex.ReplaceAllString(lower, "-"), "-")
	if slug == "" {
		slug = fmt.Sprintf("comp-%d", time.Now().UnixNano())
	}
	if len(slug) > 60 {
		slug = slug[:60]
	}
	return slug
}

type templateComponentRow struct {
	ID                 string    `db:"id"`
	TenantID           string    `db:"tenant_id"`
	Slug               string    `db:"slug"`
	Layer              string    `db:"layer"`
	SubCategory        string    `db:"sub_category"`
	Name               string    `db:"name"`
	Description        string    `db:"description"`
	IconKey            string    `db:"icon_key"`
	BaseImage          string    `db:"base_image"`
	CompatibleRuntimes []byte    `db:"compatible_runtimes"`
	InstallMode        string    `db:"install_mode"`
	PackageNames       []byte    `db:"package_names"`
	BuildLayerSnippet  string    `db:"build_layer_snippet"`
	SetupSnippet       string    `db:"setup_snippet"`
	JudgeEntrypoint    string    `db:"judge_entrypoint"`
	ToolsProvided      []byte    `db:"tools_provided"`
	ServiceEngine      string    `db:"service_engine"`
	RamDeltaMB         int       `db:"ram_delta_mb"`
	IsSystem           bool      `db:"is_system"`
	IsActive           bool      `db:"is_active"`
	SortOrder          int       `db:"sort_order"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
}

func (row *templateComponentRow) toDomain() *domain.TemplateComponent {
	item := &domain.TemplateComponent{
		ID:                 row.ID,
		TenantID:           row.TenantID,
		Slug:               row.Slug,
		Layer:              domain.ComponentLayer(row.Layer),
		SubCategory:        row.SubCategory,
		Name:               row.Name,
		Description:        row.Description,
		IconKey:            row.IconKey,
		BaseImage:          row.BaseImage,
		CompatibleRuntimes: []string{},
		InstallMode:        domain.InstallMode(row.InstallMode),
		PackageNames:       []string{},
		BuildLayerSnippet:  row.BuildLayerSnippet,
		SetupSnippet:       row.SetupSnippet,
		JudgeEntrypoint:    row.JudgeEntrypoint,
		ToolsProvided:      []string{},
		ServiceEngine:      row.ServiceEngine,
		RamDeltaMB:         row.RamDeltaMB,
		IsSystem:           row.IsSystem,
		IsActive:           row.IsActive,
		SortOrder:          row.SortOrder,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
	if len(row.CompatibleRuntimes) > 0 {
		_ = json.Unmarshal(row.CompatibleRuntimes, &item.CompatibleRuntimes)
	}
	if len(row.PackageNames) > 0 {
		_ = json.Unmarshal(row.PackageNames, &item.PackageNames)
	}
	if len(row.ToolsProvided) > 0 {
		_ = json.Unmarshal(row.ToolsProvided, &item.ToolsProvided)
	}
	return item
}

func (r *PostgresAdminGovernanceRepository) ListTemplateComponents(
	ctx context.Context,
	tenantID, layer string,
	includeInactive bool,
) ([]*domain.TemplateComponent, error) {
	query := `
		SELECT
			id::text, tenant_id::text, slug, layer, sub_category, name,
			COALESCE(description, '') AS description,
			COALESCE(icon_key, 'docker') AS icon_key,
			COALESCE(base_image, '') AS base_image,
			COALESCE(compatible_runtimes, '[]'::jsonb) AS compatible_runtimes,
			COALESCE(install_mode, 'none') AS install_mode,
			COALESCE(package_names, '[]'::jsonb) AS package_names,
			COALESCE(build_layer_snippet, '') AS build_layer_snippet,
			COALESCE(setup_snippet, '') AS setup_snippet,
			COALESCE(judge_entrypoint, '') AS judge_entrypoint,
			COALESCE(tools_provided, '[]'::jsonb) AS tools_provided,
			COALESCE(service_engine, '') AS service_engine,
			COALESCE(ram_delta_mb, 0) AS ram_delta_mb,
			COALESCE(is_system, false) AS is_system,
			COALESCE(is_active, true) AS is_active,
			COALESCE(sort_order, 100) AS sort_order,
			created_at, updated_at
		FROM template_components
		WHERE (tenant_id::text = $1 OR tenant_id = '00000000-0000-0000-0000-000000000001')
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if strings.TrimSpace(layer) != "" {
		query += fmt.Sprintf(" AND layer = $%d", argIdx)
		args = append(args, strings.TrimSpace(layer))
		argIdx++
	}
	if !includeInactive {
		query += " AND is_active = true"
	}
	query += " ORDER BY sort_order ASC, name ASC"

	var rows []templateComponentRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("error listing template components: %w", err)
	}

	result := make([]*domain.TemplateComponent, len(rows))
	for i := range rows {
		result[i] = rows[i].toDomain()
	}
	return result, nil
}

func (r *PostgresAdminGovernanceRepository) GetTemplateComponentByIDOrSlug(
	ctx context.Context,
	tenantID, idOrSlug string,
) (*domain.TemplateComponent, error) {
	query := `
		SELECT
			id::text, tenant_id::text, slug, layer, sub_category, name,
			COALESCE(description, '') AS description,
			COALESCE(icon_key, 'docker') AS icon_key,
			COALESCE(base_image, '') AS base_image,
			COALESCE(compatible_runtimes, '[]'::jsonb) AS compatible_runtimes,
			COALESCE(install_mode, 'none') AS install_mode,
			COALESCE(package_names, '[]'::jsonb) AS package_names,
			COALESCE(build_layer_snippet, '') AS build_layer_snippet,
			COALESCE(setup_snippet, '') AS setup_snippet,
			COALESCE(judge_entrypoint, '') AS judge_entrypoint,
			COALESCE(tools_provided, '[]'::jsonb) AS tools_provided,
			COALESCE(service_engine, '') AS service_engine,
			COALESCE(ram_delta_mb, 0) AS ram_delta_mb,
			COALESCE(is_system, false) AS is_system,
			COALESCE(is_active, true) AS is_active,
			COALESCE(sort_order, 100) AS sort_order,
			created_at, updated_at
		FROM template_components
		WHERE (tenant_id::text = $1 OR tenant_id = '00000000-0000-0000-0000-000000000001')
		  AND (id::text = $2 OR slug = $2)
		LIMIT 1
	`
	var row templateComponentRow
	if err := r.db.GetContext(ctx, &row, query, tenantID, strings.TrimSpace(idOrSlug)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrComponentNotFound
		}
		return nil, fmt.Errorf("error getting template component %q: %w", idOrSlug, err)
	}
	return row.toDomain(), nil
}

func (r *PostgresAdminGovernanceRepository) CreateTemplateComponent(
	ctx context.Context,
	tenantID string,
	comp *domain.TemplateComponent,
) (*domain.TemplateComponent, error) {
	slug := strings.TrimSpace(comp.Slug)
	if slug == "" {
		slug = slugifyComponentName(comp.Name)
	}

	compatJSON, _ := json.Marshal(comp.CompatibleRuntimes)
	if len(compatJSON) == 0 || string(compatJSON) == "null" {
		compatJSON = []byte(`[]`)
	}
	pkgsJSON, _ := json.Marshal(comp.PackageNames)
	if len(pkgsJSON) == 0 || string(pkgsJSON) == "null" {
		pkgsJSON = []byte(`[]`)
	}
	toolsJSON, _ := json.Marshal(comp.ToolsProvided)
	if len(toolsJSON) == 0 || string(toolsJSON) == "null" {
		toolsJSON = []byte(`[]`)
	}

	subCat := strings.TrimSpace(comp.SubCategory)
	if subCat == "" {
		subCat = "general"
	}
	iconKey := strings.TrimSpace(comp.IconKey)
	if iconKey == "" {
		iconKey = "docker"
	}
	installMode := string(comp.InstallMode)
	if installMode == "" {
		installMode = "none"
	}
	sortOrder := comp.SortOrder
	if sortOrder <= 0 {
		sortOrder = 200
	}

	query := `
		INSERT INTO template_components (
			id, tenant_id, slug, layer, sub_category, name, description, icon_key,
			base_image, compatible_runtimes, install_mode, package_names,
			build_layer_snippet, setup_snippet, judge_entrypoint, tools_provided,
			service_engine, ram_delta_mb, is_system, is_active, sort_order
		) VALUES (
			gen_random_uuid(), $1::uuid, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, false, true, $18
		)
		ON CONFLICT (tenant_id, slug) DO NOTHING
		RETURNING
			id::text, tenant_id::text, slug, layer, sub_category, name,
			COALESCE(description, '') AS description,
			COALESCE(icon_key, 'docker') AS icon_key,
			COALESCE(base_image, '') AS base_image,
			COALESCE(compatible_runtimes, '[]'::jsonb) AS compatible_runtimes,
			COALESCE(install_mode, 'none') AS install_mode,
			COALESCE(package_names, '[]'::jsonb) AS package_names,
			COALESCE(build_layer_snippet, '') AS build_layer_snippet,
			COALESCE(setup_snippet, '') AS setup_snippet,
			COALESCE(judge_entrypoint, '') AS judge_entrypoint,
			COALESCE(tools_provided, '[]'::jsonb) AS tools_provided,
			COALESCE(service_engine, '') AS service_engine,
			COALESCE(ram_delta_mb, 0) AS ram_delta_mb,
			COALESCE(is_system, false) AS is_system,
			COALESCE(is_active, true) AS is_active,
			COALESCE(sort_order, 100) AS sort_order,
			created_at, updated_at
	`
	var row templateComponentRow
	err := r.db.QueryRowContext(
		ctx, query,
		tenantID, slug, string(comp.Layer), subCat, strings.TrimSpace(comp.Name), strings.TrimSpace(comp.Description), iconKey,
		strings.TrimSpace(comp.BaseImage), compatJSON, installMode, pkgsJSON,
		strings.TrimSpace(comp.BuildLayerSnippet), strings.TrimSpace(comp.SetupSnippet), strings.TrimSpace(comp.JudgeEntrypoint), toolsJSON,
		strings.TrimSpace(comp.ServiceEngine), comp.RamDeltaMB, sortOrder,
	).Scan(
		&row.ID, &row.TenantID, &row.Slug, &row.Layer, &row.SubCategory, &row.Name,
		&row.Description, &row.IconKey, &row.BaseImage, &row.CompatibleRuntimes,
		&row.InstallMode, &row.PackageNames, &row.BuildLayerSnippet, &row.SetupSnippet,
		&row.JudgeEntrypoint, &row.ToolsProvided, &row.ServiceEngine, &row.RamDeltaMB,
		&row.IsSystem, &row.IsActive, &row.SortOrder, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrComponentSlugConflict
		}
		return nil, fmt.Errorf("error creating template component: %w", err)
	}
	return row.toDomain(), nil
}

func (r *PostgresAdminGovernanceRepository) UpdateTemplateComponent(
	ctx context.Context,
	tenantID, id string,
	dto domain.UpdateTemplateComponentDTO,
	synthesizedLayer string,
) (*domain.TemplateComponent, error) {
	existing, err := r.GetTemplateComponentByIDOrSlug(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(dto.Name) != "" {
		existing.Name = strings.TrimSpace(dto.Name)
	}
	existing.Description = strings.TrimSpace(dto.Description)
	if strings.TrimSpace(dto.IconKey) != "" {
		existing.IconKey = strings.TrimSpace(dto.IconKey)
	}
	if strings.TrimSpace(dto.BaseImage) != "" {
		existing.BaseImage = strings.TrimSpace(dto.BaseImage)
	}
	if existing.Layer == domain.ComponentLayerRuntime && strings.TrimSpace(existing.BaseImage) == "" {
		return nil, errors.New("la imagen base es obligatoria para componentes de tipo runtime")
	}
	if dto.CompatibleRuntimes != nil {
		existing.CompatibleRuntimes = dto.CompatibleRuntimes
	}
	if strings.TrimSpace(dto.InstallMode) != "" {
		existing.InstallMode = domain.InstallMode(strings.TrimSpace(dto.InstallMode))
	}
	if dto.PackageNames != nil {
		existing.PackageNames = dto.PackageNames
	}
	if synthesizedLayer != "" {
		existing.BuildLayerSnippet = synthesizedLayer
	} else if strings.TrimSpace(dto.BuildLayerSnippet) != "" {
		existing.BuildLayerSnippet = strings.TrimSpace(dto.BuildLayerSnippet)
	} else if existing.InstallMode == domain.InstallModeNone {
		existing.BuildLayerSnippet = ""
	}
	existing.SetupSnippet = strings.TrimSpace(dto.SetupSnippet)
	existing.JudgeEntrypoint = strings.TrimSpace(dto.JudgeEntrypoint)
	if dto.ToolsProvided != nil {
		existing.ToolsProvided = dto.ToolsProvided
	}
	if strings.TrimSpace(dto.ServiceEngine) != "" {
		existing.ServiceEngine = strings.TrimSpace(dto.ServiceEngine)
	}
	if dto.RamDeltaMB != nil {
		existing.RamDeltaMB = *dto.RamDeltaMB
	}
	if dto.IsActive != nil {
		existing.IsActive = *dto.IsActive
	}
	if strings.TrimSpace(dto.SubCategory) != "" {
		existing.SubCategory = strings.TrimSpace(dto.SubCategory)
	}
	if dto.SortOrder != nil {
		existing.SortOrder = *dto.SortOrder
	}

	compatJSON, _ := json.Marshal(existing.CompatibleRuntimes)
	pkgsJSON, _ := json.Marshal(existing.PackageNames)
	toolsJSON, _ := json.Marshal(existing.ToolsProvided)

	query := `
		UPDATE template_components
		SET
			name = $1,
			description = $2,
			icon_key = $3,
			base_image = $4,
			compatible_runtimes = $5,
			install_mode = $6,
			package_names = $7,
			build_layer_snippet = $8,
			setup_snippet = $9,
			judge_entrypoint = $10,
			tools_provided = $11,
			service_engine = $12,
			ram_delta_mb = $13,
			is_active = $14,
			sort_order = $15,
			sub_category = $16,
			updated_at = NOW()
		WHERE id::text = $17 AND (tenant_id::text = $18 OR tenant_id = '00000000-0000-0000-0000-000000000001')
	`
	_, err = r.db.ExecContext(
		ctx, query,
		existing.Name, existing.Description, existing.IconKey, existing.BaseImage,
		compatJSON, string(existing.InstallMode), pkgsJSON, existing.BuildLayerSnippet,
		existing.SetupSnippet, existing.JudgeEntrypoint, toolsJSON, existing.ServiceEngine,
		existing.RamDeltaMB, existing.IsActive, existing.SortOrder, existing.SubCategory, existing.ID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("error updating template component: %w", err)
	}
	return r.GetTemplateComponentByIDOrSlug(ctx, tenantID, existing.ID)
}

func (r *PostgresAdminGovernanceRepository) DeleteTemplateComponent(
	ctx context.Context,
	tenantID, id string,
) error {
	existing, err := r.GetTemplateComponentByIDOrSlug(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return domain.ErrComponentSystemLocked
	}
	query := `DELETE FROM template_components WHERE id::text = $1 AND tenant_id::text = $2 AND is_system = false`
	res, err := r.db.ExecContext(ctx, query, existing.ID, tenantID)
	if err != nil {
		return fmt.Errorf("error deleting template component: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrComponentNotFound
	}
	return nil
}

// UpdateOfficialTemplate actualiza la plantilla sin interrumpir contenedores ya en ejecución (Zero-Disruption Versioning).
func (r *PostgresAdminGovernanceRepository) UpdateOfficialTemplate(
	ctx context.Context,
	tenantID, templateID, adminID string,
	dto domain.UpdateOfficialTemplateDTO,
) (*domain.AdminTemplateReviewItem, error) {
	servicesConfigJSON, err := json.Marshal(dto.ServicesConfig)
	if err != nil || dto.ServicesConfig == nil {
		servicesConfigJSON = []byte(`{"services": []}`)
	}
	derivedProfile := domain.DeriveResourceProfile(dto.BaseRamMB)
	resourceProfileJSON, _ := json.Marshal(derivedProfile)

	toolsDeclaredJSON, err := json.Marshal(dto.ToolsDeclared)
	if err != nil || len(dto.ToolsDeclared) == 0 {
		toolsDeclaredJSON = []byte(`[]`)
	}
	stackConfigJSON := []byte(`{}`)
	if dto.StackConfig != nil {
		if b, errM := json.Marshal(dto.StackConfig); errM == nil {
			stackConfigJSON = b
		}
	}

	targetEnv := strings.TrimSpace(dto.TargetEnvironment)
	if targetEnv == "" {
		targetEnv = "IDE_PERSISTENTE"
	}
	timeoutMS := dto.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 5000
	}
	catParam := ""
	if dto.CategoryID != nil {
		catParam = strings.TrimSpace(*dto.CategoryID)
	}

	query := `
		UPDATE lab_templates
		SET
			name = $1,
			docker_image = $2,
			base_ram_mb = $3,
			description = $4,
			target_environment = $5,
			entrypoint = $6,
			timeout_ms = $7,
			sample_input = $8,
			category_id = NULLIF($9, '')::uuid,
			services_config = $10,
			resource_profile = $11,
			setup_script = $12,
			tools_declared = $13,
			stack_config = $14,
			reviewed_by = NULLIF($15, '')::uuid,
			updated_at = NOW()
		WHERE id::text = $16 AND (tenant_id::text = $17 OR tenant_id IS NULL)
	`
	res, err := r.db.ExecContext(
		ctx, query,
		strings.TrimSpace(dto.Name),
		strings.TrimSpace(dto.DockerImage),
		dto.BaseRamMB,
		strings.TrimSpace(dto.Description),
		targetEnv,
		strings.TrimSpace(dto.Entrypoint),
		timeoutMS,
		dto.SampleInput,
		catParam,
		servicesConfigJSON,
		resourceProfileJSON,
		dto.SetupScript,
		toolsDeclaredJSON,
		stackConfigJSON,
		adminID,
		templateID,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("error updating template: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("template not found")
	}
	return r.GetTemplateByID(ctx, templateID)
}
