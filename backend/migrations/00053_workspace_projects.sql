-- +goose Up
-- Fase 1: proyectos iniciales de IDE_PERSISTENTE.
-- Este esquema separa el proyecto, su borrador mutable y sus versiones
-- publicadas e inmutables. Las entregas históricas no se reescriben: se
-- clasifican como legacy_unknown y el proyecto base legacy queda en una tabla
-- de referencias sin relación con entregas ni versiones.

-- Unicidad de las claves compuestas que usan las FKs multi-tenant.
ALTER TABLE exercises ADD CONSTRAINT uq_exercises_id_tenant UNIQUE (id, tenant_id);

-- Hash del espejo de compatibilidad y origen de la versión publicada.
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS boilerplate_sha256 VARCHAR(64);
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS boilerplate_source_version_id UUID;

UPDATE exercises
SET boilerplate_sha256 = encode(sha256(convert_to(COALESCE(boilerplate, ''), 'UTF8')), 'hex')
WHERE boilerplate_sha256 IS NULL;

-- Proyecto: uno por ejercicio. La versión publicada se referencia con una FK
-- compuesta para que no pueda apuntar al proyecto o tenant de otro ejercicio.
CREATE TABLE IF NOT EXISTS workspace_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    exercise_id UUID NOT NULL,
    current_published_version_id UUID,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_workspace_projects_exercise UNIQUE (exercise_id),
    CONSTRAINT uq_workspace_projects_id_tenant UNIQUE (id, tenant_id),
    CONSTRAINT uq_workspace_projects_id_tenant_exercise UNIQUE (id, tenant_id, exercise_id),
    CONSTRAINT fk_workspace_projects_exercise
        FOREIGN KEY (exercise_id, tenant_id) REFERENCES exercises(id, tenant_id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_workspace_projects_tenant
    ON workspace_projects(tenant_id);

-- Versiones publicadas: inmutables y sin estados intermedios. published_at es
-- obligatorio porque una versión que no se publicó no es una versión.
CREATE TABLE IF NOT EXISTS workspace_project_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    project_id UUID NOT NULL,
    exercise_id UUID NOT NULL,
    version_number INT NOT NULL,
    text_snapshot TEXT NOT NULL,
    manifest JSONB NOT NULL DEFAULT '[]'::jsonb,
    content_sha256 VARCHAR(64) NOT NULL,
    file_count INT NOT NULL DEFAULT 0,
    total_bytes INT NOT NULL DEFAULT 0,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_workspace_project_versions_number UNIQUE (project_id, version_number),
    CONSTRAINT uq_workspace_project_versions_id_tenant UNIQUE (id, tenant_id),
    CONSTRAINT uq_workspace_project_versions_id_tenant_project UNIQUE (id, tenant_id, project_id),
    CONSTRAINT uq_workspace_project_versions_id_tenant_exercise UNIQUE (id, tenant_id, exercise_id),
    CONSTRAINT chk_workspace_project_versions_number CHECK (version_number > 0),
    CONSTRAINT chk_workspace_project_versions_hash CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_workspace_project_versions_manifest CHECK (jsonb_typeof(manifest) = 'array'),
    CONSTRAINT chk_workspace_project_versions_text CHECK (btrim(text_snapshot) <> ''),
    CONSTRAINT chk_workspace_project_versions_size CHECK (octet_length(text_snapshot) <= 1048576),
    CONSTRAINT chk_workspace_project_versions_files CHECK (file_count BETWEEN 0 AND 300),
    CONSTRAINT chk_workspace_project_versions_bytes CHECK (total_bytes BETWEEN 0 AND 1048576),
    CONSTRAINT fk_workspace_project_versions_project
        FOREIGN KEY (project_id, tenant_id) REFERENCES workspace_projects(id, tenant_id)
        ON DELETE RESTRICT,
    -- La versión pertenece al proyecto Y al ejercicio de ese proyecto. Sin
    -- esta FK, una versión podría declarar el ejercicio vecino.
    CONSTRAINT fk_workspace_project_versions_project_exercise
        FOREIGN KEY (project_id, tenant_id, exercise_id)
        REFERENCES workspace_projects(id, tenant_id, exercise_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_workspace_project_versions_exercise
        FOREIGN KEY (exercise_id, tenant_id) REFERENCES exercises(id, tenant_id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_workspace_project_versions_project
    ON workspace_project_versions(project_id, version_number DESC);

ALTER TABLE workspace_projects
    ADD CONSTRAINT fk_workspace_projects_current_version
    FOREIGN KEY (current_published_version_id, tenant_id, id)
    REFERENCES workspace_project_versions(id, tenant_id, project_id)
    ON DELETE RESTRICT
    DEFERRABLE INITIALLY DEFERRED;

-- Borrador mutable: uno por proyecto. base_version_id permite publicar sobre
-- una versión conocida y la FK compuesta impide mezclar tenants o proyectos.
CREATE TABLE IF NOT EXISTS workspace_project_drafts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    project_id UUID NOT NULL,
    base_version_id UUID,
    text_snapshot TEXT NOT NULL DEFAULT '',
    content_sha256 VARCHAR(64) NOT NULL DEFAULT '',
    revision BIGINT NOT NULL DEFAULT 1,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_workspace_project_drafts_project UNIQUE (project_id),
    CONSTRAINT uq_workspace_project_drafts_id_tenant UNIQUE (id, tenant_id),
    CONSTRAINT chk_workspace_project_drafts_revision CHECK (revision > 0),
    CONSTRAINT chk_workspace_project_drafts_size CHECK (octet_length(text_snapshot) <= 1048576),
    CONSTRAINT fk_workspace_project_drafts_project
        FOREIGN KEY (project_id, tenant_id) REFERENCES workspace_projects(id, tenant_id)
        ON DELETE RESTRICT,
    CONSTRAINT fk_workspace_project_drafts_base_version
        FOREIGN KEY (base_version_id, tenant_id, project_id)
        REFERENCES workspace_project_versions(id, tenant_id, project_id)
        ON DELETE RESTRICT
        DEFERRABLE INITIALLY DEFERRED
);

-- Referencias legacy: el boilerplate histórico del ejercicio. No tiene FK a
-- entregas ni se usa como base_version_id; sirve para auditar el origen.
CREATE TABLE IF NOT EXISTS workspace_legacy_project_references (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    exercise_id UUID NOT NULL,
    text_snapshot TEXT NOT NULL,
    content_sha256 VARCHAR(64) NOT NULL,
    source VARCHAR(40) NOT NULL DEFAULT 'exercise_boilerplate',
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_workspace_legacy_project_exercise UNIQUE (exercise_id),
    CONSTRAINT chk_workspace_legacy_project_hash CHECK (content_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_workspace_legacy_project_size CHECK (octet_length(text_snapshot) <= 2097152),
    CONSTRAINT chk_workspace_legacy_project_source CHECK (source IN ('exercise_boilerplate')),
    CONSTRAINT fk_workspace_legacy_project_exercise
        FOREIGN KEY (exercise_id, tenant_id) REFERENCES exercises(id, tenant_id)
        ON DELETE RESTRICT
);

ALTER TABLE exercises
    ADD CONSTRAINT fk_exercises_boilerplate_source_version
    FOREIGN KEY (boilerplate_source_version_id, tenant_id, id)
    REFERENCES workspace_project_versions(id, tenant_id, exercise_id)
    ON DELETE RESTRICT
    DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE exercises
    ADD CONSTRAINT chk_exercises_boilerplate_source_hash
    CHECK (boilerplate_source_version_id IS NULL OR boilerplate_sha256 IS NOT NULL);

-- Clasificación del entorno. Los entornos existentes son legacy; los nuevos
-- nacen sin proyecto hasta que el aprovisionamiento de una versión lo indique.
ALTER TABLE environments ADD COLUMN IF NOT EXISTS workspace_project_version_id UUID;
ALTER TABLE environments ADD COLUMN IF NOT EXISTS workspace_seed_mode VARCHAR(20);
ALTER TABLE environments ADD COLUMN IF NOT EXISTS workspace_init_status VARCHAR(20);

UPDATE environments
SET workspace_seed_mode = 'legacy',
    workspace_init_status = 'legacy'
WHERE workspace_seed_mode IS NULL;

ALTER TABLE environments ALTER COLUMN workspace_seed_mode SET DEFAULT 'none';
ALTER TABLE environments ALTER COLUMN workspace_init_status SET DEFAULT 'not_required';
UPDATE environments SET workspace_init_status = 'not_required' WHERE workspace_init_status IS NULL;
ALTER TABLE environments ALTER COLUMN workspace_seed_mode SET NOT NULL;
ALTER TABLE environments ALTER COLUMN workspace_init_status SET NOT NULL;

ALTER TABLE environments
    ADD CONSTRAINT chk_environments_workspace_seed_mode
    CHECK (workspace_seed_mode IN ('project', 'none', 'legacy'));
ALTER TABLE environments
    ADD CONSTRAINT chk_environments_workspace_init_status
    CHECK (workspace_init_status IN ('legacy', 'pending', 'volume_created', 'setup_done', 'failed', 'quarantined', 'missing', 'not_required'));
-- La correspondencia legacy es bidireccional. project y none admiten todos
-- sus estados transitorios legítimos; legacy solo puede estar en legacy.
ALTER TABLE environments
    ADD CONSTRAINT chk_environments_workspace_legacy_pair
    CHECK ((workspace_seed_mode = 'legacy') = (workspace_init_status = 'legacy'));
ALTER TABLE environments
    ADD CONSTRAINT chk_environments_workspace_version_by_mode
    CHECK (
        (workspace_seed_mode = 'project' AND workspace_project_version_id IS NOT NULL)
        OR (workspace_seed_mode IN ('none', 'legacy') AND workspace_project_version_id IS NULL)
    );
ALTER TABLE environments
    ADD CONSTRAINT chk_environments_workspace_project_exercise
    CHECK (workspace_seed_mode <> 'project' OR exercise_id IS NOT NULL);

CREATE UNIQUE INDEX IF NOT EXISTS uq_environments_id_tenant_exercise
    ON environments(id, tenant_id, exercise_id);

ALTER TABLE environments
    ADD CONSTRAINT fk_environments_workspace_project_version
    FOREIGN KEY (workspace_project_version_id, tenant_id, exercise_id)
    REFERENCES workspace_project_versions(id, tenant_id, exercise_id)
    ON DELETE RESTRICT
    DEFERRABLE INITIALLY DEFERRED;

-- Entrega: base_status clasifica la base de comparación. El juez no tiene
-- base_status (NULL) y conserva exercises.boilerplate.
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS base_status VARCHAR(20);
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS workspace_project_version_id UUID;

UPDATE submissions sub
SET base_status = 'legacy_unknown'
FROM exercises ex
WHERE ex.id = sub.exercise_id
  AND ex.environment_type = 'IDE_PERSISTENTE'
  AND sub.base_status IS NULL;

ALTER TABLE submissions
    ADD CONSTRAINT chk_submissions_base_status
    CHECK (base_status IS NULL OR base_status IN ('version', 'no_base', 'legacy_unknown'));
ALTER TABLE submissions
    ADD CONSTRAINT chk_submissions_base_status_version
    CHECK (
        (base_status = 'version' AND workspace_project_version_id IS NOT NULL)
        OR (base_status IS DISTINCT FROM 'version' AND workspace_project_version_id IS NULL)
    );
ALTER TABLE submissions
    ADD CONSTRAINT fk_submissions_workspace_project_version
    FOREIGN KEY (workspace_project_version_id, tenant_id, exercise_id)
    REFERENCES workspace_project_versions(id, tenant_id, exercise_id)
    ON DELETE RESTRICT
    DEFERRABLE INITIALLY DEFERRED;
-- La FK de entorno se agrega NOT VALID para no romper el despliegue por
-- entregas históricas que apuntan a entornos genéricos sin ejercicio. Las
-- filas nuevas y las actualizaciones sí quedan sujetas a la relación exacta.
ALTER TABLE submissions
    ADD CONSTRAINT fk_submissions_environment_scope
    FOREIGN KEY (environment_id, tenant_id, exercise_id)
    REFERENCES environments(id, tenant_id, exercise_id)
    ON DELETE RESTRICT
    NOT VALID;

CREATE INDEX IF NOT EXISTS idx_submissions_base_status
    ON submissions(base_status);
CREATE INDEX IF NOT EXISTS idx_submissions_workspace_project_version
    ON submissions(workspace_project_version_id);

-- Backfill de referencias legacy del IDE. Solo se copia el boilerplate que ya
-- existía; no se inventan versiones ni se tocan entregas.
INSERT INTO workspace_legacy_project_references
    (tenant_id, exercise_id, text_snapshot, content_sha256, source, captured_at)
SELECT e.tenant_id,
       e.id,
       e.boilerplate,
       encode(sha256(convert_to(e.boilerplate, 'UTF8')), 'hex'),
       'exercise_boilerplate',
       NOW()
FROM exercises e
WHERE e.environment_type = 'IDE_PERSISTENTE'
  AND e.tenant_id IS NOT NULL
  AND btrim(COALESCE(e.boilerplate, '')) <> ''
ON CONFLICT (exercise_id) DO NOTHING;

-- Una versión publicada no se modifica ni se borra. El historial de entregas
-- depende de que su texto permanezca estable.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION solv_workspace_project_version_immutable()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        RAISE EXCEPTION 'workspace_project_versions es inmutable'
            USING ERRCODE = '23514';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'workspace_project_versions no se puede borrar'
            USING ERRCODE = '23503';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_workspace_project_version_immutable ON workspace_project_versions;
CREATE TRIGGER trg_workspace_project_version_immutable
    BEFORE UPDATE OR DELETE ON workspace_project_versions
    FOR EACH ROW EXECUTE FUNCTION solv_workspace_project_version_immutable();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION solv_workspace_legacy_project_immutable()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        RAISE EXCEPTION 'workspace_legacy_project_references es inmutable'
            USING ERRCODE = '23514';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'workspace_legacy_project_references no se puede borrar'
            USING ERRCODE = '23503';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_workspace_legacy_project_immutable ON workspace_legacy_project_references;
CREATE TRIGGER trg_workspace_legacy_project_immutable
    BEFORE UPDATE OR DELETE ON workspace_legacy_project_references
    FOR EACH ROW EXECUTE FUNCTION solv_workspace_legacy_project_immutable();

-- El espejo exercises.boilerplate solo puede cambiar junto con su versión de
-- origen. Un update directo del texto falla; la publicación cambia ambos.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION solv_exercise_boilerplate_guard()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.boilerplate_source_version_id IS NOT NULL
       AND NEW.boilerplate IS DISTINCT FROM OLD.boilerplate
       AND NEW.boilerplate_source_version_id IS NOT DISTINCT FROM OLD.boilerplate_source_version_id THEN
        RAISE EXCEPTION 'exercises.boilerplate está gestionado por workspace_project_versions'
            USING ERRCODE = '23514';
    END IF;

    IF NEW.boilerplate_source_version_id IS NOT NULL THEN
        IF NEW.boilerplate_sha256 IS DISTINCT FROM encode(sha256(convert_to(COALESCE(NEW.boilerplate, ''), 'UTF8')), 'hex') THEN
            RAISE EXCEPTION 'exercises.boilerplate_sha256 no coincide con el espejo'
                USING ERRCODE = '23514';
        END IF;
        IF NOT EXISTS (
            SELECT 1 FROM workspace_project_versions v
            WHERE v.id = NEW.boilerplate_source_version_id
              AND v.tenant_id = NEW.tenant_id
              AND v.exercise_id = NEW.id
              AND v.text_snapshot IS NOT DISTINCT FROM NEW.boilerplate
        ) THEN
            RAISE EXCEPTION 'exercises.boilerplate no coincide con la versión de origen'
                USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_exercise_boilerplate_guard ON exercises;
CREATE TRIGGER trg_exercise_boilerplate_guard
    BEFORE UPDATE ON exercises
    FOR EACH ROW EXECUTE FUNCTION solv_exercise_boilerplate_guard();

-- +goose Down
-- Rollback destructivo y solo para entornos descartables. No se ofrece un down
-- automático de producción: las versiones y entregas son historial y el orden
-- de borrado debe revisarse antes de ejecutarlo.
DROP TRIGGER IF EXISTS trg_exercise_boilerplate_guard ON exercises;
DROP FUNCTION IF EXISTS solv_exercise_boilerplate_guard();
DROP TRIGGER IF EXISTS trg_workspace_legacy_project_immutable ON workspace_legacy_project_references;
DROP FUNCTION IF EXISTS solv_workspace_legacy_project_immutable();
DROP TRIGGER IF EXISTS trg_workspace_project_version_immutable ON workspace_project_versions;
DROP FUNCTION IF EXISTS solv_workspace_project_version_immutable();

ALTER TABLE submissions DROP CONSTRAINT IF EXISTS fk_submissions_environment_scope;
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS fk_submissions_workspace_project_version;
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS chk_submissions_base_status_version;
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS chk_submissions_base_status;
ALTER TABLE submissions DROP COLUMN IF EXISTS workspace_project_version_id;
ALTER TABLE submissions DROP COLUMN IF EXISTS base_status;

ALTER TABLE environments DROP CONSTRAINT IF EXISTS fk_environments_workspace_project_version;
ALTER TABLE environments DROP CONSTRAINT IF EXISTS chk_environments_workspace_project_exercise;
ALTER TABLE environments DROP CONSTRAINT IF EXISTS chk_environments_workspace_version_by_mode;
ALTER TABLE environments DROP CONSTRAINT IF EXISTS chk_environments_workspace_legacy_pair;
ALTER TABLE environments DROP CONSTRAINT IF EXISTS chk_environments_workspace_init_status;
ALTER TABLE environments DROP CONSTRAINT IF EXISTS chk_environments_workspace_seed_mode;
DROP INDEX IF EXISTS uq_environments_id_tenant_exercise;
ALTER TABLE environments DROP COLUMN IF EXISTS workspace_init_status;
ALTER TABLE environments DROP COLUMN IF EXISTS workspace_seed_mode;
ALTER TABLE environments DROP COLUMN IF EXISTS workspace_project_version_id;

ALTER TABLE exercises DROP CONSTRAINT IF EXISTS chk_exercises_boilerplate_source_hash;
ALTER TABLE exercises DROP CONSTRAINT IF EXISTS fk_exercises_boilerplate_source_version;
ALTER TABLE exercises DROP COLUMN IF EXISTS boilerplate_source_version_id;
ALTER TABLE exercises DROP COLUMN IF EXISTS boilerplate_sha256;
DROP TABLE IF EXISTS workspace_legacy_project_references;
DROP TABLE IF EXISTS workspace_project_drafts;
ALTER TABLE workspace_projects DROP CONSTRAINT IF EXISTS fk_workspace_projects_current_version;
DROP TABLE IF EXISTS workspace_project_versions;
DROP TABLE IF EXISTS workspace_projects;
ALTER TABLE exercises DROP CONSTRAINT IF EXISTS uq_exercises_id_tenant;
