-- +goose Up
-- Avance voluntario del docente en el aula (ADR-058). El docente emite la solución
-- de un ejercicio dentro de su aula abierta; cada alumno destinatario decide si la
-- aplica en su propio entorno. Nada se sobrescribe sin confirmación del alumno.
--
-- frozen_at: congelación manual de la clase. Bloquea emisiones nuevas, no la edición.
-- grace_until: fin de la ventana de rezago de los avances emitidos, fijado al cerrar.
ALTER TABLE class_sessions ADD COLUMN IF NOT EXISTS frozen_at TIMESTAMPTZ;
ALTER TABLE class_sessions ADD COLUMN IF NOT EXISTS grace_until TIMESTAMPTZ;

-- Un avance es una emisión: snapshot del código del ejercicio en el momento de emitir.
-- El tar.gz vive como objeto en SeaweedFS (API S3); aquí solo se guarda su clave,
-- hash y tamaño. Así la base no crece con el código ni los respaldos la incluyen.
CREATE TABLE IF NOT EXISTS class_advances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    session_id UUID NOT NULL REFERENCES class_sessions(id) ON DELETE CASCADE,
    exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    scope VARCHAR(10) NOT NULL CHECK (scope IN ('class', 'selected')),
    snapshot_key TEXT NOT NULL,
    snapshot_sha256 CHAR(64) NOT NULL,
    snapshot_size_bytes BIGINT NOT NULL CHECK (snapshot_size_bytes >= 0),
    file_count INTEGER NOT NULL DEFAULT 0 CHECK (file_count >= 0),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_class_advances_session ON class_advances(session_id);
CREATE INDEX IF NOT EXISTS idx_class_advances_tenant ON class_advances(tenant_id);

-- Destinatario de un avance. Una fila por alumno; el estado se decide una sola vez
-- (pending -> applied | dismissed | failed | expired).
CREATE TABLE IF NOT EXISTS class_advance_recipients (
    advance_id UUID NOT NULL REFERENCES class_advances(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    status VARCHAR(10) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'applied', 'dismissed', 'failed', 'expired')),
    decided_at TIMESTAMPTZ,
    backup_path TEXT,
    PRIMARY KEY (advance_id, student_id)
);

-- Consulta del aviso por minuto: avances pendientes de un alumno.
CREATE INDEX IF NOT EXISTS idx_class_advance_recipients_pending
    ON class_advance_recipients(student_id, status)
    WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS idx_class_advance_recipients_pending;
DROP TABLE IF EXISTS class_advance_recipients;
DROP INDEX IF EXISTS idx_class_advances_tenant;
DROP INDEX IF EXISTS idx_class_advances_session;
DROP TABLE IF EXISTS class_advances;
ALTER TABLE class_sessions DROP COLUMN IF EXISTS grace_until;
ALTER TABLE class_sessions DROP COLUMN IF EXISTS frozen_at;
