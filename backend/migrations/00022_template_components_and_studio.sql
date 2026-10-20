-- +goose Up
CREATE TABLE IF NOT EXISTS template_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    slug VARCHAR(64) NOT NULL,
    layer VARCHAR(20) NOT NULL CHECK (layer IN ('runtime', 'addon', 'database')),
    sub_category VARCHAR(30) NOT NULL DEFAULT 'general',
    name VARCHAR(120) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon_key VARCHAR(40) NOT NULL DEFAULT 'docker',
    base_image VARCHAR(255) NOT NULL DEFAULT '',
    compatible_runtimes JSONB NOT NULL DEFAULT '[]'::jsonb,
    install_mode VARCHAR(20) NOT NULL DEFAULT 'none' CHECK (install_mode IN ('none', 'pip', 'npm', 'apt', 'custom')),
    package_names JSONB NOT NULL DEFAULT '[]'::jsonb,
    build_layer_snippet TEXT NOT NULL DEFAULT '',
    setup_snippet TEXT NOT NULL DEFAULT '',
    judge_entrypoint VARCHAR(255) NOT NULL DEFAULT '',
    tools_provided JSONB NOT NULL DEFAULT '[]'::jsonb,
    service_engine VARCHAR(40) NOT NULL DEFAULT '',
    ram_delta_mb INT NOT NULL DEFAULT 0,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, slug)
);

ALTER TABLE lab_templates
    ADD COLUMN IF NOT EXISTS stack_config JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE template_models
    ADD COLUMN IF NOT EXISTS stack_config JSONB NOT NULL DEFAULT '{}'::jsonb;

-- Semilla institucional del Catálogo de Componentes Tecnológicos (Runtimes, Plataformas Web, Complementos y Bases de Datos)
INSERT INTO template_components (
    id, tenant_id, slug, layer, sub_category, name, description, icon_key,
    base_image, compatible_runtimes, install_mode, package_names,
    build_layer_snippet, setup_snippet, judge_entrypoint, tools_provided,
    service_engine, ram_delta_mb, is_system, is_active, sort_order
) VALUES
    -- 1. Runtimes Principales (Backend / Lenguajes)
    ('a0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'python-312', 'runtime', 'backend', 'Python 3.12', 'Entorno oficial con pip y soporte para desarrollo general, APIs y ciencia de datos', 'python', 'python:3.12-slim-bookworm', '["python"]'::jsonb, 'none', '[]'::jsonb, '', '', 'python3 solution.py', '["python3", "pip"]'::jsonb, '', 512, true, true, 10),
    ('a0000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'node-20', 'runtime', 'backend', 'Node.js 20 LTS', 'Entorno JavaScript y TypeScript con gestor de paquetes npm y npx', 'node', 'node:20-bookworm-slim', '["node", "web"]'::jsonb, 'none', '[]'::jsonb, '', '', 'node solution.js', '["node", "npm", "npx"]'::jsonb, '', 512, true, true, 20),
    ('a0000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'java-21', 'runtime', 'backend', 'Java 21 LTS', 'Entorno OpenJDK 21 para programación orientada a objetos y sistemas empresariales', 'java', 'eclipse-temurin:21-jdk-jammy', '["java"]'::jsonb, 'none', '[]'::jsonb, '', '', 'javac Solution.java && java -Xmx256m Solution', '["java", "javac", "jar"]'::jsonb, '', 1024, true, true, 30),
    ('a0000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'go-122', 'runtime', 'backend', 'Go 1.22', 'Compilador oficial de Go con herramientas de formateo y concurrencia nativa', 'go', 'golang:1.22-bookworm', '["go"]'::jsonb, 'none', '[]'::jsonb, '', '', 'go run main.go', '["go", "gofmt"]'::jsonb, '', 512, true, true, 40),
    ('a0000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'cpp-gcc13', 'runtime', 'backend', 'C / C++ (GCC 13)', 'Compiladores gcc y g++ con herramientas make y depuración para algoritmia y sistemas', 'cpp', 'gcc:13.2-bookworm', '["cpp"]'::jsonb, 'none', '[]'::jsonb, '', '', 'gcc -O2 solution.c -o solution && ./solution', '["gcc", "g++", "make"]'::jsonb, '', 384, true, true, 50),
    ('a0000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'rust-stable', 'runtime', 'backend', 'Rust', 'Compilador rustc y gestor Cargo para programación de sistemas segura', 'rust', 'rust:1.77-slim-bookworm', '["rust"]'::jsonb, 'none', '[]'::jsonb, '', '', 'rustc -O solution.rs -o solution && ./solution', '["rustc", "cargo"]'::jsonb, '', 512, true, true, 60),

    -- 2. Plataformas Web / Frontend (Combinables en Columna 1)
    ('a0000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'angular-web', 'runtime', 'web', 'Angular', 'Plataforma de desarrollo Frontend SPA con Angular CLI y soporte TypeScript', 'angular', 'node:20-bookworm-slim', '["node", "web"]'::jsonb, 'npm', '["@angular/cli"]'::jsonb, 'RUN npm install -g --no-audit --no-fund @angular/cli', '', '', '["node", "npm", "ng"]'::jsonb, '', 256, true, true, 110),
    ('a0000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', 'react-vite', 'runtime', 'web', 'React + Vite', 'Entorno ligero de interfaz web con React, Vite y recarga instantánea', 'react', 'node:20-bookworm-slim', '["node", "web"]'::jsonb, 'npm', '["vite", "typescript"]'::jsonb, 'RUN npm install -g --no-audit --no-fund vite typescript', '', '', '["node", "npm", "npx"]'::jsonb, '', 128, true, true, 120),
    ('a0000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', 'vue-vite', 'runtime', 'web', 'Vue 3 + Vite', 'Framework progresivo para interfaces web reactivas con Vite', 'vue', 'node:20-bookworm-slim', '["node", "web"]'::jsonb, 'npm', '["vite", "@vue/cli"]'::jsonb, 'RUN npm install -g --no-audit --no-fund vite', '', '', '["node", "npm"]'::jsonb, '', 128, true, true, 130),

    -- 3. Complementos y Frameworks (Columna 2 - Filtrados por compatibilidad)
    ('a0000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000001', 'fastapi', 'addon', 'framework', 'FastAPI + Uvicorn', 'Framework moderno de alto rendimiento para crear APIs REST en Python', 'fastapi', '', '["python"]'::jsonb, 'pip', '["fastapi", "uvicorn"]'::jsonb, 'RUN pip install --no-cache-dir fastapi uvicorn', '', '', '["python3", "pip"]'::jsonb, '', 64, true, true, 210),
    ('a0000000-0000-0000-0000-000000000021', '00000000-0000-0000-0000-000000000001', 'django', 'addon', 'framework', 'Django 5', 'Framework web completo en Python con ORM integrado y panel administrativo', 'django', '', '["python"]'::jsonb, 'pip', '["django", "djangorestframework"]'::jsonb, 'RUN pip install --no-cache-dir django djangorestframework', '', '', '["python3", "pip"]'::jsonb, '', 128, true, true, 220),
    ('a0000000-0000-0000-0000-000000000022', '00000000-0000-0000-0000-000000000001', 'flask', 'addon', 'framework', 'Flask', 'Micro-framework ligero para aplicaciones web y servicios en Python', 'flask', '', '["python"]'::jsonb, 'pip', '["flask"]'::jsonb, 'RUN pip install --no-cache-dir flask', '', '', '["python3", "pip"]'::jsonb, '', 64, true, true, 230),
    ('a0000000-0000-0000-0000-000000000023', '00000000-0000-0000-0000-000000000001', 'streamlit', 'addon', 'framework', 'Streamlit (UI en Python)', 'Permite crear interfaces web interactivas directamente en Python sin HTML/JS', 'streamlit', '', '["python"]'::jsonb, 'pip', '["streamlit"]'::jsonb, 'RUN pip install --no-cache-dir streamlit', '', '', '["python3", "pip"]'::jsonb, '', 128, true, true, 240),
    ('a0000000-0000-0000-0000-000000000024', '00000000-0000-0000-0000-000000000001', 'python-datascience', 'addon', 'toolkit', 'Ciencia de Datos (NumPy + Pandas)', 'Librerías esenciales para procesamiento numérico, tablas y análisis estadístico', 'python', '', '["python"]'::jsonb, 'pip', '["numpy", "pandas", "pytest"]'::jsonb, 'RUN pip install --no-cache-dir numpy pandas pytest', '', '', '["python3", "pytest"]'::jsonb, '', 256, true, true, 250),
    ('a0000000-0000-0000-0000-000000000025', '00000000-0000-0000-0000-000000000001', 'typescript-tools', 'addon', 'toolkit', 'TypeScript + Linters', 'Compilador tsc y herramientas de tipado estático para proyectos web y Node.js', 'typescript', '', '["node", "web"]'::jsonb, 'npm', '["typescript", "ts-node"]'::jsonb, 'RUN npm install -g --no-audit --no-fund typescript ts-node', '', '', '["node", "tsc"]'::jsonb, '', 64, true, true, 260),
    ('a0000000-0000-0000-0000-000000000026', '00000000-0000-0000-0000-000000000001', 'spring-maven', 'addon', 'framework', 'Maven + Gestión Java', 'Gestor de dependencias y construcción de proyectos para ecosistemas Java y Spring', 'spring', '', '["java"]'::jsonb, 'apt', '["maven"]'::jsonb, 'RUN apk add --no-cache maven || (apt-get update && apt-get install -y --no-install-recommends maven && rm -rf /var/lib/apt/lists/*)', '', '', '["java", "mvn"]'::jsonb, '', 128, true, true, 270),

    -- 4. Bases de Datos y Servicios Acoplados (Columna 3)
    ('a0000000-0000-0000-0000-000000000030', '00000000-0000-0000-0000-000000000001', 'db-none', 'database', 'none', 'Sin Base de Datos', 'El laboratorio solo utilizará archivos de código en la carpeta del estudiante', 'docker', '', '["universal"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '[]'::jsonb, 'none', 0, true, true, 300),
    ('a0000000-0000-0000-0000-000000000031', '00000000-0000-0000-0000-000000000001', 'sqlite-3', 'database', 'relational', 'SQLite 3 (En archivo)', 'Base de datos ligera guardada directamente en la carpeta de trabajo sin consumo extra de memoria', 'sqlite', '', '["universal"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '[]'::jsonb, 'sqlite', 0, true, true, 310),
    ('a0000000-0000-0000-0000-000000000032', '00000000-0000-0000-0000-000000000001', 'postgres-16', 'database', 'relational', 'PostgreSQL 16', 'Base de datos relacional lista para conectar automáticamente desde la práctica', 'postgres', '', '["universal"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '[]'::jsonb, 'postgres', 50, true, true, 320),
    ('a0000000-0000-0000-0000-000000000033', '00000000-0000-0000-0000-000000000001', 'mysql-84', 'database', 'relational', 'MySQL 8.4', 'Motor de base de datos relacional clásico preconfigurado y aislado por alumno', 'mysql', '', '["universal"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '[]'::jsonb, 'mysql', 128, true, true, 330),
    ('a0000000-0000-0000-0000-000000000034', '00000000-0000-0000-0000-000000000001', 'redis-72', 'database', 'nosql', 'Redis 7.2 (Caché)', 'Almacén ultrarrápido de datos en memoria tipo clave-valor', 'redis', '', '["universal"]'::jsonb, 'none', '[]'::jsonb, '', '', '', '[]'::jsonb, 'redis', 64, true, true, 340)
ON CONFLICT (tenant_id, slug) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    icon_key = EXCLUDED.icon_key,
    base_image = EXCLUDED.base_image,
    compatible_runtimes = EXCLUDED.compatible_runtimes,
    build_layer_snippet = EXCLUDED.build_layer_snippet,
    judge_entrypoint = EXCLUDED.judge_entrypoint,
    tools_provided = EXCLUDED.tools_provided,
    service_engine = EXCLUDED.service_engine,
    ram_delta_mb = EXCLUDED.ram_delta_mb,
    sort_order = EXCLUDED.sort_order,
    updated_at = NOW();

-- +goose Down
ALTER TABLE template_models DROP COLUMN IF EXISTS stack_config;
ALTER TABLE lab_templates DROP COLUMN IF EXISTS stack_config;
DROP TABLE IF EXISTS template_components;
