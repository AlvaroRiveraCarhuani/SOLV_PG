-- +goose Up
-- Migration 00019: Complejidad esperada en ejercicios y análisis de complejidad empírica en submissions
ALTER TABLE exercises
  ADD COLUMN IF NOT EXISTS expected_complexity VARCHAR(50);

ALTER TABLE submissions
  ADD COLUMN IF NOT EXISTS complexity_analysis JSONB;

-- +goose Down
ALTER TABLE submissions
  DROP COLUMN IF EXISTS complexity_analysis;

ALTER TABLE exercises
  DROP COLUMN IF EXISTS expected_complexity;
