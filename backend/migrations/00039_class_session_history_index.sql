-- +goose Up
-- Historial de una clase (ADR-058 §12). Los movimientos del aula viven en
-- audit_logs con resource_type = 'class_session' y resource_id = id de la
-- sesión. El índice parcial evita que leer un aula recorra todo el registro de
-- la institución, que también guarda las peticiones HTTP del middleware.
CREATE INDEX IF NOT EXISTS idx_audit_logs_class_session
    ON audit_logs (tenant_id, resource_id, created_at DESC)
    WHERE resource_type = 'class_session';

-- +goose Down
DROP INDEX IF EXISTS idx_audit_logs_class_session;
