-- +goose Up
-- Resultado de la verificación de publicación sin red (ADR-085).
-- `exam_offline_status` vale vacío (nunca se corrió), `passed`, `failed` o
-- `skipped` (el ejercicio no tiene proyecto base que verificar). El output se
-- guarda recortado: la pantalla muestra el estado y el motivo, no un log entero.
ALTER TABLE exercises
    ADD COLUMN IF NOT EXISTS exam_offline_status TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS exam_offline_output TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS exam_offline_checked_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE exercises
    DROP COLUMN IF EXISTS exam_offline_status,
    DROP COLUMN IF EXISTS exam_offline_output,
    DROP COLUMN IF EXISTS exam_offline_checked_at;

