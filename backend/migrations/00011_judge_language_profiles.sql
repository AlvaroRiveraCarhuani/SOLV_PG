-- +goose Up
-- Juez de ejercicios: perfiles de lenguaje versionados con auditoria.
-- Clave de lenguaje canonica (c++ -> cpp, c#/cs -> csharp); los perfiles
-- rechazan alias no canonicos (400 en servicio). Imagenes fijadas por digest
-- (prohibido :latest). p95_window_days configurable, default 30 dias.
-- checker_sidecar_image (digest fijado) es propiedad de administracion via el
-- endpoint de perfiles; todo cambio queda en language_profile_audits.
-- Digests verificados contra imagenes locales (docker images --digests).

CREATE TABLE IF NOT EXISTS language_profiles (
    language TEXT PRIMARY KEY,
    default_timeout_ms INT NOT NULL,
    default_memory_mb INT NOT NULL,
    image TEXT NOT NULL,
    build_command TEXT NOT NULL DEFAULT '',
    build_timeout_ms INT NOT NULL DEFAULT 10000,
    build_memory_mb INT NOT NULL DEFAULT 512,
    p95_window_days INT NOT NULL DEFAULT 30,
    checker_sidecar_image TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_language_profiles_timeout CHECK (default_timeout_ms BETWEEN 100 AND 10000),
    CONSTRAINT chk_language_profiles_memory CHECK (default_memory_mb BETWEEN 64 AND 1024)
);

CREATE TABLE IF NOT EXISTS language_profile_audits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    language TEXT NOT NULL,
    author TEXT NOT NULL DEFAULT '',
    old_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    new_values JSONB NOT NULL DEFAULT '{}'::jsonb,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_language_profile_audits_lang ON language_profile_audits(language, created_at);

-- Semilla v0 (spec D-EJ-04): python/js 2000/256, cpp/c 1000/128,
-- csharp 2500/256, java 3000/512. Idempotente por ON CONFLICT DO NOTHING.
INSERT INTO language_profiles (language, default_timeout_ms, default_memory_mb, image, build_command, checker_sidecar_image) VALUES
('python', 2000, 256, 'python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93', '', 'python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93'),
('javascript', 2000, 256, 'node:20-alpine@sha256:fb4cd12c85ee03686f6af5362a0b0d56d50c58a04632e6c0fb8363f609372293', '', 'python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93'),
('cpp', 1000, 128, 'gcc@sha256:cde79a7114216f9a1a66509932adabdd1e8620d8c8d11be19a34ee4b22d66c91', 'g++ -O2 /runner/solution.cpp -o /tmp/sol', 'python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93'),
('c', 1000, 128, 'gcc@sha256:cde79a7114216f9a1a66509932adabdd1e8620d8c8d11be19a34ee4b22d66c91', 'gcc -O2 /runner/solution.c -o /tmp/sol', 'python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93'),
('csharp', 2500, 256, 'mono@sha256:34d816779b1248b5cfd095770b64ecbaf1798e2aca693a91c11a018dce9c7ad5', 'mcs /runner/solution.cs -out:/tmp/sol.exe', 'python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93'),
('java', 3000, 512, 'eclipse-temurin:21-alpine@sha256:1ff763083f2993d57d0bf374ab10bb3e2cb873af6c13a04458ebbd3e0337dc76', 'javac -d /tmp /runner/Solution.java', 'python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93')
ON CONFLICT (language) DO NOTHING;

-- +goose Down
DELETE FROM language_profiles WHERE language IN ('python', 'javascript', 'cpp', 'c', 'csharp', 'java');
DROP INDEX IF EXISTS idx_language_profile_audits_lang;
DROP TABLE IF EXISTS language_profile_audits;
DROP TABLE IF EXISTS language_profiles;
