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

func (d *Database) GetDB() *sqlx.DB {
	return d.db
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
		user_id UUID NOT NULL,
		template_id UUID NOT NULL,
		container_id VARCHAR(64),
		volume_name VARCHAR(255),
		status VARCHAR(20) NOT NULL DEFAULT 'pending',
		ram_limit_mb INTEGER DEFAULT 150,
		oom_kills_count INTEGER DEFAULT 0,
		last_active_at TIMESTAMPTZ DEFAULT NOW(),
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);`

	exercisesTableQuery := `
	CREATE TABLE IF NOT EXISTS exercises (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		title VARCHAR(255) NOT NULL,
		description TEXT,
		type VARCHAR(50) NOT NULL DEFAULT 'algorithm',
		config JSONB NOT NULL DEFAULT '{}'::jsonb,
		created_at TIMESTAMPTZ DEFAULT NOW()
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

	d.db.Exec("DROP TABLE IF EXISTS exercises CASCADE")
	if _, err := d.db.Exec(exercisesTableQuery); err != nil {
		return fmt.Errorf("failed to create exercises table: %w", err)
	}

	// Semillas para ejercicios de prueba (Algoritmia & BD)
	seedAlgoQuery := `
	INSERT INTO exercises (id, title, description, type, config)
	VALUES (
		'e1e1e1e1-e1e1-4e1e-a1e1-e1e1e1e1e1e1',
		'Suma de Dos Números',
		'Escribe un programa que lea dos enteros por entrada estándar y devuelva su suma.',
		'algorithm',
		'{
			"algorithm": {
				"time_limit_ms": 2000,
				"memory_limit_mb": 128,
				"test_cases": [
					{"input": "2\n3", "expected_output": "5", "is_hidden": false},
					{"input": "100\n200", "expected_output": "300", "is_hidden": true}
				],
				"ast_rules": {
					"forbidden_imports": ["os", "sys", "subprocess", "System.IO"],
					"forbidden_functions": ["eval", "exec", "open"]
				}
			}
		}'::jsonb
	) ON CONFLICT (id) DO NOTHING;`

	seedDBQuery := `
	INSERT INTO exercises (id, title, description, type, config)
	VALUES (
		'd2d2d2d2-d2d2-4d2d-b2d2-d2d2d2d2d2d2',
		'Actualización de Saldo Bancario',
		'Escribe una sentencia SQL UPDATE para incrementar en 50 el saldo de la cuenta ID 1.',
		'database',
		'{
			"database": {
				"engine": "postgres",
				"init_script": "CREATE TABLE accounts (id INT PRIMARY KEY, balance INT); INSERT INTO accounts VALUES (1, 100), (2, 200);",
				"reference_solution": "UPDATE accounts SET balance = balance + 50 WHERE id = 1;",
				"validation_query": "SELECT id, balance FROM accounts ORDER BY id;",
				"expected_json": "",
				"time_limit_ms": 5000,
				"memory_limit_mb": 256
			}
		}'::jsonb
	) ON CONFLICT (id) DO NOTHING;`

	if _, err := d.db.Exec(seedAlgoQuery); err != nil {
		log.Printf("Warning: failed to seed algorithm exercise: %v", err)
	}

	if _, err := d.db.Exec(seedDBQuery); err != nil {
		log.Printf("Warning: failed to seed DB exercise: %v", err)
	}

	log.Println("Initial migrations completed successfully.")
	return nil
}
