-- +goose Up
-- Sembrar plantilla runner multilenguaje para el Entorno de Evaluación (idempotente)
INSERT INTO lab_templates (id, tenant_id, name, docker_image, base_ram_mb, target_environment, status, description, entrypoint)
VALUES (
    '00000000-0000-0000-0000-000000000099',
    '00000000-0000-0000-0000-000000000001',
    'Runner Multilenguaje Base',
    'python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93',
    256,
    'JUEZ_EFIMERO',
    'approved',
    'SYSTEM_SEED_RUNNER',
    'python3 solution.py'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO template_language_runners (template_id, language, entrypoint, docker_image)
VALUES
    ('00000000-0000-0000-0000-000000000099', 'python', 'python3 solution.py', ''),
    ('00000000-0000-0000-0000-000000000099', 'cpp', 'g++ -O2 solution.cpp -o solution && ./solution', ''),
    ('00000000-0000-0000-0000-000000000099', 'c', 'gcc -O2 solution.c -o solution && ./solution', ''),
    ('00000000-0000-0000-0000-000000000099', 'java', 'javac Main.java && java Main', ''),
    ('00000000-0000-0000-0000-000000000099', 'go', 'go run solution.go', ''),
    ('00000000-0000-0000-0000-000000000099', 'csharp', 'mono solution.exe', ''),
    ('00000000-0000-0000-0000-000000000099', 'javascript', 'node solution.js', ''),
    ('00000000-0000-0000-0000-000000000099', 'typescript', 'ts-node solution.ts', ''),
    ('00000000-0000-0000-0000-000000000099', 'rust', 'rustc -O solution.rs -o solution && ./solution', '')
ON CONFLICT (template_id, language) DO UPDATE
    SET entrypoint = EXCLUDED.entrypoint;

-- +goose Down
DELETE FROM template_language_runners WHERE template_id = '00000000-0000-0000-0000-000000000099';
DELETE FROM lab_templates WHERE id = '00000000-0000-0000-0000-000000000099';
