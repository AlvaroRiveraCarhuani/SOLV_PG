package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

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
			COALESCE(setup_script, '') AS setup_script,
			COALESCE(stack_config, '{}'::jsonb) AS stack_config
		FROM lab_templates 
		WHERE id = $1 AND (tenant_id = $2 OR tenant_id IS NULL)
	`
	var t domain.Template
	err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
		&t.ID, &t.Name, &t.DockerImage, &t.BaseRamMB, &t.ServicesConfig, &t.SetupScript, &t.StackConfig,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("error querying template: %w", err)
	}
	return &t, nil
}

// GetDependencyProfileForStack resuelve el perfil de dependencias del primer
// componente del stack que lo declara: el runtime manda, después la plataforma
// web y por último los complementos. La exportación lo usa para escribir en el
// README el comando de instalación real de la familia tecnológica.
func (r *PostgresTemplateRepository) GetDependencyProfileForStack(ctx context.Context, slugs []string) (domain.DependencyProfile, bool, error) {
	cleaned := make([]string, 0, len(slugs))
	seen := make(map[string]bool, len(slugs))
	for _, raw := range slugs {
		slug := strings.TrimSpace(raw)
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true
		cleaned = append(cleaned, slug)
	}
	if len(cleaned) == 0 {
		return domain.DependencyProfile{}, false, nil
	}

	tenantID := domain.GetTenantID(ctx)
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	query := `
		SELECT COALESCE(dependency_profile, '{}'::jsonb)
		FROM template_components
		WHERE tenant_id = $1
		  AND slug = ANY($2)
		  AND COALESCE(dependency_profile, '{}'::jsonb) <> '{}'::jsonb
		ORDER BY CASE layer WHEN 'runtime' THEN 0 WHEN 'addon' THEN 1 ELSE 2 END, sort_order ASC
		LIMIT 1
	`
	var raw []byte
	if err := r.db.QueryRowContext(ctx, query, tenantID, pq.Array(cleaned)).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return domain.DependencyProfile{}, false, nil
		}
		return domain.DependencyProfile{}, false, fmt.Errorf("error querying dependency profile: %w", err)
	}

	var profile domain.DependencyProfile
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &profile); err != nil {
			return domain.DependencyProfile{}, false, fmt.Errorf("invalid dependency profile: %w", err)
		}
	}
	if profile.IsEmpty() {
		return domain.DependencyProfile{}, false, nil
	}
	return profile, true, nil
}
