-- +goose Up
-- Pin del entorno al publicar (ADR-050, D4). La columna guarda la decisión
-- efectiva de la plantilla en el momento de la publicación: imagen, entrypoint,
-- timeout, RAM, bindings resueltos y contrato de init. Editar la plantilla no
-- reescribe este JSON; solo afecta a las publicaciones nuevas.
--
-- NULL identifica un ejercicio publicado antes del corte o todavía en borrador.
-- El arranque resuelve la plantilla viva y registra el fallback.
ALTER TABLE exercises ADD COLUMN IF NOT EXISTS environment_snapshot JSONB;

-- +goose Down
ALTER TABLE exercises DROP COLUMN IF EXISTS environment_snapshot;