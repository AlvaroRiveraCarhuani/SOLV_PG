# ADR-049: Telemetría de Host Persistida y Estados de Reconciliación (`unknown` / `orphaned`)

- **Estado:** Aprobado
- **Fecha:** 2026-10-27
- **Contexto:** Core / Observabilidad y Reconciliación de Workspaces

## Contexto y Problema

Una duda operativa ("¿esas KPIs realmente son reales?") destapó que el panel de salud y recursos del dashboard de administración `GET /api/v1/admin/metrics/health` decoraba la realidad en lugar de medirla:

1. **Historial sembrado en memoria.** `admin_handler.go` inicializaba 60 puntos con un factor sintético (`initHistoryBuffer`, `startHistoryTicker`) para que el gráfico "tuviera algo que dibujar". Un reinicio borraba la evidencia de un OOM nocturno y el gráfico inventaba tendencias que jamás ocurrieron. No hay Prometheus ni Grafana en uso: el dashboard es la única telemetría del nodo.
2. **Reconciliación de una sola dirección.** El `ZombieCollectorWorker` (ciclo 30 s) detectaba contenedores sin registro en PostgreSQL y los destruía, pero **nadie** detectaba el caso inverso: registros en `running` cuyo contenedor había desaparecido del daemon. Un `docker stop` manual o un crash del daemon dejaba el registro mintiendo.
3. **Capacidad estimada con datos falsos.** `estimated_max_containers = allocatableMB / 512` usaba el techo absoluto de RAM sin descontar el uso presente, ignoraba `max_containers` (el gate real de la admisión, `workspace_service.go:340`) y tenía un default `512` hardcodeado. Con 5.8 GB usados de 7.6 GB informaba `13` contenedores cuando el hardware jamás alojaría esa cifra junto al uso actual.
4. **Conteo de hibernados estructuralmente cero.** `hibernated_labs` se derivaba de `GetAllRunningWorkspaces`, que ya filtra `WHERE status = 'running'`: preguntarle por hibernados a una lista de running siempre devolvía 0.
5. **Literales en la interfaz.** La versión de Docker (`'Docker Engine v27.1.1 (overlay2)'`), la ruta de disco (`/var/lib/docker overlay2`) y el umbral del gráfico (`85`) eran literales embebidos en el frontend; el umbral además ignoraba el techo de política vigente por tenant.

## Decisión

### D1 — Historial persistido del host (worker + tabla `host_metrics_samples`)

- El muestreo deja de vivir en la memoria del handler HTTP y pasa a un worker `core/services/host_metrics_sampler.go` que cada 60 s toma `HostTelemetry.Snapshot()` (gopsutil + daemon Docker, puerto `domain.HostTelemetry` del ADR de telemetría) más el conteo de contenedores activos, inserta **1 fila** en `host_metrics_samples` y poda lo anterior a **7 días** (retención). Se arranca en `cmd/api/main.go` junto al `zombieCollector`.
- Al cargar el panel, el gráfico lee los últimos N samples reales desde la base (`ListSince` con ventana `?window=60m|24h`, default 60m, tope = retención). **Un reinicio del backend ya no inventa historia**: el gráfico arranca con la telemetría real acumulada.
- Se elimina de raíz el sembrado sintético (`initHistoryBuffer`, `startHistoryTicker`, `loadHistory`, `historyMu`) y el `buildLoadHistoryFallback` del frontend.
- Los campos de cada sample mantienen escala y contrato del DTO anterior (`timestamp` RFC3339, `ram_percent`, `ram_used_gb`, `cpu_percent`, `active_containers`) para no romper el gráfico existente.

**Alternativa descartada (registrada):** mantener el buffer en memoria con persistencia perezosa en shutdown. Se descartó porque un `kill -9` o una caída de energía (los dos escenarios que precisamente importan tras un OOM nocturno) no ejecutan shutdowns limpios. La escritura por worker con poda escribe el hecho antes de que exista un consumidor.

### D2 — Reconciliación bidireccional con dos estados (`unknown` y `orphaned`)

El nuevo `core/services/workspace_state_reconciler.go` corre dentro del ciclo de 30 s del `ZombieCollectorWorker` (reutilizando su mutex contra solapamientos) y para cada workspace activo con `container_id` verifica contra `ListAllManagedContainers`:

- **`unknown`** — el daemon no respondió (error/timeout de la lista o de la inspección). No destruye nada: marca el estado (para que el panel lo refleje), **retiene el cupo** de `max_containers` y acumula un contador de ciclos fallidos por workspace; al cruzar 5 ciclos (~2.5 min) emite una alerta de auditoría (`WORKSPACE_RECONCILED_UNKNOWN_PERSISTENT`) una sola vez. Al recuperarse el daemon y encontrarse el contenedor, el registro vuelve a `running`.
- **`orphaned`** — el daemon respondió y el contenedor ya no existe. **Libera cupo** (queda fuera de `GetActiveWorkspaces`) y deja el registro marcado para que el alumno lo recree con restart.
- **Brotes intermedios:** un contenedor existente pero fuera de marcha con exit 137/OOM se enruta por la vía canónica (`IncrementOOMStrike`, que fija `oom_killed` y suma strike, preservando la penalización de 3 strikes); una salida limpia retira el contenedor y marca `hibernated`. La pausa intencional (`docker pause`) no cambia `State.Running` en el SDK: queda fuera del alcance porque el `QoSOrchestratorWorker` ya la gestiona.
- `workspaces.status` es `VARCHAR(50)` sin CHECK (baseline `00001`): los estados nuevos no requieren migración de constraint. Solo se toca la whitelist de `GetActiveWorkspaces` (`IN ('running','pending','unknown')`).

**Alternativa descartada (registrada):** reutilizar `failed` para el contenedor ausente. Se rechazó por semántica: en el dominio `failed` significa "el contenedor existió y se rompió (OOM/crash)". Confundir "nunca se pudo verificar" (unknown) y "el entorno físico desapareció" (orphaned) con un fallo del proceso ocultaría en el panel la diferencia entre un daemon caído y un contenedor muerto, que exigen acciones distintas.

### D3 — Capacidad real unificada (`capacity_basis`)

`estimated_max_containers = min(policies.MaxContainers, floor(availableMB / policies.RAMLimitMB))` usando `MemAvailable` medido (`HostTelemetry.Snapshot()`), **sin** el default 512 hardcodeado y nunca por debajo de `running_labs`. La respuesta expone:

- `capacity_basis` = `"policy"` (manda `max_containers` o no se pudo medir RAM) | `"mem_available"` (manda la RAM disponible).
- `max_containers_policy` y `ram_per_lab_mb` para que la UI explique el número con el techo que la admisión realmente aplica.

### Estados de salud con motivo (`health_status` + `health_reasons`)

El veredicto pasa de `healthy`/`warning` (heurístico) a `operational | degraded | critical` calculado por `domain.EvaluateHealth` sobre hechos medidos:

- `critical` si Docker no responde (`docker.healthy == false`) o `orphaned_labs > 0`.
- `degraded` si `oom_killed_labs > 0`, `unknown_labs > 0` o la RAM del host supera el techo de seguridad de la política.
- `operational` en caso contrario, con lista de motivos vacía.

La respuesta incluye `health_reasons: string[]` con códigos estables (`docker_unreachable`, `orphaned_workspaces`, `oom_killed_workspaces`, `unknown_workspaces`, `ram_above_safety_ceiling`) y el bloque `docker: {server_version, storage_driver, healthy}` medido del daemon.

### Frontend: consumir el contrato, no literales

- `admin-metrics.service.ts` elimina el literal de Docker y `buildLoadHistoryFallback`: el estado del panel (`operational|degraded|critical`), `docker`, `health_reasons`, `capacity_basis` y `disk_path` vienen del backend.
- `hardware-kpi` muestra la ruta de disco medida y la concurrencia `running / estimated_max_containers` con subtítulo que explica `capacity_basis`.
- `load-history-chart` lee el umbral de `host_safety_ceiling_pct` (se eliminan los tres 85), calcula el delta del tooltip desde el timestamp real del sample y muestra estado vacío explícito ("Recopilando telemetría") con menos de 2 puntos.
- Estados `unknown` (neutro) y `orphaned` (ámbar) en el union del DTO de contenedores, en `admin_incidents.go` (ya no colapsan a `hibernated`) y en el `status-badge` compartido con tokens semánticos.

## Consecuencias

- **Positivas:** cada número del panel es un dato medido; el historial sobrevive reinicios; la plataforma detecta contenedores desaparecidos y daemon caído en ≤ 30 s sin destruir registros; el cupo informado coincide con el gate de admisión real; la UI explica el porqué de la capacidad y de la salud.
- **Neutras:** el contrato de `load_history` mantiene los mismos campos; la frecuencia de muestreo (1/min) y retención (7 d) caben en una tabla simple; los despliegues single-tenant (el actual) no cambian de comportamiento observable salvo en la veracidad de los datos.
- **Negativas / impacto esperado:** la reconciliación ahora escribe `unknown`/`orphaned` en la base (con su auditoría); el worker añade una escritura y una poda SQL por minuto (costo trivial en este volumen); los tests de integración nuevos requieren la base `solv_test` igual que el resto de slices.

## Relación con Otros ADRs

- **ADR-008 / ADR-043** (Recursos y QoS): el `QoSOrchestratorWorker` sigue gestionando hibernación y OOM; el reconciliador solo cierra la brecha de estados que el ciclo de 15 min dejaba abierta.
- **ADR-030 / ADR-046** (Catálogo de plantillas): el conteo de activos del sampler usa el repositorio de workspaces, no catálogo.
- **ADR-042** (Auditoría): las transiciones del reconciliador quedan registradas con `Action` estable y `Metadata` con estado origen/destino para trazabilidad time-travel.
- **Alerts (docs/OBSERVABILITY/alerts.md):** los códigos de `health_reasons` son los mismos códigos de alerta operativa documentados.