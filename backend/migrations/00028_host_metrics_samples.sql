-- +goose Up
-- Historial persistido de la telemetria del host (D1, ADR-049).
--
-- Hasta ahora el grafico del panel de salud vivia en la memoria del handler
-- HTTP: lo sembraba con un factor sintetico al arrancar y un reinicio borraba
-- la evidencia de un OOM nocturno. Con esta tabla el muestreo pasa a un worker
-- que escribe 1 fila/minuto y poda lo anterior a 7 dias; el grafico lee los
-- ultimos samples reales desde la base.
--
-- tenant_id viaja en la fila para que el panel multi-tenant lea solo su
-- historial; hoy la plataforma corre single-node/single-tenant, asi que el
-- worker escribe bajo el tenant por defecto del nodo.
--
-- El indice (sampled_at DESC) sirve a la lectura "-N minutes" del panel y a la
-- poda por ventana de retencion.

CREATE TABLE IF NOT EXISTS host_metrics_samples (
    id                BIGSERIAL PRIMARY KEY,
    sampled_at        TIMESTAMPTZ NOT NULL,
    ram_percent       DOUBLE PRECISION NOT NULL,
    ram_used_gb       DOUBLE PRECISION NOT NULL,
    cpu_percent       DOUBLE PRECISION NOT NULL,
    active_containers INTEGER NOT NULL DEFAULT 0,
    tenant_id         TEXT NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
);

CREATE INDEX IF NOT EXISTS idx_host_metrics_samples_sampled_at
    ON host_metrics_samples (sampled_at DESC);

-- +goose Down
-- Reversa idempotente: la tabla se elimina completa junto con su indice.

DROP TABLE IF EXISTS host_metrics_samples;