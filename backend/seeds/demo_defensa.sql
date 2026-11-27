-- ============================================================================
-- SOLV — Paquete de Demo Reproducible para Defensa (UAB)
-- ============================================================================
-- Idempotente y re-ejecutable (< 5 s sobre PostgreSQL).
-- Conserva: tenant UAB, template_categories, template_components,
--           template_models, lab_templates aprobadas y workspace_projects /
--           workspace_project_versions con artefactos en SeaweedFS.
-- Limpia:   usuarios de carga (student-test-*), materia CS-LOAD-2026, GEN-101,
--           plantilla rechazada, entornos/sesiones previas y datos irreproducibles.
-- ============================================================================

BEGIN;

-- ============================================================================
-- 0. Identidades canónicas UAB (Admin, Docente, Estudiante) sin duplicados .dev
-- ============================================================================
INSERT INTO tenants (id, name, slug, allowed_domains)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Universidad Adventista de Bolivia',
    'uab',
    '["@uab.edu.bo"]'::jsonb
)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    allowed_domains = EXCLUDED.allowed_domains;

DO $$
DECLARE
    v_roberto_id UUID;
    v_admin_old_id UUID := '00000000-0000-0000-0000-000000000001';
    v_carlos_id UUID;
    v_docente_dev_id UUID;
    v_valeria_id UUID := '8a000000-0000-4000-8000-000000000001';
    v_est_dev_id UUID;
BEGIN
    -- Admin: Ing. Roberto Paredes (roberto.paredes@uab.edu.bo) sobre UUID canonico
    SELECT id INTO v_roberto_id FROM users WHERE LOWER(email) = 'roberto.paredes@uab.edu.bo' LIMIT 1;
    IF v_roberto_id IS NULL THEN
        UPDATE users
        SET email = 'roberto.paredes@uab.edu.bo',
            first_name = 'Roberto',
            last_name = 'Paredes',
            name = 'Ing. Roberto Paredes',
            role = 'admin',
            origin = 'manual',
            status = 'active'
        WHERE id = v_admin_old_id;

        IF NOT FOUND THEN
            INSERT INTO users (id, email, first_name, last_name, name, role, tenant_id, origin, status)
            VALUES (v_admin_old_id, 'roberto.paredes@uab.edu.bo', 'Roberto', 'Paredes', 'Ing. Roberto Paredes', 'admin', '00000000-0000-0000-0000-000000000001', 'manual', 'active');
        END IF;
    ELSE
        UPDATE users
        SET first_name = 'Roberto',
            last_name = 'Paredes',
            name = 'Ing. Roberto Paredes',
            role = 'admin',
            origin = 'manual',
            status = 'active'
        WHERE id = v_roberto_id;
    END IF;

    -- Docente: Ing. Carlos Mendoza Rios (carlos.mendoza@uab.edu.bo)
    SELECT id INTO v_carlos_id FROM users WHERE LOWER(email) = 'carlos.mendoza@uab.edu.bo' LIMIT 1;
    SELECT id INTO v_docente_dev_id FROM users WHERE LOWER(email) = 'docente.dev@uab.edu.bo' LIMIT 1;

    IF v_carlos_id IS NULL AND v_docente_dev_id IS NOT NULL THEN
        UPDATE users
        SET email = 'carlos.mendoza@uab.edu.bo',
            first_name = 'Carlos',
            last_name = 'Mendoza Ríos',
            name = 'Ing. Carlos Mendoza Ríos',
            role = 'teacher',
            origin = 'manual',
            status = 'active'
        WHERE id = v_docente_dev_id;
        v_carlos_id := v_docente_dev_id;
    ELSIF v_carlos_id IS NOT NULL AND v_docente_dev_id IS NOT NULL AND v_carlos_id <> v_docente_dev_id THEN
        UPDATE subjects SET teacher_id = v_carlos_id WHERE teacher_id = v_docente_dev_id;
        UPDATE workspace_projects SET created_by = v_carlos_id WHERE created_by = v_docente_dev_id;
        UPDATE submission_comments SET author_id = v_carlos_id WHERE author_id = v_docente_dev_id;
        UPDATE class_sessions SET created_by = v_carlos_id WHERE created_by = v_docente_dev_id;
        UPDATE class_advances SET created_by = v_carlos_id WHERE created_by = v_docente_dev_id;
        UPDATE submissions SET graded_by = v_carlos_id WHERE graded_by = v_docente_dev_id;
        UPDATE submissions SET reviewed_by = v_carlos_id WHERE reviewed_by = v_docente_dev_id;
        UPDATE lab_templates SET reviewed_by = v_carlos_id WHERE reviewed_by = v_docente_dev_id;
        UPDATE lab_templates SET requested_by = v_carlos_id WHERE requested_by = v_docente_dev_id;
        UPDATE users SET status = 'inactive' WHERE id = v_docente_dev_id;
    ELSIF v_carlos_id IS NULL THEN
        INSERT INTO users (id, email, first_name, last_name, name, role, tenant_id, origin, status)
        VALUES ('18c1f66c-c2bd-461b-baa9-12f40f3b9411', 'carlos.mendoza@uab.edu.bo', 'Carlos', 'Mendoza Ríos', 'Ing. Carlos Mendoza Ríos', 'teacher', '00000000-0000-0000-0000-000000000001', 'manual', 'active')
        RETURNING id INTO v_carlos_id;
    END IF;

    UPDATE users
    SET first_name = 'Carlos',
        last_name = 'Mendoza Ríos',
        name = 'Ing. Carlos Mendoza Ríos',
        role = 'teacher',
        origin = 'manual',
        status = 'active'
    WHERE id = v_carlos_id;

    -- Estudiante principal: Valeria Flores Mamani (valeria.flores@uab.edu.bo)
    INSERT INTO users (id, email, first_name, last_name, name, role, tenant_id, origin, status, created_at)
    VALUES (
        v_valeria_id,
        'valeria.flores@uab.edu.bo',
        'Valeria',
        'Flores Mamani',
        'Valeria Flores Mamani',
        'student',
        '00000000-0000-0000-0000-000000000001',
        'manual',
        'active',
        NOW() - INTERVAL '40 days'
    )
    ON CONFLICT (email) DO UPDATE
    SET first_name = EXCLUDED.first_name,
        last_name = EXCLUDED.last_name,
        name = EXCLUDED.name,
        role = 'student',
        origin = 'manual',
        status = 'active';

    SELECT id INTO v_est_dev_id FROM users WHERE LOWER(email) = 'estudiante.dev@uab.edu.bo' LIMIT 1;
    IF v_est_dev_id IS NOT NULL THEN
        DELETE FROM environments WHERE student_id = v_est_dev_id;
        DELETE FROM submissions WHERE student_id = v_est_dev_id;
        DELETE FROM enrollments WHERE student_id = v_est_dev_id;
        DELETE FROM users WHERE id = v_est_dev_id;
    END IF;
END $$;

-- Cohorte estudiantil UAB realista
INSERT INTO users (id, email, first_name, last_name, name, role, tenant_id, origin, status, created_at)
VALUES
    ('8a000000-0000-4000-8000-000000000002', 'rodrigo.quispe@uab.edu.bo',   'Rodrigo', 'Quispe Huanca',    'Rodrigo Quispe Huanca',    'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '40 days'),
    ('8a000000-0000-4000-8000-000000000003', 'camila.rojas@uab.edu.bo',     'Camila',  'Rojas Terrazas',   'Camila Rojas Terrazas',    'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '39 days'),
    ('8a000000-0000-4000-8000-000000000004', 'mateo.villarroel@uab.edu.bo', 'Mateo',   'Villarroel Paz',   'Mateo Villarroel Paz',     'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '39 days'),
    ('8a000000-0000-4000-8000-000000000005', 'lucia.choque@uab.edu.bo',     'Lucía',   'Choque Apaza',     'Lucía Choque Apaza',       'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '38 days'),
    ('8a000000-0000-4000-8000-000000000006', 'diego.salazar@uab.edu.bo',    'Diego',   'Salazar Céspedes', 'Diego Salazar Céspedes',   'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '38 days'),
    ('8a000000-0000-4000-8000-000000000007', 'andrea.gutierrez@uab.edu.bo', 'Andrea',  'Gutiérrez Loayza', 'Andrea Gutiérrez Loayza',  'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active', NOW() - INTERVAL '37 days')
ON CONFLICT (email) DO UPDATE
SET first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    name = EXCLUDED.name,
    role = 'student',
    origin = 'manual',
    status = 'active';

-- ============================================================================
-- 1. Limpieza de residuos de carga, huérfanos y plantillas rechazadas
-- ============================================================================
DELETE FROM review_test_environments;
DELETE FROM environments;
DELETE FROM class_sessions;
DELETE FROM dependency_preparation_jobs;
DELETE FROM dry_run_jobs;
DELETE FROM environment_requests;
DELETE FROM run_metrics;
DELETE FROM evaluation_generated_cases;
DELETE FROM deadline_reminders_sent;
DELETE FROM notifications;
DELETE FROM submission_keystroke_events;

DELETE FROM submissions
WHERE student_id IN (
    SELECT id FROM users
    WHERE email LIKE 'student-test-%@test.com'
       OR email LIKE 'campaign-%'
       OR email LIKE 'loadtest%'
       OR email IN ('admin.dev@uab.edu.bo', 'docente.dev@uab.edu.bo', 'estudiante.dev@uab.edu.bo')
)
OR exercise_id IN (
    SELECT id FROM exercises
    WHERE subject_id IN (SELECT id FROM subjects WHERE code IN ('CS-LOAD-2026', 'GEN-101'))
       OR subject_id IS NULL
)
OR exercise_id = '9e000000-0000-4000-8000-000000000099';

DELETE FROM exercises
WHERE subject_id IN (SELECT id FROM subjects WHERE code IN ('CS-LOAD-2026', 'GEN-101'))
   OR subject_id IS NULL;

-- Ejercicios experimentales previos con versiones inmutables pasan a borrador
-- para no romper FKs ni ensuciar el portal del estudiante en vivo.
UPDATE exercises
SET status = 'draft'
WHERE id IN (
    'fe1baa8e-e45c-4610-8239-57d69e0e9430',
    'ef9664b0-2976-4ec1-affa-4776b66833fd',
    '10bf75b9-251f-4f3a-a18a-d517b4e43c43',
    '1c5361d6-634c-4cbf-b4b4-9adbf6be6c1b',
    '9e000000-0000-4000-8000-000000000099',
    'a0000000-0000-4000-8000-000000000001',
    'b0000000-0000-4000-8000-000000000002'
);

DELETE FROM enrollments
WHERE subject_id IN (SELECT id FROM subjects WHERE code IN ('CS-LOAD-2026', 'GEN-101'))
   OR student_id IN (
       SELECT id FROM users
       WHERE email LIKE 'student-test-%@test.com'
          OR email LIKE 'campaign-%'
          OR email LIKE 'loadtest%'
   );

DELETE FROM subjects WHERE code IN ('CS-LOAD-2026', 'GEN-101');

DELETE FROM users
WHERE email LIKE 'student-test-%@test.com'
   OR email LIKE 'campaign-%'
   OR email LIKE 'loadtest%'
   OR email IN ('admin.dev@uab.edu.bo', 'estudiante.dev@uab.edu.bo');

DELETE FROM lab_templates WHERE status = 'RECHAZADA' OR name = 'prueba';

-- ============================================================================
-- 2. Período académico vigente 2026-II y 3 materias activas del docente
-- ============================================================================
UPDATE academic_periods
SET is_active = false,
    is_archived = true
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'
  AND code <> '2026-II';

INSERT INTO academic_periods (id, tenant_id, name, code, start_date, end_date, is_active, is_archived)
VALUES (
    '22222222-2222-4222-8222-222222222222',
    '00000000-0000-0000-0000-000000000001',
    'Gestión 2026 - Semestre II',
    '2026-II',
    '2026-07-01',
    '2026-12-31',
    true,
    false
)
ON CONFLICT (tenant_id, code) DO UPDATE
SET name = EXCLUDED.name,
    start_date = EXCLUDED.start_date,
    end_date = EXCLUDED.end_date,
    is_active = true,
    is_archived = false,
    archived_at = NULL,
    archived_by = NULL;

DROP TABLE IF EXISTS demo_ctx;
CREATE TEMP TABLE demo_ctx AS
SELECT
    (SELECT id FROM users WHERE LOWER(email) = 'carlos.mendoza@uab.edu.bo' LIMIT 1) AS teacher_id,
    (SELECT id FROM academic_periods WHERE tenant_id = '00000000-0000-0000-0000-000000000001' AND code = '2026-II' LIMIT 1) AS period_id,
    (SELECT id FROM lab_templates WHERE name = 'Python Runner Base' AND status IN ('APROBADA', 'approved') LIMIT 1) AS python_judge_tpl_id,
    (SELECT id FROM lab_templates WHERE name = 'Python 3.12 — Proyecto CLI' AND status IN ('APROBADA', 'approved') LIMIT 1) AS python_ide_tpl_id,
    (SELECT id FROM lab_templates WHERE id = 'fc594f84-8180-42ab-8417-bf265bde0211' LIMIT 1) AS cursos_ide_tpl_id,
    (SELECT id FROM lab_templates WHERE id = '00fa60ed-225f-4acd-9303-b3375926c37d' LIMIT 1) AS node_pg_tpl_id;

INSERT INTO subject_catalog (id, tenant_id, code, name)
VALUES
    ('7c100000-0000-4000-8000-000000000001', '00000000-0000-0000-0000-000000000001', 'INF-301', 'Estructuras de Datos'),
    ('7c100000-0000-4000-8000-000000000002', '00000000-0000-0000-0000-000000000001', 'INF-101', 'Programación I'),
    ('7c100000-0000-4000-8000-000000000003', '00000000-0000-0000-0000-000000000001', 'SIS-12',  'Taller de Aplicaciones en Internet')
ON CONFLICT (tenant_id, upper(code)) DO UPDATE
SET name = EXCLUDED.name,
    updated_at = NOW();

INSERT INTO subjects (id, tenant_id, name, code, teacher_id, is_archived, created_at, updated_at, academic_period_id, subject_catalog_id, section)
VALUES
    ('7b100000-0000-4000-8000-000000000002', '00000000-0000-0000-0000-000000000001',
     'Programación I', 'INF-101', (SELECT teacher_id FROM demo_ctx), false,
     NOW() - INTERVAL '45 days', NOW(), (SELECT period_id FROM demo_ctx),
     (SELECT id FROM subject_catalog WHERE tenant_id = '00000000-0000-0000-0000-000000000001' AND upper(code) = 'INF-101' LIMIT 1), 'B'),
    ('7b100000-0000-4000-8000-000000000001', '00000000-0000-0000-0000-000000000001',
     'Estructuras de Datos', 'INF-301', (SELECT teacher_id FROM demo_ctx), false,
     NOW() - INTERVAL '38 days', NOW(), (SELECT period_id FROM demo_ctx),
     (SELECT id FROM subject_catalog WHERE tenant_id = '00000000-0000-0000-0000-000000000001' AND upper(code) = 'INF-301' LIMIT 1), 'A'),
    ('6879c008-375e-4849-a341-6517463ab81e', '00000000-0000-0000-0000-000000000001',
     'Taller de Aplicaciones en Internet', 'SIS-12', (SELECT teacher_id FROM demo_ctx), false,
     NOW() - INTERVAL '30 days', NOW(), (SELECT period_id FROM demo_ctx),
     (SELECT id FROM subject_catalog WHERE tenant_id = '00000000-0000-0000-0000-000000000001' AND upper(code) = 'SIS-12' LIMIT 1), 'A')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    code = EXCLUDED.code,
    teacher_id = EXCLUDED.teacher_id,
    is_archived = false,
    academic_period_id = EXCLUDED.academic_period_id,
    subject_catalog_id = EXCLUDED.subject_catalog_id,
    section = EXCLUDED.section,
    updated_at = NOW();

INSERT INTO enrollments (tenant_id, student_id, subject_id, enrolled_at, source_system)
SELECT
    '00000000-0000-0000-0000-000000000001',
    u.id,
    s.id,
    NOW() - INTERVAL '30 days',
    'manual'
FROM users u
CROSS JOIN subjects s
WHERE u.email IN (
    'valeria.flores@uab.edu.bo',
    'rodrigo.quispe@uab.edu.bo',
    'camila.rojas@uab.edu.bo',
    'mateo.villarroel@uab.edu.bo',
    'lucia.choque@uab.edu.bo',
    'diego.salazar@uab.edu.bo',
    'andrea.gutierrez@uab.edu.bo'
)
  AND s.code IN ('INF-101', 'INF-301', 'SIS-12')
ON CONFLICT (tenant_id, student_id, subject_id) DO NOTHING;

-- ============================================================================
-- 3. Flujo 1: Juez Virtual en INF-301 e INF-101 (AC y AST_BLOCKED en vivo)
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
    E'Implementá `esta_balanceada(expresion)` leyendo una línea desde entrada estándar (`stdin`) e imprimiendo `True` si los paréntesis `()`, corchetes `[]` y llaves `{}` están balanceados, o `False` en caso contrario.\n\nReglas:\n- Una entrada vacía está balanceada (`True`).\n- Los caracteres que no son de agrupación se ignoran.\n- No se permite importar `os` ni `subprocess`.\n\nEjemplos:\n- Entrada `{[()]}` -> Salida `True`\n- Entrada `([)]` -> Salida `False`',
    'algorithm',
    '{"algorithm": {"ast_rules": {"forbidden_imports": ["socket"], "forbidden_functions": ["open"]}, "time_limit_ms": 1500, "memory_limit_mb": 128}}'::jsonb,
    '7b100000-0000-4000-8000-000000000001',
    NOW() + INTERVAL '5 days',
    E'import sys\n\n\ndef esta_balanceada(expresion: str) -> bool:\n    # Escribi tu solucion usando una pila\n    return False\n\n\nif __name__ == "__main__":\n    entrada = sys.stdin.read().rstrip("\\r\\n")\n    print(esta_balanceada(entrada))\n',
    'published', 'python', 1500, 128, '{}'::jsonb,
    NOW() - INTERVAL '21 days', '{"id": "exact"}'::jsonb,
    E'import sys\n\n\ndef esta_balanceada(expresion: str) -> bool:\n    pares = {")": "(", "]": "[", "}": "{"}\n    pila = []\n    for c in expresion:\n        if c in "([{":\n            pila.append(c)\n        elif c in pares:\n            if not pila or pila.pop() != pares[c]:\n                return False\n    return not pila\n\n\nif __name__ == "__main__":\n    entrada = sys.stdin.read().rstrip("\\r\\n")\n    print(esta_balanceada(entrada))\n',
    false, 'easy', ARRAY['pila', 'cadenas'], 'class', false, 'JUEZ_EFIMERO',
    (SELECT python_judge_tpl_id FROM demo_ctx), 'O(N)'
),
(
    '9e000000-0000-4000-8000-000000000002', '00000000-0000-0000-0000-000000000001',
    'Búsqueda Binaria en Inventario Ordenado',
    E'Lee en la primera línea los precios separados por espacio y en la segunda línea el precio objetivo. Imprime el índice del precio o `-1` si no se encuentra.',
    'algorithm',
    '{"algorithm": {"ast_rules": {"forbidden_imports": ["os", "subprocess"], "forbidden_functions": ["eval", "exec"]}, "time_limit_ms": 1500, "memory_limit_mb": 128}}'::jsonb,
    '7b100000-0000-4000-8000-000000000001',
    NOW() + INTERVAL '4 days',
    E'import sys\n\n\ndef buscar_indice(precios: list[int], objetivo: int) -> int:\n    return -1\n\n\nif __name__ == "__main__":\n    lineas = sys.stdin.read().splitlines()\n    precios = list(map(int, lineas[0].split())) if lineas and lineas[0].strip() else []\n    objetivo = int(lineas[1].strip()) if len(lineas) > 1 else 0\n    print(buscar_indice(precios, objetivo))\n',
    'published', 'python', 1500, 128, '{}'::jsonb,
    NOW() - INTERVAL '14 days', '{"id": "exact"}'::jsonb,
    E'import sys\n\n\ndef buscar_indice(precios: list[int], objetivo: int) -> int:\n    inicio, fin = 0, len(precios) - 1\n    while inicio <= fin:\n        medio = (inicio + fin) // 2\n        if precios[medio] == objetivo:\n            return medio\n        if precios[medio] < objetivo:\n            inicio = medio + 1\n        else:\n            fin = medio - 1\n    return -1\n\n\nif __name__ == "__main__":\n    lineas = sys.stdin.read().splitlines()\n    precios = list(map(int, lineas[0].split())) if lineas and lineas[0].strip() else []\n    objetivo = int(lineas[1].strip()) if len(lineas) > 1 else 0\n    print(buscar_indice(precios, objetivo))\n',
    false, 'medium', ARRAY['búsqueda', 'complejidad'], 'class', false, 'JUEZ_EFIMERO',
    (SELECT python_judge_tpl_id FROM demo_ctx), 'O(log N)'
),
(
    '9e000000-0000-4000-8000-000000000003', '00000000-0000-0000-0000-000000000001',
    'Resumen de Ventas Diarias',
    E'Lee una línea con montos enteros separados por espacio e imprime `total maximo dias` (tres enteros separados por espacio). Si la línea está vacía imprime `0 0 0`.',
    'algorithm',
    '{"algorithm": {"ast_rules": {"forbidden_imports": ["os", "subprocess"], "forbidden_functions": ["eval", "exec"]}, "time_limit_ms": 1500, "memory_limit_mb": 128}}'::jsonb,
    '7b100000-0000-4000-8000-000000000002',
    NOW() + INTERVAL '6 days',
    E'import sys\n\n\ndef resumen_ventas(ventas: list[int]) -> tuple[int, int, int]:\n    return (0, 0, 0)\n\n\nif __name__ == "__main__":\n    datos = list(map(int, sys.stdin.read().split()))\n    total, maximo, dias = resumen_ventas(datos)\n    print(f"{total} {maximo} {dias}")\n',
    'published', 'python', 1500, 128, '{}'::jsonb,
    NOW() - INTERVAL '9 days', '{"id": "exact"}'::jsonb,
    E'import sys\n\n\ndef resumen_ventas(ventas: list[int]) -> tuple[int, int, int]:\n    return (sum(ventas), max(ventas, default=0), sum(1 for v in ventas if v > 0))\n\n\nif __name__ == "__main__":\n    datos = list(map(int, sys.stdin.read().split()))\n    total, maximo, dias = resumen_ventas(datos)\n    print(f"{total} {maximo} {dias}")\n',
    false, 'easy', ARRAY['listas', 'fundamentos'], 'class', false, 'JUEZ_EFIMERO',
    (SELECT python_judge_tpl_id FROM demo_ctx), 'O(N)'
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
    purpose = EXCLUDED.purpose,
    per_student_seed = EXCLUDED.per_student_seed,
    environment_type = EXCLUDED.environment_type,
    template_id = EXCLUDED.template_id,
    expected_complexity = EXCLUDED.expected_complexity;

INSERT INTO exercise_test_cases (exercise_id, order_index, input, expected_output, visibility, weight)
VALUES
    ('9e000000-0000-4000-8000-000000000001', 0, '{[()]}',        'True',  'public', 1.0),
    ('9e000000-0000-4000-8000-000000000001', 1, '([)]',          'False', 'public', 1.0),
    ('9e000000-0000-4000-8000-000000000001', 2, '((a+b)*[c-d])', 'True',  'hidden', 1.0),
    ('9e000000-0000-4000-8000-000000000001', 3, '}',             'False', 'hidden', 1.0),
    ('9e000000-0000-4000-8000-000000000001', 4, '',              'True',  'hidden', 1.0),

    ('9e000000-0000-4000-8000-000000000002', 0, E'120 250 300 480 510\n300', '2',  'public', 1.0),
    ('9e000000-0000-4000-8000-000000000002', 1, E'120 250 300 480 510\n200', '-1', 'public', 1.0),
    ('9e000000-0000-4000-8000-000000000002', 2, E'15\n15',                   '0',  'hidden', 1.0),
    ('9e000000-0000-4000-8000-000000000002', 3, E'15 32 48 77 90 104 233\n233', '6', 'hidden', 1.0),

    ('9e000000-0000-4000-8000-000000000003', 0, '120 80 200', '400 200 3', 'public', 1.0),
    ('9e000000-0000-4000-8000-000000000003', 1, '0 0 50',     '50 50 1',   'public', 1.0),
    ('9e000000-0000-4000-8000-000000000003', 2, '',           '0 0 0',     'hidden', 1.0)
ON CONFLICT (exercise_id, order_index) DO UPDATE
SET input = EXCLUDED.input,
    expected_output = EXCLUDED.expected_output,
    visibility = EXCLUDED.visibility,
    weight = EXCLUDED.weight;

-- ============================================================================
-- 4. Flujo 2: IDE Persistente + BD gestionada + SOLV Companion (SIS-12)
-- ============================================================================
UPDATE exercises
SET subject_id = '6879c008-375e-4849-a341-6517463ab81e',
    status = 'published'
WHERE id = '08d80113-cc02-44cb-91fa-08e671453675';

-- ============================================================================
-- 5. Flujo 3: Exámenes (Juez con per_student_seed + IDE con exam_network=none)
-- ============================================================================
INSERT INTO exercises (
    id, tenant_id, title, description, type, config, subject_id, due_date,
    boilerplate, status, language, time_limit_ms, memory_limit_mb, db_config,
    created_at, comparator_spec, reference_solution, stale, difficulty, tags,
    purpose, per_student_seed, environment_type, template_id, expected_complexity
)
VALUES (
    '9e000000-0000-4000-8000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    'Examen Parcial: Suma de Arreglos por Estudiante',
    E'Evaluación individual con casos generados por semilla determinista para cada estudiante.\n\nEntrada:\n- Línea 1: entero `N`.\n- Línea 2: `N` enteros separados por espacio.\n\nSalida:\n- La suma total de los `N` enteros.',
    'algorithm',
    '{
        "algorithm": {
            "time_limit_ms": 1500,
            "memory_limit_mb": 128,
            "input_format": {
                "version": 1,
                "input": {
                    "lines": [
                        { "id": "n", "type": "int", "min": 5, "max": 8 },
                        { "id": "valores", "type": "ints", "count": "=n", "item": { "min": 10, "max": 99 } }
                    ]
                }
            },
            "ast_rules": {
                "forbidden_imports": ["os", "subprocess"],
                "forbidden_functions": ["eval", "exec"]
            }
        }
    }'::jsonb,
    '7b100000-0000-4000-8000-000000000001',
    NOW() + INTERVAL '2 days',
    E'import sys\n\nlineas = [l.strip() for l in sys.stdin.read().splitlines() if l.strip()]\nn = int(lineas[0])\nvalores = list(map(int, lineas[1].split()))[:n]\nprint(sum(valores))\n',
    'published', 'python', 1500, 128, '{}'::jsonb,
    NOW() - INTERVAL '2 days', '{"id": "exact"}'::jsonb,
    E'import sys\n\nlineas = [l.strip() for l in sys.stdin.read().splitlines() if l.strip()]\nn = int(lineas[0])\nvalores = list(map(int, lineas[1].split()))[:n]\nprint(sum(valores))\n',
    false, 'medium', ARRAY['examen', 'semilla'], 'exam', true, 'JUEZ_EFIMERO',
    (SELECT python_judge_tpl_id FROM demo_ctx), 'O(N)'
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
    reference_solution = EXCLUDED.reference_solution,
    purpose = 'exam',
    per_student_seed = true,
    environment_type = 'JUEZ_EFIMERO',
    template_id = EXCLUDED.template_id;

INSERT INTO exercise_test_cases (exercise_id, order_index, input, expected_output, visibility, weight)
VALUES
    ('9e000000-0000-4000-8000-000000000010', 0, E'5\n10 20 30 40 50',       '150', 'example', 1.0),
    ('9e000000-0000-4000-8000-000000000010', 1, E'6\n12 15 18 21 24 27',    '117', 'hidden',  1.0),
    ('9e000000-0000-4000-8000-000000000010', 2, E'5\n99 11 22 33 44',       '209', 'hidden',  1.0)
ON CONFLICT (exercise_id, order_index) DO UPDATE
SET input = EXCLUDED.input,
    expected_output = EXCLUDED.expected_output,
    visibility = EXCLUDED.visibility,
    weight = EXCLUDED.weight;

-- Examen IDE aislado en red interna (SIS-12)
UPDATE exercises
SET subject_id = '6879c008-375e-4849-a341-6517463ab81e',
    status = 'published',
    purpose = 'exam',
    exam_network = 'none'
WHERE id = '7ae3f0c1-4455-4662-8978-a194fe34544a';

-- ============================================================================
-- 6. Flujo 4: Revisión docente, similitud y «Probar aplicación viva»
-- ============================================================================
-- Configurar app_run_command + app_port en Sistema de Reservas de Laboratorio
UPDATE exercises
SET subject_id = '6879c008-375e-4849-a341-6517463ab81e',
    status = 'published',
    config = '{
        "project": {
            "language": "python",
            "entrypoint": "main.py",
            "test_command": "python -m pytest"
        },
        "ide": {
            "app_port": 8000,
            "app_run_command": "python3 web_preview.py",
            "allow_resubmit": false,
            "auto_checkpoint": false
        }
    }'::jsonb
WHERE id = '9e000000-0000-4000-8000-000000000004';

-- Limpiar datos irreproducibles en vivo (complexity_analysis sintético) y normalizar ast_result
UPDATE submissions
SET complexity_analysis = NULL,
    generated_cases = NULL
WHERE id BETWEEN 'a1000000-0000-4000-8000-000000000001' AND 'a1000000-0000-4000-8000-000000000010';

UPDATE submissions
SET ast_result = '{"has_violations": true, "violations": [{"rule_id": "solv-python-forbidden-import-subprocess", "message": "Import of ''subprocess'' module is forbidden for security reasons", "line": 1}]}'::jsonb
WHERE id = 'a1000000-0000-4000-8000-000000000005';

-- Actualizar las 2 entregas IDE de Sistema de Reservas con web_preview.py ejecutable
-- para que «Probar aplicación viva» levante la app real en el puerto 8000.
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

# === FILE: web_preview.py ===
from http.server import BaseHTTPRequestHandler, HTTPServer
import html
import os
from reservas.service import GestorReservas

gestor = GestorReservas()
gestor.registrar("LAB-1", "2026-11-27", "08:00", "10:00", "Ing. Carlos Mendoza")
gestor.registrar("LAB-1", "2026-11-27", "10:00", "12:00", "Camila Rojas")


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        rows = "".join(
            f"<tr><td>{html.escape(r.laboratorio)}</td><td>{html.escape(r.fecha)}</td>"
            f"<td>{html.escape(r.inicio)} - {html.escape(r.fin)}</td><td>{html.escape(r.docente)}</td></tr>"
            for r in gestor.listar("LAB-1", "2026-11-27")
        )
        body = (
            "<!doctype html><html lang='es'><head><meta charset='utf-8'>"
            "<title>Reservas de Laboratorio - Camila Rojas</title>"
            "<style>body{font-family:sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem}"
            "table{width:100%;border-collapse:collapse}th,td{border:1px solid #cbd5e1;padding:.5rem;text-align:left}</style>"
            "</head><body><h1>Sistema de Reservas de Laboratorio</h1>"
            "<p>Entrega de Camila Rojas Terrazas (validación de superposición activa)</p>"
            f"<table><thead><tr><th>Lab</th><th>Fecha</th><th>Horario</th><th>Responsable</th></tr></thead><tbody>{rows}</tbody></table>"
            "</body></html>"
        ).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    port = int(os.environ.get("PORT", "8000"))
    HTTPServer(("0.0.0.0", port), Handler).serve_forever()

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

# === FILE: web_preview.py ===
from http.server import BaseHTTPRequestHandler, HTTPServer
import html
import os
from reservas.service import GestorReservas

gestor = GestorReservas()
gestor.registrar("LAB-1", "2026-11-27", "08:00", "10:00", "Mateo Villarroel")


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        rows = "".join(
            f"<tr><td>{html.escape(r.laboratorio)}</td><td>{html.escape(r.fecha)}</td>"
            f"<td>{html.escape(r.inicio)} - {html.escape(r.fin)}</td><td>{html.escape(r.docente)}</td></tr>"
            for r in gestor.listar("LAB-1", "2026-11-27")
        )
        body = (
            "<!doctype html><html lang='es'><head><meta charset='utf-8'>"
            "<title>Reservas de Laboratorio - Mateo Villarroel</title>"
            "<style>body{font-family:sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem}"
            "table{width:100%;border-collapse:collapse}th,td{border:1px solid #cbd5e1;padding:.5rem;text-align:left}</style>"
            "</head><body><h1>Sistema de Reservas de Laboratorio</h1>"
            "<p>Entrega de Mateo Villarroel Paz</p>"
            f"<table><thead><tr><th>Lab</th><th>Fecha</th><th>Horario</th><th>Responsable</th></tr></thead><tbody>{rows}</tbody></table>"
            "</body></html>"
        ).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    port = int(os.environ.get("PORT", "8000"))
    HTTPServer(("0.0.0.0", port), Handler).serve_forever()

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

-- Comentario por rango real en reservas/service.py sobre la entrega de Mateo
DELETE FROM submission_comments
WHERE submission_id IN (
    'a1000000-0000-4000-8000-000000000011',
    'a1000000-0000-4000-8000-000000000012'
);

INSERT INTO submission_comments (
    id, tenant_id, submission_id, author_id, line_number, end_line_number,
    file_path, local_line_number, comment, created_at
)
VALUES (
    'b52424b7-b439-4bff-9c18-d34f9a130836',
    '00000000-0000-0000-0000-000000000001',
    'a1000000-0000-4000-8000-000000000012',
    (SELECT teacher_id FROM demo_ctx),
    82,
    87,
    'reservas/service.py',
    11,
    'La validación del formato horario está bien encaminada, pero falta comprobar la superposición contra las reservas existentes en el mismo laboratorio y fecha.',
    NOW() - INTERVAL '2 hours'
);

COMMIT;
