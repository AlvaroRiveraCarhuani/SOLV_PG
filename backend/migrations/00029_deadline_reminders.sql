-- Migración 00029: Control de idempotencia para recordatorios de vencimiento académico (Slice 14 / Tarea T3)
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
