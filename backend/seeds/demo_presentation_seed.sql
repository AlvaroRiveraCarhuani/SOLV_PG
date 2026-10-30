-- ============================================================================
-- SOLV Academic Platform — Sembrado Realista para Grabación y Demostración
-- ============================================================================
-- Perfil: Universidad Adventista de Bolivia (UAB)
-- Periodo: Gestion 2026 - Semestre II (2026-II)
-- Roles: Administrador, Docentes titulares, Estudiantes
-- ============================================================================

BEGIN;

-- 1. Actualización de identidades de desarrollo a nombres reales y respetables
UPDATE users 
SET first_name = 'Carlos', 
    last_name = 'Mendoza Ríos', 
    origin = 'manual',
    status = 'active'
WHERE email = 'docente.dev@uab.edu.bo';

UPDATE users 
SET first_name = 'Valeria', 
    last_name = 'Flores Mamani',
    origin = 'manual',
    status = 'active'
WHERE email = 'estudiante.dev@uab.edu.bo';

UPDATE users 
SET first_name = 'Alvaro Regulo', 
    last_name = 'RIVERA CARHUANI',
    status = 'active'
WHERE email = 'regulo.rivera@uab.edu.bo';

-- 2. Docentes adicionales para supervisión institucional (/admin/docentes)
INSERT INTO users (id, first_name, last_name, email, role, tenant_id, origin, status, created_at)
VALUES 
  ('0e5e6bde-1933-4084-82b0-d481b4309ca8', 'Rodrigo', 'Morales Peñaranda', 'rodrigo.morales@uab.edu.bo', 'teacher', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '60 days'),
  ('0e5e6bde-1933-4084-82b0-d481b4309ca9', 'Elena', 'Gutiérrez Siles', 'elena.gutierrez@uab.edu.bo', 'teacher', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '45 days'),
  ('0e5e6bde-1933-4084-82b0-d481b4309caa', 'Marco Antonio', 'Torrico Gómez', 'marco.torrico@uab.edu.bo', 'teacher', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '30 days')
ON CONFLICT (id) DO UPDATE 
SET first_name = EXCLUDED.first_name, 
    last_name = EXCLUDED.last_name, 
    email = EXCLUDED.email, 
    status = EXCLUDED.status;

-- Invitación pendiente para mostrar el enlace de invitación temporal
INSERT INTO teacher_invitations (id, tenant_id, token, email, used, expires_at, created_at, origin, role_type)
VALUES (
  'a1111111-1111-1111-1111-111111111111',
  '00000000-0000-0000-0000-000000000001',
  'inv-sec-vargas-2026-token',
  'fernando.vargas@uab.edu.bo',
  false,
  NOW() + INTERVAL '7 days',
  NOW() - INTERVAL '2 hours',
  'manual',
  'titular'
)
ON CONFLICT (id) DO UPDATE 
SET email = EXCLUDED.email, 
    expires_at = EXCLUDED.expires_at, 
    used = EXCLUDED.used;

-- 3. Estudiantes adicionales para el curso y cohortes
INSERT INTO users (id, first_name, last_name, email, role, tenant_id, origin, status, created_at)
VALUES 
  ('e30c0f6d-ff63-4392-adcf-6336d6132af9', 'Mateo', 'Silva Cadima', 'mateo.silva@uab.edu.bo', 'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '50 days'),
  ('e30c0f6d-ff63-4392-adcf-6336d6132afa', 'Camila', 'Rojas Torrez', 'camila.rojas@uab.edu.bo', 'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '40 days'),
  ('e30c0f6d-ff63-4392-adcf-6336d6132afb', 'Lucas', 'Benítez Choque', 'lucas.benitez@uab.edu.bo', 'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '35 days'),
  ('e30c0f6d-ff63-4392-adcf-6336d6132afc', 'Sofía', 'Méndez Claros', 'sofia.mendez@uab.edu.bo', 'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '30 days'),
  ('e30c0f6d-ff63-4392-adcf-6336d6132afd', 'Daniel', 'Quispe Arispe', 'daniel.quispe@uab.edu.bo', 'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '25 days'),
  ('e30c0f6d-ff63-4392-adcf-6336d6132afe', 'Luciana', 'Paredes Soliz', 'luciana.paredes@uab.edu.bo', 'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '20 days')
ON CONFLICT (id) DO UPDATE 
SET first_name = EXCLUDED.first_name, 
    last_name = EXCLUDED.last_name, 
    email = EXCLUDED.email, 
    status = EXCLUDED.status;

-- 4. Plantillas de laboratorio en catálogo (/admin/plantillas)
INSERT INTO lab_templates (
  id, tenant_id, name, docker_image, base_ram_mb, description, 
  target_environment, status, smoke_test_status, security_audit_status, 
  category_id, services_config, resource_profile, setup_script, tools_declared, stack_config
) VALUES 
(
  'b0000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'Python 3.12 Data Science & IA',
  'python:3.12-slim-bookworm',
  512,
  'Entorno analítico con compilador Python 3.12, soporte de librerías numéricas y herramientas de ciencia de datos.',
  'IDE_PERSISTENTE',
  'APROBADA',
  'passed',
  'passed',
  'c0000000-0000-0000-0000-000000000001',
  '{}'::jsonb,
  '{"cgroups_v2": true, "max_pids": 256}'::jsonb,
  '#!/bin/bash\npip install --no-cache-dir numpy pytest',
  '["python3", "pip", "pytest"]'::jsonb,
  '{"language": "python", "version": "3.12"}'::jsonb
),
(
  'b0000000-0000-0000-0000-000000000002',
  '00000000-0000-0000-0000-000000000001',
  'Java 21 Enterprise & Spring Boot',
  'eclipse-temurin:21-jdk-alpine',
  1024,
  'Plataforma para desarrollo empresarial con JDK 21 Temurin, Maven y Gradle.',
  'IDE_PERSISTENTE',
  'APROBADA',
  'passed',
  'passed',
  'c0000000-0000-0000-0000-000000000004',
  '{}'::jsonb,
  '{"cgroups_v2": true, "max_pids": 512}'::jsonb,
  '#!/bin/sh\napk add --no-cache maven gradle',
  '["java", "javac", "mvn", "gradle"]'::jsonb,
  '{"language": "java", "version": "21"}'::jsonb
),
(
  'b0000000-0000-0000-0000-000000000003',
  '00000000-0000-0000-0000-000000000001',
  'C/C++ Sistemas Operativos & Concurrencia',
  'gcc:13.2.0-bookworm',
  256,
  'Compiladores nativos GCC y Clang, GDB, Valgrind y herramientas para programación de bajo nivel y concurrencia.',
  'IDE_PERSISTENTE',
  'APROBADA',
  'passed',
  'passed',
  'c0000000-0000-0000-0000-000000000003',
  '{}'::jsonb,
  '{"cgroups_v2": true, "max_pids": 128}'::jsonb,
  '#!/bin/bash\napt-get update && apt-get install -y valgrind gdb make cmake',
  '["gcc", "g++", "gdb", "valgrind", "make"]'::jsonb,
  '{"language": "cpp", "version": "17"}'::jsonb
),
(
  'b0000000-0000-0000-0000-000000000004',
  '00000000-0000-0000-0000-000000000001',
  'PostgreSQL 16 Engine & Laboratorio SQL',
  'postgres:16-alpine',
  512,
  'Servidor de base de datos relacional con cliente CLI (psql) para transacciones y modelado de datos.',
  'IDE_PERSISTENTE',
  'APROBADA',
  'passed',
  'passed',
  'c0000000-0000-0000-0000-000000000002',
  '{"db_engine": "postgres", "port": 5432}'::jsonb,
  '{"cgroups_v2": true, "max_pids": 256}'::jsonb,
  '#!/bin/sh\napk add --no-cache postgresql-client',
  '["psql", "pg_dump", "pg_restore"]'::jsonb,
  '{"database": "postgres", "version": "16"}'::jsonb
),
(
  'b0000000-0000-0000-0000-000000000005',
  '00000000-0000-0000-0000-000000000001',
  'Go 1.24 Microservicios & gRPC',
  'golang:1.24-alpine',
  512,
  'Stack ligero para arquitectura de microservicios con soporte para protobuf, gRPC y pruebas de rendimiento.',
  'IDE_PERSISTENTE',
  'PENDIENTE_AUDITORIA',
  'passed',
  'pending',
  'c0000000-0000-0000-0000-000000000002',
  '{}'::jsonb,
  '{"cgroups_v2": true, "max_pids": 256}'::jsonb,
  '#!/bin/sh\napk add --no-cache git protobuf',
  '["go", "protoc"]'::jsonb,
  '{"language": "go", "version": "1.24"}'::jsonb
),
(
  'b0000000-0000-0000-0000-000000000006',
  '00000000-0000-0000-0000-000000000001',
  'Rust 1.78 WebAssembly Sandbox',
  'rust:1.78-alpine',
  512,
  'Herramientas de compilación Rust orientadas a targets WebAssembly y optimización de binarios.',
  'IDE_PERSISTENTE',
  'PAUSADA',
  'passed',
  'passed',
  'c0000000-0000-0000-0000-000000000003',
  '{}'::jsonb,
  '{"cgroups_v2": true, "max_pids": 256}'::jsonb,
  '#!/bin/sh\napk add --no-cache wasm-pack',
  '["rustc", "cargo", "wasm-pack"]'::jsonb,
  '{"language": "rust", "version": "1.78"}'::jsonb
),
(
  'b0000000-0000-0000-0000-000000000007',
  '00000000-0000-0000-0000-000000000001',
  'PHP 7.4 Legacy Monolith',
  'php:7.4-apache',
  256,
  'Entorno deprecado no admitido para nuevos proyectos.',
  'IDE_PERSISTENTE',
  'RECHAZADA',
  'failed',
  'failed',
  'c0000000-0000-0000-0000-000000000002',
  '{}'::jsonb,
  '{"cgroups_v2": true, "max_pids": 128}'::jsonb,
  '#!/bin/sh\necho "Deprecated"',
  '["php"]'::jsonb,
  '{"language": "php", "version": "7.4"}'::jsonb
)
ON CONFLICT (id) DO UPDATE 
SET name = EXCLUDED.name, 
    status = EXCLUDED.status, 
    description = EXCLUDED.description;

-- 5. Materias y Asignaciones Curriculares (/admin/cursos)
INSERT INTO subjects (
  id, tenant_id, name, code, teacher_id, academic_period_id, template_id, is_archived, created_at, updated_at
) VALUES 
(
  '50000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'Desarrollo Web Fullstack',
  'SIS-420',
  '0e5e6bde-1933-4084-82b0-d481b4309ca7',
  'e68d7d15-13b6-40c5-9549-d0adf0cab077',
  'b0000000-0000-0000-0000-000000000004',
  false,
  NOW() - INTERVAL '30 days',
  NOW() - INTERVAL '1 day'
),
(
  '50000000-0000-0000-0000-000000000002',
  '00000000-0000-0000-0000-000000000001',
  'Estructuras de Datos y Algoritmos',
  'SIS-211',
  '0e5e6bde-1933-4084-82b0-d481b4309ca7',
  'e68d7d15-13b6-40c5-9549-d0adf0cab077',
  'b0000000-0000-0000-0000-000000000001',
  false,
  NOW() - INTERVAL '40 days',
  NOW() - INTERVAL '2 days'
),
(
  '50000000-0000-0000-0000-000000000003',
  '00000000-0000-0000-0000-000000000001',
  'Sistemas Operativos y Concurrencia',
  'SIS-325',
  '0e5e6bde-1933-4084-82b0-d481b4309ca8',
  'e68d7d15-13b6-40c5-9549-d0adf0cab077',
  'b0000000-0000-0000-0000-000000000003',
  false,
  NOW() - INTERVAL '45 days',
  NOW() - INTERVAL '3 days'
),
(
  '50000000-0000-0000-0000-000000000004',
  '00000000-0000-0000-0000-000000000001',
  'Taller de Base de Datos',
  'SIS-314',
  '0e5e6bde-1933-4084-82b0-d481b4309ca9',
  'e68d7d15-13b6-40c5-9549-d0adf0cab077',
  'b0000000-0000-0000-0000-000000000004',
  false,
  NOW() - INTERVAL '50 days',
  NOW() - INTERVAL '4 days'
),
(
  '50000000-0000-0000-0000-000000000005',
  '00000000-0000-0000-0000-000000000001',
  'Introducción a la Programación',
  'SIS-112',
  '0e5e6bde-1933-4084-82b0-d481b4309caa',
  'e68d7d15-13b6-40c5-9549-d0adf0cab077',
  'b0000000-0000-0000-0000-000000000001',
  false,
  NOW() - INTERVAL '55 days',
  NOW() - INTERVAL '5 days'
)
ON CONFLICT (id) DO UPDATE 
SET name = EXCLUDED.name, 
    code = EXCLUDED.code, 
    teacher_id = EXCLUDED.teacher_id, 
    template_id = EXCLUDED.template_id;

-- 6. Inscripciones de estudiantes (Matriculación activa)
INSERT INTO enrollments (id, tenant_id, student_id, subject_id, enrolled_at)
VALUES 
  -- Valeria Flores (estudiante.dev) inscrita en 3 materias clave
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132af8', '50000000-0000-0000-0000-000000000001', NOW() - INTERVAL '28 days'),
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132af8', '50000000-0000-0000-0000-000000000002', NOW() - INTERVAL '38 days'),
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132af8', '50000000-0000-0000-0000-000000000004', NOW() - INTERVAL '48 days'),
  -- Mateo Silva
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132af9', '50000000-0000-0000-0000-000000000001', NOW() - INTERVAL '28 days'),
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132af9', '50000000-0000-0000-0000-000000000002', NOW() - INTERVAL '38 days'),
  -- Camila Rojas
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afa', '50000000-0000-0000-0000-000000000001', NOW() - INTERVAL '28 days'),
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afa', '50000000-0000-0000-0000-000000000002', NOW() - INTERVAL '38 days'),
  -- Lucas Benítez
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afb', '50000000-0000-0000-0000-000000000002', NOW() - INTERVAL '38 days'),
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afb', '50000000-0000-0000-0000-000000000003', NOW() - INTERVAL '44 days'),
  -- Daniel Quispe
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afd', '50000000-0000-0000-0000-000000000002', NOW() - INTERVAL '38 days'),
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afd', '50000000-0000-0000-0000-000000000003', NOW() - INTERVAL '44 days'),
  -- Sofía Méndez
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afc', '50000000-0000-0000-0000-000000000001', NOW() - INTERVAL '28 days'),
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afc', '50000000-0000-0000-0000-000000000004', NOW() - INTERVAL '48 days'),
  -- Luciana Paredes
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afe', '50000000-0000-0000-0000-000000000001', NOW() - INTERVAL '28 days'),
  (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'e30c0f6d-ff63-4392-adcf-6336d6132afe', '50000000-0000-0000-0000-000000000002', NOW() - INTERVAL '38 days')
ON CONFLICT DO NOTHING;

-- 7. Ejercicios Académicos Publicados
INSERT INTO exercises (
  id, tenant_id, subject_id, template_id, title, description, 
  type, environment_type, language, status, difficulty, purpose, 
  time_limit_ms, memory_limit_mb, stale, per_student_seed, tags, 
  due_date, config, db_config, comparator_spec, boilerplate, reference_solution, created_at
) VALUES 
(
  'e0000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  '50000000-0000-0000-0000-000000000002',
  'b0000000-0000-0000-0000-000000000001',
  'Práctica 1: Búsqueda Binaria y Análisis de Complejidad',
  'Implementar el algoritmo de búsqueda binaria sobre un arreglo ordenado de enteros. La solución debe tener una complejidad temporal de O(log n) y retornar el índice correspondiente, o -1 si el objetivo no se encuentra en el arreglo.',
  'code',
  'JUEZ_EFIMERO',
  'python',
  'published',
  'medium',
  'class',
  2000,
  128,
  false,
  false,
  ARRAY['algoritmos', 'busqueda', 'python', 'complejidad'],
  NOW() + INTERVAL '15 days',
  '{"allowed_languages": ["python"]}'::jsonb,
  '{"enabled": false}'::jsonb,
  '{"type": "token_match"}'::jsonb,
  'def busqueda_binaria(arr: list[int], objetivo: int) -> int:
    """
    Retorna el índice del elemento objetivo en arr o -1 si no existe.
    Complejidad esperada: O(log n).
    """
    # Escribe tu solución aquí
    pass',
  'def busqueda_binaria(arr: list[int], objetivo: int) -> int:
    izq, der = 0, len(arr) - 1
    while izq <= der:
        medio = (izq + der) // 2
        if arr[medio] == objetivo:
            return medio
        elif arr[medio] < objetivo:
            izq = medio + 1
        else:
            der = medio - 1
    return -1',
  NOW() - INTERVAL '10 days'
),
(
  'e0000000-0000-0000-0000-000000000002',
  '00000000-0000-0000-0000-000000000001',
  '50000000-0000-0000-0000-000000000001',
  'b0000000-0000-0000-0000-000000000004',
  'Práctica 1: Endpoints REST y Modelo de Dominio',
  'Construir un módulo de catálogo de productos en TypeScript. Implementar el cálculo de subtotal y cálculo de total aplicando el impuesto del 13% sobre compras institucionales.',
  'code',
  'IDE_PERSISTENTE',
  'typescript',
  'published',
  'medium',
  'class',
  5000,
  512,
  false,
  false,
  ARRAY['fullstack', 'typescript', 'postgresql', 'rest'],
  NOW() + INTERVAL '20 days',
  '{"allowed_languages": ["typescript", "javascript"]}'::jsonb,
  '{"enabled": true, "engine": "postgres"}'::jsonb,
  '{"type": "numeric_tolerance", "tolerance": 0.01}'::jsonb,
  'export interface Product {
  id: string;
  name: string;
  price: number;
}

export class ProductService {
  calculateTotal(items: Product[]): number {
    // Retornar la suma total de precios con el 13% de IVA incluido
    return 0;
  }
}',
  'export interface Product {
  id: string;
  name: string;
  price: number;
}

export class ProductService {
  calculateTotal(items: Product[]): number {
    const subtotal = items.reduce((acc, item) => acc + item.price, 0);
    return Number((subtotal * 1.13).toFixed(2));
  }
}',
  NOW() - INTERVAL '8 days'
)
ON CONFLICT (id) DO UPDATE 
SET title = EXCLUDED.title, 
    description = EXCLUDED.description,
    status = EXCLUDED.status;

-- 8. Rúbricas de evaluación
INSERT INTO rubrics (id, exercise_id, criteria, created_at, updated_at)
VALUES 
(
  'f0000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000001',
  '[
    {"title": "Corrección Funcional", "max_score": 40, "description": "Supera todos los casos de prueba públicos y ocultos con respuestas exactas."},
    {"title": "Complejidad Algorítmica O(log n)", "max_score": 35, "description": "Aplica división binaria y no recurre a búsqueda secuencial O(n)."},
    {"title": "Casos Límite y Robustez", "max_score": 25, "description": "Gestiona arreglos vacíos, elementos fuera de rango y listas unitarias."}
  ]'::jsonb,
  NOW() - INTERVAL '10 days',
  NOW() - INTERVAL '10 days'
),
(
  'f0000000-0000-0000-0000-000000000002',
  'e0000000-0000-0000-0000-000000000002',
  '[
    {"title": "Diseño y Tipado", "max_score": 40, "description": "Uso de interfaces claras y tipado estricto en métodos de servicio."},
    {"title": "Precisión en Cálculos", "max_score": 35, "description": "Redondeo numérico a dos decimales con tasa impositiva del 13%."},
    {"title": "Calidad del Código", "max_score": 25, "description": "Estructura legible sin efectos secundarios innecesarios."}
  ]'::jsonb,
  NOW() - INTERVAL '8 days',
  NOW() - INTERVAL '8 days'
)
ON CONFLICT (id) DO UPDATE 
SET criteria = EXCLUDED.criteria;

-- 9. Casos de prueba para el Juez
INSERT INTO exercise_test_cases (
  id, exercise_id, order_index, input, expected_output, visibility, weight, created_at
) VALUES 
(
  'd0000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000001',
  1,
  '[1, 3, 5, 7, 9, 11]\n7',
  '3',
  'public',
  25.0,
  NOW() - INTERVAL '10 days'
),
(
  'd0000000-0000-0000-0000-000000000002',
  'e0000000-0000-0000-0000-000000000001',
  2,
  '[2, 4, 6, 8, 10]\n1',
  '-1',
  'public',
  25.0,
  NOW() - INTERVAL '10 days'
),
(
  'd0000000-0000-0000-0000-000000000003',
  'e0000000-0000-0000-0000-000000000001',
  3,
  '[10, 25, 30, 45, 50, 75, 90, 100]\n90',
  '6',
  'hidden',
  25.0,
  NOW() - INTERVAL '10 days'
),
(
  'd0000000-0000-0000-0000-000000000004',
  'e0000000-0000-0000-0000-000000000001',
  4,
  '[5]\n5',
  '0',
  'hidden',
  25.0,
  NOW() - INTERVAL '10 days'
),
(
  'd0000000-0000-0000-0000-000000000005',
  'e0000000-0000-0000-0000-000000000002',
  1,
  '[{"id":"1","name":"Mouse Óptico","price":100}]',
  '113',
  'public',
  50.0,
  NOW() - INTERVAL '8 days'
),
(
  'd0000000-0000-0000-0000-000000000006',
  'e0000000-0000-0000-0000-000000000002',
  2,
  '[{"id":"1","name":"Teclado Mecánico","price":200},{"id":"2","name":"Monitor 27","price":800}]',
  '1130',
  'hidden',
  50.0,
  NOW() - INTERVAL '8 days'
)
ON CONFLICT (id) DO UPDATE 
SET input = EXCLUDED.input, 
    expected_output = EXCLUDED.expected_output;

-- 10. Instancias activas y workspaces del clúster (/admin -> Salud del Servidor)
-- Nota: Valeria Flores no tiene workspace pre-instanciado para que pueda
-- ejecutar el flujo en vivo de creación y arranque desde la interfaz de estudiante.
INSERT INTO workspaces (
  id, student_id, subject_id, template_id, tenant_id, status, type,
  access_url, memory_limit_mb, oom_strike_count, last_heartbeat_at, last_oom_killed_at, created_at, updated_at
) VALUES 
(
  'c1000000-0000-0000-0000-000000000002',
  'e30c0f6d-ff63-4392-adcf-6336d6132afa', -- Camila Rojas
  '50000000-0000-0000-0000-000000000001', -- SIS-420
  'b0000000-0000-0000-0000-000000000004',
  '00000000-0000-0000-0000-000000000001',
  'running',
  'IDE_PERSISTENTE',
  'http://localhost:35039',
  1024,
  0,
  NOW() - INTERVAL '5 minutes',
  null,
  NOW() - INTERVAL '4 hours',
  NOW() - INTERVAL '5 minutes'
),
(
  'c1000000-0000-0000-0000-000000000003',
  'e30c0f6d-ff63-4392-adcf-6336d6132af9', -- Mateo Silva
  '50000000-0000-0000-0000-000000000002', -- SIS-211
  'b0000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'hibernated',
  'IDE_PERSISTENTE',
  'https://workspace-mateo.solv.uab.edu.bo',
  512,
  0,
  NOW() - INTERVAL '45 minutes',
  null,
  NOW() - INTERVAL '1 day',
  NOW() - INTERVAL '45 minutes'
),
(
  'c1000000-0000-0000-0000-000000000004',
  'e30c0f6d-ff63-4392-adcf-6336d6132afb', -- Lucas Benítez
  '50000000-0000-0000-0000-000000000003', -- SIS-325
  'b0000000-0000-0000-0000-000000000003',
  '00000000-0000-0000-0000-000000000001',
  'oom_killed',
  'IDE_PERSISTENTE',
  'https://workspace-lucas.solv.uab.edu.bo',
  256,
  1,
  NOW() - INTERVAL '22 minutes',
  NOW() - INTERVAL '22 minutes',
  NOW() - INTERVAL '6 hours',
  NOW() - INTERVAL '22 minutes'
),
(
  'c1000000-0000-0000-0000-000000000005',
  'e30c0f6d-ff63-4392-adcf-6336d6132afd', -- Daniel Quispe
  '50000000-0000-0000-0000-000000000002', -- SIS-211
  'b0000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'hibernated',
  'IDE_PERSISTENTE',
  'https://workspace-daniel.solv.uab.edu.bo',
  512,
  0,
  NOW() - INTERVAL '90 minutes',
  null,
  NOW() - INTERVAL '2 days',
  NOW() - INTERVAL '90 minutes'
)
ON CONFLICT (id) DO UPDATE 
SET status = EXCLUDED.status, 
    last_heartbeat_at = EXCLUDED.last_heartbeat_at, 
    oom_strike_count = EXCLUDED.oom_strike_count,
    last_oom_killed_at = EXCLUDED.last_oom_killed_at;

-- 11. Entregas estudiantiles para SpeedGrader y Anti-Plagio AST
INSERT INTO submissions (
  id, tenant_id, exercise_id, student_id, workspace_id, code, verdict, 
  ast_result, execution_time_ms, memory_used_mb, score, graded_by, manual_override, submitted_at
) VALUES 
(
  'a2000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000001',
  'e30c0f6d-ff63-4392-adcf-6336d6132af8', -- Valeria Flores
  null,
  'def busqueda_binaria(arr: list[int], objetivo: int) -> int:
    izq, der = 0, len(arr) - 1
    while izq <= der:
        medio = (izq + der) // 2
        if arr[medio] == objetivo:
            return medio
        elif arr[medio] < objetivo:
            izq = medio + 1
        else:
            der = medio - 1
    return -1',
  'AC',
  '{"score": 100, "similarity": 0.05, "complexity": "O(log n)", "structure_hash": "a9f8e71b"}'::jsonb,
  38,
  18,
  100,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7',
  false,
  NOW() - INTERVAL '3 days'
),
(
  'a2000000-0000-0000-0000-000000000002',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000001',
  'e30c0f6d-ff63-4392-adcf-6336d6132afa', -- Camila Rojas
  'c1000000-0000-0000-0000-000000000002',
  'def busqueda_binaria(arr, x):
    low, high = 0, len(arr) - 1
    while low <= high:
        mid = low + (high - low) // 2
        if arr[mid] == x:
            return mid
        elif arr[mid] < x:
            low = mid + 1
        else:
            high = mid - 1
    return -1',
  'AC',
  '{"score": 95, "similarity": 0.35, "complexity": "O(log n)", "structure_hash": "b2c3d4e5"}'::jsonb,
  42,
  19,
  95,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7',
  false,
  NOW() - INTERVAL '2 days'
),
(
  'a2000000-0000-0000-0000-000000000003',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000001',
  'e30c0f6d-ff63-4392-adcf-6336d6132af9', -- Mateo Silva
  'c1000000-0000-0000-0000-000000000003',
  'def busqueda_binaria(arr, x):
    izq, der = 0, len(arr)
    while izq < der:
        m = (izq + der) // 2
        if arr[m] == x:
            return m
        elif arr[m] < x:
            izq = m
        else:
            der = m - 1
    return -1',
  'WA',
  '{"score": 50, "similarity": 0.20, "complexity": "O(log n)", "structure_hash": "c3d4e5f6"}'::jsonb,
  39,
  18,
  50,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7',
  false,
  NOW() - INTERVAL '1 day'
),
(
  'a2000000-0000-0000-0000-000000000004',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000001',
  'e30c0f6d-ff63-4392-adcf-6336d6132afb', -- Lucas Benítez
  'c1000000-0000-0000-0000-000000000004',
  'def busqueda_binaria(arr, x):
    # Implementación ingenua lineal
    for i in range(len(arr)):
        if arr[i] == x:
            return i
    return -1',
  'TLE',
  '{"score": 25, "similarity": 0.05, "complexity": "O(n)", "structure_hash": "d4e5f6a1"}'::jsonb,
  2150,
  24,
  25,
  '0e5e6bde-1933-4084-82b0-d481b4309ca7',
  false,
  NOW() - INTERVAL '18 hours'
),
(
  'a2000000-0000-0000-0000-000000000005',
  '00000000-0000-0000-0000-000000000001',
  'e0000000-0000-0000-0000-000000000001',
  'e30c0f6d-ff63-4392-adcf-6336d6132afd', -- Daniel Quispe (Pendiente de calificar en SpeedGrader)
  'c1000000-0000-0000-0000-000000000005',
  'def busqueda_binaria(arr, x):
    l, r = 0, len(arr) - 1
    while l <= r:
        m = (l + r) // 2
        if arr[m] == x:
            return m
        elif arr[m] < x:
            l = m + 1
        else:
            r = m - 1
    return -1',
  'AC',
  '{"score": 90, "similarity": 0.78, "complexity": "O(log n)", "structure_hash": "e5f6a1b2", "plagiarism_flag": true, "match_student": "Camila Rojas"}'::jsonb,
  45,
  19,
  null,
  null,
  false,
  NOW() - INTERVAL '4 hours'
)
ON CONFLICT (id) DO UPDATE 
SET verdict = EXCLUDED.verdict, 
    score = EXCLUDED.score, 
    ast_result = EXCLUDED.ast_result;

-- 12. Keystroke events para el Replay del SpeedGrader (Daniel Quispe)
DELETE FROM submission_keystroke_events WHERE submission_id = 'a2000000-0000-0000-0000-000000000005';
INSERT INTO submission_keystroke_events (
  id, submission_id, timestamp_ms, event_type, position, content, paste_source_detected, created_at
) VALUES 
(gen_random_uuid(), 'a2000000-0000-0000-0000-000000000005', 1000, 'insert', 0, 'def busqueda_binaria(arr, x):\n', false, NOW() - INTERVAL '4 hours'),
(gen_random_uuid(), 'a2000000-0000-0000-0000-000000000005', 3500, 'insert', 31, '    l, r = 0, len(arr) - 1\n', false, NOW() - INTERVAL '4 hours'),
(gen_random_uuid(), 'a2000000-0000-0000-0000-000000000005', 7200, 'insert', 58, '    while l <= r:\n', false, NOW() - INTERVAL '4 hours'),
(gen_random_uuid(), 'a2000000-0000-0000-0000-000000000005', 11500, 'insert', 76, '        m = (l + r) // 2\n', false, NOW() - INTERVAL '4 hours'),
(gen_random_uuid(), 'a2000000-0000-0000-0000-000000000005', 15800, 'insert', 101, '        if arr[m] == x:\n            return m\n', false, NOW() - INTERVAL '4 hours'),
(gen_random_uuid(), 'a2000000-0000-0000-0000-000000000005', 20400, 'insert', 153, '        elif arr[m] < x:\n            l = m + 1\n', false, NOW() - INTERVAL '4 hours'),
(gen_random_uuid(), 'a2000000-0000-0000-0000-000000000005', 25100, 'insert', 205, '        else:\n            r = m - 1\n', false, NOW() - INTERVAL '4 hours'),
(gen_random_uuid(), 'a2000000-0000-0000-0000-000000000005', 28900, 'insert', 247, '    return -1\n', false, NOW() - INTERVAL '4 hours');

-- ============================================================================
-- Remediación idempotente: los laboratorios IDE_PERSISTENTE se corrigen con
-- rúbrica manual en SpeedGrader, no contra el juez I/O. Se normalizan sus
-- veredictos automáticos y se retiran sus casos de prueba de demo.
-- ============================================================================
UPDATE submissions su
SET verdict = 'PENDING',
    score = 0,
    execution_time_ms = 0,
    memory_used_mb = 0,
    graded_by = NULL,
    manual_override = FALSE,
    override_reason = ''
FROM exercises e
WHERE su.exercise_id = e.id
  AND e.environment_type = 'IDE_PERSISTENTE'
  AND su.verdict <> 'PENDING';

DELETE FROM exercise_test_cases tc
USING exercises e
WHERE tc.exercise_id = e.id
  AND e.environment_type = 'IDE_PERSISTENTE';

UPDATE exercises SET type = 'project'
WHERE type = 'code' AND environment_type = 'IDE_PERSISTENTE';

UPDATE exercises SET type = 'algorithm'
WHERE type = 'code'
  AND (environment_type = 'JUEZ_EFIMERO' OR environment_type IS NULL OR environment_type = '');

COMMIT;
