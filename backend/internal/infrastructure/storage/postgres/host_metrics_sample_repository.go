package postgres

import (
	"context"
	"fmt"
	"time"

	"solv-backend/internal/core/domain"

	"github.com/jmoiron/sqlx"
)

// PostgresHostMetricsSampleRepository persiste el historial de telemetria del
// host en host_metrics_samples (migracion 00028). No forma parte de ninguna
// interfaz del core: se inyecta tanto al sampler (insert/poda) como al handler
// del panel (lectura), cada uno con su propio contrato consumidor.
type PostgresHostMetricsSampleRepository struct {
	db *sqlx.DB
}

func NewPostgresHostMetricsSampleRepository(db *sqlx.DB) *PostgresHostMetricsSampleRepository {
	return &PostgresHostMetricsSampleRepository{db: db}
}

var _ domain.HostMetricsSampleStore = (*PostgresHostMetricsSampleRepository)(nil)

func (r *PostgresHostMetricsSampleRepository) Insert(ctx context.Context, sample *domain.HostMetricsSample) error {
	query := `
		INSERT INTO host_metrics_samples (sampled_at, ram_percent, ram_used_gb, cpu_percent, active_containers, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	if err := r.db.QueryRowContext(ctx, query,
		sample.SampledAt.UTC(),
		sample.RAMPercent,
		sample.RAMUsedGB,
		sample.CPUPercent,
		sample.ActiveContainers,
		sample.TenantID,
	).Scan(&sample.ID); err != nil {
		return fmt.Errorf("failed to insert host metrics sample: %w", err)
	}
	return nil
}

func (r *PostgresHostMetricsSampleRepository) ListSince(ctx context.Context, tenantID string, since time.Time, limit int) ([]domain.HostMetricsSample, error) {
	if tenantID == "" {
		tenantID = domain.DefaultTenantID
	}
	if limit <= 0 {
		limit = 61
	}
	query := `
		SELECT id, sampled_at, ram_percent, ram_used_gb, cpu_percent, active_containers, tenant_id
		FROM host_metrics_samples
		WHERE tenant_id = $1 AND sampled_at >= $2
		ORDER BY sampled_at ASC
		LIMIT $3
	`
	samples := make([]domain.HostMetricsSample, 0, limit)
	err := r.db.SelectContext(ctx, &samples, query, tenantID, since.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list host metrics samples: %w", err)
	}
	return samples, nil
}

func (r *PostgresHostMetricsSampleRepository) PruneBefore(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM host_metrics_samples
		WHERE sampled_at < $1
	`, olderThan.UTC())
	if err != nil {
		return 0, fmt.Errorf("failed to prune host metrics samples: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read pruned rows: %w", err)
	}
	return affected, nil
}