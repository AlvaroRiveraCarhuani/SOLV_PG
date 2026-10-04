# Tareas: Juez de ejercicios (v1 modo stdin)

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

Estimación: 1800–2200 líneas. División: PR1 → PR2 → PR3 → PR4 → PR5 → PR6. Entrega: auto-chain.

### Unidades de trabajo

| Unidad | Meta | Test enfocado | Reversión |
|--------|------|---------------|-----------|
| 1 | Migraciones 00009–00012, re-pin, semilla | `go test ./internal/infrastructure/database/` | Solo migraciones + semilla |
| 2 | Comparadores + fixtures congelados | `go test ./internal/core/services/comparators/` | Solo comparators |
| 3 | Build/run sin detención temprana + run_metrics | `go test ./...` | Solo servicio + runner |
| 4 | Perfiles + auditoría + validador | `go test ./... -run Perfil` | Solo perfiles |
| 5 | Dry-run + stale + referencia stdin | `POST dry-run` → 202, `GET job` con progreso | Solo dry-run + publish |
| 6 | Editor + candado + i18n + pinning | `ng test`, `npm run build` | Solo frontend + gates |

## Fase 1: Migraciones, re-pin y semilla (base de todo)

- [x] 1.1 Crear `backend/migrations/00009..00012_*.sql` (comparator_spec, reference/stale, profiles+audits con p95_window_days y checker_sidecar_image, run_metrics + dry_run_jobs).
- [x] 1.2 Fijar digests en `strategies/lang/lang_*.go` y semilla v0 (python/js 2000/256, cpp/c 1000/128, csharp 2500/256, java 3000/512). Aceptación: `rg ":latest" backend/internal/infrastructure/docker/strategies/lang/` → vacío.

## Fase 2: Comparadores (depende de 1.1)

- [x] 2.1 Crear `core/services/comparators/*.go` (registro + exact/unordered/float/custom). Aceptación: `go test ./internal/core/services/comparators/` → todos pasan.
- [x] 2.2 Fijar fixtures §8 (0.30000000004/0.3 AC triple; 2.000002/2.0 WA absolute y AC relative/hybrid; permutación AC; duplicado según flag).

## Fase 3: Ejecución y veredictos (depende de 1–2)

- [x] 3.1 Modificar `evaluation_service.go` (dos fases, todos los casos) y `base_runner.go` (TrimSpace solo en exact, OOM→MLE, CE build, VE checker). Aceptación: `rg` detención temprana → vacío.
- [x] 3.2 Registrar `run_metrics` por caso y p95 sobre AC con ventana `p95_window_days`. Aceptación: `go test ./...` → todos pasan.

## Fase 4: Perfiles y validador (depende de 1)

- [x] 4.1 Crear `language_profile_service.go` + handlers `delivery/http/` (GET público, PUT admin auditado; alias c++→cpp, c#→csharp; resto 400). Aceptación: contratos 400/409 con código máquina.
- [x] 4.2 Aplicar `ValidateRamAgainstHost` en alta/edición, evaluación, perfiles y dry-run. Aceptación: `rg ValidateRamAgainstHost backend/internal` → cuatro fronteras.

## Fase 5: Dry-run y publicación (depende de 3–4)

- [x] 5.1 Modelar `dry_run_jobs` (queued→running→done/failed, progreso por caso), referencia obligatoria stdin, edición marca stale y publish 409. Aceptación: `POST dry-run` → 202 con job; `GET job` muestra progreso.

## Fase 6: Editor (depende de 2 y 5)

- [ ] 6.1 Modificar `exercise-editor/` (parámetros visibles, panel dry-run con badge stale que bloquea publicar, sale `forceRecursion`, cableado blockNativeSort/blockSystemModules). Aceptación: `ng test` pasa; `npm run build` termina.
- [ ] 6.2 Extracción i18n EJ-* §4 (17 claves; EJ-ERR-RANGE interpola {campo}). Aceptación: sin faltantes ni sobrantes.

## Fase 7: Candado y gates (depende de 1–6)

- [ ] 7.1 Test candado: evaluación no acepta `template_id`. Aceptación: `rg template_id` en servicio + runner → vacío; test pasa.
- [ ] 7.2 Gates: `go test ./...` + `ng test` + pinning + i18n → todo en verde.
