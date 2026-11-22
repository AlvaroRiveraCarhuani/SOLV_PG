-- +goose Up
-- Proyecto inicial como artefacto versionado (ADR-090).
--
-- El árbol grande deja de ser texto en la base: se guarda como tar.gz en el
-- depósito de objetos y la fila conserva la referencia, el hash y el tamaño. El
-- borrador lo usa como área de preparación mientras el docente lo revisa y la
-- publicación lo copia a la versión inmutable.
--
-- Los cuatro campos son nulos en las filas existentes: el camino liviano sigue
-- igual y una versión nunca queda mezclada entre los dos modos.
ALTER TABLE workspace_project_drafts
    ADD COLUMN IF NOT EXISTS artifact_key    TEXT,
    ADD COLUMN IF NOT EXISTS artifact_sha256 CHAR(64),
    ADD COLUMN IF NOT EXISTS artifact_bytes  BIGINT,
    ADD COLUMN IF NOT EXISTS artifact_files  INTEGER;

ALTER TABLE workspace_project_drafts
    ADD CONSTRAINT workspace_project_drafts_artifact_all_or_none CHECK (
        (artifact_key IS NULL AND artifact_sha256 IS NULL AND artifact_bytes IS NULL AND artifact_files IS NULL)
        OR (artifact_key IS NOT NULL AND artifact_sha256 IS NOT NULL AND artifact_bytes IS NOT NULL AND artifact_files IS NOT NULL)
    );

ALTER TABLE workspace_project_versions
    ADD COLUMN IF NOT EXISTS artifact_key    TEXT,
    ADD COLUMN IF NOT EXISTS artifact_sha256 CHAR(64),
    ADD COLUMN IF NOT EXISTS artifact_bytes  BIGINT,
    ADD COLUMN IF NOT EXISTS artifact_files  INTEGER;

ALTER TABLE workspace_project_versions
    ADD CONSTRAINT workspace_project_versions_artifact_all_or_none CHECK (
        (artifact_key IS NULL AND artifact_sha256 IS NULL AND artifact_bytes IS NULL AND artifact_files IS NULL)
        OR (artifact_key IS NOT NULL AND artifact_sha256 IS NOT NULL AND artifact_bytes IS NOT NULL AND artifact_files IS NOT NULL)
    );

-- Las guardas de 00053 describen solo el camino liviano: texto no vacío, hasta
-- 300 archivos y 1 MiB. Una versión con artefacto tiene el snapshot vacío y un
-- árbol mucho mayor, así que el CHECK pasa a distinguir los dos modos en lugar
-- de aflojarse para todos.
ALTER TABLE workspace_project_versions
    DROP CONSTRAINT IF EXISTS chk_workspace_project_versions_text,
    DROP CONSTRAINT IF EXISTS chk_workspace_project_versions_files,
    DROP CONSTRAINT IF EXISTS chk_workspace_project_versions_bytes;

ALTER TABLE workspace_project_versions
    ADD CONSTRAINT chk_workspace_project_versions_text CHECK (
        (artifact_key IS NULL AND btrim(text_snapshot) <> '')
        OR (artifact_key IS NOT NULL AND text_snapshot = '')
    ),
    ADD CONSTRAINT chk_workspace_project_versions_files CHECK (
        (artifact_key IS NULL AND file_count BETWEEN 0 AND 300)
        OR (artifact_key IS NOT NULL AND file_count BETWEEN 1 AND 20000)
    ),
    ADD CONSTRAINT chk_workspace_project_versions_bytes CHECK (
        (artifact_key IS NULL AND total_bytes BETWEEN 0 AND 1048576)
        OR (artifact_key IS NOT NULL AND total_bytes BETWEEN 1 AND 134217728)
    );

-- +goose Down
-- Una versión con artefacto no se puede representar con las guardas de 00053:
-- su snapshot está vacío y el árbol excede los topes. Se eliminan esas filas y
-- se vacía la preparación de los borradores antes de restaurar el esquema
-- anterior. El trigger de inmutabilidad impide el borrado, así que se desactiva
-- dentro de esta misma transacción y se vuelve a activar. Los objetos del
-- depósito no se tocan: quedan para la retención.
ALTER TABLE workspace_project_versions DISABLE TRIGGER trg_workspace_project_version_immutable;

DELETE FROM workspace_project_versions WHERE artifact_key IS NOT NULL;

ALTER TABLE workspace_project_versions ENABLE TRIGGER trg_workspace_project_version_immutable;

UPDATE workspace_project_drafts
SET artifact_key = NULL,
    artifact_sha256 = NULL,
    artifact_bytes = NULL,
    artifact_files = NULL;

ALTER TABLE workspace_project_versions
    DROP CONSTRAINT IF EXISTS chk_workspace_project_versions_text,
    DROP CONSTRAINT IF EXISTS chk_workspace_project_versions_files,
    DROP CONSTRAINT IF EXISTS chk_workspace_project_versions_bytes;

ALTER TABLE workspace_project_versions
    ADD CONSTRAINT chk_workspace_project_versions_text CHECK (btrim(text_snapshot) <> ''),
    ADD CONSTRAINT chk_workspace_project_versions_files CHECK (file_count BETWEEN 0 AND 300),
    ADD CONSTRAINT chk_workspace_project_versions_bytes CHECK (total_bytes BETWEEN 0 AND 1048576);

ALTER TABLE workspace_project_versions
    DROP CONSTRAINT IF EXISTS workspace_project_versions_artifact_all_or_none,
    DROP COLUMN IF EXISTS artifact_key,
    DROP COLUMN IF EXISTS artifact_sha256,
    DROP COLUMN IF EXISTS artifact_bytes,
    DROP COLUMN IF EXISTS artifact_files;

ALTER TABLE workspace_project_drafts
    DROP CONSTRAINT IF EXISTS workspace_project_drafts_artifact_all_or_none,
    DROP COLUMN IF EXISTS artifact_key,
    DROP COLUMN IF EXISTS artifact_sha256,
    DROP COLUMN IF EXISTS artifact_bytes,
    DROP COLUMN IF EXISTS artifact_files;
