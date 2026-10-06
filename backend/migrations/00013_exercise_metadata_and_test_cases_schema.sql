-- +goose Up
-- 1. Metadata pedagógica en exercises
ALTER TABLE exercises
  ADD COLUMN IF NOT EXISTS difficulty VARCHAR(10)
    CHECK (difficulty IN ('easy','medium','hard')),
  ADD COLUMN IF NOT EXISTS tags TEXT[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS purpose VARCHAR(10) NOT NULL DEFAULT 'class'
    CHECK (purpose IN ('class','exam')),
  ADD COLUMN IF NOT EXISTS per_student_seed BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_exercises_tags
  ON exercises USING GIN (tags);

CREATE INDEX IF NOT EXISTS idx_exercises_difficulty
  ON exercises (difficulty) WHERE difficulty IS NOT NULL;

-- 2. Tabla relacional de casos de prueba
CREATE TABLE IF NOT EXISTS exercise_test_cases (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
  order_index INT NOT NULL,
  input TEXT NOT NULL DEFAULT '',
  expected_output TEXT NOT NULL CHECK (expected_output <> ''),
  visibility VARCHAR(10) NOT NULL DEFAULT 'public'
    CHECK (visibility IN ('example','public','hidden')),
  weight DOUBLE PRECISION NOT NULL DEFAULT 1.0 CHECK (weight >= 0),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (exercise_id, order_index)
);

CREATE INDEX IF NOT EXISTS idx_test_cases_visibility
  ON exercise_test_cases (exercise_id, visibility);

-- +goose Down
DROP TABLE IF EXISTS exercise_test_cases;
DROP INDEX IF EXISTS idx_exercises_tags;
DROP INDEX IF EXISTS idx_exercises_difficulty;
ALTER TABLE exercises
  DROP COLUMN IF EXISTS difficulty,
  DROP COLUMN IF EXISTS tags,
  DROP COLUMN IF EXISTS purpose,
  DROP COLUMN IF EXISTS per_student_seed;
