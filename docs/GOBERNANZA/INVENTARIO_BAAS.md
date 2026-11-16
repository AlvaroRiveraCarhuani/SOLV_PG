# Inventario de Capacidades de Endurecimiento BaaS — SOLV

> **Corte de auditoría:** 2026-11-16
> **Alcance:** 24 capacidades.
> **Estados:** `COMPLETADO` significa implementación presente y evidencia en el repositorio; `PARCIAL` significa implementación incompleta o prueba limitada; `PENDIENTE` significa que no hay implementación o no hay evidencia suficiente.

El inventario describe capacidades técnicas, no certificación operativa. El scrape de métricas, la emisión del certificado TLS y la restauración de respaldos se certifican en el [checklist pre-piloto](../SECURITY/pre-pilot-checklist.md), no en esta tabla.

---

## 1. Mapeo de capacidades

| # | Capacidad técnica | Descripción | Slice | ADR | Estado |
| :---: | :--- | :--- | :---: | :---: | :---: |
| **01** | Aislamiento de red host vía `DOCKER-USER` | Reglas de firewall en `DOCKER-USER` para evitar el bypass del firewall de Docker. | 8 | ADR-023 | `COMPLETADO` |
| **02** | Esquema académico multi-tenant unificado | Tablas de materias, inscripciones, entregas e invitaciones con filtrado por `tenant_id`. | 9 | ADR-024 | `COMPLETADO` |
| **03** | Perfiles Seccomp y AppArmor | Filtros de llamadas al sistema para limitar la superficie de los contenedores. | 9 | — | `PENDIENTE` |
| **04** | Multi-tenancy lógico con `tenant_id` | Columna `tenant_id` y middleware HTTP de aislamiento por inquilino. | 8 | ADR-018 | `COMPLETADO` |
| **05** | Autenticación ForwardAuth HttpOnly | `GET /api/v1/auth/verify` valida la cookie `solv_session` para Traefik. | 8 | ADR-017 | `COMPLETADO` |
| **06** | Modelo unificado de environments | Laboratorios e IDE efímeros en la tabla `environments` con discriminador de tipo. | 8 | ADR-019 | `COMPLETADO` |
| **07** | Sanitización y truncado de entradas y salidas | Límite de 64 KB en buffers `stdin`/`stdout` del evaluador. | 9 | — | `PENDIENTE` |
| **08** | Cuotas de disco por volumen | Medición por entorno, aviso al 80%, freno de entrega al 100%, limpieza de cachés y liberación de volúmenes sin dueño; falta la cuota dura XFS. | 9 | ADR-084 | `PARCIAL` |
| **09** | Rate limiting perimetral en Traefik | Limitación por IP y por tenant en el proxy. | 10 | — | `PENDIENTE` |
| **10** | Circuit breakers y fail-fast | Detección automática de saturación de Docker o PostgreSQL. | 10 | — | `PARCIAL` |
| **11** | Apagado ordenado | Captura de `SIGTERM`/`SIGINT` para cerrar servidor, workers y contenedores efímeros. | 10 | — | `PENDIENTE` |
| **12** | Registro de auditoría inmutable | Tabla `audit_logs` con actor, recurso, acción, estado y metadatos. | 10 | — | `COMPLETADO` |
| **13** | Pool adaptativo de conexiones | Ajuste dinámico del pool SQL de `sqlx`. | 11 | ADR-000 | `PARCIAL` |
| **14** | Warm pool de contenedores | Contenedores pre-iniciados en reserva para reducir latencia. | 11 | — | `PENDIENTE` |
| **15** | Caché y precarga de imágenes Docker | Servicio en segundo plano que precarga imágenes base. | 11 | — | `PENDIENTE` |
| **16** | Control de admisión por RAM | Rechazo de entornos cuando la memoria libre cae por debajo del 15 %. | 5 | ADR-008 | `COMPLETADO` |
| **17** | Hibernación dual por inactividad | Hibernación por heartbeat y telemetría. | 5 | ADR-011 | `COMPLETADO` |
| **18** | Penalización 3-Strikes OOMKilled | Enfriamiento de 5 minutos tras tres eliminaciones por RAM. | 5 | ADR-008 | `COMPLETADO` |
| **19** | Auto-tuning EWMA de recursos | Suavizado exponencial del consumo de RAM por plantilla. | 6 | ADR-014 | `COMPLETADO` |
| **20** | Blindaje inter-contenedor | Red Docker creada con `enable_icc=false`. | 6 | ADR-009 | `COMPLETADO` |
| **21** | Telemetría Prometheus `/metrics` | Endpoint protegido con métricas de host y contenedores. | 6 | — | `COMPLETADO` |
| **22** | Análisis semántico AST con Semgrep | Semgrep sobre el volumen del alumno y persistencia en `semgrep_audit`. | 7 | ADR-016 | `COMPLETADO` |
| **23** | TLS wildcard con DNS-01 | Desafío ACME con desec.io y Let's Encrypt para el dominio institucional. | 10 | ADR-018 | `PARCIAL` |
| **24** | Subdominios con UUID opaco | URL de acceso construida con el UUID del environment, sin datos del alumno. | 4 | ADR-004 | `COMPLETADO` |

**Resumen:** 14 completadas, 4 parciales y 6 pendientes.

---

## 2. Cambios de estado en esta revisión

| Capacidad | Estado anterior | Estado actual | Evidencia |
| :---: | :---: | :---: | :--- |
| **12** | `PENDIENTE` | `COMPLETADO` | Repositorio `audit_log_repository.go`, rutas `/api/v1/admin/audit-logs` y pantalla `/admin/auditoria`. |
| **19** | `PENDIENTE` | `COMPLETADO` | Actualización EWMA atómica en `lab_template_repository.go` y prueba de integración `slice06_ewma_autotuning_test.go`. |
| **20** | `PARCIAL` | `COMPLETADO` | Red creada con `com.docker.network.bridge.enable_icc=false` y aserción en `slice04_practice_environment_lifecycle_test.go`. |
| **22** | `PARCIAL` | `COMPLETADO` | Escaneo Semgrep, persistencia en `semgrep_audit` y flujos E2E de reglas AST en verde. |
| **23** | `PENDIENTE` | `PARCIAL` | `infra/traefik/traefik.yml` configura DNS-01 con desec; falta verificar emisión y renovación en el servidor. |
| **24** | `PENDIENTE` | `COMPLETADO` | `buildPracticeEnvironmentAccessURL` usa el UUID del environment como subdominio. |

---

## 3. Evidencia de las capacidades abiertas

| Capacidad | Qué existe | Qué falta |
| :---: | :--- | :--- |
| **03** | `no-new-privileges` y `CapDrop: ALL` en varios runners. | Perfiles Seccomp y AppArmor aplicados a los contenedores de práctica. |
| **07** | Límite de 64 MB para leer snapshots de volumen. | Truncado de `stdout`/`stderr` del evaluador a 64 KB; el límite actual no cubre la salida. |
| **08** | Barrido de medición cada 5 minutos (`disk_used_mb`, `disk_state`), cuota blanda en la política (`disk_quota_mb`), aviso al 80%, entrega rechazada con 409 al 100%, limpieza de cachés por entorno e inventario de volúmenes sin contenedor con frase y auditoría por volumen (`GET/POST /api/v1/admin/server/storage/orphans`). | Cuota dura: `StorageOpt size` no cubre volúmenes nombrados, así que hace falta un volumen XFS con `prjquota` y el asignador de proyecto por volumen. También falta separar `pg_data` del sistema de archivos de los entornos. |
| **09** | Rate limiting de aplicación en login, envíos, keystrokes y arranque de entornos. | Rate limiting perimetral en Traefik; el control actual vive en el backend. |
| **10** | Reacción manual de emergencia y reinicio del pool desde el panel admin. | Circuit breaker automático y fail-fast con umbrales medidos. |
| **11** | `http.Server` con timeouts de cabecera e inactividad. | Captura de señales, `Shutdown`, cierre de workers y limpieza de contenedores. |
| **13** | Pool SQL de `sqlx` y reset de conexiones idle desde emergencia. | Tuning dinámico de `MaxOpenConns`, `MaxIdleConns` y `ConnMaxLifetime`. |
| **14** | Creación de contenedores bajo demanda. | Contenedores pre-iniciados en reserva. |
| **15** | `PullImage` con deduplicación single-flight y pull bajo demanda. | Precarga en segundo plano de imágenes base. |
| **23** | Configuración ACME DNS-01 y script de registros desec. | Certificado emitido, renovación probada y fecha de expiración registrada. |

---

## 4. Lectura para el piloto

Las capacidades `COMPLETADO` no sustituyen las pruebas de operación. En particular:

- `/metrics` existe, pero el scrape de Prometheus y las notificaciones de alerta no están certificados.
- El respaldo tiene implementación y pruebas de integración, pero la restauración en una base temporal sigue pendiente.
- El rate limiting de aplicación está activo, pero la prueba de `429` en el servidor del piloto sigue pendiente.
- La configuración TLS existe, pero la emisión y renovación del certificado no se verificaron en esta auditoría.

Esos cuatro puntos figuran como bloqueos en el [checklist pre-piloto](../SECURITY/pre-pilot-checklist.md) y deben cerrarse con evidencia antes de iniciar el piloto.
