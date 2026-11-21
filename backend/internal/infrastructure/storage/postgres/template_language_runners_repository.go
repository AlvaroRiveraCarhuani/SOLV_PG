package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"solv-backend/internal/core/domain"
)

// Los ejecutores de lenguaje viven en template_language_runners (migración 00070).
// Un entorno de evaluación declara un ejecutor por lenguaje; el juez elige el que
// corresponde al lenguaje de la entrega.

type languageRunnerRow struct {
	TemplateID    string `db:"template_id"`
	Language      string `db:"language"`
	Entrypoint    string `db:"entrypoint"`
	DockerImage   string `db:"docker_image"`
	TimeoutMS     int    `db:"timeout_ms"`
	MemoryLimitMB int    `db:"memory_limit_mb"`
}

func (row languageRunnerRow) toDomain() domain.LanguageRunner {
	return domain.LanguageRunner{
		Language:      row.Language,
		Entrypoint:    row.Entrypoint,
		DockerImage:   row.DockerImage,
		TimeoutMS:     row.TimeoutMS,
		MemoryLimitMB: row.MemoryLimitMB,
	}
}

// replaceTemplateLanguageRunners deja el conjunto de ejecutores exactamente como
// lo recibió. Una lista vacía borra los ejecutores del entorno. El llamador
// aporta el ejecutor para que el borrado y las altas compartan la transacción de
// la edición: si un ejecutor no pasa las restricciones de la tabla, la plantilla
// conserva sus columnas y sus lenguajes en lugar de quedar a medias.
func (r *PostgresAdminGovernanceRepository) replaceTemplateLanguageRunners(
	ctx context.Context,
	exec sqlx.ExtContext,
	templateID string,
	runners []domain.LanguageRunner,
) error {
	if _, err := exec.ExecContext(ctx, `DELETE FROM template_language_runners WHERE template_id = $1`, templateID); err != nil {
		return fmt.Errorf("error limpiando ejecutores de lenguaje: %w", err)
	}
	return r.insertTemplateLanguageRunners(ctx, exec, templateID, runners)
}

func (r *PostgresAdminGovernanceRepository) insertTemplateLanguageRunners(
	ctx context.Context,
	exec sqlx.ExtContext,
	templateID string,
	runners []domain.LanguageRunner,
) error {
	for _, runner := range runners {
		_, err := exec.ExecContext(ctx, `
			INSERT INTO template_language_runners (
				template_id, language, entrypoint, docker_image, timeout_ms, memory_limit_mb
			) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (template_id, language) DO UPDATE SET
				entrypoint = EXCLUDED.entrypoint,
				docker_image = EXCLUDED.docker_image,
				timeout_ms = EXCLUDED.timeout_ms,
				memory_limit_mb = EXCLUDED.memory_limit_mb,
				updated_at = NOW()
		`, templateID, runner.Language, runner.Entrypoint, runner.DockerImage, runner.TimeoutMS, runner.MemoryLimitMB)
		if err != nil {
			return fmt.Errorf("error guardando el ejecutor %s: %w", runner.Language, err)
		}
	}
	return nil
}

func (r *PostgresAdminGovernanceRepository) loadTemplateLanguageRunners(
	ctx context.Context,
	templateID string,
) ([]domain.LanguageRunner, error) {
	rows := []languageRunnerRow{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT template_id::text AS template_id, language, entrypoint, docker_image, timeout_ms, memory_limit_mb
		FROM template_language_runners
		WHERE template_id = $1
		ORDER BY language
	`, templateID)
	if err != nil {
		return nil, fmt.Errorf("error leyendo ejecutores de lenguaje: %w", err)
	}
	runners := make([]domain.LanguageRunner, 0, len(rows))
	for _, row := range rows {
		runners = append(runners, row.toDomain())
	}
	return domain.SortLanguageRunners(runners), nil
}

// attachTemplateLanguageRunners completa una lista de plantillas con sus
// ejecutores en una sola consulta.
func (r *PostgresAdminGovernanceRepository) attachTemplateLanguageRunners(
	ctx context.Context,
	templates []*domain.AdminTemplateReviewItem,
) error {
	if len(templates) == 0 {
		return nil
	}
	ids := make([]string, 0, len(templates))
	for _, template := range templates {
		if template != nil && template.ID != "" {
			ids = append(ids, template.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	rows := []languageRunnerRow{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT template_id::text AS template_id, language, entrypoint, docker_image, timeout_ms, memory_limit_mb
		FROM template_language_runners
		WHERE template_id = ANY($1)
		ORDER BY template_id, language
	`, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("error leyendo ejecutores de lenguaje: %w", err)
	}

	byTemplate := make(map[string][]domain.LanguageRunner, len(templates))
	for _, row := range rows {
		byTemplate[row.TemplateID] = append(byTemplate[row.TemplateID], row.toDomain())
	}
	for _, template := range templates {
		if template == nil {
			continue
		}
		if runners, ok := byTemplate[template.ID]; ok {
			template.LanguageRunners = domain.SortLanguageRunners(runners)
		} else {
			template.LanguageRunners = []domain.LanguageRunner{}
		}
	}
	return nil
}
