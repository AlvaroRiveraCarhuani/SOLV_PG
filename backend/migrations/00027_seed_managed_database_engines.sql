-- +goose Up
-- Gobernanza de motores de base de datos en el catálogo institucional.
--
-- Hasta ahora MySQL, MongoDB y Redis dependían de que un administrador de sistema
-- declarara a mano el host de cada satélite (MYSQL_SATELLITE_HOST, ...) antes de que
-- la Columna 3 del Constructor de Entornos pudiera ofrecerlos. Con la
-- auto-orquestación los motores se levantan bajo demanda como contenedores satélite
-- en la red interna, así que el catálogo debe declarar para cada motor:
--   * service_engine canónico (mysql, mariadb, mongodb, redis, postgres, sqlite),
--   * base_image con tag de versión fija (prohibido :latest),
--   * ram_delta_mb de la huella residente del motor,
--   * sub_category relacional o NoSQL,
--   * y dejarlos activos (is_active = true) para que el constructor los ofrezca.
--
-- Upsert idempotente sobre el catálogo del tenant institucional: crea las semillas que
-- falten (MariaDB y MongoDB) y reescribe siempre los metadatos de las existentes.

INSERT INTO template_components (
    id, tenant_id, slug, layer, sub_category, name, description, icon_key,
    base_image, compatible_runtimes, install_mode, package_names,
    build_layer_snippet, setup_snippet, judge_entrypoint, tools_provided,
    service_engine, ram_delta_mb, is_system, is_active, sort_order
) VALUES
    ('a0000000-0000-0000-0000-000000000031', '00000000-0000-0000-0000-000000000001', 'sqlite-3', 'database', 'relational', 'SQLite 3 (En archivo)', 'Base de datos ligera guardada directamente en la carpeta de trabajo sin consumo extra de memoria', 'sqlite', '', '["universal"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '[]'::jsonb, 'sqlite', 0, true, true, 310),
    ('a0000000-0000-0000-0000-000000000032', '00000000-0000-0000-0000-000000000001', 'postgres-16', 'database', 'relational', 'PostgreSQL 16', 'Base de datos relacional compartida del aula, con base y usuario aislados por alumno y materia', 'postgres', 'postgres:18-alpine', '["universal"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '[]'::jsonb, 'postgres', 50, true, true, 320),
    ('a0000000-0000-0000-0000-000000000033', '00000000-0000-0000-0000-000000000001', 'mysql-84', 'database', 'relational', 'MySQL 8.4', 'Motor relacional clásico auto-orquestado: se levanta al iniciar el laboratorio y se apaga cuando nadie lo usa', 'mysql', 'mysql:8.4', '["universal"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '[]'::jsonb, 'mysql', 128, true, true, 330),
    ('a0000000-0000-0000-0000-000000000035', '00000000-0000-0000-0000-000000000001', 'mariadb-11', 'database', 'relational', 'MariaDB 11.4', 'Motor relacional compatible con MySQL, auto-orquestado con sintonización liviana y aislamiento por alumno', 'mysql', 'mariadb:11.4', '["universal"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '[]'::jsonb, 'mariadb', 128, true, true, 335),
    ('a0000000-0000-0000-0000-000000000036', '00000000-0000-0000-0000-000000000001', 'mongodb-70', 'database', 'nosql', 'MongoDB 7.0 (Documentos)', 'Base de datos de documentos auto-orquestada, con base aislada por alumno y caché interna acotada', 'mongodb', 'mongo:7.0', '["universal"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '[]'::jsonb, 'mongodb', 128, true, true, 336),
    ('a0000000-0000-0000-0000-000000000034', '00000000-0000-0000-0000-000000000001', 'redis-72', 'database', 'nosql', 'Redis 7.2 (Caché)', 'Almacén ultrarrápido clave-valor auto-orquestado, con espacio de claves aislado por alumno', 'redis', 'redis:7.2-alpine', '["universal"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '[]'::jsonb, 'redis', 64, true, true, 340)
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    layer = EXCLUDED.layer,
    sub_category = EXCLUDED.sub_category,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    icon_key = EXCLUDED.icon_key,
    base_image = EXCLUDED.base_image,
    compatible_runtimes = EXCLUDED.compatible_runtimes,
    install_mode = EXCLUDED.install_mode,
    package_names = EXCLUDED.package_names,
    service_engine = EXCLUDED.service_engine,
    ram_delta_mb = EXCLUDED.ram_delta_mb,
    is_active = EXCLUDED.is_active,
    sort_order = EXCLUDED.sort_order,
    updated_at = NOW();

-- +goose Down
-- Restauración idempotente del estado previo: las semillas originales vuelven a
-- declarar solo su service_engine (sin imagen propia) y desaparecen los motores que
-- esta migración dio de alta.
DELETE FROM template_components
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug IN ('mariadb-11', 'mongodb-70');

UPDATE template_components
SET base_image = '',
    service_engine = 'sqlite',
    ram_delta_mb = 0,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'sqlite-3';

UPDATE template_components
SET base_image = '',
    service_engine = 'postgres',
    ram_delta_mb = 50,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'postgres-16';

UPDATE template_components
SET base_image = '',
    service_engine = 'mysql',
    ram_delta_mb = 128,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'mysql-84';

UPDATE template_components
SET base_image = '',
    service_engine = 'redis',
    ram_delta_mb = 64,
    sub_category = 'nosql',
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'redis-72';
