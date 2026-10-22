package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"solv-backend/internal/core/domain"

	"database/sql"

	"github.com/jmoiron/sqlx"
)

// PostgresGeneratedCaseSnapshotRepository guarda el snapshot autoritativo de
// los casos generados por semilla. El servidor calcula esos casos al evaluar
// y los conserva para que la entrega del estudiante no dependa del payload
// que envía el cliente (GOB-SEGURO-04).
type PostgresGeneratedCaseSnapshotRepository struct {
	db *sqlx.DB
}

func NewPostgresGeneratedCaseSnapshotRepository(db *sqlx.DB) *PostgresGeneratedCaseSnapshotRepository {
	return &PostgresGeneratedCaseSnapshotRepository{db: db}
}

func (r *PostgresGeneratedCaseSnapshotRepository) SaveGeneratedCaseSnapshot(ctx context.Context, tenantID, exerciseID, studentID string, seed int64, cases []domain.TestCase) error {
	if exerciseID == "" || studentID == "" {
		return fmt.Errorf("exercise_id and student_id are required")
	}

	payload, err := json.Marshal(cases)
	if err != nil {
		return fmt.Errorf("failed to encode generated cases: %w", err)
	}

	query := `
		INSERT INTO evaluation_generated_cases (exercise_id, student_id, tenant_id, seed, cases, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (exercise_id, student_id)
		DO UPDATE SET seed = EXCLUDED.seed, cases = EXCLUDED.cases, created_at = NOW()
	`

	if _, err := r.db.ExecContext(ctx, query, exerciseID, studentID, tenantID, seed, string(payload)); err != nil {
		return fmt.Errorf("failed to save generated case snapshot: %w", err)
	}
	return nil
}

func (r *PostgresGeneratedCaseSnapshotRepository) GetGeneratedCaseSnapshot(ctx context.Context, exerciseID, studentID string) ([]domain.TestCase, error) {
	if exerciseID == "" || studentID == "" {
		return nil, nil
	}

	query := `
		SELECT cases
		FROM evaluation_generated_cases
		WHERE exercise_id = $1 AND student_id = $2
	`

	var payload []byte
	if err := r.db.GetContext(ctx, &payload, query, exerciseID, studentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load generated case snapshot: %w", err)
	}
	if len(payload) == 0 {
		return nil, nil
	}

	var cases []domain.TestCase
	if err := json.Unmarshal(payload, &cases); err != nil {
		return nil, fmt.Errorf("failed to decode generated case snapshot: %w", err)
	}
	return cases, nil
}
