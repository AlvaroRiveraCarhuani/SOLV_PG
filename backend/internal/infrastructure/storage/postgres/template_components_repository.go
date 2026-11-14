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

// componentUsageCountsQuery cuenta cuántas plantillas y modelos referencian cada
// tecnología de `stack_config`, comparando por id y por slug porque el
// compositor acepta ambos. Las plantillas rechazadas no cuentan: nunca se
// aprovisionan.
const componentUsageCountsQuery = `
	WITH component_refs AS (
		SELECT lt.stack_config->>'runtime_id' AS ref, lt.tenant_id
		FROM lab_templates lt WHERE COALESCE(lt.status, '') <> 'RECHAZADA'
		UNION ALL
		SELECT lt.stack_config->>'web_platform_id', lt.tenant_id
		FROM lab_templates lt WHERE COALESCE(lt.status, '') <> 'RECHAZADA'
		UNION ALL
		SELECT lt.stack_config->>'database_id', lt.tenant_id
		FROM lab_templates lt WHERE COALESCE(lt.status, '') <> 'RECHAZADA'
		UNION ALL
		SELECT jsonb_array_elements_text(COALESCE(lt.stack_config->'addon_ids', '[]'::jsonb)), lt.tenant_id
		FROM lab_templates lt WHERE COALESCE(lt.status, '') <> 'RECHAZADA'
		UNION ALL
		SELECT tm.stack_config->>'runtime_id', tm.tenant_id FROM template_models tm
		UNION ALL
		SELECT tm.stack_config->>'web_platform_id', tm.tenant_id FROM template_models tm
		UNION ALL
		SELECT tm.stack_config->>'database_id', tm.tenant_id FROM template_models tm
		UNION ALL
		SELECT jsonb_array_elements_text(COALESCE(tm.stack_config->'addon_ids', '[]'::jsonb)), tm.tenant_id
		FROM template_models tm
	)
	SELECT ref, COUNT(*)
	FROM component_refs
	WHERE ref IS NOT NULL AND ref <> ''
	  AND (tenant_id IS NULL OR tenant_id::text = $1 OR tenant_id::text = $2)
	GROUP BY ref
`

// componentReferencesInTemplateSQL arma la condición que decide si una fila de
// plantilla o modelo referencia el componente `tc`. Comparte la forma con
// componentUsageCountsQuery para que listado y borrado no discrepen.
func componentReferencesSQL(alias string) string {
	return fmt.Sprintf(`(
		%s.stack_config->>'runtime_id' IN (tc.id::text, tc.slug)
		OR %s.stack_config->>'web_platform_id' IN (tc.id::text, tc.slug)
		OR %s.stack_config->>'database_id' IN (tc.id::text, tc.slug)
		OR EXISTS (
			SELECT 1 FROM jsonb_array_elements_text(COALESCE(%s.stack_config->'addon_ids', '[]'::jsonb)) AS addon(ref)
			WHERE addon.ref IN (tc.id::text, tc.slug)
		)
	)`, alias, alias, alias, alias)
}

// applyComponentUsageCounts suma las referencias por id y por slug de cada
// tecnología. Es pura para poder probarla sin base de datos.
func applyComponentUsageCounts(items []*domain.TemplateComponent, counts map[string]int) {
	for _, item := range items {
		if item == nil {
			continue
		}
		total := counts[item.ID]
		if item.Slug != item.ID {
			total += counts[item.Slug]
		}
		item.UsageCount = total
	}
}

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
	DependencyProfile  []byte    `db:"dependency_profile"`
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
		DependencyProfile:  domain.DependencyProfile{},
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
	if len(row.DependencyProfile) > 0 {
		_ = json.Unmarshal(row.DependencyProfile, &item.DependencyProfile)
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
			COALESCE(dependency_profile, '{}'::jsonb) AS dependency_profile,
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

	counts, err := r.componentUsageCounts(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	applyComponentUsageCounts(result, counts)
	return result, nil
}

func (r *PostgresAdminGovernanceRepository) componentUsageCounts(
	ctx context.Context,
	tenantID string,
) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, componentUsageCountsQuery, tenantID, domain.DefaultTenantID)
	if err != nil {
		return nil, fmt.Errorf("error querying component usage counts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	counts := make(map[string]int)
	for rows.Next() {
		var ref string
		var count int
		if err := rows.Scan(&ref, &count); err != nil {
			return nil, fmt.Errorf("error scanning component usage row: %w", err)
		}
		counts[ref] = count
	}
	return counts, rows.Err()
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
			COALESCE(dependency_profile, '{}'::jsonb) AS dependency_profile,
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
	profileJSON, errProfile := json.Marshal(comp.DependencyProfile)
	if errProfile != nil || len(profileJSON) == 0 || string(profileJSON) == "null" {
		profileJSON = []byte(`{}`)
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
			service_engine, ram_delta_mb, is_system, is_active, sort_order,
			dependency_profile
		) VALUES (
			gen_random_uuid(), $1::uuid, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, false, true, $18, $19
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
			COALESCE(dependency_profile, '{}'::jsonb) AS dependency_profile,
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
		strings.TrimSpace(comp.ServiceEngine), comp.RamDeltaMB, sortOrder, profileJSON,
	).Scan(
		&row.ID, &row.TenantID, &row.Slug, &row.Layer, &row.SubCategory, &row.Name,
		&row.Description, &row.IconKey, &row.BaseImage, &row.CompatibleRuntimes,
		&row.InstallMode, &row.PackageNames, &row.BuildLayerSnippet, &row.SetupSnippet,
		&row.JudgeEntrypoint, &row.ToolsProvided, &row.ServiceEngine, &row.RamDeltaMB,
		&row.DependencyProfile,
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
	// El catálogo global se lee desde cualquier tenant, pero solo lo edita su
	// dueño. El borrado ya exigía pertenencia; la edición ahora coincide.
	if existing.TenantID != "" && existing.TenantID != tenantID {
		return nil, domain.ErrComponentReadOnly
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
	if dto.DependencyProfile != nil {
		existing.DependencyProfile = *dto.DependencyProfile
	}

	compatJSON, _ := json.Marshal(existing.CompatibleRuntimes)
	pkgsJSON, _ := json.Marshal(existing.PackageNames)
	toolsJSON, _ := json.Marshal(existing.ToolsProvided)
	profileJSON, errProfile := json.Marshal(existing.DependencyProfile)
	if errProfile != nil || len(profileJSON) == 0 || string(profileJSON) == "null" {
		profileJSON = []byte(`{}`)
	}

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
			dependency_profile = $17,
			updated_at = NOW()
					WHERE id::text = $18 AND tenant_id::text = $19
	`
	res, err := r.db.ExecContext(
		ctx, query,
		existing.Name, existing.Description, existing.IconKey, existing.BaseImage,
		compatJSON, string(existing.InstallMode), pkgsJSON, existing.BuildLayerSnippet,
		existing.SetupSnippet, existing.JudgeEntrypoint, toolsJSON, existing.ServiceEngine,
		existing.RamDeltaMB, existing.IsActive, existing.SortOrder, existing.SubCategory, profileJSON, existing.ID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("error updating template component: %w", err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return nil, domain.ErrComponentNotFound
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
	if existing.TenantID != "" && existing.TenantID != tenantID {
		return domain.ErrComponentReadOnly
	}

	// La guarda de uso vive en el propio DELETE: entre leer y borrar no queda
	// ventana para que una plantilla empiece a referenciar el componente.
	query := fmt.Sprintf(`
		DELETE FROM template_components tc
		WHERE tc.id::text = $1
		  AND tc.tenant_id::text = $2
		  AND tc.is_system = false
		  AND NOT EXISTS (
			SELECT 1 FROM lab_templates lt
			WHERE COALESCE(lt.status, '') <> 'RECHAZADA'
			  AND (lt.tenant_id IS NULL OR lt.tenant_id::text = $2 OR lt.tenant_id::text = $3)
			  AND %s
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM template_models tm
			WHERE (tm.tenant_id IS NULL OR tm.tenant_id::text = $2 OR tm.tenant_id::text = $3)
			  AND %s
		  )
	`, componentReferencesSQL("lt"), componentReferencesSQL("tm"))

	res, err := r.db.ExecContext(ctx, query, existing.ID, tenantID, domain.DefaultTenantID)
	if err != nil {
		return fmt.Errorf("error deleting template component: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		// La fila existe, no es del sistema y pertenece al tenant: si el DELETE no
		// encontró nada, es porque alguna plantilla o modelo la referencia.
		return domain.ErrComponentInUse
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

	// Un cambio de imagen, cuota, modalidad, herramientas o receta invalida la
	// auditoría anterior. La plantilla vuelve a la cola salvo que esté pausada o
	// suspendida: esos estados los decide una persona, no el worker.
	technicalChange := `(COALESCE(docker_image, '')::varchar <> $2::varchar
		OR COALESCE(base_ram_mb, 0) <> $3
		OR COALESCE(target_environment, 'IDE_PERSISTENTE')::varchar <> $5::varchar
		OR COALESCE(tools_declared, '[]'::jsonb) <> $13::jsonb
		OR COALESCE(stack_config, '{}'::jsonb) <> $14::jsonb)`
	query := fmt.Sprintf(`
		UPDATE lab_templates
		SET
			name = $1,
			docker_image = $2::varchar,
			base_ram_mb = $3,
			description = $4,
			target_environment = $5::varchar,
			entrypoint = $6,
			timeout_ms = $7,
			sample_input = $8,
			category_id = NULLIF($9, '')::uuid,
			services_config = $10,
			resource_profile = $11,
			setup_script = $12,
			tools_declared = $13,
			stack_config = $14,
			status = CASE
				WHEN %s AND status NOT IN ('PAUSADA', 'SUSPENDIDA', 'paused', 'suspended') THEN 'PENDIENTE_AUDITORIA'
				ELSE status
			END,
			smoke_test_status = CASE WHEN %s THEN 'pending' ELSE smoke_test_status END,
			smoke_test_output = CASE WHEN %s THEN '' ELSE smoke_test_output END,
			security_audit_status = CASE WHEN %s THEN 'pending' ELSE security_audit_status END,
			reviewed_by = NULLIF($15, '')::uuid,
			updated_at = NOW()
		WHERE id::text = $16 AND (tenant_id::text = $17 OR tenant_id IS NULL)
	`, technicalChange, technicalChange, technicalChange, technicalChange)
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
