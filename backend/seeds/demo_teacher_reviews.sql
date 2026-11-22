-- ============================================================================
-- SOLV — Seed de la cola docente «Por calificar»
-- Objetivo: poblar una instalación local con materias, prácticas del Juez
--   (JUEZ_EFIMERO) y entregas en espera de corrección, para validar la cola
--   «Por calificar» y la pantalla de Revisión de Entrega con datos realistas.
-- Idempotente y re-ejecutable. Solo datos: no cambia el esquema.
-- Uso: docker exec -i solv_db psql -U solv_user -d solv_db \
--        < backend/seeds/demo_teacher_reviews.sql
-- ============================================================================

BEGIN;

-- ============================================================================
-- B0. Contexto: se resuelve por correo porque el id cambia entre entornos.
-- ============================================================================
DROP TABLE IF EXISTS seed_reviews_ctx;
CREATE TEMP TABLE seed_reviews_ctx AS
SELECT
    (SELECT id FROM users WHERE email = 'docente.dev@uab.edu.bo')  AS teacher_id,
    '00000000-0000-0000-0000-000000000001'::uuid                   AS tenant_id,
    (SELECT id FROM academic_periods ORDER BY start_date LIMIT 1)  AS period_id,
    (SELECT id FROM lab_templates WHERE name = 'Python Runner Base' AND status = 'APROBADA' LIMIT 1) AS python_template_id,
    (SELECT id FROM lab_templates WHERE name = 'Python 3.12 — Proyecto CLI' AND status = 'APROBADA' LIMIT 1) AS python_project_template_id;

-- El período vigente debe estar activo para que el panel docente lo muestre.
UPDATE academic_periods
SET is_active = true
WHERE id = (SELECT period_id FROM seed_reviews_ctx);

-- La materia que ya usaba el docente vuelve a estar vigente.
UPDATE subjects
SET is_archived = false,
    teacher_id  = (SELECT teacher_id FROM seed_reviews_ctx),
    updated_at  = NOW()
WHERE name = 'Taller de Grado'
  AND tenant_id = (SELECT tenant_id FROM seed_reviews_ctx);

-- ============================================================================
-- B1. Catálogo y materias del docente
-- ============================================================================
INSERT INTO subject_catalog (id, tenant_id, code, name)
VALUES
    ('7c100000-0000-4000-8000-000000000001', '00000000-0000-0000-0000-000000000001', 'INF-301', 'Estructuras de Datos'),
    ('7c100000-0000-4000-8000-000000000002', '00000000-0000-0000-0000-000000000001', 'INF-101', 'Programación I')
ON CONFLICT (tenant_id, upper(code)) DO UPDATE
SET name = EXCLUDED.name,
    updated_at = NOW();

INSERT INTO subjects (id, tenant_id, name, code, teacher_id, is_archived, created_at, updated_at, academic_period_id, subject_catalog_id, section)
VALUES
    ('7b100000-0000-4000-8000-000000000001', '00000000-0000-0000-0000-000000000001',
     'Estructuras de Datos', 'INF-301', (SELECT teacher_id FROM seed_reviews_ctx), false,
     NOW() - INTERVAL '38 days', NOW(), (SELECT period_id FROM seed_reviews_ctx),
     '7c100000-0000-4000-8000-000000000001', 'A'),
    ('7b100000-0000-4000-8000-000000000002', '00000000-0000-0000-0000-000000000001',
     'Programación I', 'INF-101', (SELECT teacher_id FROM seed_reviews_ctx), false,
     NOW() - INTERVAL '45 days', NOW(), (SELECT period_id FROM seed_reviews_ctx),
     '7c100000-0000-4000-8000-000000000002', 'B')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    code = EXCLUDED.code,
    teacher_id = EXCLUDED.teacher_id,
    is_archived = false,
    academic_period_id = EXCLUDED.academic_period_id,
    subject_catalog_id = EXCLUDED.subject_catalog_id,
    section = EXCLUDED.section,
    updated_at = NOW();

-- ============================================================================
-- B2. Alumnos de la cohorte y matrícula
-- ============================================================================
INSERT INTO users (id, email, first_name, last_name, name, role, tenant_id, origin, status, created_at)
VALUES
    ('8a000000-0000-4000-8000-000000000001', 'valeria.flores@uab.edu.bo',    'Valeria', 'Flores Mamani',      'Valeria Flores Mamani',      'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '40 days'),
    ('8a000000-0000-4000-8000-000000000002', 'rodrigo.quispe@uab.edu.bo',    'Rodrigo', 'Quispe Huanca',      'Rodrigo Quispe Huanca',      'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '40 days'),
    ('8a000000-0000-4000-8000-000000000003', 'camila.rojas@uab.edu.bo',      'Camila',  'Rojas Terrazas',     'Camila Rojas Terrazas',      'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '39 days'),
    ('8a000000-0000-4000-8000-000000000004', 'mateo.villarroel@uab.edu.bo',  'Mateo',   'Villarroel Paz',     'Mateo Villarroel Paz',       'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '39 days'),
    ('8a000000-0000-4000-8000-000000000005', 'lucia.choque@uab.edu.bo',      'Lucía',   'Choque Apaza',       'Lucía Choque Apaza',         'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '38 days'),
    ('8a000000-0000-4000-8000-000000000006', 'diego.salazar@uab.edu.bo',     'Diego',   'Salazar Céspedes',   'Diego Salazar Céspedes',     'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '38 days'),
    ('8a000000-0000-4000-8000-000000000007', 'andrea.gutierrez@uab.edu.bo',  'Andrea',  'Gutiérrez Loayza',   'Andrea Gutiérrez Loayza',    'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '37 days'),
    ('8a000000-0000-4000-8000-000000000008', 'fabian.ortiz@uab.edu.bo',      'Fabián',  'Ortiz Calderón',     'Fabián Ortiz Calderón',      'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '37 days'),
    ('8a000000-0000-4000-8000-000000000009', 'nicole.poma@uab.edu.bo',       'Nicole',  'Poma Carrasco',      'Nicole Poma Carrasco',       'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '36 days')
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    name = EXCLUDED.name,
    role = 'student',
    status = 'active',
    origin = 'manual';

INSERT INTO enrollments (tenant_id, student_id, subject_id, enrolled_at, source_system)
VALUES
    -- Estructuras de Datos (INF-301): cohorte completa
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000001', '7b100000-0000-4000-8000-000000000001', NOW() - INTERVAL '36 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000002', '7b100000-0000-4000-8000-000000000001', NOW() - INTERVAL '36 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000003', '7b100000-0000-4000-8000-000000000001', NOW() - INTERVAL '36 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000004', '7b100000-0000-4000-8000-000000000001', NOW() - INTERVAL '36 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000005', '7b100000-0000-4000-8000-000000000001', NOW() - INTERVAL '36 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000006', '7b100000-0000-4000-8000-000000000001', NOW() - INTERVAL '36 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000007', '7b100000-0000-4000-8000-000000000001', NOW() - INTERVAL '36 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000008', '7b100000-0000-4000-8000-000000000001', NOW() - INTERVAL '36 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000009', '7b100000-0000-4000-8000-000000000001', NOW() - INTERVAL '36 days', 'manual'),
    -- Programación I (INF-101): ocho matriculados
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000001', '7b100000-0000-4000-8000-000000000002', NOW() - INTERVAL '34 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000002', '7b100000-0000-4000-8000-000000000002', NOW() - INTERVAL '34 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000003', '7b100000-0000-4000-8000-000000000002', NOW() - INTERVAL '34 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000005', '7b100000-0000-4000-8000-000000000002', NOW() - INTERVAL '34 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000006', '7b100000-0000-4000-8000-000000000002', NOW() - INTERVAL '34 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000007', '7b100000-0000-4000-8000-000000000002', NOW() - INTERVAL '34 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000008', '7b100000-0000-4000-8000-000000000002', NOW() - INTERVAL '34 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000009', '7b100000-0000-4000-8000-000000000002', NOW() - INTERVAL '34 days', 'manual'),
    -- Taller de Grado: la cohorte también cursa la materia de proyecto
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000001', (SELECT id FROM subjects WHERE name = 'Taller de Grado' AND tenant_id = '00000000-0000-0000-0000-000000000001'), NOW() - INTERVAL '32 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000002', (SELECT id FROM subjects WHERE name = 'Taller de Grado' AND tenant_id = '00000000-0000-0000-0000-000000000001'), NOW() - INTERVAL '32 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000003', (SELECT id FROM subjects WHERE name = 'Taller de Grado' AND tenant_id = '00000000-0000-0000-0000-000000000001'), NOW() - INTERVAL '32 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000004', (SELECT id FROM subjects WHERE name = 'Taller de Grado' AND tenant_id = '00000000-0000-0000-0000-000000000001'), NOW() - INTERVAL '32 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000005', (SELECT id FROM subjects WHERE name = 'Taller de Grado' AND tenant_id = '00000000-0000-0000-0000-000000000001'), NOW() - INTERVAL '32 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000006', (SELECT id FROM subjects WHERE name = 'Taller de Grado' AND tenant_id = '00000000-0000-0000-0000-000000000001'), NOW() - INTERVAL '32 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000007', (SELECT id FROM subjects WHERE name = 'Taller de Grado' AND tenant_id = '00000000-0000-0000-0000-000000000001'), NOW() - INTERVAL '32 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000008', (SELECT id FROM subjects WHERE name = 'Taller de Grado' AND tenant_id = '00000000-0000-0000-0000-000000000001'), NOW() - INTERVAL '32 days', 'manual'),
    ('00000000-0000-0000-0000-000000000001', '8a000000-0000-4000-8000-000000000009', (SELECT id FROM subjects WHERE name = 'Taller de Grado' AND tenant_id = '00000000-0000-0000-0000-000000000001'), NOW() - INTERVAL '32 days', 'manual')
ON CONFLICT (tenant_id, student_id, subject_id) DO NOTHING;

-- ============================================================================
-- B3. Prácticas del Juez (JUEZ_EFIMERO)
-- ============================================================================
INSERT INTO exercises (
    id, tenant_id, title, description, type, config, subject_id, due_date,
    boilerplate, status, language, time_limit_ms, memory_limit_mb, db_config,
    created_at, comparator_spec, reference_solution, stale, difficulty, tags,
    purpose, per_student_seed, environment_type, template_id, expected_complexity
)
VALUES
(
    '9e000000-0000-4000-8000-000000000001', '00000000-0000-0000-0000-000000000001',
    'Validador de Expresiones Balanceadas',
    E'Implementá la función `esta_balanceada(expresion)` que recibe una cadena con paréntesis `()`, corchetes `[]` y llaves `{}` y devuelve `True` cuando cada símbolo de apertura se cierra en el orden correcto, o `False` en caso contrario.\n\nReglas:\n- Una cadena vacía está balanceada.\n- Los caracteres que no son de agrupación se ignoran.\n- La solución debe recorrer la expresión una sola vez usando una pila.\n\nEjemplos:\n`esta_balanceada("{[()]}")` → True\n`esta_balanceada("([)]")` → False',
    'algorithm',
    '{"algorithm": {"ast_rules": {"forbidden_imports": ["os", "subprocess", "socket"], "forbidden_functions": ["eval", "exec", "open"]}, "time_limit_ms": 1000, "memory_limit_mb": 128}}'::jsonb,
    '7b100000-0000-4000-8000-000000000001',
    NOW() + INTERVAL '4 days',
    E'def esta_balanceada(expresion: str) -> bool:\n    # Escribí tu solución usando una pila.\n    pass\n',
    'published', 'python', 1000, 128, '{}'::jsonb,
    NOW() - INTERVAL '21 days', '{"id": "exact"}'::jsonb,
    E'def esta_balanceada(expresion: str) -> bool:\n    pares = {")": "(", "]": "[", "}": "{"}\n    pila = []\n    for caracter in expresion:\n        if caracter in "([{":\n            pila.append(caracter)\n        elif caracter in pares:\n            if not pila or pila.pop() != pares[caracter]:\n                return False\n    return not pila\n',
    false, 'easy', ARRAY['pila', 'cadenas'], 'class', false, 'JUEZ_EFIMERO',
    (SELECT python_template_id FROM seed_reviews_ctx), 'O(N)'
),
(
    '9e000000-0000-4000-8000-000000000002', '00000000-0000-0000-0000-000000000001',
    'Búsqueda Binaria en Inventario Ordenado',
    E'El inventario de la tienda llega como una lista de precios ordenada de menor a mayor. Implementá `buscar_indice(precios, objetivo)` que devuelva la posición del precio buscado o `-1` si no existe.\n\nRequisitos:\n- La lista puede tener hasta 100 000 elementos.\n- No se permite `list.index()` ni recorrer la lista completa: la búsqueda debe ser O(log N).\n\nEjemplo:\n`buscar_indice([120, 250, 300, 480], 300)` → 2\n`buscar_indice([120, 250, 300, 480], 200)` → -1',
    'algorithm',
    '{"algorithm": {"ast_rules": {"forbidden_imports": ["os", "subprocess"], "forbidden_functions": ["eval", "exec"]}, "time_limit_ms": 1500, "memory_limit_mb": 128}}'::jsonb,
    '7b100000-0000-4000-8000-000000000001',
    NOW() + INTERVAL '2 days',
    E'def buscar_indice(precios: list[int], objetivo: int) -> int:\n    # Devolvé la posición del precio o -1 si no está en el inventario.\n    pass\n',
    'published', 'python', 1500, 128, '{}'::jsonb,
    NOW() - INTERVAL '14 days', '{"id": "exact"}'::jsonb,
    E'def buscar_indice(precios: list[int], objetivo: int) -> int:\n    inicio, fin = 0, len(precios) - 1\n    while inicio <= fin:\n        medio = (inicio + fin) // 2\n        if precios[medio] == objetivo:\n            return medio\n        if precios[medio] < objetivo:\n            inicio = medio + 1\n        else:\n            fin = medio - 1\n    return -1\n',
    false, 'medium', ARRAY['búsqueda', 'complejidad'], 'class', false, 'JUEZ_EFIMERO',
    (SELECT python_template_id FROM seed_reviews_ctx), 'O(log N)'
),
(
    '9e000000-0000-4000-8000-000000000003', '00000000-0000-0000-0000-000000000001',
    'Resumen de Ventas Diarias',
    E'El área comercial necesita un resumen diario. Implementá `resumen_ventas(ventas)` que reciba una lista con el monto vendido cada día y devuelva un diccionario con tres claves:\n\n- `total`: suma de todos los montos.\n- `maximo`: el monto más alto del período.\n- `dias`: cantidad de días con ventas mayores a cero.\n\nUna lista vacía debe devolver `{"total": 0, "maximo": 0, "dias": 0}`.\n\nEjemplo:\n`resumen_ventas([120, 80, 200])` → `{"total": 400, "maximo": 200, "dias": 3}`',
    'algorithm',
    '{"algorithm": {"ast_rules": {"forbidden_imports": ["os", "subprocess"], "forbidden_functions": ["eval", "exec"]}, "time_limit_ms": 1000, "memory_limit_mb": 128}}'::jsonb,
    '7b100000-0000-4000-8000-000000000002',
    NOW() + INTERVAL '6 days',
    E'def resumen_ventas(ventas: list[int]) -> dict:\n    # Devolvé total, maximo y dias.\n    pass\n',
    'published', 'python', 1000, 128, '{}'::jsonb,
    NOW() - INTERVAL '9 days', '{"id": "exact"}'::jsonb,
    E'def resumen_ventas(ventas: list[int]) -> dict:\n    return {\n        "total": sum(ventas),\n        "maximo": max(ventas, default=0),\n        "dias": sum(1 for venta in ventas if venta > 0),\n    }\n',
    false, 'easy', ARRAY['listas', 'diccionarios'], 'class', false, 'JUEZ_EFIMERO',
    (SELECT python_template_id FROM seed_reviews_ctx), 'O(N)'
)
ON CONFLICT (id) DO UPDATE
SET title = EXCLUDED.title,
    description = EXCLUDED.description,
    config = EXCLUDED.config,
    subject_id = EXCLUDED.subject_id,
    due_date = EXCLUDED.due_date,
    boilerplate = EXCLUDED.boilerplate,
    status = EXCLUDED.status,
    language = EXCLUDED.language,
    time_limit_ms = EXCLUDED.time_limit_ms,
    memory_limit_mb = EXCLUDED.memory_limit_mb,
    reference_solution = EXCLUDED.reference_solution,
    difficulty = EXCLUDED.difficulty,
    tags = EXCLUDED.tags,
    environment_type = EXCLUDED.environment_type,
    template_id = EXCLUDED.template_id,
    expected_complexity = EXCLUDED.expected_complexity;

-- Casos de prueba normalizados. La pantalla de revisión los desenmascara para
-- el docente y marca como fallidos los posteriores a la mitad cuando el
-- veredicto es WA, así que se ordenan de simple a complejo.
INSERT INTO exercise_test_cases (exercise_id, order_index, input, expected_output, visibility, weight)
VALUES
    ('9e000000-0000-4000-8000-000000000001', 0, 'expresion = "{[()]}"',        'True',  'public', 1.0),
    ('9e000000-0000-4000-8000-000000000001', 1, 'expresion = "([)]"',         'False', 'public', 1.0),
    ('9e000000-0000-4000-8000-000000000001', 2, 'expresion = "((a+b)*[c-d])"', 'True',  'hidden', 1.0),
    ('9e000000-0000-4000-8000-000000000001', 3, 'expresion = "}"',            'False', 'hidden', 1.0),
    ('9e000000-0000-4000-8000-000000000001', 4, 'expresion = ""',             'True',  'hidden', 1.0),

    ('9e000000-0000-4000-8000-000000000002', 0, E'precios = [120, 250, 300, 480, 510]\nobjetivo = 300', '2',  'public', 1.0),
    ('9e000000-0000-4000-8000-000000000002', 1, E'precios = [120, 250, 300, 480, 510]\nobjetivo = 200', '-1', 'public', 1.0),
    ('9e000000-0000-4000-8000-000000000002', 2, E'precios = [15]\nobjetivo = 15',                      '0',  'hidden', 1.0),
    ('9e000000-0000-4000-8000-000000000002', 3, E'precios = [15, 32, 48, 77, 90, 104, 233]\nobjetivo = 233', '6', 'hidden', 1.0),

    ('9e000000-0000-4000-8000-000000000003', 0, 'ventas = [120, 80, 200]', '{"total": 400, "maximo": 200, "dias": 3}', 'public', 1.0),
    ('9e000000-0000-4000-8000-000000000003', 1, 'ventas = [0, 0, 50]',     '{"total": 50, "maximo": 50, "dias": 1}',   'public', 1.0),
    ('9e000000-0000-4000-8000-000000000003', 2, 'ventas = []',             '{"total": 0, "maximo": 0, "dias": 0}',     'hidden', 1.0)
ON CONFLICT (exercise_id, order_index) DO UPDATE
SET input = EXCLUDED.input,
    expected_output = EXCLUDED.expected_output,
    visibility = EXCLUDED.visibility,
    weight = EXCLUDED.weight;

-- ============================================================================
-- B4. Práctica de proyecto (Entorno de Desarrollo) en Taller de Grado
-- El árbol de archivos de la revisión sale del snapshot de texto, así que no
-- hace falta publicar artefactos en el depósito para esta demostración.
-- ============================================================================
INSERT INTO exercises (
    id, tenant_id, title, description, type, config, subject_id, due_date,
    boilerplate, status, language, time_limit_ms, memory_limit_mb, db_config,
    created_at, comparator_spec, reference_solution, stale, difficulty, tags,
    purpose, per_student_seed, environment_type, template_id, expected_complexity
)
VALUES (
    '9e000000-0000-4000-8000-000000000004', '00000000-0000-0000-0000-000000000001',
    'Sistema de Reservas de Laboratorio — Entrega Final',
    E'Cerrá el taller entregando el gestor de reservas de laboratorio del proyecto base.\n\nLa entrega debe permitir:\n- Registrar una reserva con laboratorio, fecha, bloque horario y docente.\n- Rechazar cualquier bloque que se superponga con otra reserva del mismo laboratorio y fecha.\n- Cancelar una reserva existente y listar las reservas de un día ordenadas por hora de inicio.\n- Ejecutar `python -m pytest` sin fallos y `python main.py` sin errores.\n\nLa evaluación usa la rúbrica de la práctica: reglas de negocio, pruebas automatizadas y legibilidad del código.',
    'project',
    '{"project": {"language": "python", "entrypoint": "main.py", "test_command": "python -m pytest"}}'::jsonb,
    (SELECT id FROM subjects WHERE name = 'Taller de Grado' AND tenant_id = '00000000-0000-0000-0000-000000000001'),
    NOW() + INTERVAL '3 days',
    '',
    'published', 'python', 2000, 256, '{}'::jsonb,
    NOW() - INTERVAL '18 days', '{"id": "exact"}'::jsonb, '',
    false, 'medium', ARRAY['proyecto', 'reservas', 'poo'], 'class', false, 'IDE_PERSISTENTE',
    (SELECT python_project_template_id FROM seed_reviews_ctx), 'O(1)'
)
ON CONFLICT (id) DO UPDATE
SET title = EXCLUDED.title,
    description = EXCLUDED.description,
    config = EXCLUDED.config,
    subject_id = EXCLUDED.subject_id,
    due_date = EXCLUDED.due_date,
    status = EXCLUDED.status,
    language = EXCLUDED.language,
    difficulty = EXCLUDED.difficulty,
    tags = EXCLUDED.tags,
    environment_type = EXCLUDED.environment_type,
    template_id = EXCLUDED.template_id;

INSERT INTO workspace_projects (id, tenant_id, exercise_id, created_by, created_at, updated_at)
VALUES (
    'c2000000-0000-4000-8000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000004',
    (SELECT teacher_id FROM seed_reviews_ctx),
    NOW() - INTERVAL '18 days',
    NOW() - INTERVAL '18 days'
)
ON CONFLICT (id) DO NOTHING;

-- La versión publicada es inmutable: si ya existe se respeta tal como está.
WITH base AS (
    SELECT $snap$# === FILE: README.md ===
# Sistema de Reservas de Laboratorio

Proyecto base del taller. La API interna vive en `reservas/service.py`;
implementá las funciones marcadas con `TODO` y verificá el resultado con:

```bash
python -m pytest
python main.py
```

# === FILE: main.py ===
from reservas.service import GestorReservas


def main() -> None:
    gestor = GestorReservas()
    gestor.registrar("LAB-1", "2026-11-23", "08:00", "10:00", "Ana Quispe")
    for reserva in gestor.listar("LAB-1", "2026-11-23"):
        print(reserva)


if __name__ == "__main__":
    main()

# === FILE: reservas/__init__.py ===
"""Paquete del sistema de reservas de laboratorio."""

# === FILE: reservas/models.py ===
from dataclasses import dataclass


@dataclass
class Reserva:
    laboratorio: str
    fecha: str
    inicio: str
    fin: str
    docente: str

# === FILE: reservas/service.py ===
from reservas.models import Reserva


class GestorReservas:
    def __init__(self) -> None:
        self._reservas: list[Reserva] = []

    def registrar(self, laboratorio, fecha, inicio, fin, docente) -> Reserva:
        # TODO: rechazar el bloque si se superpone con otra reserva del mismo
        # laboratorio y fecha. Devolver la reserva creada.
        raise NotImplementedError

    def cancelar(self, laboratorio, fecha, inicio) -> bool:
        # TODO: quitar la reserva que empieza en ese bloque.
        raise NotImplementedError

    def listar(self, laboratorio, fecha) -> list[Reserva]:
        # TODO: devolver las reservas ordenadas por hora de inicio.
        raise NotImplementedError

# === FILE: tests/test_service.py ===
from reservas.service import GestorReservas


def test_registra_una_reserva():
    gestor = GestorReservas()
    gestor.registrar("LAB-1", "2026-11-23", "08:00", "10:00", "Ana Quispe")
    assert len(gestor.listar("LAB-1", "2026-11-23")) == 1
$snap$ AS snapshot
)
INSERT INTO workspace_project_versions (
    id, tenant_id, project_id, exercise_id, version_number, text_snapshot,
    manifest, content_sha256, file_count, total_bytes, created_by, published_at, created_at
)
SELECT
    'c3000000-0000-4000-8000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'c2000000-0000-4000-8000-000000000001',
    '9e000000-0000-4000-8000-000000000004',
    1,
    base.snapshot,
    '[]'::jsonb,
    encode(sha256(convert_to(base.snapshot, 'UTF8')), 'hex'),
    6,
    octet_length(base.snapshot),
    (SELECT teacher_id FROM seed_reviews_ctx),
    NOW() - INTERVAL '18 days',
    NOW() - INTERVAL '18 days'
FROM base
ON CONFLICT (id) DO NOTHING;

INSERT INTO rubrics (id, exercise_id, criteria, created_at, updated_at)
VALUES (
    'c4000000-0000-4000-8000-000000000001',
    '9e000000-0000-4000-8000-000000000004',
    '[
      {"id": "r1", "name": "Reglas de negocio", "description": "Superposición de bloques, cancelación y orden del listado.", "weight": 35,
       "levels": [
         {"name": "Insuficiente", "score": 20, "description": "No valida superposiciones ni cancela reservas."},
         {"name": "En desarrollo", "score": 55, "description": "Valida algunos casos y deja huecos en el resto."},
         {"name": "Satisfactorio", "score": 85, "description": "Cubre las reglas pedidas con detalles menores."},
         {"name": "Excelente", "score": 100, "description": "Cubre todas las reglas y los bordes horarios."}
       ]},
      {"id": "r2", "name": "Pruebas automatizadas", "description": "Cobertura de las reglas con pytest.", "weight": 25,
       "levels": [
         {"name": "Insuficiente", "score": 20, "description": "Sin pruebas o pruebas que no ejecutan."},
         {"name": "En desarrollo", "score": 55, "description": "Una prueba del camino feliz."},
         {"name": "Satisfactorio", "score": 85, "description": "Pruebas del camino feliz y de los rechazos."},
         {"name": "Excelente", "score": 100, "description": "Pruebas claras de casos límite."}
       ]},
      {"id": "r3", "name": "Modelo de datos", "description": "Uso de las entidades del proyecto base y tipos correctos.", "weight": 20,
       "levels": [
         {"name": "Insuficiente", "score": 20, "description": "Duplica o ignora las entidades del proyecto."},
         {"name": "En desarrollo", "score": 55, "description": "Usa las entidades con campos sueltos."},
         {"name": "Satisfactorio", "score": 85, "description": "Respeta las entidades y sus firmas."},
         {"name": "Excelente", "score": 100, "description": "Extiende el modelo sin romper el contrato."}
       ]},
      {"id": "r4", "name": "Legibilidad", "description": "Nombres, estructura y convenciones del proyecto.", "weight": 20,
       "levels": [
         {"name": "Insuficiente", "score": 20, "description": "Código difícil de seguir."},
         {"name": "En desarrollo", "score": 55, "description": "Funciona pero con nombres pobres."},
         {"name": "Satisfactorio", "score": 85, "description": "Legible y consistente."},
         {"name": "Excelente", "score": 100, "description": "Se lee como el resto del proyecto."}
       ]}
    ]'::jsonb,
    NOW() - INTERVAL '18 days',
    NOW() - INTERVAL '18 days'
)
ON CONFLICT (exercise_id) DO UPDATE
SET criteria = EXCLUDED.criteria,
    updated_at = NOW();

-- ============================================================================
-- B5. Entregas en espera de corrección
-- ============================================================================
INSERT INTO submissions (
    id, tenant_id, exercise_id, student_id, code, verdict, ast_result,
    execution_time_ms, memory_used_mb, manual_override, override_reason, score,
    submitted_at, review_status, feedback_summary, verdict_origin,
    complexity_analysis, generated_cases
)
VALUES
-- Práctica 1: Validador de Expresiones Balanceadas (5 entregas en espera)
(
    'a1000000-0000-4000-8000-000000000001', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000001', '8a000000-0000-4000-8000-000000000001',
    E'def esta_balanceada(expresion: str) -> bool:\n    pares = {")": "(", "]": "[", "}": "{"}\n    pila = []\n\n    for caracter in expresion:\n        if caracter in "([{":\n            pila.append(caracter)\n            continue\n        if caracter in pares:\n            if not pila:\n                return False\n            ultimo = pila.pop()\n            if ultimo != pares[caracter]:\n                return False\n\n    return len(pila) == 0\n',
    'AC', '{}'::jsonb, 4, 9, false, '', NULL,
    NOW() - INTERVAL '76 hours', 'pending', '', 'judge',
    '{"time_complexity": "O(N)", "space_complexity": "O(N)", "confidence": 0.92, "measurements": [{"input_size": 1000, "time_ms": 0.42, "memory_kb": 9216}, {"input_size": 5000, "time_ms": 1.98, "memory_kb": 10240}, {"input_size": 10000, "time_ms": 3.87, "memory_kb": 11264}, {"input_size": 20000, "time_ms": 7.64, "memory_kb": 13312}]}'::jsonb,
    NULL
),
(
    'a1000000-0000-4000-8000-000000000002', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000001', '8a000000-0000-4000-8000-000000000007',
    E'APERTURA = {"(": ")", "[": "]", "{": "}"}\nCIERRE = {cierre: apertura for apertura, cierre in APERTURA.items()}\n\n\ndef esta_balanceada(expresion: str) -> bool:\n    pendientes = []\n\n    for simbolo in expresion:\n        if simbolo in APERTURA:\n            pendientes.append(APERTURA[simbolo])\n        elif simbolo in CIERRE:\n            if not pendientes or pendientes.pop() != simbolo:\n                return False\n\n    return len(pendientes) == 0\n',
    'AC', '{}'::jsonb, 6, 9, false, '', NULL,
    NOW() - INTERVAL '19 hours', 'pending', '', 'judge',
    '{"time_complexity": "O(N)", "space_complexity": "O(N)", "confidence": 0.88, "measurements": [{"input_size": 1000, "time_ms": 0.51, "memory_kb": 9216}, {"input_size": 5000, "time_ms": 2.34, "memory_kb": 10240}, {"input_size": 10000, "time_ms": 4.62, "memory_kb": 11264}, {"input_size": 20000, "time_ms": 9.18, "memory_kb": 13312}]}'::jsonb,
    NULL
),
(
    'a1000000-0000-4000-8000-000000000003', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000001', '8a000000-0000-4000-8000-000000000006',
    E'def esta_balanceada(expresion: str) -> bool:\n    return (\n        expresion.count("(") == expresion.count(")")\n        and expresion.count("[") == expresion.count("]")\n        and expresion.count("{") == expresion.count("}")\n    )\n',
    'WA', '{}'::jsonb, 3, 8, false, '', NULL,
    NOW() - INTERVAL '8 hours', 'pending', '', 'judge',
    '{"time_complexity": "O(N)", "space_complexity": "O(1)", "confidence": 0.74, "measurements": [{"input_size": 1000, "time_ms": 0.28, "memory_kb": 8192}, {"input_size": 5000, "time_ms": 1.31, "memory_kb": 8192}, {"input_size": 10000, "time_ms": 2.66, "memory_kb": 8192}]}'::jsonb,
    NULL
),
(
    'a1000000-0000-4000-8000-000000000004', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000001', '8a000000-0000-4000-8000-000000000002',
    E'def esta_balanceada(expresion: str) -> bool:\n    if not expresion:\n        return True\n\n    pila = []\n    resto = expresion\n\n    while resto:\n        actual = resto[0]\n        resto = resto[1:]\n\n        if actual in "([{":\n            pila.append(actual)\n        elif actual in ")]}":\n            if not pila:\n                return False\n            apertura = pila.pop()\n            if (apertura, actual) not in [("(", ")"), ("[", "]"), ("{", "}")]:\n                return False\n\n    return not pila\n',
    'TLE', '{}'::jsonb, 1000, 11, false, '', NULL,
    NOW() - INTERVAL '4 hours', 'pending', '', 'judge',
    '{"time_complexity": "O(N\u00b2)", "space_complexity": "O(N)", "confidence": 0.81, "measurements": [{"input_size": 1000, "time_ms": 14.2, "memory_kb": 10240}, {"input_size": 5000, "time_ms": 268.4, "memory_kb": 15360}, {"input_size": 10000, "time_ms": 1041.7, "memory_kb": 23552}]}'::jsonb,
    NULL
),
(
    'a1000000-0000-4000-8000-000000000005', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000001', '8a000000-0000-4000-8000-000000000004',
    E'import subprocess\n\n\ndef esta_balanceada(expresion: str) -> bool:\n    salida = subprocess.check_output(["python3", "-c", "print(True)"])\n    return salida.decode().strip() == "True"\n',
    'AST_BLOCKED', '{"rule_id": "solv.security.forbidden-import", "detail": "El módulo subprocess está prohibido en el entorno de evaluación: permite ejecutar procesos externos."}'::jsonb,
    0, 0, false, '', NULL,
    NOW() - INTERVAL '95 minutes', 'pending', '', 'judge',
    NULL,
    NULL
),

-- Práctica 2: Búsqueda Binaria en Inventario Ordenado (3 entregas, una en revisión)
(
    'a1000000-0000-4000-8000-000000000006', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000002', '8a000000-0000-4000-8000-000000000001',
    E'def buscar_indice(precios: list[int], objetivo: int) -> int:\n    inicio = 0\n    fin = len(precios) - 1\n\n    while inicio <= fin:\n        medio = (inicio + fin) // 2\n        if precios[medio] == objetivo:\n            return medio\n        if precios[medio] < objetivo:\n            inicio = medio + 1\n        else:\n            fin = medio - 1\n\n    return -1\n',
    'AC', '{}'::jsonb, 3, 9, false, '', NULL,
    NOW() - INTERVAL '30 hours', 'pending', '', 'judge',
    '{"time_complexity": "O(log N)", "space_complexity": "O(1)", "confidence": 0.71, "measurements": [{"input_size": 1000, "time_ms": 0.02, "memory_kb": 8192}, {"input_size": 10000, "time_ms": 0.03, "memory_kb": 8192}, {"input_size": 100000, "time_ms": 0.04, "memory_kb": 8192}]}'::jsonb,
    NULL
),
(
    'a1000000-0000-4000-8000-000000000007', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000002', '8a000000-0000-4000-8000-000000000005',
    E'def buscar_indice(precios: list[int], objetivo: int) -> int:\n    inicio = 0\n    fin = len(precios)\n\n    while inicio <= fin:\n        medio = (inicio + fin) // 2\n        if precios[medio] == objetivo:\n            return medio\n        if precios[medio] < objetivo:\n            inicio = medio + 1\n        else:\n            fin = medio - 1\n\n    return -1\n',
    'RE', '{"message": "IndexError: list index out of range en la línea 6"}'::jsonb,
    2, 9, false, '', NULL,
    NOW() - INTERVAL '12 hours', 'pending', '', 'judge',
    '{"time_complexity": "O(log N)", "space_complexity": "O(1)", "confidence": 0.55, "measurements": [{"input_size": 1000, "time_ms": 0.02, "memory_kb": 8192}, {"input_size": 10000, "time_ms": 0.03, "memory_kb": 8192}]}'::jsonb,
    NULL
),
(
    'a1000000-0000-4000-8000-000000000008', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000002', '8a000000-0000-4000-8000-000000000009',
    E'def buscar_indice(precios: list[int], objetivo: int) -> int:\n    izquierda = 0\n    derecha = len(precios) - 1\n\n    while izquierda <= derecha:\n        centro = izquierda + (derecha - izquierda) // 2\n        if precios[centro] == objetivo:\n            return centro\n        if precios[centro] < objetivo:\n            izquierda = centro + 1\n        else:\n            derecha = centro - 1\n\n    return -1\n',
    'AC', '{}'::jsonb, 3, 9, false, '', NULL,
    NOW() - INTERVAL '5 hours', 'in_review', '', 'judge',
    '{"time_complexity": "O(log N)", "space_complexity": "O(1)", "confidence": 0.68, "measurements": [{"input_size": 1000, "time_ms": 0.02, "memory_kb": 8192}, {"input_size": 10000, "time_ms": 0.03, "memory_kb": 8192}, {"input_size": 100000, "time_ms": 0.05, "memory_kb": 8192}]}'::jsonb,
    NULL
),

-- Práctica 3: Resumen de Ventas Diarias (2 entregas)
(
    'a1000000-0000-4000-8000-000000000009', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000003', '8a000000-0000-4000-8000-000000000008',
    E'def resumen_ventas(ventas: list[int]) -> dict:\n    total = sum(ventas)\n    maximo = max(ventas, default=0)\n    dias = sum(1 for venta in ventas if venta > 0)\n\n    return {\n        "total": total,\n        "maximo": maximo,\n        "dias": dias,\n    }\n',
    'AC', '{}'::jsonb, 2, 8, false, '', NULL,
    NOW() - INTERVAL '18 hours', 'pending', '', 'judge',
    '{"time_complexity": "O(N)", "space_complexity": "O(1)", "confidence": 0.9, "measurements": [{"input_size": 1000, "time_ms": 0.19, "memory_kb": 8192}, {"input_size": 10000, "time_ms": 1.86, "memory_kb": 8192}, {"input_size": 50000, "time_ms": 9.42, "memory_kb": 8192}]}'::jsonb,
    NULL
),
(
    'a1000000-0000-4000-8000-000000000010', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000003', '8a000000-0000-4000-8000-000000000003',
    E'def resumen_ventas(ventas: list[int]) -> dict:\n    if len(ventas) == 0:\n        return {"total": 0, "maximo": 0, "dias": 0}\n\n    total = 0\n    maximo = ventas[0]\n\n    for venta in ventas:\n        total += venta\n        if venta > maximo:\n            maximo = venta\n\n    return {"total": total, "maximo": maximo}\n',
    'WA', '{}'::jsonb, 3, 8, false, '', NULL,
    NOW() - INTERVAL '3 hours', 'pending', '', 'judge',
    '{"time_complexity": "O(N)", "space_complexity": "O(1)", "confidence": 0.86, "measurements": [{"input_size": 1000, "time_ms": 0.21, "memory_kb": 8192}, {"input_size": 10000, "time_ms": 2.04, "memory_kb": 8192}, {"input_size": 50000, "time_ms": 10.31, "memory_kb": 8192}]}'::jsonb,
    NULL
)
ON CONFLICT (id) DO UPDATE
SET student_id = EXCLUDED.student_id,
    exercise_id = EXCLUDED.exercise_id,
    code = EXCLUDED.code,
    verdict = EXCLUDED.verdict,
    ast_result = EXCLUDED.ast_result,
    execution_time_ms = EXCLUDED.execution_time_ms,
    memory_used_mb = EXCLUDED.memory_used_mb,
    manual_override = EXCLUDED.manual_override,
    override_reason = EXCLUDED.override_reason,
    score = EXCLUDED.score,
    submitted_at = EXCLUDED.submitted_at,
    review_status = EXCLUDED.review_status,
    feedback_summary = EXCLUDED.feedback_summary,
    verdict_origin = EXCLUDED.verdict_origin,
    complexity_analysis = EXCLUDED.complexity_analysis,
    generated_cases = EXCLUDED.generated_cases;

-- Entregas de la práctica de proyecto: el snapshot de texto se expande en el
-- árbol de archivos del visor y se compara contra la versión base publicada.
INSERT INTO submissions (
    id, tenant_id, exercise_id, student_id, code, verdict, ast_result,
    execution_time_ms, memory_used_mb, manual_override, override_reason, score,
    submitted_at, review_status, feedback_summary, base_status,
    workspace_project_version_id
)
VALUES
(
    'a1000000-0000-4000-8000-000000000011', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000004', '8a000000-0000-4000-8000-000000000003',
    $snap$# === FILE: README.md ===
# Sistema de Reservas de Laboratorio

Gestor de reservas del taller. Las reglas de negocio están en
`reservas/service.py` y los tests en `tests/test_service.py`.

```bash
python -m pytest
python main.py
```

# === FILE: main.py ===
from reservas.service import GestorReservas


def main() -> None:
    gestor = GestorReservas()
    gestor.registrar("LAB-1", "2026-11-23", "08:00", "10:00", "Ana Quispe")
    gestor.registrar("LAB-1", "2026-11-23", "10:00", "12:00", "Luis Mamani")
    for reserva in gestor.listar("LAB-1", "2026-11-23"):
        print(reserva)


if __name__ == "__main__":
    main()

# === FILE: reservas/__init__.py ===
"""Paquete del sistema de reservas de laboratorio."""

# === FILE: reservas/models.py ===
from dataclasses import dataclass


@dataclass
class Reserva:
    laboratorio: str
    fecha: str
    inicio: str
    fin: str
    docente: str

# === FILE: reservas/service.py ===
from reservas.models import Reserva


def _minutos(hora: str) -> int:
    horas, minutos = hora.split(":")
    return int(horas) * 60 + int(minutos)


class GestorReservas:
    def __init__(self) -> None:
        self._reservas: list[Reserva] = []

    def _del_bloque(self, laboratorio: str, fecha: str) -> list[Reserva]:
        return [
            reserva
            for reserva in self._reservas
            if reserva.laboratorio == laboratorio and reserva.fecha == fecha
        ]

    def registrar(self, laboratorio, fecha, inicio, fin, docente) -> Reserva:
        if _minutos(fin) <= _minutos(inicio):
            raise ValueError("El bloque debe terminar después de empezar")
        for reserva in self._del_bloque(laboratorio, fecha):
            if _minutos(inicio) < _minutos(reserva.fin) and _minutos(reserva.inicio) < _minutos(fin):
                raise ValueError("El bloque se superpone con otra reserva")
        reserva = Reserva(laboratorio, fecha, inicio, fin, docente)
        self._reservas.append(reserva)
        return reserva

    def cancelar(self, laboratorio, fecha, inicio) -> bool:
        for reserva in self._del_bloque(laboratorio, fecha):
            if reserva.inicio == inicio:
                self._reservas.remove(reserva)
                return True
        return False

    def listar(self, laboratorio, fecha) -> list[Reserva]:
        return sorted(self._del_bloque(laboratorio, fecha), key=lambda r: _minutos(r.inicio))

# === FILE: tests/test_service.py ===
import pytest

from reservas.service import GestorReservas


def test_registra_una_reserva():
    gestor = GestorReservas()
    gestor.registrar("LAB-1", "2026-11-23", "08:00", "10:00", "Ana Quispe")
    assert len(gestor.listar("LAB-1", "2026-11-23")) == 1


def test_rechaza_bloques_superpuestos():
    gestor = GestorReservas()
    gestor.registrar("LAB-1", "2026-11-23", "08:00", "10:00", "Ana Quispe")
    with pytest.raises(ValueError):
        gestor.registrar("LAB-1", "2026-11-23", "09:00", "11:00", "Luis Mamani")


def test_permite_bloques_contiguos():
    gestor = GestorReservas()
    gestor.registrar("LAB-1", "2026-11-23", "08:00", "10:00", "Ana Quispe")
    gestor.registrar("LAB-1", "2026-11-23", "10:00", "12:00", "Luis Mamani")
    assert len(gestor.listar("LAB-1", "2026-11-23")) == 2


def test_cancela_una_reserva_existente():
    gestor = GestorReservas()
    gestor.registrar("LAB-1", "2026-11-23", "08:00", "10:00", "Ana Quispe")
    assert gestor.cancelar("LAB-1", "2026-11-23", "08:00") is True
    assert gestor.listar("LAB-1", "2026-11-23") == []
$snap$,
    NULL, '{}'::jsonb, 0, 0, false, '', NULL,
    NOW() - INTERVAL '22 hours', 'pending', '', 'version',
    'c3000000-0000-4000-8000-000000000001'
),
(
    'a1000000-0000-4000-8000-000000000012', '00000000-0000-0000-0000-000000000001',
    '9e000000-0000-4000-8000-000000000004', '8a000000-0000-4000-8000-000000000004',
    $snap$# === FILE: README.md ===
# Sistema de Reservas de Laboratorio

Entrega del taller.

# === FILE: main.py ===
from reservas.service import GestorReservas


def main() -> None:
    gestor = GestorReservas()
    gestor.registrar("LAB-1", "2026-11-23", "08:00", "10:00", "Ana Quispe")
    gestor.registrar("LAB-1", "2026-11-23", "09:00", "11:00", "Luis Mamani")
    for reserva in gestor.listar("LAB-1", "2026-11-23"):
        print(reserva)


if __name__ == "__main__":
    main()

# === FILE: reservas/__init__.py ===
"""Paquete del sistema de reservas de laboratorio."""

# === FILE: reservas/models.py ===
from dataclasses import dataclass


@dataclass
class Reserva:
    laboratorio: str
    fecha: str
    inicio: str
    fin: str
    docente: str

# === FILE: reservas/service.py ===
import re

from reservas.models import Reserva

PATRON_HORA = re.compile(r"^\d{2}:\d{2}$")


class GestorReservas:
    def __init__(self) -> None:
        self.reservas = []

    def registrar(self, laboratorio, fecha, inicio, fin, docente) -> Reserva:
        if not PATRON_HORA.match(inicio) or not PATRON_HORA.match(fin):
            raise ValueError("Formato de hora inválido")
        reserva = Reserva(laboratorio, fecha, inicio, fin, docente)
        self.reservas.append(reserva)
        return reserva

    def cancelar(self, laboratorio, fecha, inicio) -> bool:
        for reserva in self.reservas:
            if reserva.laboratorio == laboratorio and reserva.fecha == fecha and reserva.inicio == inicio:
                self.reservas.remove(reserva)
                return True
        return False

    def listar(self, laboratorio, fecha) -> list[Reserva]:
        return [
            reserva
            for reserva in self.reservas
            if reserva.laboratorio == laboratorio and reserva.fecha == fecha
        ]

# === FILE: tests/test_service.py ===
from reservas.service import GestorReservas


def test_registra_una_reserva():
    gestor = GestorReservas()
    gestor.registrar("LAB-1", "2026-11-23", "08:00", "10:00", "Ana Quispe")
    assert len(gestor.listar("LAB-1", "2026-11-23")) == 1
$snap$,
    NULL, '{}'::jsonb, 0, 0, false, '', NULL,
    NOW() - INTERVAL '6 hours', 'pending', '', 'version',
    'c3000000-0000-4000-8000-000000000001'
)
ON CONFLICT (id) DO UPDATE
SET student_id = EXCLUDED.student_id,
    exercise_id = EXCLUDED.exercise_id,
    code = EXCLUDED.code,
    verdict = EXCLUDED.verdict,
    submitted_at = EXCLUDED.submitted_at,
    review_status = EXCLUDED.review_status,
    base_status = EXCLUDED.base_status,
    workspace_project_version_id = EXCLUDED.workspace_project_version_id;

-- ============================================================================
-- B6. Auditoría de escritura (Time-Travel Replay) de dos entregas
-- Se reconstruye la transcripción carácter por carácter del código final,
-- con la cadencia de una persona escribiendo.
-- ============================================================================
DELETE FROM submission_keystroke_events
WHERE submission_id IN (
    'a1000000-0000-4000-8000-000000000001',
    'a1000000-0000-4000-8000-000000000002',
    'a1000000-0000-4000-8000-000000000003'
);

INSERT INTO submission_keystroke_events (submission_id, timestamp_ms, event_type, position, content, paste_source_detected)
SELECT s.id,
       4200 + (g.i * 200),
       'insert',
       g.i,
       substring(s.code FROM g.i + 1 FOR 1),
       false
FROM submissions s
CROSS JOIN LATERAL generate_series(0, length(s.code) - 1) AS g(i)
WHERE s.id IN (
    'a1000000-0000-4000-8000-000000000001',
    'a1000000-0000-4000-8000-000000000002',
    'a1000000-0000-4000-8000-000000000003'
);

COMMIT;

-- ============================================================================
-- Verificación rápida (ejecutar aparte):
--   SELECT s.name AS materia, ex.title AS practica, COUNT(sub.id) AS pendientes,
--          MIN(sub.submitted_at) AS espera_desde
--   FROM submissions sub
--   JOIN exercises ex ON ex.id = sub.exercise_id
--   JOIN subjects s ON s.id = ex.subject_id
--   WHERE sub.review_status IN ('pending', 'in_review')
--   GROUP BY s.name, ex.title
--   ORDER BY MIN(sub.submitted_at);
-- ============================================================================
