-- +goose Up
CREATE TABLE IF NOT EXISTS course_modules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  title VARCHAR(255) NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  order_index INT NOT NULL,
  pass_score INT NOT NULL DEFAULT 60 CHECK (pass_score BETWEEN 0 AND 100),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE (subject_id, order_index)
);

CREATE TABLE IF NOT EXISTS module_prerequisites (
  module_id UUID NOT NULL REFERENCES course_modules(id) ON DELETE CASCADE,
  prerequisite_module_id UUID NOT NULL REFERENCES course_modules(id) ON DELETE CASCADE,
  PRIMARY KEY (module_id, prerequisite_module_id),
  CHECK (module_id <> prerequisite_module_id)
);

ALTER TABLE exercises
  ADD COLUMN IF NOT EXISTS module_id UUID REFERENCES course_modules(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_exercises_module ON exercises(module_id);

-- +goose Down
DROP INDEX IF EXISTS idx_exercises_module;
ALTER TABLE exercises DROP COLUMN IF EXISTS module_id;
DROP TABLE IF EXISTS module_prerequisites;
DROP TABLE IF EXISTS course_modules;
