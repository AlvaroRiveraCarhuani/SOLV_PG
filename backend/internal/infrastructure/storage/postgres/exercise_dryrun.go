package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"solv-backend/internal/core/domain"
)

type dryRunJobDB struct {
	ID              string          `db:"id"`
	ExerciseID      string          `db:"exercise_id"`
	Status          string          `db:"status"`
	ProgressCurrent int             `db:"progress_current"`
	ProgressTotal   int             `db:"progress_total"`
	Result          json.RawMessage `db:"result"`
	Error           string          `db:"error"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

func (r *PostgresExerciseRepository) CreateDryRunJob(ctx context.Context, job *domain.DryRunJob) error {
	resultBytes := []byte("{}")
	if job.Result != nil {
		if b, err := json.Marshal(job.Result); err == nil {
			resultBytes = b
		}
	}
	query := `
		INSERT INTO dry_run_jobs (id, exercise_id, status, progress_current, progress_total, result, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.ExerciseID,
		string(job.Status),
		job.ProgressCurrent,
		job.ProgressTotal,
		resultBytes,
		job.Error,
	)
	if err != nil {
		return fmt.Errorf("failed to create dry_run_job: %w", err)
	}
	return nil
}

func (r *PostgresExerciseRepository) GetDryRunJob(ctx context.Context, jobID string) (*domain.DryRunJob, error) {
	query := `
		SELECT id, exercise_id, status, progress_current, progress_total, result, error, created_at, updated_at
		FROM dry_run_jobs
		WHERE id = $1
	`
	var dbJob dryRunJobDB
	err := r.db.GetContext(ctx, &dbJob, query, jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("dry_run_job not found")
		}
		return nil, fmt.Errorf("failed to get dry_run_job: %w", err)
	}

	job := &domain.DryRunJob{
		ID:              dbJob.ID,
		ExerciseID:      dbJob.ExerciseID,
		Status:          domain.DryRunJobStatus(dbJob.Status),
		ProgressCurrent: dbJob.ProgressCurrent,
		ProgressTotal:   dbJob.ProgressTotal,
		Error:           dbJob.Error,
		CreatedAt:       dbJob.CreatedAt,
		UpdatedAt:       dbJob.UpdatedAt,
	}

	if len(dbJob.Result) > 0 && string(dbJob.Result) != "{}" {
		var res domain.EvaluationResult
		if err := json.Unmarshal(dbJob.Result, &res); err == nil {
			job.Result = &res
		}
	}

	return job, nil
}

func (r *PostgresExerciseRepository) UpdateDryRunJobProgress(
	ctx context.Context,
	jobID string,
	status domain.DryRunJobStatus,
	current, total int,
	result *domain.EvaluationResult,
	errMsg string,
) error {
	resultBytes := []byte("{}")
	if result != nil {
		if b, err := json.Marshal(result); err == nil {
			resultBytes = b
		}
	}
	query := `
		UPDATE dry_run_jobs
		SET status = $1,
		    progress_current = $2,
		    progress_total = $3,
		    result = $4,
		    error = $5,
		    updated_at = NOW()
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query,
		string(status),
		current,
		total,
		resultBytes,
		errMsg,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("failed to update dry_run_job progress: %w", err)
	}
	return nil
}
