-- +goose Up
-- Guardas de versión para el proyecto grande, reaplicadas (ADR-090).
--
-- 00067 reemplazó los CHECK de 00053 para distinguir texto de artefacto, pero
-- una instalación que ya había aplicado esa migración antes de ese ajuste
-- conserva las guardas viejas: goose no vuelve a correr una versión aplicada.
-- Esta migración deja las mismas guardas de forma idempotente, así una base
-- existente acepta publicar una versión con artefacto igual que una nueva.
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
-- Volver a las guardas de 00053 exige que no queden versiones con artefacto:
-- su snapshot vacío las viola. El trigger de inmutabilidad impide borrarlas, así
-- que se desactiva dentro de esta transacción y se vuelve a activar. Los objetos
-- del depósito no se tocan.
ALTER TABLE workspace_project_versions DISABLE TRIGGER trg_workspace_project_version_immutable;

DELETE FROM workspace_project_versions WHERE artifact_key IS NOT NULL;

ALTER TABLE workspace_project_versions ENABLE TRIGGER trg_workspace_project_version_immutable;

ALTER TABLE workspace_project_versions
    DROP CONSTRAINT IF EXISTS chk_workspace_project_versions_text,
    DROP CONSTRAINT IF EXISTS chk_workspace_project_versions_files,
    DROP CONSTRAINT IF EXISTS chk_workspace_project_versions_bytes;

ALTER TABLE workspace_project_versions
    ADD CONSTRAINT chk_workspace_project_versions_text CHECK (btrim(text_snapshot) <> ''),
    ADD CONSTRAINT chk_workspace_project_versions_files CHECK (file_count BETWEEN 0 AND 300),
    ADD CONSTRAINT chk_workspace_project_versions_bytes CHECK (total_bytes BETWEEN 0 AND 1048576);
