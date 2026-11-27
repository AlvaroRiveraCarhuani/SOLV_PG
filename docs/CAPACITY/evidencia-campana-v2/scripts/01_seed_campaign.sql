-- Siembra de la campaña experimental v2.
-- Crea datos PROPIOS de la campaña (materia, 10 prácticas de juez efímero,
-- 1 práctica IDE persistente, 40 estudiantes de aula + 1 estudiante IDE).
-- Todo queda identificado con prefijos d0000000/f0000000/f1000000/c0000000…0002
-- para que scripts/cleanup.sh lo elimine sin tocar datos de la campaña v1.
BEGIN;

DO $$
DECLARE
    v_tenant      UUID := '00000000-0000-0000-0000-000000000001';
    v_subject     UUID := 'd0000000-0000-0000-0000-0000000000a1';
    v_ide_subject UUID := 'd0000000-0000-0000-0000-0000000000b1';
    v_template    UUID;
    v_ide_tpl     UUID := 'b395313f-701f-49cf-807e-638230241b3c';
    v_ex          UUID;
    v_user        UUID;
    v_i           INT;
BEGIN
    SELECT id INTO v_template
      FROM lab_templates
     WHERE target_environment = 'JUEZ_EFIMERO'
       AND status IN ('approved', 'APROBADA')
       AND name LIKE 'Python%'
     LIMIT 1;

    IF v_template IS NULL THEN
        RAISE EXCEPTION 'no existe plantilla JUEZ_EFIMERO de Python aprobada';
    END IF;

    INSERT INTO subjects (id, tenant_id, name, code)
    VALUES (v_subject, v_tenant, 'Capacidad de Aula — Campaña v2 (temporal)', 'CS-CAP-V2')
    ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, is_archived = false;

    INSERT INTO subjects (id, tenant_id, name, code)
    VALUES (v_ide_subject, v_tenant, 'Capacidad IDE — Campaña v2 (temporal)', 'CS-CAP-V2-IDE')
    ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, is_archived = false;

    -- 10 prácticas de juez efímero. purpose='class' permite reentrega, de modo
    -- que el 403 resubmission_not_allowed no contamine la medición de envíos.
    FOR v_i IN 1..10 LOOP
        v_ex := ('f0000000-0000-0000-0000-0000000000' || LPAD(v_i::TEXT, 2, '0'))::UUID;

        INSERT INTO exercises (
            id, tenant_id, subject_id, title, description, type, difficulty, tags,
            purpose, status, language, time_limit_ms, memory_limit_mb,
            reference_solution, config, template_id, environment_type
        )
        VALUES (
            v_ex, v_tenant, v_subject,
            'Práctica de capacidad ' || v_i || ': suma de enteros',
            'Dados enteros separados por espacio, imprime su suma.',
            'algorithm', 'easy', ARRAY['capacidad', 'campana-v2'],
            'class', 'published', 'python', 2000, 128,
            'print(sum(map(int, input().split())))',
            '{"algorithm": {"time_limit_ms": 2000, "memory_limit_mb": 128, "ast_rules": {"forbidden_imports": [], "forbidden_functions": []}}, "input_format": {"type": "integers", "count": 2}}'::jsonb,
            v_template, 'JUEZ_EFIMERO'
        )
        ON CONFLICT (id) DO UPDATE SET
            status = 'published',
            purpose = 'class',
            template_id = EXCLUDED.template_id,
            config = EXCLUDED.config;

        DELETE FROM exercise_test_cases WHERE exercise_id = v_ex;
        INSERT INTO exercise_test_cases (exercise_id, order_index, input, expected_output, visibility, weight)
        VALUES
            (v_ex, 1, '5 10', '15', 'public', 1.0),
            (v_ex, 2, '100 250', '350', 'example', 1.0),
            (v_ex, 3, '-5 5', '0', 'hidden', 1.0);
    END LOOP;

    -- Práctica de IDE persistente para E6 (frío vs caliente).
    INSERT INTO exercises (
        id, tenant_id, subject_id, title, description, type, difficulty, tags,
        purpose, status, language, time_limit_ms, memory_limit_mb,
        reference_solution, config, template_id, environment_type
    )
    VALUES (
        'f1000000-0000-0000-0000-000000000001', v_tenant, v_ide_subject,
        'Proyecto IDE de capacidad (temporal)',
        'Práctica de proyecto para medir provisión en frío y reanudación.',
        'project', 'easy', ARRAY['capacidad', 'campana-v2', 'ide'],
        'class', 'published', 'go', 2000, 128, '',
        '{"ide": {"allow_resubmit": true, "auto_checkpoint": false}}'::jsonb,
        v_ide_tpl, 'IDE_PERSISTENTE'
    )
    ON CONFLICT (id) DO UPDATE SET
        status = 'published',
        purpose = 'class',
        template_id = EXCLUDED.template_id,
        config = EXCLUDED.config;

    -- Dos proyectos IDE adicionales del mismo alumno y materia: permiten
    -- sostener 3 entornos IDE en paralelo durante el soak (la unicidad es por
    -- estudiante + materia + ejercicio).
    FOR v_i IN 2..3 LOOP
        v_ex := ('f1000000-0000-0000-0000-00000000000' || v_i::TEXT)::UUID;
        INSERT INTO exercises (
            id, tenant_id, subject_id, title, description, type, difficulty, tags,
            purpose, status, language, time_limit_ms, memory_limit_mb,
            reference_solution, config, template_id, environment_type
        )
        VALUES (
            v_ex, v_tenant, v_ide_subject,
            'Proyecto IDE de capacidad ' || v_i || ' (temporal)',
            'Práctica de proyecto para el soak de 30 minutos.',
            'project', 'easy', ARRAY['capacidad', 'campana-v2', 'ide'],
            'class', 'published', 'go', 2000, 128, '',
            '{"ide": {"allow_resubmit": true, "auto_checkpoint": false}}'::jsonb,
            v_ide_tpl, 'IDE_PERSISTENTE'
        )
        ON CONFLICT (id) DO UPDATE SET
            status = 'published',
            template_id = EXCLUDED.template_id,
            config = EXCLUDED.config;
    END LOOP;

    -- 40 estudiantes de aula.
    FOR v_i IN 1..40 LOOP
        v_user := ('c0000000-0000-0000-0002-' || LPAD(v_i::TEXT, 12, '0'))::UUID;

        INSERT INTO users (id, tenant_id, first_name, last_name, email, role, status)
        VALUES (v_user, v_tenant, 'Estudiante', 'Campaña v2 ' || v_i,
                'campaign-v2-s' || LPAD(v_i::TEXT, 2, '0') || '@test.local',
                'student', 'active')
        ON CONFLICT (id) DO UPDATE SET status = 'active';

        INSERT INTO enrollments (tenant_id, student_id, subject_id)
        VALUES (v_tenant, v_user, v_subject)
        ON CONFLICT (tenant_id, student_id, subject_id) DO NOTHING;
    END LOOP;

    -- Estudiante dedicado al IDE de E6.
    v_user := 'c0000000-0000-0000-0002-0000000000f1';
    INSERT INTO users (id, tenant_id, first_name, last_name, email, role, status)
    VALUES (v_user, v_tenant, 'Estudiante', 'Campaña v2 IDE',
            'campaign-v2-ide@test.local', 'student', 'active')
    ON CONFLICT (id) DO UPDATE SET status = 'active';

    INSERT INTO enrollments (tenant_id, student_id, subject_id)
    VALUES (v_tenant, v_user, v_ide_subject)
    ON CONFLICT (tenant_id, student_id, subject_id) DO NOTHING;

    RAISE NOTICE 'campaña v2 sembrada: subject % / ide % / template %', v_subject, v_ide_subject, v_template;
END $$;

COMMIT;
