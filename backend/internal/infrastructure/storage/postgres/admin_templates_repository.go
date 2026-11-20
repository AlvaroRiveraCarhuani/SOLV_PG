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

// approvedTemplateStatusClause helper único de normalización de estado de plantillas aprobadas.
func approvedTemplateStatusClause(alias string) string {
	if alias != "" {
		return fmt.Sprintf("(%s.status = 'approved' OR %s.status = 'APROBADA')", alias, alias)
	}
	return "(status = 'approved' OR status = 'APROBADA')"
}

func isTemplateApproved(status string) bool {
	return status == "approved" || status == "APROBADA"
}

type adminTemplateRow struct {
	ID                     string     `db:"id"`
	TenantID               *string    `db:"tenant_id"`
	Name                   string     `db:"name"`
	DockerImage            string     `db:"docker_image"`
	BaseRamMB              int        `db:"base_ram_mb"`
	Status                 string     `db:"status"`
	RejectionReason        string     `db:"rejection_reason"`
	ReviewedBy             *string    `db:"reviewed_by"`
	ReviewedAt             *time.Time `db:"reviewed_at"`
	RequestedBy            *string    `db:"requested_by"`
	RequestedByName        *string    `db:"requested_by_name"`
	Description            string     `db:"description"`
	TargetEnvironment      string     `db:"target_environment"`
	Entrypoint             string     `db:"entrypoint"`
	TimeoutMS              int        `db:"timeout_ms"`
	SampleInput            string     `db:"sample_input"`
	CategoryID             *string    `db:"category_id"`
	CategoryName           *string    `db:"category_name"`
	ModelID                *string    `db:"model_id"`
	ServicesConfig         []byte     `db:"services_config"`
	ResourceProfile        []byte     `db:"resource_profile"`
	SetupScript            string     `db:"setup_script"`
	ToolsDeclared          []byte     `db:"tools_declared"`
	StackConfig            []byte     `db:"stack_config"`
	SmokeTestStatus        string     `db:"smoke_test_status"`
	SmokeTestOutput        string     `db:"smoke_test_output"`
	SecurityAuditStatus    string     `db:"security_audit_status"`
	CVECriticalCount       int        `db:"cve_critical_count"`
	CVEHighCount           int        `db:"cve_high_count"`
	SecurityAuditedAt      *time.Time `db:"security_audited_at"`
	EOLStatus              string     `db:"eol_status"`
	EOLDate                string     `db:"eol_date"`
	EOLMessage             string     `db:"eol_message"`
	EOLCheckedAt           *time.Time `db:"eol_checked_at"`
	CreatedAt              time.Time  `db:"created_at"`
	PublishedExerciseCount int        `db:"published_exercise_count"`
}

func (row *adminTemplateRow) toDomain() *domain.AdminTemplateReviewItem {
	item := &domain.AdminTemplateReviewItem{
		ID:                     row.ID,
		TenantID:               row.TenantID,
		Name:                   row.Name,
		DockerImage:            row.DockerImage,
		BaseRamMB:              row.BaseRamMB,
		Status:                 row.Status,
		RejectionReason:        row.RejectionReason,
		ReviewedBy:             row.ReviewedBy,
		ReviewedAt:             row.ReviewedAt,
		RequestedBy:            row.RequestedBy,
		RequestedByName:        row.RequestedByName,
		Description:            row.Description,
		TargetEnvironment:      row.TargetEnvironment,
		Entrypoint:             row.Entrypoint,
		TimeoutMS:              row.TimeoutMS,
		SampleInput:            row.SampleInput,
		CategoryID:             row.CategoryID,
		CategoryName:           row.CategoryName,
		ModelID:                row.ModelID,
		SetupScript:            row.SetupScript,
		SmokeTestStatus:        row.SmokeTestStatus,
		SmokeTestOutput:        row.SmokeTestOutput,
		SecurityAuditStatus:    row.SecurityAuditStatus,
		CVECriticalCount:       row.CVECriticalCount,
		CVEHighCount:           row.CVEHighCount,
		SecurityAuditedAt:      row.SecurityAuditedAt,
		EOLStatus:              row.EOLStatus,
		EOLDate:                row.EOLDate,
		EOLMessage:             row.EOLMessage,
		EOLCheckedAt:           row.EOLCheckedAt,
		CreatedAt:              row.CreatedAt,
		PublishedExerciseCount: row.PublishedExerciseCount,
		ToolsDeclared:          []string{},
		LanguageRunners:        []domain.LanguageRunner{},
	}
	if len(row.ServicesConfig) > 0 {
		_ = json.Unmarshal(row.ServicesConfig, &item.ServicesConfig)
	}
	if len(row.ResourceProfile) > 0 {
		_ = json.Unmarshal(row.ResourceProfile, &item.ResourceProfile)
	}
	if len(row.ToolsDeclared) > 0 {
		_ = json.Unmarshal(row.ToolsDeclared, &item.ToolsDeclared)
	}
	if len(row.StackConfig) > 0 {
		_ = json.Unmarshal(row.StackConfig, &item.StackConfig)
	}
	return item
}

func (r *PostgresAdminGovernanceRepository) ListTemplates(
	ctx context.Context,
	tenantID, status, search string,
) ([]*domain.AdminTemplateReviewItem, error) {
	baseQuery := `
		SELECT 
			lt.id,
			lt.tenant_id,
			lt.name,
			lt.docker_image,
			lt.base_ram_mb,
			COALESCE(lt.status, 'approved') AS status,
			COALESCE(lt.rejection_reason, '') AS rejection_reason,
			lt.reviewed_by,
			lt.reviewed_at,
			lt.requested_by,
			NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), '') AS requested_by_name,
			COALESCE(lt.description, '') AS description,
			COALESCE(lt.target_environment, 'IDE_PERSISTENTE') AS target_environment,
			COALESCE(lt.entrypoint, '') AS entrypoint,
			COALESCE(lt.timeout_ms, 5000) AS timeout_ms,
			COALESCE(lt.sample_input, '') AS sample_input,
			lt.category_id::text,
			COALESCE(tc.name, '') AS category_name,
			lt.model_id::text,
			COALESCE(lt.services_config, '{"services": []}'::jsonb) AS services_config,
			COALESCE(lt.resource_profile, '{"min_mb": 256, "high_mb": 768, "max_mb": 1024}'::jsonb) AS resource_profile,
			COALESCE(lt.setup_script, '') AS setup_script,
			COALESCE(lt.tools_declared, '[]'::jsonb) AS tools_declared,
			COALESCE(lt.stack_config, '{}'::jsonb) AS stack_config,
			COALESCE(lt.smoke_test_status, 'pending') AS smoke_test_status,
			COALESCE(lt.smoke_test_output, '') AS smoke_test_output,
			COALESCE(lt.security_audit_status, 'pending') AS security_audit_status,
			COALESCE(lt.cve_critical_count, 0) AS cve_critical_count,
			COALESCE(lt.cve_high_count, 0) AS cve_high_count,
			COALESCE(lt.eol_status, 'supported') AS eol_status,
			COALESCE(lt.eol_date, '') AS eol_date,
			COALESCE(lt.eol_message, '') AS eol_message,
			lt.eol_checked_at,
			lt.security_audited_at,
			lt.created_at,
			(SELECT COUNT(*)
			 FROM exercises e
			 WHERE e.template_id = lt.id
			   AND e.status = 'published'
			   AND ($1 = '' OR lt.tenant_id IS NULL OR e.tenant_id = NULLIF($1, '')::uuid)) AS published_exercise_count
		FROM lab_templates lt
		LEFT JOIN users u ON u.id = lt.requested_by
		LEFT JOIN template_categories tc ON tc.id = lt.category_id
		WHERE ($1 = '' OR lt.tenant_id = NULLIF($1, '')::uuid OR lt.tenant_id IS NULL)
	`
	args := []interface{}{tenantID}
	argIdx := 2

	if status != "" {
		if isTemplateApproved(status) {
			baseQuery += fmt.Sprintf(` AND %s`, approvedTemplateStatusClause("lt"))
		} else {
			baseQuery += fmt.Sprintf(` AND lt.status = $%d`, argIdx)
			args = append(args, status)
			argIdx++
		}
	}

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		baseQuery += fmt.Sprintf(` AND (LOWER(lt.name) LIKE $%d OR LOWER(lt.docker_image) LIKE $%d OR LOWER(u.first_name) LIKE $%d OR LOWER(u.last_name) LIKE $%d)`, argIdx, argIdx, argIdx, argIdx)
		args = append(args, searchPattern)
	}

	baseQuery += ` ORDER BY lt.created_at DESC`

	var rows []adminTemplateRow
	err := r.db.SelectContext(ctx, &rows, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error listing templates: %w", err)
	}

	list := make([]*domain.AdminTemplateReviewItem, len(rows))
	for i := range rows {
		list[i] = rows[i].toDomain()
	}
	if err := r.attachTemplateLanguageRunners(ctx, list); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *PostgresAdminGovernanceRepository) ReviewTemplate(
	ctx context.Context,
	tenantID, templateID, adminID, status, rejectionReason string,
	baseRamMB *int,
) (*domain.AdminTemplateReviewItem, error) {
	// Re-aprobar una plantilla rechazada no la devuelve al catálogo: la
	// reingresa a la cola con el smoke y Trivy limpios. El worker decide si
	// vuelve a APROBADA o si el rechazo se mantiene.
	query := `
		UPDATE lab_templates
		SET 
			status = CASE
				WHEN $1 IN ('approved', 'APROBADA') AND status IN ('RECHAZADA', 'rejected') THEN 'PENDIENTE_AUDITORIA'
				ELSE $1
			END,
			rejection_reason = CASE
				WHEN $1 IN ('approved', 'APROBADA') AND status IN ('RECHAZADA', 'rejected') THEN ''
				ELSE $2
			END,
			smoke_test_status = CASE
				WHEN $1 IN ('approved', 'APROBADA') AND status IN ('RECHAZADA', 'rejected') THEN 'pending'
				ELSE smoke_test_status
			END,
			smoke_test_output = CASE
				WHEN $1 IN ('approved', 'APROBADA') AND status IN ('RECHAZADA', 'rejected') THEN ''
				ELSE smoke_test_output
			END,
			security_audit_status = CASE
				WHEN $1 IN ('approved', 'APROBADA') AND status IN ('RECHAZADA', 'rejected') THEN 'pending'
				ELSE security_audit_status
			END,
			cve_critical_count = CASE
				WHEN $1 IN ('approved', 'APROBADA') AND status IN ('RECHAZADA', 'rejected') THEN 0
				ELSE cve_critical_count
			END,
			cve_high_count = CASE
				WHEN $1 IN ('approved', 'APROBADA') AND status IN ('RECHAZADA', 'rejected') THEN 0
				ELSE cve_high_count
			END,
			security_audit_report = CASE
				WHEN $1 IN ('approved', 'APROBADA') AND status IN ('RECHAZADA', 'rejected') THEN '{}'::jsonb
				ELSE security_audit_report
			END,
			security_audited_at = CASE
				WHEN $1 IN ('approved', 'APROBADA') AND status IN ('RECHAZADA', 'rejected') THEN NULL
				ELSE security_audited_at
			END,
			reviewed_by = $3,
			reviewed_at = NOW(),
			base_ram_mb = COALESCE($4, base_ram_mb),
			resource_profile = CASE 
				WHEN $4::int IS NOT NULL AND $4::int > 0 THEN jsonb_build_object(
					'min_mb', ($4::int / 2),
					'high_mb', (($4::int * 3) / 2),
					'max_mb', ($4::int * 2)
				)
				ELSE resource_profile
			END
		WHERE id = $5 AND ($6 = '' OR tenant_id = NULLIF($6, '')::uuid OR tenant_id IS NULL)
		RETURNING 
			id,
			tenant_id,
			name,
			docker_image,
			base_ram_mb,
			status,
			rejection_reason,
			reviewed_by,
			reviewed_at,
			requested_by,
			COALESCE(description, '') AS description,
			COALESCE(target_environment, 'IDE_PERSISTENTE') AS target_environment,
			COALESCE(entrypoint, '') AS entrypoint,
			COALESCE(timeout_ms, 5000) AS timeout_ms,
			COALESCE(sample_input, '') AS sample_input,
			category_id::text,
			model_id::text,
			COALESCE(services_config, '{"services": []}'::jsonb) AS services_config,
			COALESCE(resource_profile, '{"min_mb": 256, "high_mb": 768, "max_mb": 1024}'::jsonb) AS resource_profile,
			COALESCE(setup_script, '') AS setup_script,
			COALESCE(tools_declared, '[]'::jsonb) AS tools_declared,
			COALESCE(stack_config, '{}'::jsonb) AS stack_config,
			COALESCE(smoke_test_status, 'pending') AS smoke_test_status,
			COALESCE(smoke_test_output, '') AS smoke_test_output,
			COALESCE(security_audit_status, 'pending') AS security_audit_status,
			COALESCE(cve_critical_count, 0) AS cve_critical_count,
			COALESCE(cve_high_count, 0) AS cve_high_count,
			COALESCE(eol_status, 'supported') AS eol_status,
			COALESCE(eol_date, '') AS eol_date,
			COALESCE(eol_message, '') AS eol_message,
			eol_checked_at,
			security_audited_at,
			created_at
	`

	var row adminTemplateRow
	err := r.db.QueryRowContext(
		ctx,
		query,
		status,
		rejectionReason,
		adminID,
		baseRamMB,
		templateID,
		tenantID,
	).Scan(
		&row.ID,
		&row.TenantID,
		&row.Name,
		&row.DockerImage,
		&row.BaseRamMB,
		&row.Status,
		&row.RejectionReason,
		&row.ReviewedBy,
		&row.ReviewedAt,
		&row.RequestedBy,
		&row.Description,
		&row.TargetEnvironment,
		&row.Entrypoint,
		&row.TimeoutMS,
		&row.SampleInput,
		&row.CategoryID,
		&row.ModelID,
		&row.ServicesConfig,
		&row.ResourceProfile,
		&row.SetupScript,
		&row.ToolsDeclared,
		&row.StackConfig,
		&row.SmokeTestStatus,
		&row.SmokeTestOutput,
		&row.SecurityAuditStatus,
		&row.CVECriticalCount,
		&row.CVEHighCount,
		&row.EOLStatus,
		&row.EOLDate,
		&row.EOLMessage,
		&row.EOLCheckedAt,
		&row.SecurityAuditedAt,
		&row.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("error reviewing template: %w", err)
	}

	return row.toDomain(), nil
}

// UpdateTemplateQuota ajusta memoria y descripción sin tocar la imagen. No pide
// un job de build porque no hay build: la comprobación de arranque se repite en
// segundo plano cuando la plantilla está aprobada.
func (r *PostgresAdminGovernanceRepository) UpdateTemplateQuota(
	ctx context.Context,
	tenantID, templateID, adminID string,
	dto domain.UpdateTemplateQuotaDTO,
) (*domain.AdminTemplateReviewItem, error) {
	profile := domain.DeriveResourceProfile(dto.BaseRamMB)
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		profileJSON = []byte(`{"min_mb": 256, "high_mb": 768, "max_mb": 1024}`)
	}
	var description interface{}
	if dto.Description != nil {
		description = strings.TrimSpace(*dto.Description)
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE lab_templates
		SET base_ram_mb = $1,
		    resource_profile = $2,
		    description = COALESCE($3, description),
		    smoke_test_status = CASE WHEN $4 THEN 'pending' ELSE smoke_test_status END,
		    smoke_test_output = CASE WHEN $4 THEN '' ELSE smoke_test_output END,
		    reviewed_by = NULLIF($5, '')::uuid,
		    reviewed_at = NOW(),
		    updated_at = NOW()
		WHERE id::text = $6 AND ($7 = '' OR tenant_id = NULLIF($7, '')::uuid OR tenant_id IS NULL)
	`, dto.BaseRamMB, profileJSON, description, dto.ResetSmoke, adminID, templateID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error updating template quota: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, errors.New("template not found")
	}
	return r.GetTemplateByID(ctx, templateID)
}

func (r *PostgresAdminGovernanceRepository) CreateOfficialTemplate(
	ctx context.Context,
	tenantID, adminID string,
	dto domain.CreateOfficialTemplateDTO,
) (*domain.AdminTemplateReviewItem, error) {
	servicesConfigJSON, err := json.Marshal(dto.ServicesConfig)
	if err != nil {
		servicesConfigJSON = []byte(`{"services": []}`)
	}
	derivedProfile := domain.DeriveResourceProfile(dto.BaseRamMB)
	resourceProfileJSON, err := json.Marshal(derivedProfile)
	if err != nil {
		resourceProfileJSON = []byte(`{"min_mb": 256, "high_mb": 768, "max_mb": 1024}`)
	}
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
	initialStatus := domain.InitialTemplateStatus(dto.DockerImage, dto.StackConfig != nil)
	// El alta no declara una comprobación que no ocurrió: el worker de
	// auditoría escribe el resultado real del smoke test y de Trivy.
	initialSmokeStatus := "pending"
	initialSmokeOutput := ""

	targetEnv := dto.TargetEnvironment
	if targetEnv == "" {
		targetEnv = "IDE_PERSISTENTE"
	}
	timeoutMS := dto.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 5000
	}

	query := `
		INSERT INTO lab_templates (
			id, tenant_id, name, docker_image, base_ram_mb, status, description, 
			target_environment, entrypoint, timeout_ms, sample_input, category_id, model_id,
			services_config, resource_profile, setup_script,
			tools_declared, stack_config, smoke_test_status, smoke_test_output, security_audit_status,
			cve_critical_count, cve_high_count, eol_status, eol_date, eol_message,
			requested_by, reviewed_by, reviewed_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $18, $5,
			$6, $7, $8, $9, NULLIF($10, '')::uuid, NULLIF($11, '')::uuid,
			$12, $13, $14,
			$15, $16, $19, $20, 'pending',
			0, 0, 'supported', '', '',
			NULLIF($17, '')::uuid, NULL, NULL
		)
		ON CONFLICT (tenant_id, name) DO NOTHING
		RETURNING 
			id,
			tenant_id,
			name,
			docker_image,
			base_ram_mb,
			status,
			rejection_reason,
			reviewed_by,
			reviewed_at,
			requested_by,
			COALESCE(description, '') AS description,
			COALESCE(target_environment, 'IDE_PERSISTENTE') AS target_environment,
			COALESCE(entrypoint, '') AS entrypoint,
			COALESCE(timeout_ms, 5000) AS timeout_ms,
			COALESCE(sample_input, '') AS sample_input,
			category_id::text,
			model_id::text,
			COALESCE(services_config, '{"services": []}'::jsonb) AS services_config,
			COALESCE(resource_profile, '{"min_mb": 256, "high_mb": 768, "max_mb": 1024}'::jsonb) AS resource_profile,
			COALESCE(setup_script, '') AS setup_script,
			COALESCE(tools_declared, '[]'::jsonb) AS tools_declared,
			COALESCE(stack_config, '{}'::jsonb) AS stack_config,
			COALESCE(smoke_test_status, 'pending') AS smoke_test_status,
			COALESCE(smoke_test_output, '') AS smoke_test_output,
			COALESCE(security_audit_status, 'pending') AS security_audit_status,
			COALESCE(cve_critical_count, 0) AS cve_critical_count,
			COALESCE(cve_high_count, 0) AS cve_high_count,
			COALESCE(eol_status, 'supported') AS eol_status,
			COALESCE(eol_date, '') AS eol_date,
			COALESCE(eol_message, '') AS eol_message,
			eol_checked_at,
			security_audited_at,
			created_at
	`

	var row adminTemplateRow
	err = r.db.QueryRowContext(
		ctx,
		query,
		tenantID,
		dto.Name,
		dto.DockerImage,
		dto.BaseRamMB,
		dto.Description,
		targetEnv,
		dto.Entrypoint,
		timeoutMS,
		dto.SampleInput,
		dto.CategoryID,
		dto.ModelID,
		servicesConfigJSON,
		resourceProfileJSON,
		dto.SetupScript,
		toolsDeclaredJSON,
		stackConfigJSON,
		adminID,
		initialStatus,
		initialSmokeStatus,
		initialSmokeOutput,
	).Scan(
		&row.ID,
		&row.TenantID,
		&row.Name,
		&row.DockerImage,
		&row.BaseRamMB,
		&row.Status,
		&row.RejectionReason,
		&row.ReviewedBy,
		&row.ReviewedAt,
		&row.RequestedBy,
		&row.Description,
		&row.TargetEnvironment,
		&row.Entrypoint,
		&row.TimeoutMS,
		&row.SampleInput,
		&row.CategoryID,
		&row.ModelID,
		&row.ServicesConfig,
		&row.ResourceProfile,
		&row.SetupScript,
		&row.ToolsDeclared,
		&row.StackConfig,
		&row.SmokeTestStatus,
		&row.SmokeTestOutput,
		&row.SecurityAuditStatus,
		&row.CVECriticalCount,
		&row.CVEHighCount,
		&row.EOLStatus,
		&row.EOLDate,
		&row.EOLMessage,
		&row.EOLCheckedAt,
		&row.SecurityAuditedAt,
		&row.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTemplateNameConflict
		}
		return nil, fmt.Errorf("error creating official template: %w", err)
	}

	created := row.toDomain()
	if len(dto.LanguageRunners) > 0 {
		if err := r.insertTemplateLanguageRunners(ctx, created.ID, dto.LanguageRunners); err != nil {
			// La plantilla sin sus ejecutores no es un entorno válido: se deshace
			// el alta en lugar de dejar un entorno a medias.
			_, _ = r.db.ExecContext(ctx, `DELETE FROM lab_templates WHERE id = $1`, created.ID)
			return nil, err
		}
		created.LanguageRunners = dto.LanguageRunners
	}
	return created, nil
}

func (r *PostgresAdminGovernanceRepository) ListPendingAuditTemplates(ctx context.Context) ([]*domain.AdminTemplateReviewItem, error) {
	query := `
		SELECT 
			lt.id,
			lt.tenant_id,
			lt.name,
			lt.docker_image,
			lt.base_ram_mb,
			COALESCE(lt.status, 'PENDIENTE_AUDITORIA') AS status,
			COALESCE(lt.rejection_reason, '') AS rejection_reason,
			lt.reviewed_by,
			lt.reviewed_at,
			lt.requested_by,
			NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), '') AS requested_by_name,
			COALESCE(lt.description, '') AS description,
			COALESCE(lt.target_environment, 'IDE_PERSISTENTE') AS target_environment,
			COALESCE(lt.entrypoint, '') AS entrypoint,
			COALESCE(lt.timeout_ms, 5000) AS timeout_ms,
			COALESCE(lt.sample_input, '') AS sample_input,
			lt.category_id::text AS category_id,
			lt.model_id::text AS model_id,
			COALESCE(lt.services_config, '{"services": []}'::jsonb) AS services_config,
			COALESCE(lt.resource_profile, '{"min_mb": 256, "high_mb": 768, "max_mb": 1024}'::jsonb) AS resource_profile,
			COALESCE(lt.setup_script, '') AS setup_script,
			COALESCE(lt.tools_declared, '[]'::jsonb) AS tools_declared,
			COALESCE(lt.stack_config, '{}'::jsonb) AS stack_config,
			COALESCE(lt.smoke_test_status, 'pending') AS smoke_test_status,
			COALESCE(lt.smoke_test_output, '') AS smoke_test_output,
			COALESCE(lt.security_audit_status, 'pending') AS security_audit_status,
			COALESCE(lt.cve_critical_count, 0) AS cve_critical_count,
			COALESCE(lt.cve_high_count, 0) AS cve_high_count,
			COALESCE(lt.eol_status, 'supported') AS eol_status,
			COALESCE(lt.eol_date, '') AS eol_date,
			COALESCE(lt.eol_message, '') AS eol_message,
			lt.eol_checked_at,
			lt.security_audited_at,
			lt.created_at
		FROM lab_templates lt
		LEFT JOIN users u ON u.id = lt.requested_by
		WHERE lt.status = 'PENDIENTE_AUDITORIA'
		   OR lt.security_audit_status IN ('pending', 'error')
		   OR lt.smoke_test_status IN ('pending', 'error')
		ORDER BY lt.created_at ASC
	`
	var rows []adminTemplateRow
	err := r.db.SelectContext(ctx, &rows, query)
	if err != nil {
		return nil, fmt.Errorf("error listing pending audit templates: %w", err)
	}

	list := make([]*domain.AdminTemplateReviewItem, len(rows))
	for i := range rows {
		list[i] = rows[i].toDomain()
	}
	if err := r.attachTemplateLanguageRunners(ctx, list); err != nil {
		return nil, err
	}
	return list, nil
}
