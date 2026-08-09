# ADR-027 & ADR-028: Operabilidad B2B, Backups y Driver PostgreSQL

## Resumen de Decisiones del Slice 11

* **ADR-027:** [Operabilidad B2B — Audit Logs, Rate Limiting y Lock de Migraciones](../../ARQUITECTURA/ADR/ADR-027-operabilidad-b2b.md)
  * Rate limiting en memoria por usuario en `POST /api/v1/workspaces/start` (5 req/min) con cabeceras informativas RFC 6585.
  * Audit logs asíncronos con `AuditWorkerPool` (1000 slots, 5 workers fijos) y captura de `status_code`.
  * Lock consultivo `pg_advisory_lock(1337)` para ejecución limpia de migraciones entre réplicas.

* **ADR-028:** [Selección del Driver PostgreSQL (`lib/pq`) frente a `pgx`](../../ARQUITECTURA/ADR/ADR-028-driver-postgresql-libpq.md)
  * Mantenimiento de `github.com/lib/pq` + `jmoiron/sqlx` para estabilidad y cero riesgo de regresión en la defensa académica.
