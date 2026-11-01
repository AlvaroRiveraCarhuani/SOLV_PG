-- +goose Up
-- Entornos por práctica. Alumnos y docentes tienen un entorno por ejercicio
-- dentro de la misma materia; el volumen se separa por exercise_id y el
-- entorno por materia conserva exercise_id NULL.
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS exercise_id UUID
    REFERENCES exercises(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_workspaces_exercise
    ON workspaces (exercise_id)
    WHERE exercise_id IS NOT NULL;

-- Una persona no puede tener dos entornos de la misma práctica. El entorno por
-- materia (exercise_id NULL) no participa del índice.
DROP INDEX IF EXISTS idx_workspaces_teacher_practice;
CREATE UNIQUE INDEX IF NOT EXISTS idx_workspaces_practice_scope
    ON workspaces (tenant_id, student_id, subject_id, exercise_id)
    WHERE exercise_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_workspaces_practice_scope;
DROP INDEX IF EXISTS idx_workspaces_teacher_practice;
DROP INDEX IF EXISTS idx_workspaces_exercise;
ALTER TABLE workspaces DROP COLUMN IF EXISTS exercise_id;
