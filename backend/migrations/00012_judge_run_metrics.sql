-- +goose Up
-- Juez de ejercicios: telemetria por caso y trabajos dry-run asincronos.
-- run_metrics registra por caso: lenguaje en clave canonica, digest de imagen,
-- duration_ms y veredicto (base del p95 sobre AC con ventana p95_window_days).
-- dry_run_jobs modela queued -> running -> done | failed con progreso por caso.

CREATE TABLE IF NOT EXISTS run_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exercise_id UUID REFERENCES exercises(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    image_digest TEXT NOT NULL DEFAULT '',
    duration_ms INT NOT NULL DEFAULT 0,
    verdict TEXT NOT NULL,
    case_index INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_run_metrics_lang_verdict ON run_metrics(language, verdict, created_at);

CREATE TABLE IF NOT EXISTS dry_run_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exercise_id UUID NOT NULL REFERENCES exercises(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'queued',
    progress_current INT NOT NULL DEFAULT 0,
    progress_total INT NOT NULL DEFAULT 0,
    result JSONB NOT NULL DEFAULT '{}'::jsonb,
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_dry_run_jobs_status CHECK (status IN ('queued', 'running', 'done', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_dry_run_jobs_exercise ON dry_run_jobs(exercise_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_dry_run_jobs_exercise;
DROP TABLE IF EXISTS dry_run_jobs;
DROP INDEX IF EXISTS idx_run_metrics_lang_verdict;
DROP TABLE IF EXISTS run_metrics;
