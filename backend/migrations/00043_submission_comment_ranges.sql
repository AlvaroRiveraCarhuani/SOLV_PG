-- +goose Up
-- Comentarios multilínea por rango (fase 3 de revisión por modalidad).
-- line_number sigue siendo la línea global inicial sobre `code`; end_line_number
-- es opcional y, cuando existe, cierra el bloque comentado. No se migran datos:
-- los comentarios existentes quedan como comentarios de una sola línea.

ALTER TABLE submission_comments
    ADD COLUMN IF NOT EXISTS end_line_number INT;

ALTER TABLE submission_comments
    DROP CONSTRAINT IF EXISTS chk_submission_comments_range;

ALTER TABLE submission_comments
    ADD CONSTRAINT chk_submission_comments_range
    CHECK (end_line_number IS NULL OR end_line_number >= line_number);

-- +goose Down
ALTER TABLE submission_comments
    DROP CONSTRAINT IF EXISTS chk_submission_comments_range;

ALTER TABLE submission_comments
    DROP COLUMN IF EXISTS end_line_number;
