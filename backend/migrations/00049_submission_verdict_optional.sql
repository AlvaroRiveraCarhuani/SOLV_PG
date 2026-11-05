-- +goose Up
-- El veredicto es una salida del juez, no un campo obligatorio de toda entrega.
--
-- Una entrega de IDE_PERSISTENTE no se evalúa con el juez: se corrige con
-- rúbrica. Hasta hoy la columna exigía un valor y la entrega se guardaba con el
-- centinela 'PENDING', que no significa "evaluación en curso" sino "no hay
-- evaluación". Con la columna opcional, la ausencia de veredicto se representa
-- como ausencia, y el estado del trabajo sigue viviendo en review_status.
--
-- Orden de despliegue: primero el código que lee las consultas con
-- COALESCE(verdict, ''), después esta migración. Las escrituras todavía envían
-- 'PENDING'; se retiran en una etapa posterior, cuando la columna ya admite
-- nulos. Ninguna fila existente se modifica acá.

ALTER TABLE submissions ALTER COLUMN verdict DROP NOT NULL;

-- +goose Down
-- Revertir reintroduce el centinela, porque la columna no admite ausencia. Es
-- una acción de despliegue para volver a una versión anterior del código, no
-- una política de datos: no debe ejecutarse para "limpiar" entregas de IDE.
UPDATE submissions SET verdict = 'PENDING' WHERE verdict IS NULL;

ALTER TABLE submissions ALTER COLUMN verdict SET NOT NULL;
