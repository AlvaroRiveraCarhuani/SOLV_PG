-- +goose Up
-- Control de idempotencia para recordatorios de vencimiento academico.
--
-- Registra los recordatorios de vencimiento enviados a estudiantes para ejercicios
-- dentro de la ventana de urgencia (~12 horas).
-- La restriccion unica (tenant_id, exercise_id, student_id) evita envios duplicados
-- al ejecutarse el worker periodico.

CREATE TABLE IF NOT EXISTS deadline_reminders_sent (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_deadline_reminder_exercise_student UNIQUE (tenant_id, exercise_id, student_id)
);

CREATE INDEX IF NOT EXISTS idx_deadline_reminders_tenant_exercise 
ON deadline_reminders_sent(tenant_id, exercise_id);

-- +goose Down
-- Reversa idempotente: se elimina la tabla y sus indices asociados.

DROP TABLE IF EXISTS deadline_reminders_sent;

