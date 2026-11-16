-- +goose Up
-- Última medición del volumen contra la cuota del nodo. La escribe el barrido de
-- disco cada pocos minutos; la pantalla solo lee el valor guardado, sin tocar el
-- daemon de Docker en cada refresco.
ALTER TABLE environments ADD COLUMN IF NOT EXISTS disk_used_mb BIGINT NOT NULL DEFAULT 0;
ALTER TABLE environments ADD COLUMN IF NOT EXISTS disk_state TEXT NOT NULL DEFAULT 'ok';

-- +goose Down
ALTER TABLE environments DROP COLUMN IF EXISTS disk_state;
ALTER TABLE environments DROP COLUMN IF EXISTS disk_used_mb;
