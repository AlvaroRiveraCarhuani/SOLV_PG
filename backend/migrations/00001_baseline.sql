-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- SOLV – Baseline schema (todo el esquema actual de postgres.go)
-- Esta migración representa el estado del esquema al adoptar
-- goose. No altera datos existentes: todos los DDL son
-- IF NOT EXISTS o ADD COLUMN IF NOT EXISTS.
-- ============================================================

-- Tabla base de tenants
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    allowed_domains JSONB NOT NULL DEFAULT '[]'::jsonb,
    config JSONB DEFAULT '{}'::jsonb,
    maintenance_mode BOOLEAN DEFAULT FALSE,
    maintenance_until TIMESTAMPTZ,
    maintenance_reason TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Tabla de usuarios
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) DEFAULT '',
    first_name VARCHAR(255) DEFAULT '',
    last_name VARCHAR(255) DEFAULT '',
    picture TEXT,
    role VARCHAR(50) NOT NULL DEFAULT 'student',
    tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT,
    origin VARCHAR(50) DEFAULT 'manual',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    suspension_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_login_at TIMESTAMPTZ
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS first_name VARCHAR(255) DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_name VARCHAR(255) DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS origin VARCHAR(50) DEFAULT 'manual';
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active';
ALTER TABLE users ADD COLUMN IF NOT EXISTS suspension_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant_role_status ON users(tenant_id, role, status);

-- Categorías de plantillas
CREATE TABLE IF NOT EXISTS template_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    name VARCHAR(100) NOT NULL,
    description TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uk_template_categories_tenant_name UNIQUE (tenant_id, name)
);
CREATE INDEX IF NOT EXISTS idx_template_categories_tenant ON template_categories(tenant_id);

-- Plantillas de laboratorio (lab_templates)
CREATE TABLE IF NOT EXISTS lab_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    docker_image VARCHAR(255) NOT NULL,
    base_ram_mb INT NOT NULL,
    description TEXT DEFAULT '',
    target_environment VARCHAR(50) NOT NULL DEFAULT 'IDE_PERSISTENTE',
    status VARCHAR(50) DEFAULT 'PENDIENTE_AUDITORIA',
    rejection_reason TEXT DEFAULT '',
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    requested_by UUID REFERENCES users(id) ON DELETE SET NULL,
    services_config JSONB NOT NULL DEFAULT '{"database": {"enabled": false}}'::jsonb,
    resource_profile JSONB NOT NULL DEFAULT '{"min_mb": 256, "high_mb": 768, "max_mb": 1024}'::jsonb,
    setup_script TEXT NOT NULL DEFAULT '',
    tools_declared JSONB NOT NULL DEFAULT '[]'::jsonb,
    smoke_test_status VARCHAR(50) DEFAULT 'pending',
    smoke_test_output TEXT DEFAULT '',
    security_audit_status VARCHAR(50) DEFAULT 'pending',
    cve_critical_count INT DEFAULT 0,
    cve_high_count INT DEFAULT 0,
    security_audit_report JSONB DEFAULT '{}'::jsonb,
    security_audited_at TIMESTAMPTZ,
    eol_status VARCHAR(50) DEFAULT 'supported',
    eol_date VARCHAR(50) DEFAULT '',
    eol_message TEXT DEFAULT '',
    eol_checked_at TIMESTAMPTZ,
    entrypoint TEXT DEFAULT '',
    timeout_ms INT DEFAULT 5000,
    sample_input TEXT DEFAULT '',
    category_id UUID,
    model_id UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS target_environment VARCHAR(50) NOT NULL DEFAULT 'IDE_PERSISTENTE';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'PENDIENTE_AUDITORIA';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS rejection_reason TEXT DEFAULT '';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS requested_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS services_config JSONB NOT NULL DEFAULT '{"database": {"enabled": false}}'::jsonb;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS resource_profile JSONB NOT NULL DEFAULT '{"min_mb": 256, "high_mb": 768, "max_mb": 1024}'::jsonb;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS setup_script TEXT NOT NULL DEFAULT '';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS tools_declared JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS smoke_test_status VARCHAR(50) DEFAULT 'pending';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS smoke_test_output TEXT DEFAULT '';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS security_audit_status VARCHAR(50) DEFAULT 'pending';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS cve_critical_count INT DEFAULT 0;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS cve_high_count INT DEFAULT 0;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS security_audit_report JSONB DEFAULT '{}'::jsonb;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS security_audited_at TIMESTAMPTZ;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS eol_status VARCHAR(50) DEFAULT 'supported';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS eol_date VARCHAR(50) DEFAULT '';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS eol_message TEXT DEFAULT '';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS eol_checked_at TIMESTAMPTZ;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS entrypoint TEXT DEFAULT '';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS timeout_ms INT DEFAULT 5000;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS sample_input TEXT DEFAULT '';
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS category_id UUID;
ALTER TABLE lab_templates ADD COLUMN IF NOT EXISTS model_id UUID;

-- Saneamiento de duplicados históricos antes de imponer unicidad
DELETE FROM lab_templates a
USING lab_templates b
WHERE a.ctid < b.ctid
  AND a.tenant_id IS NOT DISTINCT FROM b.tenant_id
  AND a.name = b.name;

CREATE UNIQUE INDEX IF NOT EXISTS idx_lab_templates_tenant_name ON lab_templates(tenant_id, name);
CREATE INDEX IF NOT EXISTS idx_lab_templates_status ON lab_templates(status);
CREATE INDEX IF NOT EXISTS idx_lab_templates_tenant ON lab_templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_lab_templates_env ON lab_templates(target_environment);
CREATE INDEX IF NOT EXISTS idx_lab_templates_audit ON lab_templates(security_audit_status);
CREATE INDEX IF NOT EXISTS idx_lab_templates_eol ON lab_templates(eol_status);
CREATE INDEX IF NOT EXISTS idx_lab_templates_category ON lab_templates(category_id);
CREATE INDEX IF NOT EXISTS idx_lab_templates_model ON lab_templates(model_id);

-- Modelos oficiales de plantillas
CREATE TABLE IF NOT EXISTS template_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    category_id UUID NOT NULL REFERENCES template_categories(id) ON DELETE RESTRICT,
    source_template_id UUID REFERENCES lab_templates(id) ON DELETE SET NULL,
    title VARCHAR(150) NOT NULL,
    description TEXT DEFAULT '',
    docker_image VARCHAR(255) NOT NULL,
    base_ram_mb INT NOT NULL,
    tools JSONB NOT NULL DEFAULT '[]'::jsonb,
    target_environment VARCHAR(50) NOT NULL DEFAULT 'IDE_PERSISTENTE',
    entrypoint TEXT DEFAULT '',
    timeout_ms INT DEFAULT 5000,
    sample_input TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uk_template_models_tenant_title UNIQUE (tenant_id, title)
);
ALTER TABLE template_models ADD COLUMN IF NOT EXISTS source_template_id UUID REFERENCES lab_templates(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_template_models_tenant ON template_models(tenant_id);
CREATE INDEX IF NOT EXISTS idx_template_models_category ON template_models(category_id);
CREATE INDEX IF NOT EXISTS idx_template_models_env ON template_models(target_environment);
CREATE INDEX IF NOT EXISTS idx_template_models_source_template ON template_models(source_template_id);

-- FKs de lab_templates → template_categories y template_models (ON DELETE SET NULL)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'lab_templates_category_id_fkey') THEN
        ALTER TABLE lab_templates ADD CONSTRAINT lab_templates_category_id_fkey
            FOREIGN KEY (category_id) REFERENCES template_categories(id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'lab_templates_model_id_fkey') THEN
        ALTER TABLE lab_templates ADD CONSTRAINT lab_templates_model_id_fkey
            FOREIGN KEY (model_id) REFERENCES template_models(id) ON DELETE SET NULL;
    END IF;
END $$;

-- Tabla de perfiles de plantilla (legacy, se mantiene por compatibilidad)
CREATE TABLE IF NOT EXISTS lab_template_profiles (
    signature_hash VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    base_image VARCHAR(255) NOT NULL,
    setup_script TEXT NOT NULL DEFAULT '',
    resource_profile JSONB NOT NULL DEFAULT '{}'::jsonb,
    tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
ALTER TABLE lab_template_profiles ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_lab_template_profiles_tenant_id ON lab_template_profiles(tenant_id);

-- Tabla de ejercicios
CREATE TABLE IF NOT EXISTS exercises (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL DEFAULT 'algorithm',
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    subject_id UUID,
    due_date TIMESTAMPTZ,
    boilerplate TEXT NOT NULL DEFAULT '',
    status VARCHAR(30) NOT NULL DEFAULT 'draft',
    language VARCHAR(50) NOT NULL DEFAULT 'python',
    time_limit_ms INT NOT NULL DEFAULT 1000,
    memory_limit_mb INT NOT NULL DEFAULT 128,
    db_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT;
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS subject_id UUID;
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS due_date TIMESTAMPTZ;
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS boilerplate TEXT NOT NULL DEFAULT '';
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS status VARCHAR(30) NOT NULL DEFAULT 'draft';
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS language VARCHAR(50) NOT NULL DEFAULT 'python';
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS time_limit_ms INT NOT NULL DEFAULT 1000;
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS memory_limit_mb INT NOT NULL DEFAULT 128;
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS db_config JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_exercises_tenant_id ON exercises(tenant_id);

-- Tabla de workspaces
CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL,
    subject_id UUID NOT NULL,
    tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT,
    type VARCHAR(50) NOT NULL DEFAULT 'IDE_PERSISTENTE',
    container_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    access_url TEXT NOT NULL DEFAULT '',
    memory_limit_mb INT NOT NULL DEFAULT 256,
    last_heartbeat_at TIMESTAMPTZ DEFAULT NOW(),
    last_oom_killed_at TIMESTAMPTZ,
    oom_strike_count INT NOT NULL DEFAULT 0,
    semgrep_audit JSONB DEFAULT '{}'::jsonb,
    template_id UUID REFERENCES lab_templates(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS type VARCHAR(50) NOT NULL DEFAULT 'IDE_PERSISTENTE';
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS memory_limit_mb INT NOT NULL DEFAULT 256;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS last_heartbeat_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS last_oom_killed_at TIMESTAMPTZ;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS oom_strike_count INT NOT NULL DEFAULT 0;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS semgrep_audit JSONB DEFAULT '{}'::jsonb;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id) ON DELETE RESTRICT;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS template_id UUID REFERENCES lab_templates(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_workspaces_tenant_id ON workspaces(tenant_id);
CREATE INDEX IF NOT EXISTS idx_workspaces_template_id ON workspaces(template_id);

-- Esquema académico
CREATE TABLE IF NOT EXISTS subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL,
    classroom_course_id VARCHAR(255),
    teacher_id UUID REFERENCES users(id) ON DELETE SET NULL,
    template_id UUID REFERENCES lab_templates(id) ON DELETE SET NULL,
    is_archived BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
ALTER TABLE subjects ADD COLUMN IF NOT EXISTS teacher_id UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE subjects ADD COLUMN IF NOT EXISTS template_id UUID REFERENCES lab_templates(id) ON DELETE SET NULL;
ALTER TABLE subjects ADD COLUMN IF NOT EXISTS is_archived BOOLEAN DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_subjects_tenant ON subjects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_subjects_teacher ON subjects(teacher_id);
CREATE INDEX IF NOT EXISTS idx_subjects_template_id ON subjects(template_id);

-- FK exercises → subjects (se agrega aquí porque subjects ya existe)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'exercises_subject_id_fkey') THEN
        ALTER TABLE exercises ADD CONSTRAINT exercises_subject_id_fkey
            FOREIGN KEY (subject_id) REFERENCES subjects(id) ON DELETE SET NULL;
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_exercises_subject_id ON exercises(subject_id);
CREATE INDEX IF NOT EXISTS idx_exercises_due_date ON exercises(due_date);
CREATE INDEX IF NOT EXISTS idx_exercises_status ON exercises(status);
CREATE INDEX IF NOT EXISTS idx_exercises_subject_status ON exercises(subject_id, status);

CREATE TABLE IF NOT EXISTS enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    enrolled_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_enrollment_per_tenant UNIQUE (tenant_id, student_id, subject_id)
);
CREATE INDEX IF NOT EXISTS idx_enrollments_student ON enrollments(student_id);

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

CREATE INDEX IF NOT EXISTS idx_submissions_tenant ON submissions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_submissions_exercise ON submissions(exercise_id);

CREATE TABLE IF NOT EXISTS teacher_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    token VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    origin VARCHAR(50) DEFAULT 'manual',
    role_type VARCHAR(50) DEFAULT 'titular',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
ALTER TABLE teacher_invitations ADD COLUMN IF NOT EXISTS origin VARCHAR(50) DEFAULT 'manual';
ALTER TABLE teacher_invitations ADD COLUMN IF NOT EXISTS role_type VARCHAR(50) DEFAULT 'titular';

-- Periodos académicos
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
CREATE INDEX IF NOT EXISTS idx_subjects_period ON subjects(academic_period_id);

-- FK workspaces → subjects
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_workspaces_subject') THEN
        ALTER TABLE workspaces ADD CONSTRAINT fk_workspaces_subject
            FOREIGN KEY (subject_id) REFERENCES subjects(id) ON DELETE RESTRICT;
    END IF;
END $$;

-- FKs de multi-tenancy
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

-- Comentarios pedagógicos
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

-- Audit logs
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

-- Notificaciones
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    recipient_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel VARCHAR(20) NOT NULL DEFAULT 'in_app' CHECK (channel IN ('in_app', 'email', 'both')),
    severity VARCHAR(20) NOT NULL DEFAULT 'info' CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    title VARCHAR(128) NOT NULL,
    message TEXT NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    email_sent_at TIMESTAMPTZ,
    occurrence_count INT NOT NULL DEFAULT 1,
    link VARCHAR(255) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_recipient_unread
    ON notifications (tenant_id, recipient_user_id, created_at DESC) WHERE is_read = false;
CREATE INDEX IF NOT EXISTS idx_notifications_recipient_all
    ON notifications (tenant_id, recipient_user_id, created_at DESC);

-- Backups
CREATE TABLE IF NOT EXISTS backup_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    local_frequency_hours INT NOT NULL DEFAULT 6,
    local_retention_days INT NOT NULL DEFAULT 7,
    remote_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    remote_provider VARCHAR(32) DEFAULT 'backblaze_b2',
    remote_bucket_name VARCHAR(128) DEFAULT '',
    remote_endpoint VARCHAR(256) DEFAULT '',
    remote_access_key VARCHAR(128) DEFAULT '',
    remote_secret_key_encrypted TEXT DEFAULT '',
    remote_retention_days INT NOT NULL DEFAULT 30,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_tenant_backup_config UNIQUE (tenant_id)
);

CREATE TABLE IF NOT EXISTS backup_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    file_name VARCHAR(256) NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    sha256_checksum VARCHAR(64) NOT NULL,
    storage_tier VARCHAR(20) NOT NULL DEFAULT 'local' CHECK (storage_tier IN ('local', 'remote', 'both')),
    status VARCHAR(20) NOT NULL DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'success', 'failed')),
    error_message TEXT DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_backup_executions_tenant ON backup_executions(tenant_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_backup_executions_status ON backup_executions(status);

-- ============================================================
-- Seeds idempotentes (datos que no son schema, pero son base
-- del sistema y deben existir siempre)
-- ============================================================

-- Seed tenant UAB (configurable vía env; este seed es solo para dev/onboarding inicial)
INSERT INTO tenants (id, name, slug, allowed_domains, config)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Universidad Adventista de Bolivia',
    'uab',
    '["@uab.edu.bo"]'::jsonb,
    '{"institution_name": "Universidad Adventista de Bolivia", "logo_url": "/assets/uab-logo.png", "base_domain": "solv.uab.edu.bo", "support_email": "soporte.solv@uab.edu.bo"}'::jsonb
) ON CONFLICT (id) DO NOTHING;

-- UPDATE masivo de tenant_id para registros pre-multitenancy
UPDATE users SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
UPDATE lab_templates SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
UPDATE lab_template_profiles SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
UPDATE exercises SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;
UPDATE workspaces SET tenant_id = '00000000-0000-0000-0000-000000000001' WHERE tenant_id IS NULL;

-- NOT NULL en tenant_id (solo si ya no hay NULLs)
ALTER TABLE users ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE lab_templates ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE lab_template_profiles ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE exercises ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE workspaces ALTER COLUMN tenant_id SET NOT NULL;

-- Periodo académico semilla
INSERT INTO academic_periods (id, tenant_id, code, name, start_date, end_date, is_active, created_at)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    '00000000-0000-0000-0000-000000000001',
    '2026-I',
    'Primer Semestre 2026',
    '2026-02-01',
    '2026-06-30',
    true,
    NOW()
) ON CONFLICT (tenant_id, code) DO NOTHING;

-- Materia general (anchor para workspaces pre-académicos)
INSERT INTO subjects (id, tenant_id, name, code)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Materia General', 'GEN-101')
ON CONFLICT (id) DO NOTHING;

UPDATE subjects SET academic_period_id = '11111111-1111-1111-1111-111111111111'
WHERE tenant_id = '00000000-0000-0000-0000-000000000001' AND academic_period_id IS NULL;

UPDATE workspaces SET subject_id = '00000000-0000-0000-0000-000000000001'
WHERE subject_id NOT IN (SELECT id FROM subjects);

-- Seeds de categorías oficiales
INSERT INTO template_categories (id, tenant_id, name, description)
VALUES
    ('c0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Ciencia de Datos & IA', 'Entornos especializados en análisis de datos, visualización y aprendizaje automático.'),
    ('c0000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Desarrollo Web & Cloud', 'Herramientas para desarrollo fullstack, APIs backend y microservicios modernos.'),
    ('c0000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'Sistemas & Computación', 'Compiladores nativos y herramientas de bajo nivel para arquitectura y sistemas operativos.'),
    ('c0000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'Empresarial & Backend', 'Plataformas consolidadas para desarrollo empresarial y programación orientada a objetos.')
ON CONFLICT (tenant_id, name) DO NOTHING;

-- Seeds de modelos oficiales (status no aplica a template_models; solo a lab_templates)
INSERT INTO template_models (id, tenant_id, category_id, title, description, docker_image, base_ram_mb, tools, target_environment, entrypoint, timeout_ms, sample_input)
VALUES
    ('d0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'Python 3.12 Data Science', 'Entorno Debian con Python 3.12 y pip optimizado para ciencias y analítica.', 'python:3.12-slim-bookworm', 1024, '["python3", "pip"]'::jsonb, 'IDE_PERSISTENTE', '', 5000, ''),
    ('d0000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'Python Machine Learning & Visión', 'Especializado en redes neuronales, OpenCV y procesamiento multimedia.', 'python:3.11-slim', 2048, '["python3", "pip", "ffmpeg"]'::jsonb, 'IDE_PERSISTENTE', '', 5000, ''),
    ('d0000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000002', 'Node.js 20 LTS Fullstack', 'JavaScript & TypeScript para desarrollo web moderno backend y microservicios.', 'node:20-bookworm-slim', 768, '["node", "npm"]'::jsonb, 'IDE_PERSISTENTE', '', 5000, ''),
    ('d0000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000002', 'Go 1.22 Microservicios', 'SDK oficial de Go para concurrencia masiva y APIs de alto rendimiento.', 'golang:1.22-bookworm', 512, '["go", "git"]'::jsonb, 'IDE_PERSISTENTE', '', 5000, ''),
    ('d0000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000003', 'C/C++ GCC 13 Algoritmia', 'Entorno clásico con compiladores GNU y depurador para sistemas operativos.', 'gcc:13.2-bookworm', 512, '["gcc", "g++", "make", "gdb"]'::jsonb, 'IDE_PERSISTENTE', '', 5000, ''),
    ('d0000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000003', 'Rust 1.80 Sistemas Seguros', 'Lenguaje moderno para desarrollo de infraestructura con seguridad de memoria.', 'rust:1.80-slim-bookworm', 1024, '["rustc", "cargo"]'::jsonb, 'IDE_PERSISTENTE', '', 5000, ''),
    ('d0000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000004', 'Java 21 OpenJDK LTS', 'Entorno de ejecución Java para ingeniería de software y programación orientada a objetos.', 'eclipse-temurin:21-alpine', 1024, '["java", "javac"]'::jsonb, 'IDE_PERSISTENTE', '', 5000, ''),
    ('d0000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000003', 'Juez Algorítmico C++ GCC 13', 'Evaluación rápida de algoritmos competitivos y estructuras de datos.', 'gcc:13.2-bookworm', 128, '["gcc", "g++"]'::jsonb, 'JUEZ_EFIMERO', 'g++ -O3 solution.cpp -o solution && ./solution', 2000, ''),
    ('d0000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'Juez Algorítmico Python 3.12', 'Sandbox de evaluación de scripts algorítmicos en Python.', 'python:3.12-slim-bookworm', 256, '["python3"]'::jsonb, 'JUEZ_EFIMERO', 'python3 solution.py', 3000, ''),
    ('d0000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000004', 'Juez Algorítmico Java 21', 'Compilación y ejecución de clases Java en contenedor efímero aislado.', 'eclipse-temurin:21-alpine', 512, '["java", "javac"]'::jsonb, 'JUEZ_EFIMERO', 'javac Solution.java && java Solution', 5000, '')
ON CONFLICT (tenant_id, title) DO NOTHING;

-- Seed ejercicio algorítmico base
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
) ON CONFLICT (id) DO NOTHING;

-- Seed ejercicio de BD
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
) ON CONFLICT (id) DO NOTHING;

-- Migración de lab_instances legado (si existe)
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'lab_instances') THEN
        INSERT INTO workspaces (id, student_id, subject_id, container_id, status, type, access_url, memory_limit_mb, semgrep_audit, created_at, updated_at)
        SELECT id, user_id, template_id, container_id, status, 'JUEZ_EFIMERO', '', ram_limit_mb, semgrep_audit, created_at, updated_at
        FROM lab_instances
        ON CONFLICT (id) DO NOTHING;
        DROP TABLE lab_instances CASCADE;
    END IF;
END $$;

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS submission_comments, backup_executions, backup_configs, notifications, audit_logs, teacher_invitations, submissions, enrollments, workspaces, subjects, academic_periods, exercises, lab_template_profiles, template_models, lab_templates, template_categories, users, tenants CASCADE;
