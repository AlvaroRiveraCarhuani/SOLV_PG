# Feature: Veracidad de la Telemetria del Dashboard de Salud

Objetivo: que cada numero de `/admin/dashboard` (Salud y Recursos) sea un dato medido, no una estimacion oculta ni un valor hardcodeado.
Branch: `feat/host-telemetry-truthfulness` (desde `main`) | Delivery strategy: ask-on-risk | Test runner: `go test ./...` & `npm run test:ci`

Forecast de lineas autoradas: ~1400 (backend ~850, frontend ~450, docs ~100). Supera el presupuesto de 400 → requiere decision de estrategia de entrega antes de `apply` (chained vs `size:exception`).

## Decisiones cerradas

- **D1 — Historial persistido.** El muestreo de host deja de vivir en memoria del handler HTTP y pasa a un worker que escribe en `host_metrics_samples` (1 fila/min/nodo, retencion 7 dias). Al arrancar, el grafico lee los ultimos 60 samples reales de la DB. Se elimina de raiz el sembrado sintetico de `admin_handler.go:87-104`. Motivo: sin Prometheus/Grafana en uso, el dashboard es la unica telemetria; un reinicio no puede borrar la evidencia de un OOM nocturno.
- **D2 — Dos estados, no uno.** La reconciliacion distingue `unknown` (no se pudo verificar: Docker no respondio) de `orphaned` (se verifico: el contenedor no existe).
  - `unknown`: no modifica datos, reintenta en el siguiente ciclo, alerta si persiste >5 ciclos, **retiene cupo** de `max_containers`.
  - `orphaned`: **libera cupo**, marca el registro para que el alumno recree el entorno.
  - No se reutiliza `failed`: en el dominio ya significa "el contenedor existio y se rompio" (OOM/crash).
- **D3 — Capacidad real unificada.** `estimated_max_containers = min(policies.MaxContainers, floor(availableMB / policies.RAMLimitMB))`, usando `MemAvailable` (via `HostMonitor.GetHostMemoryStats()`), sin el default `512` hardcodeado. La respuesta expone `capacity_basis` para que la UI explique cual limite manda.

## Specs

- **S1 — Telemetria detras de un puerto de dominio.**
  - Nuevo puerto `domain.HostTelemetry` con `Snapshot(ctx) (HostTelemetrySnapshot, error)` devolviendo: RAM usada/total/porcentaje, vCPUs, CPU%, disco `{path, used_bytes, total_bytes, percent}`, y `DockerInfo{server_version, storage_driver, healthy}`.
  - Implementacion en `backend/internal/infrastructure/system/host_telemetry.go` (gopsutil) compuesta con `backend/internal/infrastructure/docker/docker_info.go` (`ServerVersion()` + `Ping()`).
  - `delivery/http/admin_health.go` y `admin_handler.go` dejan de importar `gopsutil` directamente.
  - Criterio: `grep -r gopsutil backend/internal/delivery` no devuelve resultados.

- **S2 — Capacidad que no miente.**
  - `estimated_max_containers` se calcula segun D3; nunca menor que `running_labs`.
  - Respuesta expone `capacity_basis` (`"policy"` si manda `max_containers`, `"mem_available"` si manda la RAM), `max_containers_policy`, `ram_per_lab_mb`.
  - El default `512` en `admin_health.go:123` se elimina; `ram_per_lab_mb` sale de `policies.RAMLimitMB` (default de dominio 256/512 segun politica vigente).
  - Criterio: con 5.8 GB usados de 7.6 GB, el valor deja de ser `13`; y nunca contradice el gate real de admision de `workspace_service.go:340`.

- **S3 — Conteos por estado, tomados de la fuente correcta.**
  - Nuevo metodo de repositorio `CountWorkspacesByStatus(ctx, tenantID) (map[string]int, error)` con `GROUP BY status`.
  - `hibernated_labs`, `oom_killed_labs`, `unknown_labs` y `orphaned_labs` salen de ese conteo, no de filtrar `GetAllRunningWorkspaces` (que ya aplica `WHERE status = 'running'`).
  - Criterio: pausar un laboratorio incrementa `hibernated_labs`; hoy es estructuralmente imposible que sea distinto de `0`.

- **S4 — Historial real y persistente (D1).**
  - Migracion `backend/migrations/00028_host_metrics_samples.sql` (Up/Down): tabla `host_metrics_samples(id, sampled_at TIMESTAMPTZ NOT NULL, ram_percent, ram_used_gb, cpu_percent, active_containers, tenant_id)` con indice `(sampled_at DESC)`.
  - Worker `core/services/host_metrics_sampler.go`: cada 60 s toma `HostTelemetry.Snapshot()` + conteo de activos, inserta una fila y poda `sampled_at < now() - interval '7 days'`. Se arranca en `cmd/api/main.go` junto a `zombieCollector`.
  - `delivery/http/admin_handler.go`: se eliminan `loadHistory`, `historyMu`, `initHistoryBuffer` y `startHistoryTicker`. `getHistorySnapshots` pasa a leer del repositorio con ventana configurable (`?window=60m|24h`, default `60m`, tope 10080 min).
  - Criterio: reiniciar el backend no inventa historia y el grafico arranca con los ultimos samples reales.

- **S5 — Reconciliacion bidireccional con dos estados (D2).**
  - `core/services/workspace_state_reconciler.go`: para cada fila cuyo estado este en `('running','pending')`, verificar contra `ListAllManagedContainers`.
    - Docker responde y el contenedor no existe → `orphaned`.
    - Docker falla/timeout → `unknown` (sin tocar el resto del registro), contador de ciclos fallidos por workspace.
    - Docker responde y el contenedor esta `paused`/`exited` → `hibernated`/`failed` segun corresponda.
  - Se ejecuta en el ciclo de `ZombieCollectorWorker` (30 s) reutilizando su `singleflight`/mutex. Registra en auditoria cada transicion.
  - `GetActiveWorkspaces` (gate de cupo) incluye `'unknown'` en la whitelist `IN ('running','pending','unknown')` para retener cupo; `'orphaned'` queda fuera y libera.
  - Criterio: matar un contenedor a mano produce `orphaned` en el siguiente ciclo; detener el daemon de Docker produce `unknown` y no destruye ningun registro.

- **S6 — Salud real, con motivo.**
  - `health_status` pasa a `operational | degraded | critical`, calculado como: `critical` si `DockerInfo.healthy == false` o hay `orphaned_labs` > 0; `degraded` si `oom_killed_labs > 0` o `unknown_labs > 0` o RAM sobre el techo de seguridad; `operational` en caso contrario.
  - La respuesta incluye `health_reasons: string[]` con codigos estables y `docker: {server_version, storage_driver, healthy}`.
  - Criterio: `docker stop` con la DB limpia lleva el badge a `critical`; hoy dice "Operativo y Estable".

- **S7 — Frontend alineado al contrato real.**
  - `admin-metrics.service.ts`: elimina el literal `'Docker Engine v27.1.1 (overlay2)'` (linea 109) y el bloque `buildLoadHistoryFallback`; consume `docker`, `capacity_basis`, `health_reasons`, `disk_path`.
  - `hardware-kpi.component.ts`: footer de disco renderiza `disk_path` real; tarjeta de concurrencia muestra `running / estimated_max_containers` con subtitulo que explica el `capacity_basis`; footer de hibernados muestra el conteo real.
  - `load-history-chart.component.ts`: umbral leido de `host_safety_ceiling_pct` (se eliminan los tres `85` hardcodeados en lineas 41, 61 y 73); eje X proporcional al timestamp real; tooltip calcula el delta desde `snapshot.timestamp`, no `59 - index`; estado vacio explicito ("Recopilando telemetria") cuando hay menos de 2 puntos.
  - Badge de salud renderiza `health_status` + tooltip con `health_reasons` traducidos.
  - Estados `unknown` y `orphaned`: se agregan al union de `DockerContainerDTO.status` y a `admin_incidents.go:120` (hoy un estado desconocido se mapea a `hibernated`); tokens semanticos neutro y ambar respectivamente.
  - Cumplimiento: cero hex en SCSS, solo tokens (`--tenant-*`, `--color-*`, `--border-color`); fechas con `dateText` dentro de `<time>`; valores tecnicos con `[machineData]`; selectores kebab-case sin prefijo `solv-`.

- **S8 — Verificacion y documentacion indivisible.**
  - `cd backend && go vet ./... && go test ./...`; `cd frontend && npm run lint:styles && npm run lint:ts && CI=true npm run build && npm run test:ci`.
  - Nueva `docs/ARQUITECTURA/ADR/ADR-049-telemetria-de-host-persistida-y-estados-de-reconciliacion.md` documentando D1, D2 y D3 con sus alternativas descartadas.
  - Actualizar contrato en `docs/API_CONTRACTS.md` (`GET /api/v1/admin/metrics/health` con sus campos nuevos) y `docs/OBSERVABILITY/alerts.md` (origen de alertas `unknown`/`orphaned`).

## Tasks

- [x] **T1** (S1): Crear puerto `domain.HostTelemetry` e implementaciones `infrastructure/system/host_telemetry.go` y `infrastructure/docker/docker_info.go`; migrar `delivery/http` a consumir el puerto. Tests unitarios con dobles. Route: inline. Commit: `feat(telemetry): add host telemetry port`
  - Estado: **cerrado**. Puerto, ambas implementaciones, 4 tests, wiring en `cmd/api/main.go` (`adminHandler.SetTelemetry(system.NewSystemTelemetry("/var/lib/docker", dockerClient)))`) y `admin_health.go` limpio de gopsutil. `delivery/http` ya no lee `/proc` ni consulta Docker por su cuenta; el gopsutil restante en `admin_handler.go` es del historial en memoria que T4 va a eliminar. Sin commit (pedido del usuario).
- [x] **T2** (S2): Recalcular `estimated_max_containers` con `min(policy, mem_available/ram_per_lab)` y exponer `capacity_basis`, `max_containers_policy`, `ram_per_lab_mb`. Tests de la formula con 0, 1 y N labs. Route: inline. Commit: `fix(capacity): compute max containers from available memory and policy cap`
  - Estado: **cerrado**. `domain/capacity.go` + 4 tests. `admin_health.go` consume `ServerPoliciesService` en vez de parsear `tenant.Config` (duplicacion eliminada). Sin commit.
- [x] **T3** (S3): Agregar `CountWorkspacesByStatus` al repositorio y reconstruir `hibernated/oom/unknown/orphaned_labs` sobre el conteo real. Tests de integracion con los cuatro estados. Route: inline. Commit: `fix(metrics): count workspaces by status instead of filtering running rows`
  - Estado: **cerrado de verdad**. `CountWorkspacesByStatus` en el repositorio Postgres; interfaz declarada en el consumidor para no tocar `domain/ports.go`. 3 tests unitarios del consumidor (agregado, fallo, ausencia) **más 3 tests de integración reales contra `solv_test`** en `tests/integration/slice17_workspace_status_counts_test.go`: desglose completo de 4 estados, aislamiento entre tenants y conteo real. Estos prueban el SQL `GROUP BY` real, no un mock. Requiere `getTestDSN()` (default `solv_test` local) y un subject sembrado por FK. Sin commit.
- [x] **T4** (S4): Migracion `00028`, `host_metrics_sampler.go` con retencion de 7 dias, lectura de historial desde DB en `admin_health.go`, y borrado de `initHistoryBuffer`/`startHistoryTicker`/`loadHistory`. Tests del sampler y de la ventana `60m|24h`. Route: delegated. Commit: `feat(telemetry): persist host metrics samples and drop seeded history`
  - Estado: **cerrado**. Migracion `00028_host_metrics_samples.sql` (goose, verificado aplicado contra solv_test), `domain/host_metrics_sample.go` con `HostMetricsSample` + `HostMetricsSampleStore` + constantes, repositorio `infrastructure/storage/postgres/host_metrics_sample_repository.go`, worker `core/services/host_metrics_sampler.go` (1 fila/min, poda 7 dias, muestra inicial al arrancar) y wiring en `cmd/api/main.go` (`NewHostMetricsSampler` + `SetMetricsSampleStore`). `admin_handler.go` quedó sin gopsutil, sin `historyMu`/`loadHistory` y con `getHistorySnapshots` leyendo del store con `?window=60m|24h` (default 60m, tope 7 dias, acepta `N d`). 4 tests del sampler (persistencia, fail-open del contador, telemetria caida sin insert, error de insert) + 3 tests del handler (window defaults/24h/tope y lectura con tenant). Sin commit.
- [x] **T5** (S5): Implementar `workspace_state_reconciler.go` con `unknown`/`orphaned`, integrarlo al ciclo de 30 s y ampliar la whitelist de cupo en `GetActiveWorkspaces`. Tests de los tres escenarios (Docker ok / Docker caido / contenedor ausente) con dobles, sin tocar el daemon real. Route: delegated. Commit: `feat(reconcile): detect drifted workspace state with unknown and orphaned statuses`
  - Estado: **cerrado**. `core/services/workspace_state_reconciler.go` con estados `unknown`/`orphaned` (nuevos en `domain/workspace.go`), recuperacion unknown→running, salida limpia→hibernated (remueve contenedor) y exit 137→vía canónica `IncrementOOMStrike`; alerta de auditoria unica al cruzar 5 ciclos (`WORKSPACE_RECONCILED_UNKNOWN_PERSISTENT`), transiciones auditadas (ActorID `system`). Integrado al ciclo de 30 s del `ZombieCollectorWorker` (`SetStateReconciler`, dentro de su mutex). `GetActiveWorkspaces` ahora usa `IN ('running','pending','unknown')`. 7 tests con dobles (sin daemon real): 3 escenarios especificados + recuperacion, salida limpia y 137. Sin commit.
- [x] **T6** (S6): Calcular `health_status` y `health_reasons` desde Docker ping, reconciliacion y OOMs. Tests de la matriz de transicion. Route: inline. Commit: `fix(health): derive health status from docker daemon and reconciliation`
  - Estado: **cerrado**. `domain/health.go` con `EvaluateHealth` puro (operational/degraded/critical y 5 codigos estables de motivo), consumido por `admin_health.go` con `unknown_labs`/`orphaned_labs`/`health_reasons` nuevos en la respuesta y `ramPercent` medido (default techo 85 si la politica no trae). 9 tests de matriz en `domain/health_test.go` (nominal, docker caido, orphaned, oom, unknown, RAM sobre techo, techo por defecto, condiciones apiladas, degraded no critical). Sin commit.
- [x] **T7** (S7): Ajustar `admin-metrics.service.ts`, `hardware-kpi.component.ts`, `load-history-chart.component.ts` y el badge de salud al contrato nuevo; agregar estados `unknown`/`orphaned` al DTO y a `admin_incidents.go`. Tests de componente con Signals y de los limites del grafico. Route: delegated. Commit: `feat(dashboard): render measured telemetry with dynamic thresholds`
  - Estado: **cerrado**. Service sin literal de Docker ni `buildLoadHistoryFallback`; consume `docker`, `health_status`, `health_reasons`, `capacity_basis`, `disk_path`. `hardware-kpi` con ruta de disco medida y `running / estimated_max_containers` + subtitulo de `capacity_basis`. `load-history-chart` con umbral por input (se eliminaron los tres 85), eje X proporcional a timestamps reales, tooltip con delta desde `snapshot.timestamp` y estado vacio "Recopilando telemetria". Badge de salud con estado y tooltip de razones traducidas. Estados `unknown`/`orphaned` en el union de `DockerContainerSummary.status`, mapeo de `admin_incidents.go` (ya no colapsan a `hibernated`) y `status-badge` (neutro/ambar con tokens). Specs nuevos: hardware-kpi (7) y load-history-chart (6, limites del grafico); suite completa 105/105 archivos y 785 tests + gates en verde; `lint:styles` y build OK. Sin commit.
- [x] **T8** (S8): Ejecutar la matriz de verificacion, escribir `ADR-049` y actualizar `API_CONTRACTS.md` y `OBSERVABILITY/alerts.md`. Route: inline. Commit: `docs(adr): document persisted host telemetry and reconciliation states`
  - Estado: **cerrado**. ADR-049 (D1/D2/D3 + alternativas descartadas + consecuencias) y entrada en `INDEX.md`; seccion nueva en `API_CONTRACTS.md` (Slice 17, `GET /api/v1/admin/metrics/health` con todos los campos nuevos y tabla de estados) y alerta nueva en `OBSERVABILITY/alerts.md` (origen `unknown`/`orphaned`, codigos de motivo). Matriz de verificacion: `go vet ./...` OK, `go test ./...` OK salvo `TestSemgrepPrecheckSuite` (CLI semgrep del host roto por opentelemetry/Python 3.14, ajeno a este feature), integracion contra `solv_test` OK incluida la migracion `00028`; frontend `lint:styles` OK, `lint:ts` con 6 errores pre-existentes en archivos ajenos (template-studio, ai-exercise-generator-modal, teacher-ai-copilot), `CI=true ng build` OK, `npm run test:ci` OK (785 + 26 tests de gates). Sin commit.

## Log

- **L1** (original, verbatim): "una duda, ahroa mismo quiero saber si esas kpis realmente son reales, y su grafico de abajo y toda esta pantalla, en sentido de que dice 0 / 13 Activos, pero no hay ningun activo, de donde sale ese 13 , verifica tambien los demas"
- **L2** (verbatim): "con esos 3 fixes cubririamos todos esos datos de la pantalla o que"
- **L3** (verbatim): "dale, pero coom solucionaremos todas, ,prefiero que me menciones todas y la asolucion de punta, para despues ponerlas en odd como tareas, y asi vasmos despues avanzando, ahroa mismo reevalua tu solucion y menciame si realmente son de punta"
- **L4** (verbatim): "que decisioens son mias, no entendi bien, y en que afecta cada cosa"
- **L5** (verbatim): "Si pero grafana y prometheus no se usan xdd. para ti cual es la recomendacion de punta para estos 2 puntos. Quiero que me recomiendes coss"
- **L6** (verbatim): "dale, documentalo con las desiciones, me convenciste con tus 2 propuestas, asi que primeramente documenta las tareas odd, avisame"
- **E1** — Hallazgos verificados en codigo: `estimated_max_containers` = `allocatableMB/512` sobre el techo de 85% sin descontar uso (`admin_health.go:122-131`); `hibernated_labs` estructuralmente 0 porque `GetAllRunningWorkspaces` ya filtra `status='running'` (`workspace_repository.go:222`); `docker_version` hardcodeado (`admin-metrics.service.ts:109`); historial sembrado con factor sintetico (`admin_handler.go:87-104`); umbral 85 hardcodeado en el grafico (lineas 41/61/73); reconciliacion solo Docker→DB (`zombie_collector.go`); `gopsutil` importado desde `delivery/http`.
- **E2** — Gate real de admision es `policies.MaxContainers` (default 40) en `workspace_service.go:340`, no la estimacion por RAM. El dashboard desinformaba en ambos sentidos.
- **E3** — `workspaces.status` es `VARCHAR(50)` sin CHECK (`00001_baseline.sql:242`): agregar estados no requiere migracion de constraint, pero si tocar `GetActiveWorkspaces` y el mapeo de `admin_incidents.go:120`.
- **E4** — Backend corre como binario en el host (`docs/DEPLOYMENT.md:85`), por lo que `mem.VirtualMemory()` y `disk.Usage("/")` son valores reales del servidor fisico.
- **N1** — Siguiente paso: decidir estrategia de entrega (chained PRs vs `size:exception`) antes de `T1`, por el forecast de ~1400 lineas.
