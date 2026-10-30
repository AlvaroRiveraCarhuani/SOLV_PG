-- +goose Up
-- Corrección de datos de la migración 00034.
--
-- Las entregas de un laboratorio IDE se creaban con verdict 'PENDING' y
-- score 0. El backfill de 00034 interpretó ese 0 como una nota real y las
-- marcó 'not_required'; la libreta, además, creó una celda con 0 para cada
-- una. El resultado: la cola quedaba vacía y la libreta volvía a mostrar
-- ceros por trabajo que nadie corrigió.

-- 1. Devolver al estado de revisión real y borrar la nota fabricada.
UPDATE submissions sub
SET review_status = 'pending',
    score = NULL
FROM exercises ex
WHERE sub.exercise_id = ex.id
  AND ex.environment_type = 'IDE_PERSISTENTE'
  AND sub.verdict = 'PENDING'
  AND sub.review_status = 'not_required'
  AND sub.score = 0
  AND sub.manual_override IS NOT TRUE
  AND sub.graded_by IS NULL;

-- 2. Retirar las celdas de libreta que nacieron de ese 0. Solo las sugeridas
--    por el sistema: una nota manual del docente se respeta.
DELETE FROM exercise_grades eg
USING submissions sub
WHERE eg.selected_submission_id = sub.id
  AND eg.source = 'auto'
  AND eg.score = 0
  AND eg.overridden_by IS NULL
  AND sub.verdict = 'PENDING'
  AND sub.score IS NULL;

-- +goose Down
-- La corrección es irreversible por diseño: revertirla volvería a contar como
-- calificadas entregas que nunca se corrigieron y reintroduciría el 0 en la
-- libreta. Si hay que rehacer el estado, se corrige el código y se vuelve a
-- ejecutar este archivo sobre una base restaurada.
