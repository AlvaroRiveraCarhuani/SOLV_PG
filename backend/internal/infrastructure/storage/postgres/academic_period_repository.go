package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"solv-backend/internal/core/domain"
)

type PostgresAcademicPeriodRepository struct {
	db *sqlx.DB
}

func NewPostgresAcademicPeriodRepository(db *sqlx.DB) *PostgresAcademicPeriodRepository {
	return &PostgresAcademicPeriodRepository{db: db}
}

func (r *PostgresAcademicPeriodRepository) Create(ctx context.Context, period *domain.AcademicPeriod) error {
	query := `
		INSERT INTO academic_periods (id, tenant_id, name, code, start_date, end_date, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		period.ID,
		period.TenantID,
		period.Name,
		period.Code,
		period.StartDate,
		period.EndDate,
		period.IsActive,
	)
	if err != nil {
		return fmt.Errorf("error creating academic period: %w", err)
	}
	return nil
}

func (r *PostgresAcademicPeriodRepository) GetByID(ctx context.Context, tenantID, id string) (*domain.AcademicPeriod, error) {
	query := `
		SELECT id, tenant_id, name, code, start_date, end_date, is_active, created_at
		FROM academic_periods
		WHERE tenant_id = $1 AND id = $2
	`
	var p domain.AcademicPeriod
	err := r.db.GetContext(ctx, &p, query, tenantID, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("academic period not found")
		}
		return nil, fmt.Errorf("error querying academic period: %w", err)
	}
	return &p, nil
}

func (r *PostgresAcademicPeriodRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.AcademicPeriod, error) {
	query := `
		SELECT id, tenant_id, name, code, start_date, end_date, is_active, created_at
		FROM academic_periods
		WHERE tenant_id = $1
		ORDER BY start_date DESC, created_at DESC
	`
	var list []*domain.AcademicPeriod
	err := r.db.SelectContext(ctx, &list, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error querying academic periods: %w", err)
	}
	return list, nil
}

func (r *PostgresAcademicPeriodRepository) Update(ctx context.Context, period *domain.AcademicPeriod) error {
	query := `
		UPDATE academic_periods
		SET name = $1, code = $2, start_date = $3, end_date = $4, is_active = $5
		WHERE tenant_id = $6 AND id = $7
	`
	res, err := r.db.ExecContext(ctx, query,
		period.Name,
		period.Code,
		period.StartDate,
		period.EndDate,
		period.IsActive,
		period.TenantID,
		period.ID,
	)
	if err != nil {
		return fmt.Errorf("error updating academic period: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("academic period not found or not updated")
	}
	return nil
}

func (r *PostgresAcademicPeriodRepository) Delete(ctx context.Context, tenantID, id string) error {
	// Verificar si existen materias vinculadas
	var count int
	countQuery := `SELECT COUNT(*) FROM subjects WHERE tenant_id = $1 AND academic_period_id = $2`
	err := r.db.GetContext(ctx, &count, countQuery, tenantID, id)
	if err != nil {
		return fmt.Errorf("error checking associated subjects: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete academic period with %d associated subjects", count)
	}

	deleteQuery := `DELETE FROM academic_periods WHERE tenant_id = $1 AND id = $2`
	res, err := r.db.ExecContext(ctx, deleteQuery, tenantID, id)
	if err != nil {
		return fmt.Errorf("error deleting academic period: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("academic period not found")
	}
	return nil
}

func (r *PostgresAcademicPeriodRepository) ArchiveExpiredPeriods(ctx context.Context) (int64, error) {
	// 1. Archivar periodos cuya end_date ya pasó
	queryPeriods := `
		UPDATE academic_periods
		SET is_active = false
		WHERE end_date < CURRENT_DATE AND is_active = true
	`
	res, err := r.db.ExecContext(ctx, queryPeriods)
	if err != nil {
		return 0, fmt.Errorf("error archiving expired periods: %w", err)
	}
	archivedPeriods, _ := res.RowsAffected()

	// 2. Archivar materias de periodos inactivos o vencidos
	querySubjects := `
		UPDATE subjects
		SET is_archived = true
		WHERE academic_period_id IN (
			SELECT id FROM academic_periods WHERE end_date < CURRENT_DATE OR is_active = false
		) AND is_archived = false
	`
	_, _ = r.db.ExecContext(ctx, querySubjects)

	return archivedPeriods, nil
}
