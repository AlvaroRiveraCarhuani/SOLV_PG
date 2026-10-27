package postgres

import (
	"context"
	"fmt"
	"time"

	"solv-backend/internal/core/domain"

	"github.com/jmoiron/sqlx"
)

type PostgresDeadlineReminderRepository struct {
	db *sqlx.DB
}

func NewPostgresDeadlineReminderRepository(db *sqlx.DB) *PostgresDeadlineReminderRepository {
	return &PostgresDeadlineReminderRepository{db: db}
}

func (r *PostgresDeadlineReminderRepository) FindPendingReminders(ctx context.Context, windowStart, windowEnd time.Time) ([]*domain.DeadlineReminderCandidate, error) {
	query := `
		SELECT
			e.tenant_id::text AS tenant_id,
			e.id::text AS exercise_id,
			e.title AS exercise_title,
			s.id::text AS subject_id,
			s.name AS subject_name,
			e.due_date,
			u.id::text AS student_id,
			TRIM(u.first_name || ' ' || u.last_name) AS student_name,
			u.email AS student_email,
			EXISTS (
				SELECT 1 FROM workspaces w 
				WHERE w.student_id = u.id 
				  AND w.subject_id = e.subject_id
			) AS has_started_workspace
		FROM exercises e
		JOIN subjects s ON s.id = e.subject_id
		JOIN enrollments en ON en.subject_id = e.subject_id AND en.tenant_id = e.tenant_id
		JOIN users u ON u.id = en.student_id AND u.tenant_id = e.tenant_id
		WHERE e.status = 'published'
		  AND e.due_date IS NOT NULL
		  AND e.due_date >= $1 
		  AND e.due_date <= $2
		  AND NOT EXISTS (
			  SELECT 1 FROM submissions sub 
			  WHERE sub.exercise_id = e.id 
				AND sub.student_id = u.id
		  )
		  AND NOT EXISTS (
			  SELECT 1 FROM deadline_reminders_sent drs 
			  WHERE drs.tenant_id = e.tenant_id 
				AND drs.exercise_id = e.id 
				AND drs.student_id = u.id
		  )
		ORDER BY e.due_date ASC, u.id ASC
	`

	var results []*domain.DeadlineReminderCandidate
	if err := r.db.SelectContext(ctx, &results, query, windowStart, windowEnd); err != nil {
		return nil, fmt.Errorf("failed to find pending deadline reminders: %w", err)
	}

	return results, nil
}

func (r *PostgresDeadlineReminderRepository) MarkReminderSent(ctx context.Context, tenantID, exerciseID, studentID string) error {
	query := `
		INSERT INTO deadline_reminders_sent (tenant_id, exercise_id, student_id, sent_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (tenant_id, exercise_id, student_id) DO NOTHING
	`
	if _, err := r.db.ExecContext(ctx, query, tenantID, exerciseID, studentID); err != nil {
		return fmt.Errorf("failed to record sent deadline reminder: %w", err)
	}
	return nil
}

func (r *PostgresDeadlineReminderRepository) IsReminderSent(ctx context.Context, tenantID, exerciseID, studentID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM deadline_reminders_sent
			WHERE tenant_id = $1 AND exercise_id = $2 AND student_id = $3
		)
	`
	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, tenantID, exerciseID, studentID); err != nil {
		return false, fmt.Errorf("failed to check deadline reminder status: %w", err)
	}
	return exists, nil
}
