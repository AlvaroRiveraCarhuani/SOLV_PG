package database

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Database struct {
	db *sqlx.DB
}

func NewPostgresDB(dsn string) (*Database, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &Database{db: db}, nil
}
func (d *Database) RunInitialMigrations() error {
	usersTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		first_name VARCHAR(100) NOT NULL,
		last_name VARCHAR(100) NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		role VARCHAR(50) DEFAULT 'student'
	);`

	labTemplatesTableQuery := `
	CREATE TABLE IF NOT EXISTS lab_templates (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(100) NOT NULL,
		docker_image VARCHAR(255) NOT NULL,
		base_ram_mb INTEGER NOT NULL
	);`

	labInstancesTableQuery := `
	CREATE TABLE IF NOT EXISTS lab_instances (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		template_id UUID NOT NULL REFERENCES lab_templates(id) ON DELETE CASCADE,
		container_name VARCHAR(255) UNIQUE NOT NULL,
		traefik_url VARCHAR(255) UNIQUE,
		status VARCHAR(50) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	log.Println("Running initial database migrations...")

	if _, err := d.db.Exec(usersTableQuery); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	if _, err := d.db.Exec(labTemplatesTableQuery); err != nil {
		return fmt.Errorf("failed to create lab_templates table: %w", err)
	}

	if _, err := d.db.Exec(labInstancesTableQuery); err != nil {
		return fmt.Errorf("failed to create lab_instances table: %w", err)
	}

	log.Println("Initial migrations completed successfully.")
	return nil
}
