package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
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

	if err := db.Ping(); err != nil {
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
	_, errLock := db.Exec("SELECT pg_advisory_lock(1337)")
	if errLock != nil {
		log.Printf("Notice: could not acquire migration advisory lock: %v", errLock)
	} else {
		defer func() { _, _ = db.Exec("SELECT pg_advisory_unlock(1337)") }()
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

