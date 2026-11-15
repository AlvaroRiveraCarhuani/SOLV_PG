package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

type Database struct {
	db *sqlx.DB
}

func NewPostgresDB(dsn string) (*Database, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) GetDB() *sqlx.DB {
	return d.db
}

// RunMigrations ejecuta todas las migraciones pendientes en el directorio dado
// usando goose. Adquiere el advisory lock 1337 antes de correr para garantizar
// que solo un proceso aplica migraciones a la vez en entornos multi-réplica.
func RunMigrations(db *sql.DB, migrationsDir string) error {
	_, errLock := db.ExecContext(context.Background(), "SELECT pg_advisory_lock(1337)")
	if errLock != nil {
		log.Printf("Notice: could not acquire migration advisory lock: %v", errLock)
	} else {
		defer func() { _, _ = db.ExecContext(context.Background(), "SELECT pg_advisory_unlock(1337)") }()
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

// CurrentMigrationVersion devuelve la última migración aplicada por goose
// (columna version_id de goose_db_version). Una tabla inexistente o vacía
// devuelve 0 sin error: un arranque limpio no es un fallo.
func CurrentMigrationVersion(ctx context.Context, db *sql.DB) (int64, error) {
	var applied int64
	err := db.QueryRowContext(ctx, `SELECT version_id FROM goose_db_version WHERE is_applied ORDER BY id DESC LIMIT 1`).Scan(&applied)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("current migration version: %w", err)
	}
	return applied, nil
}

// RunMigrationsDown revierte todas las migraciones en el directorio dado hasta 0 usando goose.
func RunMigrationsDown(db *sql.DB, migrationsDir string) error {
	_, errLock := db.ExecContext(context.Background(), "SELECT pg_advisory_lock(1337)")
	if errLock != nil {
		log.Printf("Notice: could not acquire migration advisory lock: %v", errLock)
	} else {
		defer func() { _, _ = db.ExecContext(context.Background(), "SELECT pg_advisory_unlock(1337)") }()
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}
	if err := goose.DownTo(db, migrationsDir, 0); err != nil {
		return fmt.Errorf("goose down: %w", err)
	}
	return nil
}
