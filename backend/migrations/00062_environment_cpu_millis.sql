-- +goose Up
-- Cuota de CPU aplicada al contenedor y perfil de memoria que declaró la
-- plantilla. Se guardan para que la ficha muestre el límite real con el que se
-- abrió el entorno, no el que la política tenga hoy.
ALTER TABLE environments ADD COLUMN IF NOT EXISTS cpu_millis BIGINT NOT NULL DEFAULT 0;
ALTER TABLE environments ADD COLUMN IF NOT EXISTS memory_profile_mb BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE environments DROP COLUMN IF EXISTS memory_profile_mb;
ALTER TABLE environments DROP COLUMN IF EXISTS cpu_millis;
