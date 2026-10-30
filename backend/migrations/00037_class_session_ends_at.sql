-- +goose Up
-- Hora de fin del aula. El docente abre la clase con una duración y la sesión se
-- cierra sola al vencer: una aula olvidada no puede bloquear la materia para
-- siempre, porque el índice parcial único impide abrir otra mientras siga en
-- estado 'open'.
--
-- opens_at marca el inicio y ends_at el fin previsto; closes_at sigue siendo el
-- fin real (cuando el docente cierra o cuando vence la sesión).

ALTER TABLE class_sessions ADD COLUMN IF NOT EXISTS ends_at TIMESTAMPTZ;

-- Las aulas creadas antes de esta columna no tienen fin declarado. Se les asigna
-- el cierre registrado o, si siguen abiertas, 90 minutos desde su apertura: no
-- se puede dejar una fila sin fin cuando el fin es lo que la cierra.
UPDATE class_sessions
SET ends_at = COALESCE(closes_at, opens_at + INTERVAL '90 minutes')
WHERE ends_at IS NULL;

ALTER TABLE class_sessions ALTER COLUMN ends_at SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_class_sessions_open_ends_at
    ON class_sessions(tenant_id, ends_at)
    WHERE status = 'open';

-- +goose Down
DROP INDEX IF EXISTS idx_class_sessions_open_ends_at;
ALTER TABLE class_sessions ALTER COLUMN ends_at DROP NOT NULL;
ALTER TABLE class_sessions DROP COLUMN IF EXISTS ends_at;
