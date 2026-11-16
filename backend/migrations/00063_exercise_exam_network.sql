-- +goose Up
-- Política de red del examen por ejercicio. `none` es el default: el entorno
-- corre en una red interna sin puerta de enlace. `egress` es la excepción del
-- docente y exige motivo; el por qué queda en audit_logs.
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS exam_network TEXT NOT NULL DEFAULT 'none';
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS exam_network_reason TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE exercises DROP COLUMN IF EXISTS exam_network_reason;
ALTER TABLE exercises DROP COLUMN IF EXISTS exam_network;
