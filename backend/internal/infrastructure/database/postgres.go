package database

import (
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
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

func (d *Database) RunInitialMigrations() error {
	usersTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		picture TEXT,
		role VARCHAR(50) NOT NULL DEFAULT 'student',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);`

	labTemplatesTableQuery := `
	CREATE TABLE IF NOT EXISTS lab_templates (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		docker_image VARCHAR(255) NOT NULL,
		base_ram_mb INT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);`

	labTemplateProfilesTableQuery := `
	CREATE TABLE IF NOT EXISTS lab_template_profiles (
		signature_hash VARCHAR(64) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		base_image VARCHAR(255) NOT NULL,
		setup_script TEXT NOT NULL DEFAULT '',
		resource_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
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

	workspacesTableQuery := `
	CREATE TABLE IF NOT EXISTS workspaces (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		student_id UUID NOT NULL,
		subject_id UUID NOT NULL,
		type VARCHAR(50) NOT NULL DEFAULT 'IDE_PERSISTENTE',
		container_id VARCHAR(255),
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		access_url TEXT NOT NULL DEFAULT '',
		memory_limit_mb INT NOT NULL DEFAULT 256,
		last_heartbeat_at TIMESTAMPTZ DEFAULT NOW(),
		last_oom_killed_at TIMESTAMPTZ,
		oom_strike_count INT NOT NULL DEFAULT 0,
		semgrep_audit JSONB DEFAULT '{}'::jsonb,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);`

	alterWorkspacesQuery := `
	ALTER TABLE workspaces 
	ADD COLUMN IF NOT EXISTS type VARCHAR(50) NOT NULL DEFAULT 'IDE_PERSISTENTE',
	ADD COLUMN IF NOT EXISTS memory_limit_mb INT NOT NULL DEFAULT 256,
	ADD COLUMN IF NOT EXISTS last_heartbeat_at TIMESTAMPTZ DEFAULT NOW(),
	ADD COLUMN IF NOT EXISTS last_oom_killed_at TIMESTAMPTZ,
	ADD COLUMN IF NOT EXISTS oom_strike_count INT NOT NULL DEFAULT 0,
	ADD COLUMN IF NOT EXISTS semgrep_audit JSONB DEFAULT '{}'::jsonb;`

	migrateLabInstancesQuery := `
	DO $$ 
	BEGIN
		IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'lab_instances') THEN
			INSERT INTO workspaces (id, student_id, subject_id, container_id, status, type, access_url, memory_limit_mb, semgrep_audit, created_at, updated_at)
			SELECT id, user_id, template_id, container_id, status, 'JUEZ_EFIMERO', '', ram_limit_mb, semgrep_audit, created_at, updated_at
			FROM lab_instances
			ON CONFLICT (id) DO NOTHING;
			
			DROP TABLE lab_instances CASCADE;
		END IF;
	END $$;`

	log.Println("Running initial database migrations...")

	if _, err := d.db.Exec(usersTableQuery); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	if _, err := d.db.Exec(labTemplatesTableQuery); err != nil {
		return fmt.Errorf("failed to create lab_templates table: %w", err)
	}

	if _, err := d.db.Exec(labTemplateProfilesTableQuery); err != nil {
		return fmt.Errorf("failed to create lab_template_profiles table: %w", err)
	}

	d.db.Exec("DROP TABLE IF EXISTS exercises CASCADE")
	if _, err := d.db.Exec(exercisesTableQuery); err != nil {
		return fmt.Errorf("failed to create exercises table: %w", err)
	}

	if _, err := d.db.Exec(workspacesTableQuery); err != nil {
		return fmt.Errorf("failed to create workspaces table: %w", err)
	}

	if _, err := d.db.Exec(alterWorkspacesQuery); err != nil {
		log.Printf("Notice: alter workspaces query: %v", err)
	}

	if _, err := d.db.Exec(migrateLabInstancesQuery); err != nil {
		log.Printf("Notice: migrate lab_instances query: %v", err)
	}

	// Migraciones de Multi-tenancy (Orden crítico de ejecución)
	multitenancyMigrationQuery := `
	-- 1. Crear tabla tenants
	CREATE TABLE IF NOT EXISTS tenants (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		slug VARCHAR(100) UNIQUE NOT NULL,
		allowed_domains JSONB NOT NULL DEFAULT '["@uab.edu.bo"]'::jsonb,
		config JSONB DEFAULT '{}'::jsonb,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Seed UAB
	INSERT INTO tenants (id, name, slug, allowed_domains, config) VALUES (
		'00000000-0000-0000-0000-000000000001',
		'Universidad Adventista de Bolivia',
		'uab',
		'["@uab.edu.bo"]'::jsonb,
		'{"institution_name": "Universidad Adventista de Bolivia", "logo_url": "/assets/uab-logo.png", "base_domain": "solv.uab.edu.bo", "support_email": "soporte.solv@uab.edu.bo"}'::jsonb
	) ON CONFLICT (id) DO NOTHING;

	-- 2. Añadir tenant_id como NULLABLE
	ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id UUID;
	ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS tenant_id UUID;
	ALTER TABLE lab_template_profiles ADD COLUMN IF NOT EXISTS tenant_id UUID;
	ALTER TABLE exercises ADD COLUMN IF NOT EXISTS tenant_id UUID;
	ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS tenant_id UUID;

	-- 3. UPDATE masivo para registros existentes al tenant UAB
	UPDATE users SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
	UPDATE lab_templates SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
	UPDATE lab_template_profiles SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
	UPDATE exercises SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
	UPDATE workspaces SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;

	-- 4. ALTER COLUMN tenant_id SET NOT NULL
	ALTER TABLE users ALTER COLUMN tenant_id SET NOT NULL;
	ALTER TABLE lab_templates ALTER COLUMN tenant_id SET NOT NULL;
	ALTER TABLE lab_template_profiles ALTER COLUMN tenant_id SET NOT NULL;
	ALTER TABLE exercises ALTER COLUMN tenant_id SET NOT NULL;
	ALTER TABLE workspaces ALTER COLUMN tenant_id SET NOT NULL;

	-- 5. Añadir foreign keys con ON DELETE RESTRICT
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_users_tenant') THEN
			ALTER TABLE users ADD CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_lab_templates_tenant') THEN
			ALTER TABLE lab_templates ADD CONSTRAINT fk_lab_templates_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_lab_template_profiles_tenant') THEN
			ALTER TABLE lab_template_profiles ADD CONSTRAINT fk_lab_template_profiles_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_exercises_tenant') THEN
			ALTER TABLE exercises ADD CONSTRAINT fk_exercises_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_workspaces_tenant') THEN
			ALTER TABLE workspaces ADD CONSTRAINT fk_workspaces_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT;
		END IF;
	END $$;

	-- 6. Crear índices en columnas tenant_id
	CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_lab_templates_tenant_id ON lab_templates(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_lab_template_profiles_tenant_id ON lab_template_profiles(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_exercises_tenant_id ON exercises(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_workspaces_tenant_id ON workspaces(tenant_id);
	`

	if _, err := d.db.Exec(multitenancyMigrationQuery); err != nil {
		return fmt.Errorf("failed to run multitenancy migration: %w", err)
	}

	// Semillas para ejercicios de prueba (Algoritmia & BD)
	seedAlgoQuery := `
	INSERT INTO exercises (id, title, description, type, config, tenant_id)
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
		}'::jsonb,
		'00000000-0000-0000-0000-000000000001'
	) ON CONFLICT (id) DO NOTHING;`

	seedDBQuery := `
	INSERT INTO exercises (id, title, description, type, config, tenant_id)
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
		}'::jsonb,
		'00000000-0000-0000-0000-000000000001'
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
