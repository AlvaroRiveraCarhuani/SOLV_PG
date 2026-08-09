# ADR-027: Operabilidad B2B — Audit Logs, Rate Limiting y Lock de Migraciones

## Estado
Aprobado e Implementado (Slice 11 Parte 1).

## Contexto
Para el despliegue multi-institucional y BaaS-ready de SOLV, se requieren mecanismos de operabilidad, trazabilidad de acciones administrativas/docentes y defensa contra abusos de recursos informáticos en entornos multi-tenant.

## Decisiones de Arquitectura

### 1. Rate Limiting por Usuario en Workspaces (`CRIT-10`)
- **Implementación:** Limitador de tasa en memoria (`sync.Map` de `user_id` a `*rate.Limiter`) con cubeta de fichas (token bucket) configurado a 5 solicitudes por minuto.
- **Cabeceras Informativas RFC 6585:** En cada respuesta HTTP a `POST /api/v1/workspaces/start` se retornan las cabeceras `X-RateLimit-Limit`, `X-RateLimit-Remaining` y `X-RateLimit-Reset`. Si se supera el límite, se responde HTTP `429 Too Many Requests` con cabecera `Retry-After: 60`.
- **Limitación:** Al ser un limitador en memoria, el estado no se comparte entre réplicas en despliegues distribuidos multi-nodo.
- **Perspectiva Futura:** La arquitectura permite abstraer una interfaz `RateLimiter` para desacoplar el motor en memoria y reemplazarlo por un almacén distribuido (Redis / KeyDB) en la fase multi-nodo.

### 2. Audit Logs Asíncronos con Worker Pool (`CRIT-11`)
- **Trazabilidad:** Registro automático de mutaciones HTTP (`POST`, `PUT`, `DELETE`) ejecutadas por roles `teacher` y `admin` en la tabla `audit_logs`.
- **Status Code:** Inclusión y persistencia del código de estado HTTP (`status_code`) mediante un envoltorio `statusCapturingResponseWriter`.
- **Worker Pool:** Procesamiento asíncrono compuesto por un canal bufferizado de 1000 ranuras y 5 workers concurrentes fijos. Si la cola se satura, el log se descarta registrando una advertencia en los logs sin bloquear la respuesta HTTP.

### 3. Lock de Migraciones (`CRIT-19`)
- **Coordinación:** Uso del bloqueo consultivo a nivel de sesión PostgreSQL `pg_advisory_lock(1337)` al arrancar `RunInitialMigrations()`.
- **Exclusión Mutua:** Garantiza la ejecución limpia e indivisible de las migraciones esquema cuando múltiples réplicas o instancias del backend inician simultáneamente.

## Consecuencias
- **Positivas:** Trazabilidad completa de operaciones docentes, protección contra agotamiento de recursos del host Asus y arranque coordinado entre réplicas.
- **Riesgos:** Pérdida eventual de logs de auditoría únicamente bajo saturación extrema del buffer (1000 eventos pendientes).
