-- +goose Up
-- Un veredicto guardado no dice por sí mismo quién lo escribió. Esta columna lo
-- declara para que la interfaz no presente un resultado heredado como si fuera
-- una evaluación del juez.
--
-- Valores:
--   judge    -> lo escribió el camino de entrega del juez
--   rubric   -> lo escribió la rúbrica (comportamiento anterior a la separación)
--   override -> lo escribió un ajuste manual docente
--   legacy   -> fila histórica que no se puede atribuir a un camino concreto

ALTER TABLE submissions ADD COLUMN IF NOT EXISTS verdict_origin VARCHAR(20);

ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_verdict_origin_check;
ALTER TABLE submissions ADD CONSTRAINT submissions_verdict_origin_check
  CHECK (verdict_origin IS NULL OR verdict_origin IN ('judge', 'rubric', 'override', 'legacy'));

-- Clasificación por evidencia: la rúbrica dejaba una firma textual exacta y el
-- ajuste manual dejaba manual_override. Nada más se atribuye por suposición.

UPDATE submissions sub
SET verdict_origin = 'rubric'
FROM exercises ex
WHERE sub.exercise_id = ex.id
  AND ex.environment_type = 'IDE_PERSISTENTE'
  AND sub.verdict IS NOT NULL
  AND sub.override_reason = 'Evaluado por rúbrica'
  AND sub.verdict_origin IS NULL;

UPDATE submissions sub
SET verdict_origin = 'override'
FROM exercises ex
WHERE sub.exercise_id = ex.id
  AND ex.environment_type = 'IDE_PERSISTENTE'
  AND sub.verdict IS NOT NULL
  AND sub.manual_override IS TRUE
  AND sub.verdict_origin IS NULL;

-- El resto no es atribuible: incluye los veredictos de juez declarados por el
-- navegador, que no son verificables por el servidor.
UPDATE submissions
SET verdict_origin = 'legacy'
WHERE verdict IS NOT NULL
  AND verdict_origin IS NULL;

-- +goose Down
-- Revertir descarta la clasificación de origen, que es el único dato nuevo de
-- esta migración. No toca veredictos, notas ni estados de revisión.
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_verdict_origin_check;
ALTER TABLE submissions DROP COLUMN IF EXISTS verdict_origin;
