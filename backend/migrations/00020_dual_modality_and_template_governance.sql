-- +goose Up
-- 1. Sembrar plantillas runner base faltantes por lenguaje (idempotente)
INSERT INTO lab_templates (tenant_id, name, docker_image, base_ram_mb, target_environment, status, description)
SELECT '00000000-0000-0000-0000-000000000001', 'Python Runner Base', 'python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93', COALESCE((SELECT MODE() WITHIN GROUP (ORDER BY memory_limit_mb) FROM exercises WHERE LOWER(language) IN ('python', 'python3')), 128), 'JUEZ_EFIMERO', 'approved', 'SYSTEM_SEED_RUNNER'
WHERE NOT EXISTS (
    SELECT 1 FROM lab_templates WHERE target_environment = 'JUEZ_EFIMERO' AND (docker_image LIKE 'python%' OR name = 'Python Runner Base')
);

INSERT INTO lab_templates (tenant_id, name, docker_image, base_ram_mb, target_environment, status, description)
SELECT '00000000-0000-0000-0000-000000000001', 'JavaScript Runner Base', 'node:20-alpine@sha256:fb4cd12c85ee03686f6af5362a0b0d56d50c58a04632e6c0fb8363f609372293', COALESCE((SELECT MODE() WITHIN GROUP (ORDER BY memory_limit_mb) FROM exercises WHERE LOWER(language) IN ('javascript', 'js', 'node')), 128), 'JUEZ_EFIMERO', 'approved', 'SYSTEM_SEED_RUNNER'
WHERE NOT EXISTS (
    SELECT 1 FROM lab_templates WHERE target_environment = 'JUEZ_EFIMERO' AND (docker_image LIKE 'node%' OR name = 'JavaScript Runner Base')
);

INSERT INTO lab_templates (tenant_id, name, docker_image, base_ram_mb, target_environment, status, description)
SELECT '00000000-0000-0000-0000-000000000001', 'C/C++ Runner Base', 'gcc@sha256:cde79a7114216f9a1a66509932adabdd1e8620d8c8d11be19a34ee4b22d66c91', COALESCE((SELECT MODE() WITHIN GROUP (ORDER BY memory_limit_mb) FROM exercises WHERE LOWER(language) IN ('cpp', 'c++', 'c')), 128), 'JUEZ_EFIMERO', 'approved', 'SYSTEM_SEED_RUNNER'
WHERE NOT EXISTS (
    SELECT 1 FROM lab_templates WHERE target_environment = 'JUEZ_EFIMERO' AND (docker_image LIKE 'gcc%' OR name = 'C/C++ Runner Base')
);

INSERT INTO lab_templates (tenant_id, name, docker_image, base_ram_mb, target_environment, status, description)
SELECT '00000000-0000-0000-0000-000000000001', 'Java Runner Base', 'eclipse-temurin:21-alpine@sha256:1ff763083f2993d57d0bf374ab10bb3e2cb873af6c13a04458ebbd3e0337dc76', COALESCE((SELECT MODE() WITHIN GROUP (ORDER BY memory_limit_mb) FROM exercises WHERE LOWER(language) = 'java'), 512), 'JUEZ_EFIMERO', 'approved', 'SYSTEM_SEED_RUNNER'
WHERE NOT EXISTS (
    SELECT 1 FROM lab_templates WHERE target_environment = 'JUEZ_EFIMERO' AND (docker_image LIKE 'eclipse-temurin%' OR name = 'Java Runner Base')
);

INSERT INTO lab_templates (tenant_id, name, docker_image, base_ram_mb, target_environment, status, description)
SELECT '00000000-0000-0000-0000-000000000001', 'C# Runner Base', 'mono@sha256:34d816779b1248b5cfd095770b64ecbaf1798e2aca693a91c11a018dce9c7ad5', COALESCE((SELECT MODE() WITHIN GROUP (ORDER BY memory_limit_mb) FROM exercises WHERE LOWER(language) IN ('csharp', 'c#', 'cs')), 256), 'JUEZ_EFIMERO', 'approved', 'SYSTEM_SEED_RUNNER'
WHERE NOT EXISTS (
    SELECT 1 FROM lab_templates WHERE target_environment = 'JUEZ_EFIMERO' AND (docker_image LIKE 'mono%' OR name = 'C# Runner Base')
);

INSERT INTO lab_templates (tenant_id, name, docker_image, base_ram_mb, target_environment, status, description)
SELECT '00000000-0000-0000-0000-000000000001', 'Go Runner Base', 'golang:1.24-alpine', COALESCE((SELECT MODE() WITHIN GROUP (ORDER BY memory_limit_mb) FROM exercises WHERE LOWER(language) IN ('go', 'golang')), 256), 'JUEZ_EFIMERO', 'approved', 'SYSTEM_SEED_RUNNER'
WHERE NOT EXISTS (
    SELECT 1 FROM lab_templates WHERE target_environment = 'JUEZ_EFIMERO' AND (docker_image LIKE 'golang%' OR name = 'Go Runner Base')
);

-- 2. Agregar columnas environment_type y template_id
ALTER TABLE exercises
  ADD COLUMN IF NOT EXISTS environment_type VARCHAR(50) NOT NULL DEFAULT 'JUEZ_EFIMERO'
    CHECK (environment_type IN ('JUEZ_EFIMERO', 'IDE_PERSISTENTE')),
  ADD COLUMN IF NOT EXISTS template_id UUID;

-- 3. Backfill de environment_type
UPDATE exercises SET environment_type = 'JUEZ_EFIMERO' WHERE environment_type IS NULL OR environment_type = '';

-- 4. Backfill de template_id por lenguaje
UPDATE exercises e SET template_id = (
  SELECT lt.id FROM lab_templates lt
  WHERE lt.target_environment = 'JUEZ_EFIMERO' AND lt.status = 'approved'
    AND (
      (LOWER(e.language) IN ('python', 'python3') AND (lt.docker_image LIKE 'python%' OR LOWER(lt.name) LIKE '%python%'))
      OR (LOWER(e.language) IN ('javascript', 'js', 'node') AND (lt.docker_image LIKE 'node%' OR LOWER(lt.name) LIKE '%node%' OR LOWER(lt.name) LIKE '%javascript%'))
      OR (LOWER(e.language) IN ('cpp', 'c++', 'c') AND (lt.docker_image LIKE 'gcc%' OR LOWER(lt.name) LIKE '%c++%' OR LOWER(lt.name) LIKE '%gcc%'))
      OR (LOWER(e.language) IN ('java') AND (lt.docker_image LIKE 'eclipse-temurin%' OR lt.docker_image LIKE 'java%' OR LOWER(lt.name) LIKE '%java%'))
      OR (LOWER(e.language) IN ('csharp', 'c#', 'cs') AND (lt.docker_image LIKE 'mono%' OR LOWER(lt.name) LIKE '%csharp%' OR LOWER(lt.name) LIKE '%mono%'))
      OR (LOWER(e.language) IN ('go', 'golang') AND (lt.docker_image LIKE 'golang%' OR LOWER(lt.name) LIKE '%go%'))
    )
  ORDER BY (lt.description = 'SYSTEM_SEED_RUNNER') DESC, lt.created_at ASC
  LIMIT 1
) WHERE e.template_id IS NULL;

-- Fallback general para cualquier ejercicio que aun tenga template_id NULL
UPDATE exercises e SET template_id = (
  SELECT lt.id FROM lab_templates lt
  WHERE lt.target_environment = 'JUEZ_EFIMERO' AND lt.status = 'approved'
  ORDER BY (lt.description = 'SYSTEM_SEED_RUNNER') DESC, lt.created_at ASC
  LIMIT 1
) WHERE e.template_id IS NULL;

-- 5. NOT NULL y FK
ALTER TABLE exercises ALTER COLUMN template_id SET NOT NULL,
  ADD CONSTRAINT fk_exercises_template FOREIGN KEY (template_id) REFERENCES lab_templates(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_exercises_environment ON exercises(environment_type);
CREATE INDEX IF NOT EXISTS idx_exercises_template ON exercises(template_id);

-- +goose Down
ALTER TABLE exercises DROP CONSTRAINT IF EXISTS fk_exercises_template;
DROP INDEX IF EXISTS idx_exercises_template;
DROP INDEX IF EXISTS idx_exercises_environment;
ALTER TABLE exercises DROP COLUMN IF EXISTS template_id, DROP COLUMN IF EXISTS environment_type;
DELETE FROM lab_templates WHERE description = 'SYSTEM_SEED_RUNNER';
