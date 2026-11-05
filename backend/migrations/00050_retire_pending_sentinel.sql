-- +goose Up
-- El centinela 'PENDING' nació para llenar una columna que no admitía
-- ausencia. No significaba "evaluación en curso" sino "no hay evaluación del
-- juez", y eso ahora se representa con ausencia.
--
-- Se convierte solo el centinela en entregas de IDE. No se toca ninguna nota,
-- ningún estado de revisión ni ningún veredicto que provenga de una corrección
-- humana: esos registros conservan su información tal como quedó.
--
-- Los veredictos históricos de IDE que sí tienen contenido (por ejemplo, el
-- 'AC' que escribía la rúbrica) quedan intactos: no son resultados del juez y
-- no se convierten en evaluación verificada. Se reconocen por
-- override_reason = 'Evaluado por rúbrica' y quedan documentados como legado.

UPDATE submissions sub
SET verdict = NULL
FROM exercises ex
WHERE sub.exercise_id = ex.id
  AND ex.environment_type = 'IDE_PERSISTENTE'
  AND sub.verdict = 'PENDING';

-- +goose Down
-- Revertir devolvería el centinela a entregas que hoy declaran su ausencia de
-- veredicto, y no distingue las filas históricas de las nuevas. Es
-- irreversible por diseño: si hay que rehacer el estado, se restaura la base
-- y se vuelve a aplicar el esquema.
