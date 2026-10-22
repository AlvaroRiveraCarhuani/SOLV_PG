package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"solv-backend/internal/core/domain"
	"strings"
)

func (r *PostgresAdminGovernanceRepository) DuplicateTemplate(
	ctx context.Context,
	tenantID, templateID, adminID string,
) (*domain.AdminTemplateReviewItem, error) {
	// 1. Obtener la plantilla original
	origQuery := `
		SELECT name, docker_image, base_ram_mb, COALESCE(description, '') AS description,
		       COALESCE(target_environment, 'IDE_PERSISTENTE') AS target_environment,
		       COALESCE(entrypoint, '') AS entrypoint, COALESCE(timeout_ms, 5000) AS timeout_ms,
		       COALESCE(sample_input, '') AS sample_input, category_id::text, model_id::text,
		       COALESCE(services_config, '{"services": []}'::jsonb) AS services_config,
		       COALESCE(resource_profile, '{"min_mb": 256, "high_mb": 768, "max_mb": 1024}'::jsonb) AS resource_profile,
		       COALESCE(setup_script, '') AS setup_script,
		       COALESCE(tools_declared, '[]'::jsonb) AS tools_declared,
		       COALESCE(stack_config, '{}'::jsonb) AS stack_config
		FROM lab_templates
		WHERE id = $1 AND (tenant_id = $2 OR tenant_id IS NULL)
	`
	var (
		origName, dockerImg, desc, targetEnv, entrypoint, sampleInput, setupScript string
		baseRam, timeoutMS                                                         int
		catID, modelID                                                             *string
		servicesJSON, profileJSON, toolsJSON, stackJSON                            []byte
	)
	err := r.db.QueryRowContext(ctx, origQuery, templateID, tenantID).Scan(
		&origName, &dockerImg, &baseRam, &desc, &targetEnv, &entrypoint, &timeoutMS,
		&sampleInput, &catID, &modelID,
		&servicesJSON, &profileJSON, &setupScript, &toolsJSON, &stackJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("plantilla original no encontrada")
		}
		return nil, fmt.Errorf("error leyendo plantilla original: %w", err)
	}

	copyProfile := domain.DeriveResourceProfile(baseRam)
	copyProfileJSON, errMarshal := json.Marshal(copyProfile)
	if errMarshal == nil {
		profileJSON = copyProfileJSON
	}

	// 2. Determinar un nombre de copia único para evitar colisión 500 y no violar VARCHAR(100)
	reCopy := regexp.MustCompile(`^\(Copia(\s+\d+)?\)\s*`)
	cleanBase := strings.TrimSpace(reCopy.ReplaceAllString(origName, ""))
	if cleanBase == "" {
		cleanBase = origName
	}

	buildCandidate := func(num int) string {
		var prefix string
		if num <= 1 {
			prefix = "(Copia) "
		} else {
			prefix = fmt.Sprintf("(Copia %d) ", num)
		}
		maxBase := 100 - len([]rune(prefix))
		baseRunes := []rune(cleanBase)
		if len(baseRunes) > maxBase {
			baseRunes = baseRunes[:maxBase]
		}
		return prefix + string(baseRunes)
	}

	var existingNames []string
	nameQuery := `
		SELECT name FROM lab_templates 
		WHERE (tenant_id = $1 OR tenant_id IS NULL) 
		  AND (name = $2 OR name LIKE $3)
	`
	basePattern := "%" + cleanBase
	_ = r.db.SelectContext(ctx, &existingNames, nameQuery, tenantID, buildCandidate(1), basePattern)
	namesMap := make(map[string]bool)
	for _, n := range existingNames {
		namesMap[n] = true
	}

	candidateName := buildCandidate(1)
	if namesMap[candidateName] {
		for i := 2; i < 1000; i++ {
			c := buildCandidate(i)
			if !namesMap[c] {
				candidateName = c
				break
			}
		}
	}

	// 3. Insertar la copia
	insertQuery := `
		INSERT INTO lab_templates (
			id, tenant_id, name, docker_image, base_ram_mb, status, description, 
			target_environment, entrypoint, timeout_ms, sample_input, category_id, model_id,
			services_config, resource_profile, setup_script,
			tools_declared, stack_config, smoke_test_status, smoke_test_output, security_audit_status,
			cve_critical_count, cve_high_count, eol_status, eol_date, eol_message,
			reviewed_by, reviewed_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, 'PENDIENTE_AUDITORIA', $5,
			$6, $7, $8, $9, NULLIF($10, '')::uuid, NULLIF($11, '')::uuid,
			$12, $13, $14,
			$15, $16, 'pending', '', 'pending',
			0, 0, 'supported', '', '',
			NULLIF($17, '')::uuid, NOW()
		)
		RETURNING 
			id, tenant_id, name, docker_image, base_ram_mb, status, rejection_reason,
			reviewed_by, reviewed_at, requested_by,
			COALESCE(description, '') AS description,
			COALESCE(target_environment, 'IDE_PERSISTENTE') AS target_environment,
			COALESCE(entrypoint, '') AS entrypoint,
			COALESCE(timeout_ms, 5000) AS timeout_ms,
			COALESCE(sample_input, '') AS sample_input,
			category_id::text, model_id::text,
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
			eol_checked_at, security_audited_at, created_at
	`

	var row adminTemplateRow
	catParam := ""
	if catID != nil {
		catParam = *catID
	}
	modParam := ""
	if modelID != nil {
		modParam = *modelID
	}

	err = r.db.QueryRowContext(
		ctx, insertQuery,
		tenantID, candidateName, dockerImg, baseRam, desc,
		targetEnv, entrypoint, timeoutMS, sampleInput, catParam, modParam,
		servicesJSON, profileJSON, setupScript, toolsJSON, stackJSON,
		adminID,
	).Scan(
		&row.ID, &row.TenantID, &row.Name, &row.DockerImage, &row.BaseRamMB, &row.Status, &row.RejectionReason,
		&row.ReviewedBy, &row.ReviewedAt, &row.RequestedBy,
		&row.Description, &row.TargetEnvironment, &row.Entrypoint, &row.TimeoutMS, &row.SampleInput,
		&row.CategoryID, &row.ModelID,
		&row.ServicesConfig, &row.ResourceProfile, &row.SetupScript, &row.ToolsDeclared, &row.StackConfig,
		&row.SmokeTestStatus, &row.SmokeTestOutput, &row.SecurityAuditStatus,
		&row.CVECriticalCount, &row.CVEHighCount,
		&row.EOLStatus, &row.EOLDate, &row.EOLMessage,
		&row.EOLCheckedAt, &row.SecurityAuditedAt, &row.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error duplicando plantilla: %w", err)
	}

	return row.toDomain(), nil
}

func (r *PostgresAdminGovernanceRepository) UpdateEOLStatus(
	ctx context.Context,
	templateID string,
	status, eolDate, message string,
) error {
	query := `
		UPDATE lab_templates
		SET 
			eol_status = $1,
			eol_date = $2,
			eol_message = $3,
			eol_checked_at = NOW(),
			updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query, status, eolDate, message, templateID)
	if err != nil {
		return fmt.Errorf("error actualizando estado EOL: %w", err)
	}
	return nil
}

func (r *PostgresAdminGovernanceRepository) UpdateAuditResults(
	ctx context.Context,
	templateID string,
	smokeStatus, smokeOutput, secStatus string,
	cveCritical, cveHigh int,
	secReportJSON []byte,
	finalStatus string,
) error {
	if len(secReportJSON) == 0 {
		secReportJSON = []byte(`{}`)
	}
	query := `
		UPDATE lab_templates
		SET 
			smoke_test_status = $1,
			smoke_test_output = $2,
			security_audit_status = $3,
			cve_critical_count = $4,
			cve_high_count = $5,
			security_audit_report = $6,
			security_audited_at = NOW(),
			status = CAST($7 AS VARCHAR),
			rejection_reason = CASE
				WHEN CAST($7 AS VARCHAR) = 'RECHAZADA' AND COALESCE(rejection_reason, '') = '' THEN COALESCE(NULLIF($2, ''), 'Auditoría técnica de seguridad no superada')
				ELSE rejection_reason
			END,
			updated_at = NOW()
		WHERE id = $8
	`
	_, err := r.db.ExecContext(ctx, query, smokeStatus, smokeOutput, secStatus, cveCritical, cveHigh, secReportJSON, finalStatus, templateID)
	if err != nil {
		return fmt.Errorf("error updating audit results: %w", err)
	}
	return nil
}

func (r *PostgresAdminGovernanceRepository) TerminateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	query := `
		UPDATE workspaces
		SET status = 'failed', updated_at = NOW()
		WHERE tenant_id = $1 AND (status = 'running' OR status = 'pending')
	`
	res, err := r.db.ExecContext(ctx, query, tenantID)
	if err != nil {
		return 0, fmt.Errorf("error terminating all workspaces: %w", err)
	}
	rows, _ := res.RowsAffected()
	return rows, nil
}

func (r *PostgresAdminGovernanceRepository) HibernateAllWorkspaces(ctx context.Context, tenantID string) (int64, error) {
	query := `
		UPDATE workspaces
		SET status = 'hibernated', updated_at = NOW()
		WHERE tenant_id = $1 AND status = 'running'
	`
	res, err := r.db.ExecContext(ctx, query, tenantID)
	if err != nil {
		return 0, fmt.Errorf("error hibernating all workspaces: %w", err)
	}
	rows, _ := res.RowsAffected()
	return rows, nil
}
