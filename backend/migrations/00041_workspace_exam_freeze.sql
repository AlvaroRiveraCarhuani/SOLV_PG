-- +goose Up
-- Congelamiento del examen (ADR-005 §5). Al cerrar el examen el volumen del
-- alumno queda montado en solo lectura y la hora se guarda acá: el estado
-- sobrevive a un reinicio del backend y la reanudación no puede deshacerlo.
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS frozen_at TIMESTAMPTZ;

-- El barrido busca laboratorios congelados sin recorrer toda la tabla.
CREATE INDEX IF NOT EXISTS idx_workspaces_frozen_at
    ON workspaces (frozen_at)
    WHERE frozen_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_workspaces_frozen_at;
ALTER TABLE workspaces DROP COLUMN IF EXISTS frozen_at;
