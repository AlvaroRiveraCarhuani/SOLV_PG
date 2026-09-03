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
	// CRIT-19: Advisory lock de migraciones (bloqueante)
	_, errLock := d.db.Exec("SELECT pg_advisory_lock(1337)")
	if errLock != nil {
		log.Printf("Notice: Could not acquire migration advisory lock 1337: %v. Continuing if pre-existing...", errLock)
	} else {
		defer func() {
			_, _ = d.db.Exec("SELECT pg_advisory_unlock(1337)")
		}()
	}

	usersTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		name VARCHAR(255) DEFAULT '',
		first_name VARCHAR(255) DEFAULT '',
		last_name VARCHAR(255) DEFAULT '',
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
	ALTER TABLE users ADD COLUMN IF NOT EXISTS first_name VARCHAR(255) DEFAULT '';
	ALTER TABLE users ADD COLUMN IF NOT EXISTS last_name VARCHAR(255) DEFAULT '';
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

	-- 7. Esquema Académico (Slice 9 / CRIT-02)
	CREATE TABLE IF NOT EXISTS subjects (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
		name VARCHAR(255) NOT NULL,
		code VARCHAR(50) NOT NULL,
		classroom_course_id VARCHAR(255),
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS enrollments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
		student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
		enrolled_at TIMESTAMPTZ DEFAULT NOW(),
		CONSTRAINT unique_enrollment_per_tenant UNIQUE (tenant_id, student_id, subject_id)
	);

	CREATE TABLE IF NOT EXISTS submissions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
		exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
		student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
		code TEXT NOT NULL DEFAULT '',
		verdict VARCHAR(50) NOT NULL,
		ast_result JSONB DEFAULT '{}'::jsonb,
		execution_time_ms INT NOT NULL DEFAULT 0,
		memory_used_mb INT NOT NULL DEFAULT 0,
		manual_override BOOLEAN DEFAULT FALSE,
		override_reason TEXT DEFAULT '',
		score INT,
		graded_by UUID REFERENCES users(id) ON DELETE SET NULL,
		submitted_at TIMESTAMPTZ DEFAULT NOW()
	);

	ALTER TABLE submissions ADD COLUMN IF NOT EXISTS manual_override BOOLEAN DEFAULT FALSE;
	ALTER TABLE submissions ADD COLUMN IF NOT EXISTS override_reason TEXT DEFAULT '';
	ALTER TABLE submissions ADD COLUMN IF NOT EXISTS score INT;
	ALTER TABLE submissions ADD COLUMN IF NOT EXISTS graded_by UUID REFERENCES users(id) ON DELETE SET NULL;

	CREATE TABLE IF NOT EXISTS teacher_invitations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
		token VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255) NOT NULL,
		used BOOLEAN DEFAULT FALSE,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Tabla de comentarios pedagógicos in-line en código (Slice 13)
	CREATE TABLE IF NOT EXISTS submission_comments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
		submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
		author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		line_number INT NOT NULL,
		comment TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_submission_comments_sub ON submission_comments(submission_id);
	CREATE INDEX IF NOT EXISTS idx_submission_comments_tenant ON submission_comments(tenant_id);

	-- Slice 14: Modo Mantenimiento (ADR-031) y Periodos Académicos (ADR-029)
	ALTER TABLE tenants ADD COLUMN IF NOT EXISTS maintenance_mode BOOLEAN DEFAULT FALSE;
	ALTER TABLE tenants ADD COLUMN IF NOT EXISTS maintenance_until TIMESTAMPTZ;
	ALTER TABLE tenants ADD COLUMN IF NOT EXISTS maintenance_reason TEXT DEFAULT '';

	CREATE TABLE IF NOT EXISTS academic_periods (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
		name VARCHAR(100) NOT NULL,
		code VARCHAR(50) NOT NULL,
		start_date DATE NOT NULL,
		end_date DATE NOT NULL,
		is_active BOOLEAN NOT NULL DEFAULT true,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		CONSTRAINT uq_tenant_period_code UNIQUE(tenant_id, code)
	);
	CREATE INDEX IF NOT EXISTS idx_academic_periods_tenant ON academic_periods(tenant_id);

	ALTER TABLE subjects ADD COLUMN IF NOT EXISTS academic_period_id UUID REFERENCES academic_periods(id) ON DELETE SET NULL;
	ALTER TABLE subjects ADD COLUMN IF NOT EXISTS is_archived BOOLEAN DEFAULT FALSE;
	CREATE INDEX IF NOT EXISTS idx_subjects_period ON subjects(academic_period_id);

	-- Foreign key FK_workspaces_subject con saneamiento de registros preexistentes
	INSERT INTO subjects (id, tenant_id, name, code)
	VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Materia General', 'GEN-101')
	ON CONFLICT (id) DO NOTHING;

	UPDATE workspaces SET subject_id = '00000000-0000-0000-0000-000000000001'
	WHERE subject_id NOT IN (SELECT id FROM subjects);

	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_workspaces_subject') THEN
			ALTER TABLE workspaces ADD CONSTRAINT fk_workspaces_subject FOREIGN KEY (subject_id) REFERENCES subjects(id) ON DELETE RESTRICT;
		END IF;
	END $$;

	-- Índices académicos requeridos (CRIT-02)
	CREATE INDEX IF NOT EXISTS idx_subjects_tenant ON subjects(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_submissions_tenant ON submissions(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_submissions_exercise ON submissions(exercise_id);
	CREATE INDEX IF NOT EXISTS idx_enrollments_student ON enrollments(student_id);

	-- Extensión de tabla exercises para Slice 13 (CREAR_LAB)
	ALTER TABLE exercises 
	ADD COLUMN IF NOT EXISTS subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL,
	ADD COLUMN IF NOT EXISTS due_date TIMESTAMPTZ,
	ADD COLUMN IF NOT EXISTS boilerplate TEXT NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS status VARCHAR(30) NOT NULL DEFAULT 'draft',
	ADD COLUMN IF NOT EXISTS language VARCHAR(50) NOT NULL DEFAULT 'python',
	ADD COLUMN IF NOT EXISTS time_limit_ms INT NOT NULL DEFAULT 1000,
	ADD COLUMN IF NOT EXISTS memory_limit_mb INT NOT NULL DEFAULT 128,
	ADD COLUMN IF NOT EXISTS db_config JSONB NOT NULL DEFAULT '{}'::jsonb;

	CREATE INDEX IF NOT EXISTS idx_exercises_subject_id ON exercises(subject_id);
	CREATE INDEX IF NOT EXISTS idx_exercises_due_date ON exercises(due_date);
	CREATE INDEX IF NOT EXISTS idx_exercises_status ON exercises(status);
	CREATE INDEX IF NOT EXISTS idx_exercises_subject_status ON exercises(subject_id, status);

	-- Extensión de subjects para docente asignado
	ALTER TABLE subjects ADD COLUMN IF NOT EXISTS teacher_id UUID REFERENCES users(id) ON DELETE SET NULL;
	CREATE INDEX IF NOT EXISTS idx_subjects_teacher ON subjects(teacher_id);

	-- Tabla audit_logs (CRIT-11)
	CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
		actor_id UUID NOT NULL,
		action VARCHAR(255) NOT NULL,
		resource_type VARCHAR(100) NOT NULL,
		resource_id UUID,
		status_code INT NOT NULL DEFAULT 200,
		metadata JSONB DEFAULT '{}'::jsonb,
		ip_address INET,
		user_agent TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant ON audit_logs(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
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
