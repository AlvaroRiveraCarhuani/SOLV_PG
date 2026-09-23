-- +goose Up
ALTER TABLE template_categories ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE template_categories ADD COLUMN IF NOT EXISTS sort_order INT DEFAULT 0;

ALTER TABLE template_models ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE template_models ADD COLUMN IF NOT EXISTS sort_order INT DEFAULT 0;

-- Actualizar orden y estado de categorías existentes
UPDATE template_categories SET
    sort_order = CASE name
        WHEN 'Ciencia de Datos & IA' THEN 1
        WHEN 'Desarrollo Web & Cloud' THEN 2
        WHEN 'Sistemas & Computación' THEN 3
        WHEN 'Empresarial & Backend' THEN 4
        ELSE 5
    END,
    is_active = true
WHERE tenant_id = '00000000-0000-0000-0000-000000000001';

-- Seeds de categorías institucionales (si no existen)
INSERT INTO template_categories (id, tenant_id, name, description, is_active, sort_order)
VALUES
    ('c0000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Ciencia de Datos & IA', 'Modelos para análisis de datos, machine learning e inteligencia artificial', true, 1),
    ('c0000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Desarrollo Web & Cloud', 'Modelos para aplicaciones web, APIs y microservicios modernos', true, 2),
    ('c0000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', 'Sistemas & Computación', 'Modelos para bajo nivel, sistemas operativos, compiladores y algoritmia', true, 3),
    ('c0000000-0000-0000-0000-000000000004', '00000000-0000-0000-0000-000000000001', 'Empresarial & Backend', 'Modelos para desarrollo corporativo, arquitecturas empresariales y JVM', true, 4)
ON CONFLICT (id) DO UPDATE SET
    is_active = EXCLUDED.is_active,
    sort_order = EXCLUDED.sort_order;

-- +goose Down
ALTER TABLE template_models DROP COLUMN IF EXISTS sort_order;
ALTER TABLE template_models DROP COLUMN IF EXISTS is_active;
ALTER TABLE template_categories DROP COLUMN IF EXISTS sort_order;
ALTER TABLE template_categories DROP COLUMN IF EXISTS is_active;
