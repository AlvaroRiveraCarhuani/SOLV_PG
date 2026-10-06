-- +goose Up
-- Migration 00017: Guardar snapshot de casos de prueba generados por estudiante en examenes
ALTER TABLE submissions
  ADD COLUMN IF NOT EXISTS generated_cases JSONB;

-- +goose Down
ALTER TABLE submissions
  DROP COLUMN IF EXISTS generated_cases;
