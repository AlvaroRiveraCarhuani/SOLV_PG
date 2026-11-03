-- +goose Up
-- Solicitudes de entorno del docente.
--
-- Una solicitud es una intención pedagógica, no una plantilla: guarda lo que el
-- docente pidió, las piezas que el constructor resolvió contra el catálogo, el
-- uso declarado (practicar, evaluar o ambos) y quién la pidió dentro del tenant.
-- La plantilla definitiva se sigue creando en el constructor del administrador;
-- aquí solo se enlaza cuando existe.
--
-- `stack_signature` es la huella de la receta (imagen base, capas de build,
-- servicios satélite y modalidad). Dos solicitudes con la misma huella son el
-- mismo entorno: la segunda se suma a la primera agregando su materia en
-- `subjects`, en lugar de abrir una ficha paralela.

CREATE TABLE IF NOT EXISTS environment_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    requested_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(160) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    teacher_text TEXT NOT NULL DEFAULT '',
    uses JSONB NOT NULL DEFAULT '[]'::jsonb,
    target_environment VARCHAR(50) NOT NULL DEFAULT 'IDE_PERSISTENTE',
    stack_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    components JSONB NOT NULL DEFAULT '[]'::jsonb,
    missing_components JSONB NOT NULL DEFAULT '[]'::jsonb,
    subjects JSONB NOT NULL DEFAULT '[]'::jsonb,
    stack_signature VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'EN_REVISION',
    review_note TEXT NOT NULL DEFAULT '',
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    resolved_template_id UUID REFERENCES lab_templates(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_environment_requests_status
        CHECK (status IN ('EN_REVISION', 'APROBADA', 'NECESITA_CAMBIO', 'NO_VIABLE')),
    CONSTRAINT chk_environment_requests_environment
        CHECK (target_environment IN ('IDE_PERSISTENTE', 'JUEZ_EFIMERO', 'AMBOS'))
);

CREATE INDEX IF NOT EXISTS idx_environment_requests_tenant_status
    ON environment_requests(tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_environment_requests_requester
    ON environment_requests(tenant_id, requested_by, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_environment_requests_signature
    ON environment_requests(tenant_id, stack_signature);

-- +goose Down
DROP TABLE IF EXISTS environment_requests;
