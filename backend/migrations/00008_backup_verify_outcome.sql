-- +goose Up
-- ADR-040 (verify integrity): persistencia del resultado de verificación en la fila de ejecución.
-- Agrega columnas nullables last_verify_ok / last_verify_at a backup_executions.
-- NULL = nunca verificado; se puebla solo desde VerifyBackup (nunca desde Trigger).

ALTER TABLE backup_executions ADD COLUMN IF NOT EXISTS last_verify_ok BOOLEAN;
ALTER TABLE backup_executions ADD COLUMN IF NOT EXISTS last_verify_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE backup_executions DROP COLUMN IF EXISTS last_verify_at;
ALTER TABLE backup_executions DROP COLUMN IF EXISTS last_verify_ok;
