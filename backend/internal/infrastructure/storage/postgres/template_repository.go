package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"solv-backend/internal/core/domain"
)

type PostgresTemplateRepository struct {
	db *sqlx.DB
}

func NewPostgresTemplateRepository(db *sqlx.DB) *PostgresTemplateRepository {
	return &PostgresTemplateRepository{db: db}
}

func (r *PostgresTemplateRepository) GetTemplateByID(ctx context.Context, id string) (*domain.Template, error) {
	tenantID := domain.GetTenantID(ctx)
	query := `SELECT id, name, docker_image, base_ram_mb FROM lab_templates WHERE id = $1 AND tenant_id = $2`
	var t domain.Template
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(&t.ID, &t.Name, &t.DockerImage, &t.BaseRamMB)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("error querying template: %w", err)
	}
	return &t, nil
}
