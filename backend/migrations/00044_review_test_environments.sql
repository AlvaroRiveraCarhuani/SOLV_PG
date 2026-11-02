-- +goose Up
-- Preview efímero de una entrega IDE. El contenedor ejecuta una copia del
-- snapshot en un volumen propio; nunca monta el workspace vivo del alumno.
-- El TTL lo gobierna el QoS Worker por inactividad (ADR-011).

CREATE TABLE IF NOT EXISTS review_test_environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    container_id TEXT,
    volume_name TEXT NOT NULL,
    access_url TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'starting',
    error_code VARCHAR(64) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_rx_bytes BIGINT NOT NULL DEFAULT 0,
    last_tx_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE review_test_environments
    DROP CONSTRAINT IF EXISTS chk_review_test_environments_status;

ALTER TABLE review_test_environments
    ADD CONSTRAINT chk_review_test_environments_status
    CHECK (status IN ('starting', 'running', 'expired', 'failed'));

-- Un solo preview activo por entrega. Un fallo o un vencimiento libera el
-- índice y permite reintentar sin borrar el historial.
CREATE UNIQUE INDEX IF NOT EXISTS uq_review_test_environments_active
    ON review_test_environments(submission_id)
    WHERE status IN ('starting', 'running');

CREATE INDEX IF NOT EXISTS idx_review_test_environments_active_expiry
    ON review_test_environments(expires_at)
    WHERE status IN ('starting', 'running');

CREATE INDEX IF NOT EXISTS idx_review_test_environments_tenant
    ON review_test_environments(tenant_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_review_test_environments_tenant;
DROP INDEX IF EXISTS idx_review_test_environments_active_expiry;
DROP INDEX IF EXISTS uq_review_test_environments_active;
DROP TABLE IF EXISTS review_test_environments;
