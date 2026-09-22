-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- Limpieza de residuo de tests de integración en solv_db
-- Incidente documentado: 139 filas de tests contaminaron la BD
-- de desarrollo (categorías 'Category Test %', plantillas
-- 'Template %' con status rejected y rejection_reason 'test%').
--
-- Esta migración es idempotente: elimina solo los residuos
-- conocidos sin tocar datos reales.
-- ============================================================

-- 1. Eliminar plantillas de test que no deberían estar en producción
--    Criterio: status 'RECHAZADA' (normalizado de 'rejected') con
--    rejection_reason que referencia contexto de test.
--    Usar CTE para reportar cuántas se eliminan.
WITH deleted_templates AS (
    DELETE FROM lab_templates
    WHERE
        (status = 'RECHAZADA' OR status = 'rejected')
        AND (
            rejection_reason ILIKE '%test%'
            OR name ILIKE 'Template %Test%'
            OR name ILIKE 'Test Template%'
        )
    RETURNING id, name
)
INSERT INTO audit_logs (tenant_id, actor_id, action, resource_type, metadata)
SELECT
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000000'::uuid,
    'MIGRATION_CLEANUP_TEST_TEMPLATE',
    'lab_template',
    jsonb_build_object('deleted_name', name, 'reason', 'test_residue_cleanup_00003')
FROM deleted_templates
WHERE EXISTS (SELECT 1 FROM tenants WHERE id = '00000000-0000-0000-0000-000000000001');

-- 2. Eliminar categorías de test (nombre LIKE 'Category Test %')
--    Solo si no tienen plantillas reales asociadas (las de test
--    ya fueron eliminadas en el paso anterior).
WITH deleted_cats AS (
    DELETE FROM template_categories
    WHERE name ILIKE 'Category Test %'
      AND NOT EXISTS (
          SELECT 1 FROM lab_templates lt
          WHERE lt.category_id = template_categories.id
      )
      AND NOT EXISTS (
          SELECT 1 FROM template_models tm
          WHERE tm.category_id = template_categories.id
      )
    RETURNING id, name
)
INSERT INTO audit_logs (tenant_id, actor_id, action, resource_type, metadata)
SELECT
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000000'::uuid,
    'MIGRATION_CLEANUP_TEST_CATEGORY',
    'template_category',
    jsonb_build_object('deleted_name', name, 'reason', 'test_residue_cleanup_00003')
FROM deleted_cats
WHERE EXISTS (SELECT 1 FROM tenants WHERE id = '00000000-0000-0000-0000-000000000001');

-- 3. Reinstaurar seeds de categorías y modelos en caso de que
--    alguna limpieza previa haya eliminado los seeds oficiales.
INSERT INTO template_categories (id, tenant_id, name, description)
VALUES
    ('c0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Ciencia de Datos & IA', 'Entornos especializados en análisis de datos, visualización y aprendizaje automático.'),
    ('c0000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Desarrollo Web & Cloud', 'Herramientas para desarrollo fullstack, APIs backend y microservicios modernos.'),
    ('c0000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'Sistemas & Computación', 'Compiladores nativos y herramientas de bajo nivel para arquitectura y sistemas operativos.'),
    ('c0000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'Empresarial & Backend', 'Plataformas consolidadas para desarrollo empresarial y programación orientada a objetos.')
ON CONFLICT (tenant_id, name) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- Down es no-op: los datos eliminados eran basura de test, no hay que restaurar.
SELECT 1;
