-- +goose Up
-- Migration 00025: Snapshot autoritativo de los casos generados por semilla
--
-- Los casos generados de los exámenes con `per_student_seed` se calculan en el
-- servidor durante la evaluación. El rol student nunca debe recibir la salida
-- esperada de esos casos, por lo que el snapshot se conserva del lado del
-- servidor y es la única fuente de verdad para la entrega del estudiante.
CREATE TABLE IF NOT EXISTS evaluation_generated_cases (
  exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
  student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
  seed BIGINT NOT NULL DEFAULT 0,
  cases JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (exercise_id, student_id)
);

CREATE INDEX IF NOT EXISTS idx_evaluation_generated_cases_student
  ON evaluation_generated_cases (student_id);

-- +goose Down
DROP TABLE IF EXISTS evaluation_generated_cases;
