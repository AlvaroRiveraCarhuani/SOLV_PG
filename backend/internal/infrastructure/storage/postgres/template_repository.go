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
	// Las tareas de background (por ejemplo, el apagado de motores ociosos) leen
	// sin tenant en el contexto. La columna es uuid: mandar cadena vacía hace
	// fallar la consulta y la plantilla queda ilegible, así que sin tenant se usa
	// el del nodo, igual que el resto de los repositorios.
	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		SELECT 
			id, name, docker_image, base_ram_mb,
			COALESCE(services_config, '{"services": []}'::jsonb) AS services_config,
			COALESCE(setup_script, '') AS setup_script
		FROM lab_templates 
		WHERE id = $1 AND (tenant_id = $2 OR tenant_id IS NULL)
	`
	var t domain.Template
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&t.ID, &t.Name, &t.DockerImage, &t.BaseRamMB, &t.ServicesConfig, &t.SetupScript,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("error querying template: %w", err)
	}
	return &t, nil
}
