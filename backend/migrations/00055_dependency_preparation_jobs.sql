-- +goose Up
-- Un trabajo por versión publicada. La cola vive en la base para que un
-- reinicio del nodo no pierda lo que todavía no se preparó, y el estado queda
-- visible para el docente que publicó la versión.
--
-- La clave se guarda con el trabajo, no se recalcula al leer: si el perfil o la
-- imagen cambian, la clave vieja sigue siendo la que identifica el árbol que se
-- preparó con ese perfil, y una versión nueva produce una clave nueva.
CREATE TABLE IF NOT EXISTS dependency_preparation_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    version_id UUID NOT NULL,
    project_id UUID NOT NULL,
    component_id UUID REFERENCES template_components(id) ON DELETE SET NULL,
    preparation_key VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    log_path TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    tree_path TEXT NOT NULL DEFAULT '',
    bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_dependency_preparation_jobs_version UNIQUE (version_id),
    CONSTRAINT chk_dependency_preparation_jobs_status
        CHECK (status IN ('pending', 'running', 'prepared', 'failed', 'skipped')),
    CONSTRAINT chk_dependency_preparation_jobs_key
        CHECK (preparation_key = '' OR preparation_key ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_dependency_preparation_jobs_attempts CHECK (attempts >= 0),
    CONSTRAINT chk_dependency_preparation_jobs_bytes CHECK (bytes >= 0),
    -- La versión pertenece a ese proyecto y a ese tenant; sin la FK compuesta
    -- un trabajo podría apuntar a la versión de otro proyecto.
    CONSTRAINT fk_dependency_preparation_jobs_version
        FOREIGN KEY (version_id, tenant_id, project_id)
        REFERENCES workspace_project_versions(id, tenant_id, project_id)
        ON DELETE RESTRICT
);

-- El worker busca siempre el siguiente pendiente por antigüedad.
CREATE INDEX IF NOT EXISTS idx_dependency_preparation_jobs_pending
    ON dependency_preparation_jobs(status, created_at);

CREATE INDEX IF NOT EXISTS idx_dependency_preparation_jobs_key
    ON dependency_preparation_jobs(preparation_key)
    WHERE preparation_key <> '';

-- +goose Down
DROP TABLE IF EXISTS dependency_preparation_jobs;
