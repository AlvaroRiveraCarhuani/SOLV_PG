-- +goose Up
ALTER TABLE class_advances ADD COLUMN IF NOT EXISTS title VARCHAR(255) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE class_advances DROP COLUMN IF EXISTS title;
