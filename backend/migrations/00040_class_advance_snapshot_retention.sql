-- +goose Up
-- Retención de snapshots de avance (ADR-058 §6). El objeto tar.gz vive en
-- SeaweedFS; esta marca evita volver a recorrer avances ya limpiados y permite
-- distinguir un snapshot eliminado de uno que todavía se puede leer.
ALTER TABLE class_advances ADD COLUMN IF NOT EXISTS snapshot_purged_at TIMESTAMPTZ;

-- El barrido busca solo las filas pendientes de limpiar. El índice parcial no
-- crece con los avances que ya quedaron marcados.
CREATE INDEX IF NOT EXISTS idx_class_advances_pending_purge
    ON class_advances (snapshot_purged_at)
    WHERE snapshot_purged_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_class_advances_pending_purge;
ALTER TABLE class_advances DROP COLUMN IF EXISTS snapshot_purged_at;
