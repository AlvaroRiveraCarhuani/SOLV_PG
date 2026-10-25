-- +goose Up
-- Saneamiento del catálogo tecnológico institucional y alineación de los metadatos
-- de memoria residente para los Kits Universitarios, más el alta del Catálogo
-- Universitario completo (lenguajes, plataformas web y complementos de los 8
-- ecosistemas) mediante upsert idempotente.
--
-- Reglas de negocio:
--   * fastapi / django / flask  -> sub_category 'framework' con ram_delta_mb 0 (ocupan disco, no RAM residente).
--   * streamlit                 -> sub_category 'devserver' con ram_delta_mb 128 (proceso web residente).
--   * python-datascience        -> sub_category 'heavy' con ram_delta_mb 256 (librerías pesadas residentes).
--   * spring-maven              -> sub_category 'toolkit' con ram_delta_mb 0 (herramienta de construcción).
--   * tools_provided declara los binarios ejecutables reales de cada complemento.
-- Cada sentencia es idempotente: reescribe siempre el valor objetivo sobre el catálogo
-- del tenant institucional, sin depender del estado previo.

-- 1. Apache Maven (Java / Spring): toolkit de construcción, sin memoria residente
UPDATE template_components
SET name = 'Apache Maven (Java / Spring)',
    icon_key = 'spring',
    sub_category = 'toolkit',
    ram_delta_mb = 0,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'spring-maven';

-- 2. FastAPI + Uvicorn: framework, binarios reales uvicorn/fastapi, sin memoria residente
UPDATE template_components
SET sub_category = 'framework',
    ram_delta_mb = 0,
    tools_provided = '["uvicorn", "fastapi"]'::jsonb,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'fastapi';

-- 3. Django: framework, binario real django-admin, sin memoria residente
UPDATE template_components
SET sub_category = 'framework',
    ram_delta_mb = 0,
    tools_provided = '["django-admin"]'::jsonb,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'django';

-- 4. Flask: framework, binario real flask, sin memoria residente
UPDATE template_components
SET sub_category = 'framework',
    ram_delta_mb = 0,
    tools_provided = '["flask"]'::jsonb,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'flask';

-- 5. Streamlit: servidor de desarrollo residente (devserver) con 128 MB
UPDATE template_components
SET sub_category = 'devserver',
    ram_delta_mb = 128,
    tools_provided = '["streamlit"]'::jsonb,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'streamlit';

-- 6. Ciencia de Datos: complemento pesado residente (heavy) con 256 MB
UPDATE template_components
SET sub_category = 'heavy',
    ram_delta_mb = 256,
    tools_provided = '["pytest", "python3"]'::jsonb,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'python-datascience';

-- 7. Catálogo Universitario completo: lenguajes y plataformas web faltantes más los
--    complementos (Frameworks, ORMs y Kits de herramientas) de los 8 ecosistemas.
--    Upsert idempotente: crea los registros que falten y re-sanea los existentes.
INSERT INTO template_components (
    id, tenant_id, slug, layer, sub_category, name, description, icon_key,
    base_image, compatible_runtimes, install_mode, package_names,
    build_layer_snippet, setup_snippet, judge_entrypoint, tools_provided,
    service_engine, ram_delta_mb, is_system, is_active, sort_order
) VALUES
    -- A. Lenguajes principales faltantes
    ('a0000000-0000-0000-0000-000000000008', '00000000-0000-0000-0000-000000000001', 'php-83', 'runtime', 'backend', 'PHP 8.3 CLI', 'Intérprete oficial PHP con Composer para desarrollo backend y aplicaciones web', 'php', 'php:8.3-cli-bookworm', '["php"]'::jsonb, 'none', '[]'::jsonb, '', '', 'php solution.php', '["php"]'::jsonb, '', 512, true, true, 70),
    ('a0000000-0000-0000-0000-000000000009', '00000000-0000-0000-0000-000000000001', 'dotnet-8', 'runtime', 'backend', '.NET 8 SDK (C#)', 'SDK oficial de Microsoft .NET 8 para desarrollo en C# y aplicaciones empresariales', 'dotnet', 'mcr.microsoft.com/dotnet/sdk:8.0-bookworm-slim', '["dotnet"]'::jsonb, 'none', '[]'::jsonb, '', '', 'dotnet run', '["dotnet"]'::jsonb, '', 1024, true, true, 80),

    -- B. Plataformas web faltantes
    ('a0000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000001', 'nextjs-web', 'runtime', 'web', 'Next.js (React Full-Stack)', 'Framework React de producción con enrutamiento y renderizado híbrido', 'react', 'node:20-bookworm-slim', '["node", "web"]'::jsonb, 'npm', '["create-next-app", "typescript"]'::jsonb, 'RUN npm install -g --no-audit --no-fund create-next-app typescript', '', '', '["node", "npm", "npx"]'::jsonb, '', 256, true, true, 140),
    ('a0000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000001', 'vite-web', 'runtime', 'web', 'Vite Estándar (HTML / CSS / TS)', 'Empaquetador web moderno ultrarrápido para proyectos TypeScript, JavaScript y CSS', 'vite', 'node:20-bookworm-slim', '["node", "web"]'::jsonb, 'npm', '["vite", "typescript"]'::jsonb, 'RUN npm install -g --no-audit --no-fund vite typescript', '', '', '["node", "npm", "npx"]'::jsonb, '', 128, true, true, 150),

    -- C. Python
    ('a0000000-0000-0000-0000-000000000040', '00000000-0000-0000-0000-000000000001', 'python-sqlalchemy', 'addon', 'orm', 'SQLAlchemy + Drivers SQL', 'ORM SQLAlchemy con Alembic y conectores para PostgreSQL, MySQL y SQLite', 'python', '', '["python"]'::jsonb, 'pip', '["sqlalchemy", "alembic", "psycopg2-binary", "pymysql"]'::jsonb, 'RUN pip install --no-cache-dir sqlalchemy alembic psycopg2-binary pymysql', '', '', '["python3", "alembic"]'::jsonb, '', 0, true, true, 242),
    ('a0000000-0000-0000-0000-000000000041', '00000000-0000-0000-0000-000000000001', 'python-nosql-clients', 'addon', 'orm', 'Cliente MongoDB & Redis', 'Conectores oficiales PyMongo y Redis-py para bases de datos NoSQL y caché', 'python', '', '["python"]'::jsonb, 'pip', '["pymongo", "redis"]'::jsonb, 'RUN pip install --no-cache-dir pymongo redis', '', '', '["python3", "pip"]'::jsonb, '', 0, true, true, 244),
    ('a0000000-0000-0000-0000-000000000042', '00000000-0000-0000-0000-000000000001', 'python-ml', 'addon', 'heavy', 'Machine Learning (Scikit-Learn + SciPy)', 'Modelos predictivos, clustering y cálculo científico avanzado en Python', 'python', '', '["python"]'::jsonb, 'pip', '["scikit-learn", "scipy", "matplotlib"]'::jsonb, 'RUN pip install --no-cache-dir scikit-learn scipy matplotlib', '', '', '["python3", "pip"]'::jsonb, '', 256, true, true, 252),
    ('a0000000-0000-0000-0000-000000000043', '00000000-0000-0000-0000-000000000001', 'python-testing-ruff', 'addon', 'toolkit', 'Pruebas y Calidad (Pytest + HTTPX + Ruff)', 'Ejecutor de pruebas Pytest, cliente HTTPX para APIs y linter ultrarrápido Ruff', 'python', '', '["python"]'::jsonb, 'pip', '["pytest", "httpx", "ruff"]'::jsonb, 'RUN pip install --no-cache-dir pytest httpx ruff', '', '', '["pytest", "ruff"]'::jsonb, '', 0, true, true, 254),

    -- C. Java
    ('a0000000-0000-0000-0000-000000000044', '00000000-0000-0000-0000-000000000001', 'java-springboot', 'addon', 'framework', 'Spring Boot (Web + REST)', 'Entorno preparado con Maven y utilidades para servicios Spring Boot', 'spring', '', '["java"]'::jsonb, 'apt', '["maven"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends maven && rm -rf /var/lib/apt/lists/*', '', '', '["java", "mvn"]'::jsonb, '', 0, true, true, 268),
    ('a0000000-0000-0000-0000-000000000045', '00000000-0000-0000-0000-000000000001', 'java-hibernate-jdbc', 'addon', 'orm', 'Hibernate ORM / JPA + JDBC Drivers', 'Soporte de persistencia JPA y drivers JDBC para PostgreSQL, MySQL y SQLite vía Maven', 'java', '', '["java"]'::jsonb, 'apt', '["maven", "libsqlite3-dev"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends maven libsqlite3-dev && rm -rf /var/lib/apt/lists/*', '', '', '["java", "mvn"]'::jsonb, '', 0, true, true, 271),
    ('a0000000-0000-0000-0000-000000000046', '00000000-0000-0000-0000-000000000001', 'java-gradle', 'addon', 'toolkit', 'Gradle Build Tool', 'Sistema de automatización de compilación y dependencias Gradle para Java', 'java', '', '["java"]'::jsonb, 'apt', '["gradle"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends gradle && rm -rf /var/lib/apt/lists/*', '', '', '["java", "gradle"]'::jsonb, '', 0, true, true, 272),
    ('a0000000-0000-0000-0000-000000000047', '00000000-0000-0000-0000-000000000001', 'java-junit5', 'addon', 'toolkit', 'JUnit 5 + Consola de Pruebas', 'Soporte de ejecución de pruebas unitarias JUnit 5 mediante Maven/Gradle', 'java', '', '["java"]'::jsonb, 'apt', '["maven"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends maven && rm -rf /var/lib/apt/lists/*', '', '', '["java", "mvn"]'::jsonb, '', 0, true, true, 273),

    -- C. Node.js / Web
    ('a0000000-0000-0000-0000-000000000048', '00000000-0000-0000-0000-000000000001', 'node-express', 'addon', 'framework', 'Express + API REST', 'Servidor HTTP ligero Express con soporte CORS y recarga en TypeScript (tsx)', 'node', '', '["node", "web"]'::jsonb, 'npm', '["express", "cors", "tsx"]'::jsonb, 'RUN npm install -g --no-audit --no-fund express cors tsx', '', '', '["node", "npx"]'::jsonb, '', 0, true, true, 256),
    ('a0000000-0000-0000-0000-000000000049', '00000000-0000-0000-0000-000000000001', 'node-nestjs', 'addon', 'framework', 'NestJS CLI', 'Framework modular progresivo de Node.js con arquitectura de controladores e inyección de dependencias', 'node', '', '["node", "web"]'::jsonb, 'npm', '["@nestjs/cli"]'::jsonb, 'RUN npm install -g --no-audit --no-fund @nestjs/cli', '', '', '["node", "nest"]'::jsonb, '', 0, true, true, 257),
    ('a0000000-0000-0000-0000-000000000050', '00000000-0000-0000-0000-000000000001', 'node-prisma', 'addon', 'orm', 'Prisma ORM + Clientes SQL', 'ORM tipado Prisma con conectores para PostgreSQL, MySQL y SQLite', 'node', '', '["node", "web"]'::jsonb, 'npm', '["prisma", "@prisma/client", "pg", "mysql2"]'::jsonb, 'RUN npm install -g --no-audit --no-fund prisma @prisma/client pg mysql2', '', '', '["node", "npx"]'::jsonb, '', 0, true, true, 258),
    ('a0000000-0000-0000-0000-000000000051', '00000000-0000-0000-0000-000000000001', 'node-typeorm', 'addon', 'orm', 'TypeORM + Drivers SQL', 'ORM basado en decoradores para TypeScript y Node.js con soporte PostgreSQL', 'node', '', '["node", "web"]'::jsonb, 'npm', '["typeorm", "reflect-metadata", "pg"]'::jsonb, 'RUN npm install -g --no-audit --no-fund typeorm reflect-metadata pg', '', '', '["node", "npx"]'::jsonb, '', 0, true, true, 259),
    ('a0000000-0000-0000-0000-000000000052', '00000000-0000-0000-0000-000000000001', 'node-vitest-jest', 'addon', 'toolkit', 'Pruebas Unitarias (Vitest + Jest)', 'Ejecutores de pruebas unitarias modernas para proyectos JavaScript y TypeScript', 'typescript', '', '["node", "web"]'::jsonb, 'npm', '["vitest", "jest"]'::jsonb, 'RUN npm install -g --no-audit --no-fund vitest jest', '', '', '["node", "npx"]'::jsonb, '', 0, true, true, 262),

    -- C. C / C++
    ('a0000000-0000-0000-0000-000000000053', '00000000-0000-0000-0000-000000000001', 'cpp-sql-drivers', 'addon', 'orm', 'Conectores C/C++ SQLite3 y PostgreSQL', 'Cabeceras y bibliotecas de desarrollo libsqlite3-dev y libpq-dev para persistencia en C/C++', 'cpp', '', '["c", "cpp"]'::jsonb, 'apt', '["libsqlite3-dev", "libpq-dev"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends libsqlite3-dev libpq-dev && rm -rf /var/lib/apt/lists/*', '', '', '["pkg-config"]'::jsonb, '', 0, true, true, 278),
    ('a0000000-0000-0000-0000-000000000054', '00000000-0000-0000-0000-000000000001', 'cpp-openmp', 'addon', 'toolkit', 'Concurrencia y OpenMP / Pthreads', 'Bibliotecas de programación paralela multiproceso OpenMP (libomp-dev) y Pthreads', 'cpp', '', '["c", "cpp"]'::jsonb, 'apt', '["libomp-dev"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends libomp-dev && rm -rf /var/lib/apt/lists/*', '', '', '["gcc", "g++"]'::jsonb, '', 0, true, true, 288),

    -- C. Go
    ('a0000000-0000-0000-0000-000000000055', '00000000-0000-0000-0000-000000000001', 'go-gin-chi', 'addon', 'framework', 'Gin / Chi (APIs HTTP en Go)', 'Enrutadores y frameworks HTTP de alto rendimiento para microservicios en Go', 'go', '', '["go"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '["go"]'::jsonb, '', 0, true, true, 290),
    ('a0000000-0000-0000-0000-000000000056', '00000000-0000-0000-0000-000000000001', 'go-gorm', 'addon', 'orm', 'GORM + Drivers SQL (PostgreSQL / SQLite)', 'Biblioteca ORM para Go y soporte de compilación CGO para SQLite y PostgreSQL', 'go', '', '["go"]'::jsonb, 'apt', '["libsqlite3-dev"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends libsqlite3-dev && rm -rf /var/lib/apt/lists/*', '', '', '["go"]'::jsonb, '', 0, true, true, 291),
    ('a0000000-0000-0000-0000-000000000057', '00000000-0000-0000-0000-000000000001', 'go-golangci-lint', 'addon', 'toolkit', 'Herramientas de Pruebas y Linter (Go)', 'Analizador estático y herramientas de verificación para proyectos Go', 'go', '', '["go"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '["go", "gofmt"]'::jsonb, '', 0, true, true, 292),

    -- C. Rust
    ('a0000000-0000-0000-0000-000000000058', '00000000-0000-0000-0000-000000000001', 'rust-axum', 'addon', 'framework', 'Axum / Actix (Web en Rust)', 'Dependencias del sistema pkg-config y libssl-dev para compilación de servidores web en Rust', 'rust', '', '["rust"]'::jsonb, 'apt', '["pkg-config", "libssl-dev"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*', '', '', '["cargo", "rustc"]'::jsonb, '', 0, true, true, 293),
    ('a0000000-0000-0000-0000-000000000059', '00000000-0000-0000-0000-000000000001', 'rust-sqlx', 'addon', 'orm', 'Conectores SQL en Rust (SQLite / PostgreSQL)', 'Bibliotecas nativas libsqlite3-dev y libpq-dev para consultas SQL verificadas en Rust', 'rust', '', '["rust"]'::jsonb, 'apt', '["libsqlite3-dev", "libpq-dev", "pkg-config"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends libsqlite3-dev libpq-dev pkg-config && rm -rf /var/lib/apt/lists/*', '', '', '["cargo", "pkg-config"]'::jsonb, '', 0, true, true, 294),
    ('a0000000-0000-0000-0000-000000000060', '00000000-0000-0000-0000-000000000001', 'rust-clippy', 'addon', 'toolkit', 'Clippy + Rustfmt (Calidad en Rust)', 'Linter oficial Clippy y formateador rustfmt integrados en la cadena de herramientas', 'rust', '', '["rust"]'::jsonb, 'custom', '["clippy", "rustfmt"]'::jsonb, 'RUN rustup component add clippy rustfmt', '', '', '["cargo", "rustc"]'::jsonb, '', 0, true, true, 295),

    -- C. PHP
    ('a0000000-0000-0000-0000-000000000061', '00000000-0000-0000-0000-000000000001', 'php-laravel', 'addon', 'framework', 'Laravel + Composer', 'Gestor de dependencias Composer y utilidades para aplicaciones web con Laravel en PHP', 'laravel', '', '["php"]'::jsonb, 'apt', '["composer", "unzip"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends composer unzip && rm -rf /var/lib/apt/lists/*', '', '', '["php", "composer"]'::jsonb, '', 0, true, true, 296),
    ('a0000000-0000-0000-0000-000000000062', '00000000-0000-0000-0000-000000000001', 'php-pdo-drivers', 'addon', 'orm', 'Conectores PDO (PostgreSQL / MySQL / SQLite)', 'Bibliotecas de desarrollo para extensiones PDO de bases de datos relacionales en PHP', 'php', '', '["php"]'::jsonb, 'apt', '["libpq-dev", "libsqlite3-dev"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends libpq-dev libsqlite3-dev && rm -rf /var/lib/apt/lists/*', '', '', '["php"]'::jsonb, '', 0, true, true, 297),
    ('a0000000-0000-0000-0000-000000000063', '00000000-0000-0000-0000-000000000001', 'php-phpunit', 'addon', 'toolkit', 'PHPUnit + Composer Tools', 'Entorno con Composer listo para ejecución de pruebas unitarias con PHPUnit', 'php', '', '["php"]'::jsonb, 'apt', '["composer"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends composer && rm -rf /var/lib/apt/lists/*', '', '', '["php", "composer"]'::jsonb, '', 0, true, true, 298),

    -- C. .NET
    ('a0000000-0000-0000-0000-000000000064', '00000000-0000-0000-0000-000000000001', 'dotnet-aspnet', 'addon', 'framework', 'ASP.NET Core Web API', 'Plantillas y herramientas del SDK de .NET 8 para creación de servicios REST y Web APIs', 'dotnet', '', '["dotnet"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '["dotnet"]'::jsonb, '', 0, true, true, 299),
    ('a0000000-0000-0000-0000-000000000065', '00000000-0000-0000-0000-000000000001', 'dotnet-ef', 'addon', 'orm', 'Entity Framework Core + Drivers SQL', 'Soporte nativo y bibliotecas SQLite3 para migraciones y persistencia con EF Core', 'dotnet', '', '["dotnet"]'::jsonb, 'apt', '["libsqlite3-dev"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends libsqlite3-dev && rm -rf /var/lib/apt/lists/*', '', '', '["dotnet"]'::jsonb, '', 0, true, true, 301),
    ('a0000000-0000-0000-0000-000000000066', '00000000-0000-0000-0000-000000000001', 'dotnet-xunit', 'addon', 'toolkit', 'xUnit + Pruebas Unitarias .NET', 'Ejecutor de pruebas dotnet test con plantillas xUnit integradas en el SDK', 'dotnet', '', '["dotnet"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '["dotnet"]'::jsonb, '', 0, true, true, 302)
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
    judge_entrypoint = EXCLUDED.judge_entrypoint,
    tools_provided = EXCLUDED.tools_provided,
    service_engine = EXCLUDED.service_engine,
    ram_delta_mb = EXCLUDED.ram_delta_mb,
    sort_order = EXCLUDED.sort_order,
    updated_at = NOW();

-- +goose Down
-- Restauración idempotente de los valores previos al saneamiento.

-- Baja de los registros del Catálogo Universitario creados por esta migración
DELETE FROM template_components
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug IN (
      'php-83', 'dotnet-8', 'nextjs-web', 'vite-web',
      'python-sqlalchemy', 'python-nosql-clients', 'python-ml', 'python-testing-ruff',
      'java-springboot', 'java-hibernate-jdbc', 'java-gradle', 'java-junit5',
      'node-express', 'node-nestjs', 'node-prisma', 'node-typeorm', 'node-vitest-jest',
      'cpp-sql-drivers', 'cpp-openmp',
      'go-gin-chi', 'go-gorm', 'go-golangci-lint',
      'rust-axum', 'rust-sqlx', 'rust-clippy',
      'php-laravel', 'php-pdo-drivers', 'php-phpunit',
      'dotnet-aspnet', 'dotnet-ef', 'dotnet-xunit'
  );

UPDATE template_components
SET name = 'Maven + Gestión Java',
    icon_key = 'spring',
    sub_category = 'framework',
    ram_delta_mb = 128,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'spring-maven';

UPDATE template_components
SET sub_category = 'framework',
    ram_delta_mb = 64,
    tools_provided = '["python3", "pip"]'::jsonb,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'fastapi';

UPDATE template_components
SET sub_category = 'framework',
    ram_delta_mb = 128,
    tools_provided = '["python3", "pip"]'::jsonb,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'django';

UPDATE template_components
SET sub_category = 'framework',
    ram_delta_mb = 64,
    tools_provided = '["python3", "pip"]'::jsonb,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'flask';

UPDATE template_components
SET sub_category = 'framework',
    ram_delta_mb = 128,
    tools_provided = '["python3", "pip"]'::jsonb,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'streamlit';

UPDATE template_components
SET sub_category = 'toolkit',
    ram_delta_mb = 256,
    tools_provided = '["python3", "pytest"]'::jsonb,
    updated_at = NOW()
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND slug = 'python-datascience';
