-- +goose Up
-- Separación explícita de perfiles C (GCC 13) y C++ (G++ 13), soporte de gestores de paquetes de primera clase en servidor (pnpm, yarn, bun, uv, cargo, go, composer, dotnet) y semilla institucional idempotente completa

-- 0. Ampliar la restricción de install_mode para que el backend gobierne y sintetice de forma segura todos los gestores modernos sin depender de custom RUN en cliente
ALTER TABLE template_components DROP CONSTRAINT IF EXISTS template_components_install_mode_check;
ALTER TABLE template_components ADD CONSTRAINT template_components_install_mode_check
    CHECK (install_mode IN ('none', 'pip', 'uv', 'npm', 'pnpm', 'yarn', 'bun', 'apt', 'cargo', 'go', 'composer', 'dotnet', 'custom'));

-- 1. Asegurar el catálogo institucional completo (auto-reparable si se limpió en pruebas zero-state)
INSERT INTO template_components (
    id, tenant_id, slug, layer, sub_category, name, description, icon_key,
    base_image, compatible_runtimes, install_mode, package_names,
    build_layer_snippet, setup_snippet, judge_entrypoint, tools_provided,
    service_engine, ram_delta_mb, is_system, is_active, sort_order
) VALUES
    -- Columna 1A: Lenguajes Principales (Backend / Sistemas)
    ('a0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'python-312', 'runtime', 'backend', 'Python 3.12', 'Entorno oficial con pip y soporte para desarrollo general, APIs y ciencia de datos', 'python', 'python:3.12-slim-bookworm', '["python"]'::jsonb, 'none', '[]'::jsonb, '', '', 'python3 solution.py', '["python3", "pip"]'::jsonb, '', 512, true, true, 10),
    ('a0000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'node-20', 'runtime', 'backend', 'Node.js 20 LTS', 'Entorno JavaScript y TypeScript con gestor de paquetes npm y npx', 'node', 'node:20-bookworm-slim', '["node", "web"]'::jsonb, 'none', '[]'::jsonb, '', '', 'node solution.js', '["node", "npm", "npx"]'::jsonb, '', 512, true, true, 20),
    ('a0000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'java-21', 'runtime', 'backend', 'Java 21 LTS', 'Entorno OpenJDK 21 Temurin para programación orientada a objetos y sistemas empresariales', 'java', 'eclipse-temurin:21-jdk-jammy', '["java"]'::jsonb, 'none', '[]'::jsonb, '', '', 'javac Solution.java && java -Xmx256m Solution', '["java", "javac", "jar"]'::jsonb, '', 1024, true, true, 30),
    ('a0000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'go-122', 'runtime', 'backend', 'Go 1.22', 'Compilador oficial de Go con herramientas de formateo y concurrencia nativa', 'go', 'golang:1.22-bookworm', '["go"]'::jsonb, 'none', '[]'::jsonb, '', '', 'go run main.go', '["go", "gofmt"]'::jsonb, '', 512, true, true, 40),
    ('a0000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'c-gcc13', 'runtime', 'backend', 'C (GCC 13)', 'Compilador oficial gcc (C17) con libc y herramientas make para programación estructurada y sistemas', 'c', 'gcc:13.2-bookworm', '["c", "cpp"]'::jsonb, 'none', '[]'::jsonb, '', '', 'gcc -O2 solution.c -o solution && ./solution', '["gcc", "make"]'::jsonb, '', 384, true, true, 45),
    ('a0000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'cpp-gcc13', 'runtime', 'backend', 'C++ (G++ 13)', 'Compilador oficial g++ (C++20) con biblioteca estándar STL y make sobre Debian Bookworm', 'cpp', 'gcc:13.2-bookworm', '["cpp", "c"]'::jsonb, 'none', '[]'::jsonb, '', '', 'g++ -O2 solution.cpp -o solution && ./solution', '["g++", "gcc", "make"]'::jsonb, '', 384, true, true, 50),
    ('a0000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'rust-stable', 'runtime', 'backend', 'Rust', 'Compilador rustc y gestor Cargo para programación de sistemas segura', 'rust', 'rust:1.77-slim-bookworm', '["rust"]'::jsonb, 'none', '[]'::jsonb, '', '', 'rustc -O solution.rs -o solution && ./solution', '["rustc", "cargo"]'::jsonb, '', 512, true, true, 60),

    -- Columna 1B: Plataformas Web / Frontend
    ('a0000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000001', 'angular-web', 'runtime', 'web', 'Angular', 'Plataforma de desarrollo Frontend SPA con Angular CLI y soporte TypeScript', 'angular', 'node:20-bookworm-slim', '["node", "web"]'::jsonb, 'npm', '["@angular/cli"]'::jsonb, 'RUN npm install -g --no-audit --no-fund @angular/cli', '', '', '["node", "npm", "ng"]'::jsonb, '', 256, true, true, 110),
    ('a0000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000001', 'react-vite', 'runtime', 'web', 'React + Vite', 'Entorno ligero de interfaz web con React, Vite y recarga instantánea', 'react', 'node:20-bookworm-slim', '["node", "web"]'::jsonb, 'npm', '["vite", "typescript"]'::jsonb, 'RUN npm install -g --no-audit --no-fund vite typescript', '', '', '["node", "npm", "npx"]'::jsonb, '', 128, true, true, 120),
    ('a0000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000001', 'vue-vite', 'runtime', 'web', 'Vue 3 + Vite', 'Framework progresivo para interfaces web reactivas con Vite', 'vue', 'node:20-bookworm-slim', '["node", "web"]'::jsonb, 'npm', '["vite", "@vue/cli"]'::jsonb, 'RUN npm install -g --no-audit --no-fund vite', '', '', '["node", "npm"]'::jsonb, '', 128, true, true, 130),

    -- Columna 2: Complementos y Kits por Ecosistema
    ('a0000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000001', 'fastapi', 'addon', 'framework', 'FastAPI + Uvicorn', 'Framework moderno de alto rendimiento para crear APIs REST en Python', 'fastapi', '', '["python"]'::jsonb, 'pip', '["fastapi", "uvicorn"]'::jsonb, 'RUN pip install --no-cache-dir fastapi uvicorn', '', '', '["python3", "pip"]'::jsonb, '', 64, true, true, 210),
    ('a0000000-0000-0000-0000-000000000021', '00000000-0000-0000-0000-000000000001', 'django', 'addon', 'framework', 'Django 5', 'Framework web completo en Python con ORM integrado y panel administrativo', 'django', '', '["python"]'::jsonb, 'pip', '["django", "djangorestframework"]'::jsonb, 'RUN pip install --no-cache-dir django djangorestframework', '', '', '["python3", "pip"]'::jsonb, '', 128, true, true, 220),
    ('a0000000-0000-0000-0000-000000000022', '00000000-0000-0000-0000-000000000001', 'flask', 'addon', 'framework', 'Flask', 'Micro-framework ligero para aplicaciones web y servicios en Python', 'flask', '', '["python"]'::jsonb, 'pip', '["flask"]'::jsonb, 'RUN pip install --no-cache-dir flask', '', '', '["python3", "pip"]'::jsonb, '', 64, true, true, 230),
    ('a0000000-0000-0000-0000-000000000023', '00000000-0000-0000-0000-000000000001', 'streamlit', 'addon', 'framework', 'Streamlit (UI en Python)', 'Permite crear interfaces web interactivas directamente en Python sin HTML/JS', 'streamlit', '', '["python"]'::jsonb, 'pip', '["streamlit"]'::jsonb, 'RUN pip install --no-cache-dir streamlit', '', '', '["python3", "pip"]'::jsonb, '', 128, true, true, 240),
    ('a0000000-0000-0000-0000-000000000024', '00000000-0000-0000-0000-000000000001', 'python-datascience', 'addon', 'toolkit', 'Ciencia de Datos (NumPy + Pandas)', 'Librerías esenciales para procesamiento numérico, tablas y análisis estadístico', 'python', '', '["python"]'::jsonb, 'pip', '["numpy", "pandas", "pytest"]'::jsonb, 'RUN pip install --no-cache-dir numpy pandas pytest', '', '', '["python3", "pytest"]'::jsonb, '', 256, true, true, 250),
    ('a0000000-0000-0000-0000-000000000025', '00000000-0000-0000-0000-000000000001', 'typescript-tools', 'addon', 'toolkit', 'TypeScript + Linters', 'Compilador tsc y herramientas de tipado estático para proyectos web y Node.js', 'typescript', '', '["node", "web"]'::jsonb, 'npm', '["typescript", "ts-node"]'::jsonb, 'RUN npm install -g --no-audit --no-fund typescript ts-node', '', '', '["node", "tsc"]'::jsonb, '', 64, true, true, 260),
    ('a0000000-0000-0000-0000-000000000026', '00000000-0000-0000-0000-000000000001', 'spring-maven', 'addon', 'framework', 'Maven + Gestión Java', 'Gestor de dependencias y construcción de proyectos para ecosistemas Java y Spring', 'spring', '', '["java"]'::jsonb, 'apt', '["maven"]'::jsonb, 'RUN apk add --no-cache maven || (apt-get update && apt-get install -y --no-install-recommends maven && rm -rf /var/lib/apt/lists/*)', '', '', '["java", "mvn"]'::jsonb, '', 128, true, true, 270),
    ('a0000000-0000-0000-0000-000000000027', '00000000-0000-0000-0000-000000000001', 'cpp-debug-valgrind', 'addon', 'toolkit', 'Depuración y Memoria (GDB + Valgrind)', 'Depurador paso a paso GDB y detector de fugas de memoria Valgrind para prácticas de punteros y estructuras de datos', 'cpp', '', '["c", "cpp"]'::jsonb, 'apt', '["gdb", "valgrind"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends gdb valgrind && rm -rf /var/lib/apt/lists/*', '', '', '["gdb", "valgrind"]'::jsonb, '', 0, true, true, 275),
    ('a0000000-0000-0000-0000-000000000028', '00000000-0000-0000-0000-000000000001', 'cpp-cmake', 'addon', 'toolkit', 'Construcción CMake + Ninja', 'Sistema de generación de proyectos multiplataforma CMake y constructor rápido Ninja para proyectos C/C++', 'cpp', '', '["c", "cpp"]'::jsonb, 'apt', '["cmake", "ninja-build"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends cmake ninja-build && rm -rf /var/lib/apt/lists/*', '', '', '["cmake", "ninja"]'::jsonb, '', 0, true, true, 280),
    ('a0000000-0000-0000-0000-000000000029', '00000000-0000-0000-0000-000000000001', 'cpp-googletest', 'addon', 'toolkit', 'Pruebas Unitarias C/C++ (GoogleTest)', 'Cabeceras y bibliotecas de GoogleTest junto con pkg-config para evaluación automatizada en C y C++', 'cpp', '', '["c", "cpp"]'::jsonb, 'apt', '["libgtest-dev", "pkg-config"]'::jsonb, 'RUN apt-get update && apt-get install -y --no-install-recommends libgtest-dev pkg-config && rm -rf /var/lib/apt/lists/*', '', '', '["pkg-config", "libgtest-dev"]'::jsonb, '', 0, true, true, 285),

    -- Columna 3: Bases de Datos y Servicios Acoplados
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
    install_mode = EXCLUDED.install_mode,
    package_names = EXCLUDED.package_names,
    build_layer_snippet = EXCLUDED.build_layer_snippet,
    judge_entrypoint = EXCLUDED.judge_entrypoint,
    tools_provided = EXCLUDED.tools_provided,
    service_engine = EXCLUDED.service_engine,
    ram_delta_mb = EXCLUDED.ram_delta_mb,
    sort_order = EXCLUDED.sort_order,
    updated_at = NOW();

-- Eliminar duplicado manual python-3-12 si ya existe el oficial python-312
DELETE FROM template_components
WHERE slug = 'python-3-12'
  AND tenant_id = '00000000-0000-0000-0000-000000000001'
  AND EXISTS (
      SELECT 1 FROM template_components
      WHERE slug = 'python-312' AND tenant_id = '00000000-0000-0000-0000-000000000001'
  );

-- +goose Down
DELETE FROM template_components
WHERE slug IN ('c-gcc13', 'cpp-debug-valgrind', 'cpp-cmake', 'cpp-googletest')
  AND tenant_id = '00000000-0000-0000-0000-000000000001';

UPDATE template_components
SET
    name = 'C / C++ (GCC 13)',
    description = 'Compiladores gcc y g++ con herramientas make y depuración para algoritmia y sistemas',
    compatible_runtimes = '["cpp"]'::jsonb,
    judge_entrypoint = 'gcc -O2 solution.c -o solution && ./solution',
    tools_provided = '["gcc", "g++", "make"]'::jsonb,
    sort_order = 50,
    updated_at = NOW()
WHERE slug = 'cpp-gcc13';
