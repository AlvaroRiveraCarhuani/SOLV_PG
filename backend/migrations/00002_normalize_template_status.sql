-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- Normalización de vocabulario de estados en lab_templates
-- v1 → v2
--   'approved'  → 'APROBADA'
--   'pending'   → 'PENDIENTE_AUDITORIA'
--   'rejected'  → 'RECHAZADA'
--   'paused'    → 'PAUSADA'
--
-- El helper approvedTemplateStatusClause en el repositorio ya
-- maneja ambos vocabularios con OR durante la ventana de
-- transición; esta migración completa la unificación.
-- ============================================================

UPDATE lab_templates SET status = 'APROBADA'            WHERE status = 'approved';
UPDATE lab_templates SET status = 'PENDIENTE_AUDITORIA' WHERE status = 'pending';
UPDATE lab_templates SET status = 'RECHAZADA'           WHERE status = 'rejected';
UPDATE lab_templates SET status = 'PAUSADA'             WHERE status = 'paused';

-- Corregir el DEFAULT de la columna: ya no debe silenciosamente publicar plantillas
ALTER TABLE lab_templates ALTER COLUMN status SET DEFAULT 'PENDIENTE_AUDITORIA';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Revert: volver a vocabulary v1 (solo para rollback de emergencia)
UPDATE lab_templates SET status = 'approved'  WHERE status = 'APROBADA';
UPDATE lab_templates SET status = 'pending'   WHERE status = 'PENDIENTE_AUDITORIA';
UPDATE lab_templates SET status = 'rejected'  WHERE status = 'RECHAZADA';
UPDATE lab_templates SET status = 'paused'    WHERE status = 'PAUSADA';
ALTER TABLE lab_templates ALTER COLUMN status SET DEFAULT 'approved';
-- +goose StatementEnd
