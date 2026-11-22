-- +goose Up
-- Entrega de un proyecto grande como artefacto (ADR-090).
--
-- El árbol del alumno deja de viajar como texto en `submissions.code`: se
-- empaqueta en tar.gz y se guarda en el depósito de objetos. La fila conserva la
-- referencia, la huella y el tamaño, y la revisión lo lee de ahí.
--
-- Las cuatro columnas son nulas en las filas existentes: el camino liviano sigue
-- guardando texto y una entrega nunca queda mezclada entre los dos modos.
ALTER TABLE submissions
    ADD COLUMN IF NOT EXISTS artifact_key    TEXT,
    ADD COLUMN IF NOT EXISTS artifact_sha256 CHAR(64),
    ADD COLUMN IF NOT EXISTS artifact_bytes  BIGINT,
    ADD COLUMN IF NOT EXISTS artifact_files  INTEGER;

ALTER TABLE submissions
    ADD CONSTRAINT submissions_artifact_all_or_none CHECK (
        (artifact_key IS NULL AND artifact_sha256 IS NULL AND artifact_bytes IS NULL AND artifact_files IS NULL)
        OR (artifact_key IS NOT NULL AND artifact_sha256 IS NOT NULL AND artifact_bytes IS NOT NULL AND artifact_files IS NOT NULL)
    );

-- Con artefacto el texto queda vacío a propósito: si hubiera ambos, nadie sabría
-- cuál es el árbol entregado.
ALTER TABLE submissions
    ADD CONSTRAINT submissions_artifact_text_exclusive CHECK (
        artifact_key IS NULL OR code = ''
    );

-- +goose Down
-- Revertir descarta las referencias del depósito: el árbol de esas entregas no
-- se puede representar en `code` sin volver a leer los objetos. Los objetos no se
-- tocan, quedan para la retención.
DELETE FROM submissions WHERE artifact_key IS NOT NULL;

ALTER TABLE submissions
    DROP CONSTRAINT IF EXISTS submissions_artifact_text_exclusive,
    DROP CONSTRAINT IF EXISTS submissions_artifact_all_or_none,
    DROP COLUMN IF EXISTS artifact_key,
    DROP COLUMN IF EXISTS artifact_sha256,
    DROP COLUMN IF EXISTS artifact_bytes,
    DROP COLUMN IF EXISTS artifact_files;
