package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

// PostgresTenantProvisioningRepository da de alta una institución completa.
type PostgresTenantProvisioningRepository struct {
	db *sqlx.DB
}

func NewPostgresTenantProvisioningRepository(db *sqlx.DB) *PostgresTenantProvisioningRepository {
	return &PostgresTenantProvisioningRepository{db: db}
}

// ProvisionTenant crea tenant, primer administrador, periodo y materia ancla en
// una transacción. Si algo falla no queda una institución a medio construir con
// la que nadie puede iniciar sesión.
func (r *PostgresTenantProvisioningRepository) ProvisionTenant(
	ctx context.Context,
	req domain.TenantProvisioningRequest,
) (*domain.TenantProvisioningResult, error) {
	result := &domain.TenantProvisioningResult{Slug: req.Slug}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error abriendo la transacción de alta: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existingTenant string
	err = tx.GetContext(ctx, &existingTenant, `SELECT id FROM tenants WHERE slug = $1`, req.Slug)
	switch {
	case err == nil:
		return nil, domain.ErrTenantSlugExists
	case !errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("error verificando el identificador %q: %w", req.Slug, err)
	}

	var existingUser string
	err = tx.GetContext(ctx, &existingUser, `SELECT id FROM users WHERE LOWER(email) = LOWER($1)`, req.AdminEmail)
	switch {
	case err == nil:
		return nil, domain.ErrAdminEmailExists
	case !errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("error verificando el correo del administrador: %w", err)
	}

	domainsJSON, err := json.Marshal(req.AllowedDomains)
	if err != nil {
		return nil, fmt.Errorf("error serializando los dominios permitidos: %w", err)
	}
	configJSON, err := json.Marshal(map[string]string{
		"institution_name": req.Name,
		"support_email":    req.AdminEmail,
	})
	if err != nil {
		return nil, fmt.Errorf("error serializando la configuración inicial: %w", err)
	}

	if err := tx.GetContext(ctx, &result.TenantID, `
		INSERT INTO tenants (id, name, slug, allowed_domains, config)
		VALUES (gen_random_uuid(), $1, $2, $3::jsonb, $4::jsonb)
		RETURNING id`,
		req.Name, req.Slug, string(domainsJSON), string(configJSON)); err != nil {
		return nil, fmt.Errorf("error creando la institución: %w", err)
	}

	if err := tx.GetContext(ctx, &result.AdminID, `
		INSERT INTO users (id, tenant_id, email, first_name, last_name, role, status, origin, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, 'admin', 'active', 'manual', NOW())
		RETURNING id`,
		result.TenantID, req.AdminEmail, req.AdminFirstName, req.AdminLastName); err != nil {
		return nil, fmt.Errorf("error creando el administrador: %w", err)
	}

	if err := tx.GetContext(ctx, &result.PeriodID, `
		INSERT INTO academic_periods (id, tenant_id, code, name, start_date, end_date, is_active, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, TRUE, NOW())
		RETURNING id`,
		result.TenantID, req.PeriodCode, req.PeriodName, req.PeriodStart, req.PeriodEnd); err != nil {
		return nil, fmt.Errorf("error creando el periodo académico: %w", err)
	}

	// Materia ancla: los entornos anteriores al modelo académico necesitan una
	// materia donde colgarse y el panel asume que existe.
	if err := tx.GetContext(ctx, &result.SubjectID, `
		INSERT INTO subjects (id, tenant_id, name, code, academic_period_id, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, 'Materia General', 'GEN-101', $2, NOW(), NOW())
		RETURNING id`,
		result.TenantID, result.PeriodID); err != nil {
		return nil, fmt.Errorf("error creando la materia ancla: %w", err)
	}

	var catalogID string
	if err := tx.GetContext(ctx, &catalogID, `
		INSERT INTO subject_catalog (tenant_id, code, name)
		VALUES ($1, 'GEN-101', 'Materia General')
		ON CONFLICT DO NOTHING
		RETURNING id`,
		result.TenantID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("error creando el catálogo base: %w", err)
	}
	if catalogID != "" {
		result.CatalogIDs = append(result.CatalogIDs, catalogID)
		if _, err := tx.ExecContext(ctx, `UPDATE subjects SET subject_catalog_id = $2 WHERE id = $1`, result.SubjectID, catalogID); err != nil {
			return nil, fmt.Errorf("error enlazando la materia ancla al catálogo: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error confirmando el alta de la institución: %w", err)
	}

	return result, nil
}

// ListTenants devuelve las instituciones de la instalación, para que quien
// opera sepa qué hay sin entrar a la base.
func (r *PostgresTenantProvisioningRepository) ListTenants(ctx context.Context) ([]*domain.TenantSummary, error) {
	var rows []*domain.TenantSummary
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT id, name, slug, created_at FROM tenants ORDER BY created_at`); err != nil {
		return nil, fmt.Errorf("error listando las instituciones: %w", err)
	}
	return rows, nil
}
