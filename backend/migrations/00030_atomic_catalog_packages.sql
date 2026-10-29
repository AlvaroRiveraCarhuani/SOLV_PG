-- +goose Up
-- Desacoplamiento de paquetes atomicos del catalogo de plantillas:
--   * Renombrar SQLAlchemy eliminando fuga de abstraccion ("+ Drivers SQL") y asignando su icon_key oficial "sqlalchemy".
--   * Separar conectores NoSQL ("python-nosql-clients") en PyMongo (MongoDB) y Redis-py (Redis) con sus iconos oficiales.
--   * Registrar primitivas atomicas para Ciencia de Datos: NumPy ("python-numpy") y Pandas ("python-pandas").
--   * Registrar primitiva atomica limpia para Testing: Pytest ("python-pytest").

-- 1. SQLAlchemy: actualizar nombre limpio e icon_key oficial
UPDATE template_components
SET name = 'SQLAlchemy',
    icon_key = 'sqlalchemy',
    description = 'ORM SQLAlchemy con Alembic y conectores para PostgreSQL, MySQL y SQLite',
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'python-sqlalchemy';

-- 2. Conectores NoSQL: actualizar el registro existente para PyMongo exclusivo
UPDATE template_components
SET name = 'PyMongo (Cliente MongoDB)',
    description = 'Conector oficial de MongoDB para Python con soporte de documentos y BSON',
    icon_key = 'mongodb',
    package_names = '["pymongo"]'::jsonb,
    build_layer_snippet = 'RUN pip install --no-cache-dir pymongo',
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'python-nosql-clients';

-- 3. Alta de paquetes atomicos independientes (NumPy, Pandas, Pytest, Redis-py)
INSERT INTO template_components (
    id, tenant_id, slug, layer, sub_category, name, description, icon_key,
    base_image, compatible_runtimes, install_mode, package_names,
    build_layer_snippet, setup_snippet, judge_entrypoint, tools_provided,
    service_engine, ram_delta_mb, is_system, is_active, sort_order
) VALUES
    -- A. Redis para Python (desacoplado de Mongo)
    ('a0000000-0000-0000-0000-000000000070', '00000000-0000-0000-0000-000000000001', 'python-redis', 'addon', 'orm', 'Redis-py (Cliente Redis)', 'Cliente oficial de Redis para Python con soporte de cache, pub/sub y estructuras en memoria', 'redis', '', '["python"]'::jsonb, 'pip', '["redis"]'::jsonb, 'RUN pip install --no-cache-dir redis', '', '', '["python3"]'::jsonb, '', 0, true, true, 245),

    -- B. NumPy atomico
    ('a0000000-0000-0000-0000-000000000071', '00000000-0000-0000-0000-000000000001', 'python-numpy', 'addon', 'toolkit', 'NumPy', 'Calculo numerico multidimensional, algebra lineal y procesamiento matricial de alto rendimiento', 'numpy', '', '["python"]'::jsonb, 'pip', '["numpy"]'::jsonb, 'RUN pip install --no-cache-dir numpy', '', '', '["python3"]'::jsonb, '', 0, true, true, 248),

    -- C. Pandas atomico
    ('a0000000-0000-0000-0000-000000000072', '00000000-0000-0000-0000-000000000001', 'python-pandas', 'addon', 'heavy', 'Pandas', 'Estructuras de datos DataFrame y herramientas de analisis y manipulacion tabular', 'pandas', '', '["python"]'::jsonb, 'pip', '["pandas"]'::jsonb, 'RUN pip install --no-cache-dir pandas', '', '', '["python3"]'::jsonb, '', 128, true, true, 249),

    -- D. Pytest atomico
    ('a0000000-0000-0000-0000-000000000073', '00000000-0000-0000-0000-000000000001', 'python-pytest', 'addon', 'toolkit', 'Pytest', 'Framework estandar para ejecucion de pruebas unitarias y automatizadas en Python', 'pytest', '', '["python"]'::jsonb, 'pip', '["pytest"]'::jsonb, 'RUN pip install --no-cache-dir pytest', '', '', '["pytest"]'::jsonb, '', 0, true, true, 253)

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
    build_layer_snippet = EXCLUDED.build_layer_snippet,
    tools_provided = EXCLUDED.tools_provided,
    service_engine = EXCLUDED.service_engine,
    ram_delta_mb = EXCLUDED.ram_delta_mb,
    is_active = EXCLUDED.is_active,
    sort_order = EXCLUDED.sort_order,
    updated_at = NOW();

-- +goose Down
-- Reversion de los cambios del catalogo atomico

DELETE FROM template_components
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug IN ('python-redis', 'python-numpy', 'python-pandas', 'python-pytest');

UPDATE template_components
SET name = 'SQLAlchemy + Drivers SQL',
    icon_key = 'python',
    description = 'ORM SQLAlchemy con Alembic y conectores para PostgreSQL, MySQL y SQLite',
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'python-sqlalchemy';

UPDATE template_components
SET name = 'Cliente MongoDB & Redis',
    description = 'Conectores oficiales PyMongo y Redis-py para bases de datos NoSQL y caché',
    icon_key = 'python',
    package_names = '["pymongo", "redis"]'::jsonb,
    build_layer_snippet = 'RUN pip install --no-cache-dir pymongo redis',
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'python-nosql-clients';
