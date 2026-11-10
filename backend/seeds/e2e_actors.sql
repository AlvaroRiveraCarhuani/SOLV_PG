-- Identidades sintéticas para los flujos E2E. No usa nombres, correos ni
-- identificadores de personas reales. Idempotente: se puede aplicar en cada
-- corrida sobre una base de tests o sobre un stack E2E aislado.
BEGIN;

INSERT INTO tenants (id, name, slug)
VALUES ('00000000-0000-0000-0000-000000000001', 'SOLV E2E', 'solv-e2e')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    slug = EXCLUDED.slug;

INSERT INTO users (id, email, first_name, last_name, role, tenant_id, origin, status)
VALUES
  ('e2e00000-0000-4000-8000-000000000001', 'admin@e2e.solv.test', 'Admin', 'E2E', 'admin', '00000000-0000-0000-0000-000000000001', 'manual', 'active'),
  ('e2e00000-0000-4000-8000-000000000002', 'teacher@e2e.solv.test', 'Docente', 'E2E', 'teacher', '00000000-0000-0000-0000-000000000001', 'manual', 'active'),
  ('e2e00000-0000-4000-8000-000000000003', 'student@e2e.solv.test', 'Estudiante', 'E2E', 'student', '00000000-0000-0000-0000-000000000001', 'manual', 'active')
ON CONFLICT (id) DO UPDATE
SET email = EXCLUDED.email,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    role = EXCLUDED.role,
    tenant_id = EXCLUDED.tenant_id,
    origin = EXCLUDED.origin,
    status = EXCLUDED.status;

COMMIT;
