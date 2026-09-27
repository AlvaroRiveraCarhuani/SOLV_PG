-- +goose Up
-- ADR-029 (hardening): archivado formal e irreversible de períodos académicos.
-- Cierra la divergencia entre el wireframe oficial (docs/UI/ADMIN/CONFIGURACION.md)
-- y la implementación: hasta ahora "archivado" era derivado (is_active=false +
-- fecha vencida) y reactivable con un simple PUT. Con is_archived persistido, el
-- congelamiento institucional es un estado real de la BD, con trazabilidad de
-- quién y cuándo archivó (Ley 4 de UX: inmutabilidad de actas y entregas).

ALTER TABLE academic_periods ADD COLUMN IF NOT EXISTS is_archived BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE academic_periods ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;
ALTER TABLE academic_periods ADD COLUMN IF NOT EXISTS archived_by UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_academic_periods_archived ON academic_periods(tenant_id, is_archived);

-- Los períodos que ya estaban vencidos e inactivos se consideran archivados de
-- facto por el sweep histórico: se normaliza su estado al nuevo modelo.
UPDATE academic_periods
SET is_archived = TRUE, archived_at = COALESCE(archived_at, NOW())
WHERE is_active = FALSE AND end_date < CURRENT_DATE;

-- +goose Down
DROP INDEX IF EXISTS idx_academic_periods_archived;
ALTER TABLE academic_periods DROP COLUMN IF EXISTS archived_by;
ALTER TABLE academic_periods DROP COLUMN IF EXISTS archived_at;
ALTER TABLE academic_periods DROP COLUMN IF EXISTS is_archived;
