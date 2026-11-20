-- +goose Up
-- Un entorno de evaluación puede declarar varios lenguajes. Cada lenguaje
-- aporta su propio comando de compilación y ejecución, y opcionalmente su
-- imagen Docker. Una imagen vacía significa que el juez usa la imagen del
-- catálogo para ese lenguaje (language_profiles / estrategia del motor).
--
-- La clave es (template_id, language): un lenguaje se declara una sola vez por
-- entorno. El lenguaje se guarda ya canonizado por el dominio (c++ -> cpp,
-- c# -> csharp, js -> javascript).

CREATE TABLE IF NOT EXISTS template_language_runners (
    template_id UUID NOT NULL REFERENCES lab_templates(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    entrypoint TEXT NOT NULL,
    docker_image TEXT NOT NULL DEFAULT '',
    timeout_ms INT NOT NULL DEFAULT 0,
    memory_limit_mb INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT pk_template_language_runners PRIMARY KEY (template_id, language),
    CONSTRAINT chk_template_language_runners_language CHECK (language <> ''),
    CONSTRAINT chk_template_language_runners_entrypoint CHECK (entrypoint <> ''),
    CONSTRAINT chk_template_language_runners_timeout CHECK (timeout_ms BETWEEN 0 AND 60000),
    CONSTRAINT chk_template_language_runners_memory CHECK (memory_limit_mb BETWEEN 0 AND 4096)
);

CREATE INDEX IF NOT EXISTS idx_template_language_runners_language
    ON template_language_runners(language);

-- Relleno de los entornos de evaluación que ya existen: se deduce su lenguaje
-- del runtime de la imagen y se conserva el entrypoint vigente. Es idempotente
-- y no inventa lenguajes para imágenes que no reconoce.
INSERT INTO template_language_runners (template_id, language, entrypoint, docker_image)
SELECT lt.id, derived.language, TRIM(lt.entrypoint), ''
FROM lab_templates lt
CROSS JOIN LATERAL (
    SELECT CASE
        WHEN LOWER(lt.docker_image) LIKE '%python%' THEN 'python'
        WHEN LOWER(lt.docker_image) LIKE '%node%'
          OR LOWER(lt.docker_image) LIKE '%javascript%' THEN 'javascript'
        WHEN LOWER(lt.docker_image) LIKE '%temurin%'
          OR LOWER(lt.docker_image) LIKE '%openjdk%'
          OR LOWER(lt.docker_image) LIKE '%java%' THEN 'java'
        WHEN LOWER(lt.docker_image) LIKE '%mono%'
          OR LOWER(lt.docker_image) LIKE '%csharp%' THEN 'csharp'
        WHEN LOWER(lt.docker_image) LIKE '%golang%'
          OR LOWER(lt.docker_image) LIKE 'go:%' THEN 'go'
        WHEN LOWER(lt.docker_image) LIKE '%rust%' THEN 'rust'
        WHEN LOWER(lt.docker_image) LIKE '%php%' THEN 'php'
        WHEN LOWER(lt.docker_image) LIKE '%gcc%'
          OR LOWER(lt.docker_image) LIKE '%clang%'
          OR LOWER(lt.docker_image) LIKE '%cpp%' THEN
            CASE
                WHEN LOWER(lt.entrypoint) LIKE '%solution.c %'
                  OR LOWER(lt.entrypoint) LIKE '%.c -o%'
                  OR LOWER(lt.entrypoint) LIKE '%.c%' AND LOWER(lt.entrypoint) NOT LIKE '%.cpp%' THEN 'c'
                ELSE 'cpp'
            END
        ELSE ''
    END AS language
) AS derived
WHERE lt.target_environment IN ('JUEZ_EFIMERO', 'JUEZ_VIRTUAL', 'AMBOS')
  AND COALESCE(TRIM(lt.entrypoint), '') <> ''
  AND derived.language <> ''
ON CONFLICT (template_id, language) DO NOTHING;

-- +goose Down
DROP INDEX IF EXISTS idx_template_language_runners_language;
DROP TABLE IF EXISTS template_language_runners;
