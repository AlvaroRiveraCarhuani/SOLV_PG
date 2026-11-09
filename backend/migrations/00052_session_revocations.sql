-- +goose Up
-- El logout debe invalidar el JWT en todas las instancias y sobrevivir
-- reinicios. Se guarda solo el identificador de sesión (jti o hash SHA-256 del
-- token heredado), nunca el JWT completo.
CREATE TABLE IF NOT EXISTS session_revocations (
    session_id TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_session_revocations_expires_at
    ON session_revocations (expires_at);

-- +goose Down
DROP TABLE IF EXISTS session_revocations;