-- +goose Up
-- Ejercicios: el discriminador de modalidad es environment_type (JUEZ_EFIMERO /
-- IDE_PERSISTENTE), ya restringido por CHECK. La columna type debe quedar
-- acotada a los valores canonicos (algorithm | database | project) y dejar de
-- aceptar valores legacy de la demo ('code').
--
-- 1. Backfill de type legacy 'code' según environment_type:
--    - IDE_PERSISTENTE -> project (entrega por snapshot + rúbrica/SpeedGrader)
--    - JUEZ_EFIMERO / sin entorno -> algorithm (juez I/O)
UPDATE exercises SET type = 'project'
WHERE type = 'code' AND environment_type = 'IDE_PERSISTENTE';

UPDATE exercises SET type = 'algorithm'
WHERE type = 'code'
  AND (environment_type = 'JUEZ_EFIMERO' OR environment_type IS NULL OR environment_type = '');

-- 2. CHECK constraint: type solo acepta los valores canonicos.
ALTER TABLE exercises ADD CONSTRAINT exercises_type_check
CHECK (type IN ('algorithm', 'database', 'project'));

-- +goose Down
ALTER TABLE exercises DROP CONSTRAINT IF EXISTS exercises_type_check;
