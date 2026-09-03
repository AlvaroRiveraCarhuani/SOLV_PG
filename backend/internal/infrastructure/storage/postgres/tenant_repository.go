package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

type PostgresTenantRepository struct {
	db *sqlx.DB
}

func NewPostgresTenantRepository(db *sqlx.DB) *PostgresTenantRepository {
	return &PostgresTenantRepository{db: db}
}

func (r *PostgresTenantRepository) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	query := `SELECT id, name, slug, allowed_domains, config, created_at, updated_at FROM tenants WHERE id = $1`
	var t domain.Tenant
	err := r.db.GetContext(ctx, &t, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("error querying tenant by id: %w", err)
	}
	return &t, nil
}

func (r *PostgresTenantRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	query := `SELECT id, name, slug, allowed_domains, config, created_at, updated_at FROM tenants WHERE slug = $1`
	var t domain.Tenant
	err := r.db.GetContext(ctx, &t, query, slug)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("error querying tenant by slug: %w", err)
	}
	return &t, nil
}

func (r *PostgresTenantRepository) GetAll(ctx context.Context) ([]*domain.Tenant, error) {
	query := `SELECT id, name, slug, allowed_domains, config, created_at, updated_at FROM tenants`
	var list []*domain.Tenant
	err := r.db.SelectContext(ctx, &list, query)
	if err != nil {
		return nil, fmt.Errorf("error querying all tenants: %w", err)
	}
	return list, nil
}

func (r *PostgresTenantRepository) UpdateConfig(ctx context.Context, id string, config []byte) error {
	query := `UPDATE tenants SET config = $1, updated_at = NOW() WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, config, id)
	if err != nil {
		return fmt.Errorf("error updating tenant config: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("tenant not found or not updated")
	}
	return nil
}

func (r *PostgresTenantRepository) SetMaintenance(ctx context.Context, tenantID string, enabled bool, until *time.Time, reason string) error {
	query := `UPDATE tenants SET maintenance_mode = $1, maintenance_until = $2, maintenance_reason = $3, updated_at = NOW() WHERE id = $4`
	res, err := r.db.ExecContext(ctx, query, enabled, until, reason, tenantID)
	if err != nil {
		return fmt.Errorf("error setting maintenance mode: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}

func (r *PostgresTenantRepository) GetMaintenance(ctx context.Context, tenantID string) (*domain.MaintenanceStatus, error) {
	query := `SELECT maintenance_mode, maintenance_until, maintenance_reason FROM tenants WHERE id = $1`
	var row struct {
		MaintenanceMode   bool           `db:"maintenance_mode"`
		MaintenanceUntil  *time.Time     `db:"maintenance_until"`
		MaintenanceReason sql.NullString `db:"maintenance_reason"`
	}
	err := r.db.GetContext(ctx, &row, query, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("error querying maintenance status: %w", err)
	}

	return &domain.MaintenanceStatus{
		MaintenanceMode:   row.MaintenanceMode,
		MaintenanceUntil:  row.MaintenanceUntil,
		MaintenanceReason: row.MaintenanceReason.String,
	}, nil
}
