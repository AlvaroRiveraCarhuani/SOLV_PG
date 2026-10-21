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
	ID                  string     `db:"id"`
	TenantID            *string    `db:"tenant_id"`
	Name                string     `db:"name"`
	DockerImage         string     `db:"docker_image"`
	BaseRamMB           int        `db:"base_ram_mb"`
	Status              string     `db:"status"`
	RejectionReason     string     `db:"rejection_reason"`
	ReviewedBy          *string    `db:"reviewed_by"`
	ReviewedAt          *time.Time `db:"reviewed_at"`
	RequestedBy         *string    `db:"requested_by"`
	RequestedByName     *string    `db:"requested_by_name"`
	Description         string     `db:"description"`
	TargetEnvironment   string     `db:"target_environment"`
	Entrypoint          string     `db:"entrypoint"`
	TimeoutMS           int        `db:"timeout_ms"`
	SampleInput         string     `db:"sample_input"`
	CategoryID          *string    `db:"category_id"`
	CategoryName        *string    `db:"category_name"`
	ModelID             *string    `db:"model_id"`
	ServicesConfig      []byte     `db:"services_config"`
	ResourceProfile     []byte     `db:"resource_profile"`
	SetupScript         string     `db:"setup_script"`
	ToolsDeclared       []byte     `db:"tools_declared"`
	StackConfig         []byte     `db:"stack_config"`
	SmokeTestStatus     string     `db:"smoke_test_status"`
	SmokeTestOutput     string     `db:"smoke_test_output"`
	SecurityAuditStatus string     `db:"security_audit_status"`
	CVECriticalCount    int        `db:"cve_critical_count"`
	CVEHighCount        int        `db:"cve_high_count"`
	SecurityAuditedAt   *time.Time `db:"security_audited_at"`
	EOLStatus           string     `db:"eol_status"`
	EOLDate             string     `db:"eol_date"`
	EOLMessage          string     `db:"eol_message"`
	EOLCheckedAt        *time.Time `db:"eol_checked_at"`
	CreatedAt           time.Time  `db:"created_at"`
}

func (row *adminTemplateRow) toDomain() *domain.AdminTemplateReviewItem {
	item := &domain.AdminTemplateReviewItem{
		ID:                  row.ID,
		TenantID:            row.TenantID,
		Name:                row.Name,
		DockerImage:         row.DockerImage,
		BaseRamMB:           row.BaseRamMB,
		Status:              row.Status,
		RejectionReason:     row.RejectionReason,
		ReviewedBy:          row.ReviewedBy,
		ReviewedAt:          row.ReviewedAt,
		RequestedBy:         row.RequestedBy,
		RequestedByName:     row.RequestedByName,
		Description:         row.Description,
		TargetEnvironment:   row.TargetEnvironment,
		Entrypoint:          row.Entrypoint,
		TimeoutMS:           row.TimeoutMS,
		SampleInput:         row.SampleInput,
		CategoryID:          row.CategoryID,
		CategoryName:        row.CategoryName,
		ModelID:             row.ModelID,
		SetupScript:         row.SetupScript,
		SmokeTestStatus:     row.SmokeTestStatus,
		SmokeTestOutput:     row.SmokeTestOutput,
		SecurityAuditStatus: row.SecurityAuditStatus,
		CVECriticalCount:    row.CVECriticalCount,
		CVEHighCount:        row.CVEHighCount,
		SecurityAuditedAt:   row.SecurityAuditedAt,
		EOLStatus:           row.EOLStatus,
		EOLDate:             row.EOLDate,
		EOLMessage:          row.EOLMessage,
		EOLCheckedAt:        row.EOLCheckedAt,
		CreatedAt:           row.CreatedAt,
		ToolsDeclared:       []string{},
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
			lt.created_at
		FROM lab_templates lt
		LEFT JOIN users u ON u.id = lt.requested_by
		LEFT JOIN template_categories tc ON tc.id = lt.category_id
		WHERE (lt.tenant_id = $1 OR lt.tenant_id IS NULL)
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
	return list, nil
}

func (r *PostgresAdminGovernanceRepository) ReviewTemplate(
	ctx context.Context,
	tenantID, templateID, adminID, status, rejectionReason string,
	baseRamMB *int,
) (*domain.AdminTemplateReviewItem, error) {
	query := `
		UPDATE lab_templates
		SET 
			status = $1,
			rejection_reason = $2,
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
		WHERE id = $5 AND (tenant_id = $6 OR tenant_id IS NULL)
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
	initialStatus := "PENDIENTE_AUDITORIA"
	initialSmokeStatus := "pending"
	initialSmokeOutput := ""
	if dto.StackConfig != nil {
		if b, errM := json.Marshal(dto.StackConfig); errM == nil {
			stackConfigJSON = b
		}
		initialStatus = "APROBADA"
		initialSmokeStatus = "passed"
		initialSmokeOutput = "Verificado en Constructor de Entornos"
	}

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
			cve_critical_count, cve_high_count, eol_status, eol_date, eol_message, reviewed_by, reviewed_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $18, $5,
			$6, $7, $8, $9, NULLIF($10, '')::uuid, NULLIF($11, '')::uuid,
			$12, $13, $14,
			$15, $16, $19, $20, 'pending',
			0, 0, 'supported', '', '', NULLIF($17, '')::uuid, NOW()
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

	return row.toDomain(), nil
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
		WHERE lt.status = 'PENDIENTE_AUDITORIA' OR lt.security_audit_status = 'pending'
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
	return list, nil
}
