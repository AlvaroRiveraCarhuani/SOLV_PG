-- +goose Up
-- El script de inicialización de base de datos deja de vivir dentro de
-- exercises.config (JSONB) y pasa a su propia tabla (ADR-073). El config de un
-- ejercicio se lee en cada apertura del editor y en cada arranque de laboratorio;
-- cargar ahí el texto completo del docente castiga la lectura y mezcla contenido
-- voluminoso con la configuración.
CREATE TABLE IF NOT EXISTS exercise_db_init_scripts (
    exercise_id UUID PRIMARY KEY REFERENCES exercises(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    byte_size   INTEGER NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Backfill desde el config existente. Se acepta el espejo histórico
-- config.database.init_script porque los ejercicios guardados antes escribían el
-- script en los dos lugares.
INSERT INTO exercise_db_init_scripts (exercise_id, content, byte_size)
SELECT id,
       btrim(COALESCE(config -> 'ide' ->> 'db_init_script', config -> 'database' ->> 'init_script', '')),
       length(btrim(COALESCE(config -> 'ide' ->> 'db_init_script', config -> 'database' ->> 'init_script', '')))
FROM exercises
WHERE btrim(COALESCE(config -> 'ide' ->> 'db_init_script', config -> 'database' ->> 'init_script', '')) <> ''
ON CONFLICT (exercise_id) DO NOTHING;

-- Retirar el texto del JSONB. Se hace en dos pasos para no crear objetos vacíos
-- en los ejercicios que nunca declararon ese bloque.
UPDATE exercises
SET config = jsonb_set(config, '{ide}', (config -> 'ide') - 'db_init_script')
WHERE config -> 'ide' ? 'db_init_script';

UPDATE exercises
SET config = jsonb_set(config, '{database}', (config -> 'database') - 'init_script')
WHERE config -> 'database' ? 'init_script';

-- +goose Down
-- Restaurar el script en el config antes de eliminar la tabla, para que la
-- reversión no pierda el contenido del docente.
UPDATE exercises e
SET config = jsonb_set(
        e.config,
        '{ide}',
        COALESCE(e.config -> 'ide', '{}'::jsonb) || jsonb_build_object('db_init_script', s.content),
        true
    )
FROM exercise_db_init_scripts s
WHERE s.exercise_id = e.id;

DROP TABLE IF EXISTS exercise_db_init_scripts;
