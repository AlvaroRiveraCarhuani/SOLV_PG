-- Script de carga de datos para OA-21 (Prueba de carga de pico de deadline)
BEGIN;

DO $$
DECLARE
    v_tenant_id UUID := '00000000-0000-0000-0000-000000000001';
    v_subject_id UUID := 'c0000000-0000-0000-0000-000000000001';
    v_exercise_id UUID;
    v_student_id UUID;
    v_i INT;
    v_j INT;
BEGIN
    -- 1. Crear o asegurar asignatura / curso de prueba
    INSERT INTO subjects (id, tenant_id, name, code)
    VALUES (v_subject_id, v_tenant_id, 'Algoritmos y Estructuras de Datos - Load Test', 'CS-LOAD-2026')
    ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

    -- 2. Crear 10 ejercicios publicados de algoritmia simple (Suma de números en Python)
    FOR v_i IN 1..10 LOOP
        v_exercise_id := ('e0000000-0000-0000-0000-0000000000' || LPAD(v_i::TEXT, 2, '0'))::UUID;
        
        INSERT INTO exercises (
            id, tenant_id, subject_id, title, description, type,
            difficulty, tags, purpose, status, language,
            time_limit_ms, memory_limit_mb, reference_solution,
            config
        )
        VALUES (
            v_exercise_id,
            v_tenant_id,
            v_subject_id,
            'Ejercicio de Carga ' || v_i || ': Suma de Enteros',
            'Dado un conjunto de enteros separados por espacio, calcula e imprime la suma total.',
            'algorithm',
            'easy',
            ARRAY['algoritmos', 'load-test', 'suma'],
            'exam',
            'published',
            'python',
            2000,
            128,
            'print(sum(map(int, input().split())))',
            '{"algorithm": {"time_limit_ms": 2000, "memory_limit_mb": 128, "ast_rules": {"forbidden_imports": [], "forbidden_functions": []}}, "input_format": {"type": "integers", "count": 2}}'::jsonb
        )
        ON CONFLICT (id) DO UPDATE SET
            status = 'published',
            reference_solution = EXCLUDED.reference_solution,
            config = EXCLUDED.config;

        -- Casos de prueba para cada ejercicio
        DELETE FROM exercise_test_cases WHERE exercise_id = v_exercise_id;

        INSERT INTO exercise_test_cases (exercise_id, order_index, input, expected_output, visibility, weight)
        VALUES
            (v_exercise_id, 1, '5 10', '15', 'public', 1.0),
            (v_exercise_id, 2, '100 250', '350', 'example', 1.0),
            (v_exercise_id, 3, '-5 5', '0', 'hidden', 1.0);
    END LOOP;

    -- 3. Crear 200 estudiantes de prueba y matricularlos
    FOR v_j IN 1..200 LOOP
        v_student_id := ('00000000-0000-0000-0001-' || LPAD(v_j::TEXT, 12, '0'))::UUID;

        INSERT INTO users (
            id, tenant_id, first_name, last_name, email, role, status
        )
        VALUES (
            v_student_id,
            v_tenant_id,
            'Estudiante',
            'Test ' || v_j,
            'student-test-' || v_j || '@test.com',
            'student',
            'active'
        )
        ON CONFLICT (email) DO UPDATE SET status = 'active';

        INSERT INTO enrollments (tenant_id, student_id, subject_id)
        VALUES (v_tenant_id, v_student_id, v_subject_id)
        ON CONFLICT (tenant_id, student_id, subject_id) DO NOTHING;
    END LOOP;

END $$;

COMMIT;
