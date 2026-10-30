-- +goose Up
-- Estado de revisión explícito y celda de nota por alumno y práctica.
--
-- Antes de esta migración "por calificar" se deducía del veredicto (WA/RE/TLE)
-- y la libreta calculaba max(score) en cada lectura, con 0 cuando la entrega no
-- tenía nota. Así, una entrega IDE sin corregir valía 0 y el trabajo manual del
-- docente quedaba fuera de la cola.

-- 1. Estado de revisión en la entrega.
ALTER TABLE submissions
  ADD COLUMN IF NOT EXISTS review_status VARCHAR(20) NOT NULL DEFAULT 'not_required',
  ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS feedback_summary TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS returned_at TIMESTAMPTZ;

-- Recrear el CHECK es idempotente y evita bloques DO, que el migrador parte
-- por el punto y coma interno.
ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_review_status_check;
ALTER TABLE submissions ADD CONSTRAINT submissions_review_status_check
  CHECK (review_status IN ('not_required', 'pending', 'in_review', 'graded', 'returned'));

CREATE INDEX IF NOT EXISTS idx_submissions_review_status
  ON submissions(tenant_id, review_status);
CREATE INDEX IF NOT EXISTS idx_submissions_exercise_review
  ON submissions(exercise_id, review_status);

-- 2. Backfill: solo los laboratorios IDE exigen corrección manual.
UPDATE submissions sub
SET review_status = 'pending'
FROM exercises ex
WHERE sub.exercise_id = ex.id
  AND ex.environment_type = 'IDE_PERSISTENTE'
  AND sub.score IS NULL
  AND sub.review_status = 'not_required';

-- 3. Celda de nota por alumno y práctica.
CREATE TABLE IF NOT EXISTS exercise_grades (
  student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
  subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
  selected_submission_id UUID REFERENCES submissions(id) ON DELETE SET NULL,
  score INT,
  source VARCHAR(10) NOT NULL DEFAULT 'auto',
  overridden_by UUID REFERENCES users(id) ON DELETE SET NULL,
  reason TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (student_id, exercise_id)
);

ALTER TABLE exercise_grades DROP CONSTRAINT IF EXISTS exercise_grades_source_check;
ALTER TABLE exercise_grades ADD CONSTRAINT exercise_grades_source_check
  CHECK (source IN ('auto', 'manual'));

ALTER TABLE exercise_grades DROP CONSTRAINT IF EXISTS exercise_grades_score_check;
ALTER TABLE exercise_grades ADD CONSTRAINT exercise_grades_score_check
  CHECK (score IS NULL OR (score >= 0 AND score <= 100));

CREATE INDEX IF NOT EXISTS idx_exercise_grades_subject
  ON exercise_grades(subject_id, tenant_id);

-- 4. Backfill de celdas con la mejor entrega ya calificada de cada pareja.
--    Si ninguna entrega tiene nota, no se crea celda: la libreta la muestra
--    como pendiente en lugar de inventar un 0.
INSERT INTO exercise_grades (
  student_id, exercise_id, subject_id, tenant_id, selected_submission_id, score, source
)
SELECT DISTINCT ON (sub.student_id, sub.exercise_id)
  sub.student_id,
  sub.exercise_id,
  ex.subject_id,
  sub.tenant_id,
  sub.id,
  best.score,
  'auto'
FROM submissions sub
JOIN exercises ex ON ex.id = sub.exercise_id
CROSS JOIN LATERAL (
  SELECT COALESCE(
    sub.score,
    CASE WHEN sub.verdict = 'AC' OR COALESCE(sub.manual_override, FALSE) THEN 100 ELSE NULL END
  ) AS score
) best
WHERE ex.subject_id IS NOT NULL
  AND best.score IS NOT NULL
ORDER BY sub.student_id, sub.exercise_id, best.score DESC, sub.submitted_at ASC
ON CONFLICT (student_id, exercise_id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS exercise_grades;

DROP INDEX IF EXISTS idx_submissions_exercise_review;
DROP INDEX IF EXISTS idx_submissions_review_status;

ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_review_status_check;
ALTER TABLE submissions
  DROP COLUMN IF EXISTS returned_at,
  DROP COLUMN IF EXISTS feedback_summary,
  DROP COLUMN IF EXISTS reviewed_by,
  DROP COLUMN IF EXISTS reviewed_at,
  DROP COLUMN IF EXISTS review_status;
