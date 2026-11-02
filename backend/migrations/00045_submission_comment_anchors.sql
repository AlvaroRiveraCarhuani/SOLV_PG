-- +goose Up
-- Anclaje por archivo para comentarios IDE. line_number sigue siendo la línea
-- global sobre `code` y se conserva para retrocompatibilidad; file_path y
-- local_line_number permiten que SOLV Companion abra el comentario por archivo
-- sin recalcular el snapshot. En modo juez ambos quedan NULL.
-- No se migran datos: los comentarios existentes se resuelven al leer.

ALTER TABLE submission_comments
    ADD COLUMN IF NOT EXISTS file_path TEXT;

ALTER TABLE submission_comments
    ADD COLUMN IF NOT EXISTS local_line_number INT;

ALTER TABLE submission_comments
    DROP CONSTRAINT IF EXISTS chk_submission_comments_anchor;

ALTER TABLE submission_comments
    ADD CONSTRAINT chk_submission_comments_anchor
    CHECK (
        (file_path IS NULL AND local_line_number IS NULL)
        OR (
            file_path IS NOT NULL
            AND btrim(file_path) <> ''
            AND local_line_number > 0
        )
    );

CREATE INDEX IF NOT EXISTS idx_submission_comments_file
    ON submission_comments(submission_id, file_path)
    WHERE file_path IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_submission_comments_file;

ALTER TABLE submission_comments
    DROP CONSTRAINT IF EXISTS chk_submission_comments_anchor;

ALTER TABLE submission_comments
    DROP COLUMN IF EXISTS local_line_number;

ALTER TABLE submission_comments
    DROP COLUMN IF EXISTS file_path;
