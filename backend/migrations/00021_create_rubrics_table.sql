-- +goose Up
-- SQL migration for creating rubrics table (IDE Persistente evaluation)

CREATE TABLE IF NOT EXISTS rubrics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
  criteria JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (exercise_id)
);

CREATE INDEX IF NOT EXISTS idx_rubrics_exercise ON rubrics(exercise_id);

-- +goose Down
DROP TABLE IF EXISTS rubrics;
