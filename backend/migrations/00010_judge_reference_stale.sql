-- +goose Up
-- Juez de ejercicios: referencia obligatoria y ciclo de publicacion (v1 stdin).
-- reference_solution persiste la solucion de referencia para dry-run y
-- recalificacion. stale bloquea la publicacion (409) hasta un nuevo dry-run
-- valido. last_valid_dry_run_at registra el ultimo dry-run valido.
-- Aditivo: columnas nuevas, ningun estado intermedio bloquea sin dry-run.

ALTER TABLE exercises ADD COLUMN IF NOT EXISTS reference_solution TEXT NOT NULL DEFAULT '';
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS stale BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS last_valid_dry_run_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_exercises_stale ON exercises(tenant_id, stale);

-- +goose Down
DROP INDEX IF EXISTS idx_exercises_stale;
ALTER TABLE exercises DROP COLUMN IF EXISTS last_valid_dry_run_at;
ALTER TABLE exercises DROP COLUMN IF EXISTS stale;
ALTER TABLE exercises DROP COLUMN IF EXISTS reference_solution;
