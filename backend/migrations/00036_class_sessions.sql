-- +goose Up
-- Aula de una clase: la sesión que faltaba para poder decir quién está, quién
-- falta y quién se trabó. Es una envoltura opcional de la práctica: el alumno
-- puede trabajar sin aula abierta y nada de esta migración lo impide.
--
-- Una materia no puede tener dos aulas abiertas a la vez: el índice parcial
-- sobre status='open' lo garantiza en la base, y el servicio lo traduce a un
-- conflicto de dominio en lugar de un error de base de datos.

CREATE TABLE IF NOT EXISTS class_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    exercise_id UUID REFERENCES exercises(id) ON DELETE SET NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    opens_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closes_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    instructions TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_class_sessions_open_subject
    ON class_sessions(tenant_id, subject_id)
    WHERE status = 'open';
CREATE INDEX IF NOT EXISTS idx_class_sessions_tenant ON class_sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_class_sessions_subject ON class_sessions(subject_id, opens_at DESC);

-- Presencia: la fila existe desde que el alumno entra al aula y last_seen_at se
-- refresca con cada señal. joined_at/left_at son el registro de asistencia, no
-- un módulo de pasar lista.
CREATE TABLE IF NOT EXISTS class_session_participants (
    session_id UUID NOT NULL REFERENCES class_sessions(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    PRIMARY KEY (session_id, student_id)
);

CREATE INDEX IF NOT EXISTS idx_session_participants_tenant
    ON class_session_participants(tenant_id, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_session_participants_student
    ON class_session_participants(student_id, session_id);

-- En qué aula nació el entorno. Nulo en los entornos abiertos fuera de clase.
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS session_id UUID
    REFERENCES class_sessions(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_workspaces_session ON workspaces(session_id);

-- +goose Down
DROP INDEX IF EXISTS idx_workspaces_session;
ALTER TABLE workspaces DROP COLUMN IF EXISTS session_id;
DROP TABLE IF EXISTS class_session_participants;
DROP TABLE IF EXISTS class_sessions;
