package postgres

import (
	"context"
	"fmt"
	"solv-backend/internal/core/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PostgresKeystrokeRepository struct {
	db *sqlx.DB
}

func NewPostgresKeystrokeRepository(db *sqlx.DB) *PostgresKeystrokeRepository {
	return &PostgresKeystrokeRepository{db: db}
}

func (r *PostgresKeystrokeRepository) SaveBatch(ctx context.Context, submissionID string, events []domain.SubmissionKeystrokeEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO submission_keystroke_events (id, submission_id, timestamp_ms, event_type, position, content, paste_source_detected, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, ev := range events {
		id := ev.ID
		if id == "" {
			id = uuid.NewString()
		}
		pasteDetected := ev.PasteSourceDetected || (ev.EventType == domain.KeystrokeEventPaste && len(ev.Content) > 50)
		_, err := stmt.ExecContext(ctx, id, submissionID, ev.TimestampMS, string(ev.EventType), ev.Position, ev.Content, pasteDetected)
		if err != nil {
			return fmt.Errorf("failed to insert keystroke event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit keystroke events batch: %w", err)
	}

	return nil
}

func (r *PostgresKeystrokeRepository) GetBySubmission(ctx context.Context, submissionID string) ([]domain.SubmissionKeystrokeEvent, error) {
	query := `
		SELECT id, submission_id, timestamp_ms, event_type, position, COALESCE(content, '') AS content, paste_source_detected, created_at
		FROM submission_keystroke_events
		WHERE submission_id = $1
		ORDER BY timestamp_ms ASC
	`
	var events []domain.SubmissionKeystrokeEvent
	err := r.db.SelectContext(ctx, &events, query, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query keystroke events: %w", err)
	}
	if events == nil {
		events = make([]domain.SubmissionKeystrokeEvent, 0)
	}
	return events, nil
}
