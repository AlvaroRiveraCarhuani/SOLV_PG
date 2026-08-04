package postgres

import (
	"context"
	"database/sql"
	"fmt"

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
