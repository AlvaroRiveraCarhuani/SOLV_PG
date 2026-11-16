-- +goose Up
-- Perfil del entorno por momento académico. El perfil se deriva del propósito
-- del ejercicio en el dominio; la columna guarda con qué reglas se ejecutó la
-- última apertura y sirve para auditar. El intento de examen arranca en cero
-- (práctica) y forma parte del volumen del intento.
ALTER TABLE environments ADD COLUMN IF NOT EXISTS environment_profile TEXT NOT NULL DEFAULT 'practice';
ALTER TABLE environments ADD COLUMN IF NOT EXISTS exam_attempt INT NOT NULL DEFAULT 0;

-- El arranque de un examen busca los intentos de una práctica concreta sin
-- recorrer la tabla. La unicidad por intento se agrega cuando el alta de
-- intentos exista: hoy hay filas previas sin ese contrato.
CREATE INDEX IF NOT EXISTS idx_environments_exam_attempt
    ON environments (student_id, subject_id, exercise_id, exam_attempt)
    WHERE exercise_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_environments_exam_attempt;
ALTER TABLE environments DROP COLUMN IF EXISTS exam_attempt;
ALTER TABLE environments DROP COLUMN IF EXISTS environment_profile;
