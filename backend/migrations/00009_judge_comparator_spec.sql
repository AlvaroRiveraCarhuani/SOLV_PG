-- +goose Up
-- Juez de ejercicios: comparador declarado por ejercicio.
-- Columna dedicada comparator_spec JSONB NOT NULL con default {"id":"exact"}
-- (identificador mas parametros). El override por caso vive dentro de
-- config (config.test_cases[].comparator) y prevalece sobre el ejercicio.
-- Aditivo: conserva comportamiento (exact era el unico criterio previo).

ALTER TABLE exercises ADD COLUMN IF NOT EXISTS comparator_spec JSONB NOT NULL DEFAULT '{"id":"exact"}'::jsonb;

-- +goose Down
ALTER TABLE exercises DROP COLUMN IF EXISTS comparator_spec;
