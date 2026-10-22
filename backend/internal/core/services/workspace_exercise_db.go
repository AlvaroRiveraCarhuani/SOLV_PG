package services

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	// exerciseMigrationTableDDL es la tabla de control que hace idempotente la
	// ejecución del db_init_script de un ejercicio dentro de la BD satélite.
	exerciseMigrationTableDDL = `CREATE TABLE IF NOT EXISTS _solv_exercise_migrations (exercise_id TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`

	exerciseDBInitTimeout = 30 * time.Second
)

// satelliteDatabaseDSN deriva el DSN de la base satélite a partir del DSN de la
// base principal, sin exigir configuración adicional.
func satelliteDatabaseDSN(baseDSN, dbName string) (string, error) {
	trimmed := strings.TrimSpace(baseDSN)
	if trimmed == "" {
		return "", fmt.Errorf("DATABASE_URL is not configured: cannot reach satellite database %s", dbName)
	}

	if strings.HasPrefix(trimmed, "postgres://") || strings.HasPrefix(trimmed, "postgresql://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", fmt.Errorf("invalid DATABASE_URL: %w", err)
		}
		parsed.Path = "/" + dbName
		return parsed.String(), nil
	}

	parts := strings.Fields(trimmed)
	replaced := false
	for i, part := range parts {
		if strings.HasPrefix(part, "dbname=") {
			parts[i] = "dbname=" + dbName
			replaced = true
		}
	}
	if !replaced {
		parts = append(parts, "dbname="+dbName)
	}
	return strings.Join(parts, " "), nil
}

// applyExerciseDBInitScript aplica el db_init_script del ejercicio una sola vez
// por exercise_id sobre la BD satélite del alumno.
func applyExerciseDBInitScript(ctx context.Context, satelliteDBName, exerciseID, script string) error {
	dsn, err := satelliteDatabaseDSN(os.Getenv("DATABASE_URL"), satelliteDBName)
	if err != nil {
		return err
	}

	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to satellite database %s: %w", satelliteDBName, err)
	}
	defer func() { _ = db.Close() }()

	execCtx, cancel := context.WithTimeout(ctx, exerciseDBInitTimeout)
	defer cancel()

	if _, err := db.ExecContext(execCtx, exerciseMigrationTableDDL); err != nil {
		return fmt.Errorf("failed to ensure exercise migrations table: %w", err)
	}

	var applied bool
	if err := db.GetContext(execCtx, &applied,
		`SELECT EXISTS(SELECT 1 FROM _solv_exercise_migrations WHERE exercise_id = $1)`, exerciseID); err != nil {
		return fmt.Errorf("failed to check applied db_init_script: %w", err)
	}
	if applied {
		return nil
	}

	tx, err := db.BeginTxx(execCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to open db_init_script transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(execCtx, script); err != nil {
		return fmt.Errorf("db_init_script failed: %w", err)
	}
	if _, err := tx.ExecContext(execCtx,
		`INSERT INTO _solv_exercise_migrations (exercise_id) VALUES ($1) ON CONFLICT (exercise_id) DO NOTHING`,
		exerciseID); err != nil {
		return fmt.Errorf("failed to register applied db_init_script: %w", err)
	}

	return tx.Commit()
}
