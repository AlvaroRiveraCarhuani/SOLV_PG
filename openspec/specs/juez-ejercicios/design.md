# Diseño: juez de ejercicios (motor de evaluación, v1 modo stdin)

La especificación `spec.md` de este módulo es la única fuente de decisiones. Estrategia: comparadores como registro de funciones puras, ejecución build+run sin detención temprana, perfiles en tabla versionada con auditoría, imágenes fijadas por digest y métricas por caso. Alcance v1: solo stdin; función hereda el ciclo con su harness posterior, sin publicación en v1.

## Enfoque técnico

```
Envío ──→ AST + Semgrep ──→ build ──→ run por caso ──→ comparador ──→ veredictos + run_metrics
Docente ──→ publish / dry-run / profiles ──→ servicio ──→ exercises / dry_run_jobs / language_profiles
```

Lógica en `core/`; Docker, Semgrep y SQL en `infrastructure/`; HTTP en `delivery/http`. Se conservan cookie de sesión, `tenant_id`, workspaces efímeros, Semgrep previo y veto a `:latest`.

## Decisiones de arquitectura

| Opción | Descartada | Decisión |
|---|---|---|
| Digest (`nombre@sha256:…`) en `lang_*.go` y perfiles | Tags mutables | Pinning obligatorio; `:latest` prohibido en runners, modelos y perfiles |
| Tabla versionada servida por endpoint | Config estática | `language_profiles` + `language_profile_audits` (autor, valores, motivo) |
| Validador único de dominio | Validadores por capa | `ValidateRamAgainstHost` en las cuatro fronteras; la prueba interactiva no lo invoca |
| Clave canónica en escritura | Alias persistidos | `c++→cpp`, `c#→csharp`; perfiles rechazan alias (400) |
| v1 solo stdin | Función en v1 | Sin publicación en modo función hasta su harness |
| `TrimSpace` solo en `exact` | Recorte global | Resto de comparadores sin recorte previo |

Toda calibración previa al pinning es inválida. Re-pin: registrar digests en `lang_*.go` y semilla en la misma ventana; la base queda pendiente de revisión.

## Alias y reglas

| Entrada | Canónica |
|---|---|
| `c++` | `cpp` |
| `c#`, `cs` | `csharp` |
| `c`, `python`, `java`, `javascript`, `cpp` | sin cambio |

Normalizar en: alta/edición, evaluación, perfiles, `run_metrics`, dry-run. `blockNativeSort→ForbiddenFunctions`, `blockSystemModules→ForbiddenImports`, antes de `ValidateCode` y `forbidden.yaml`. `forceRecursion` sale de la UI; el backend lo ignora.

## Flujos

Publicación (`409` si stale o falta referencia):

```
Docente ──PUT publish──→ handler ──→ servicio ──→ exercises
```

Dry-run (`queued → running → done | failed`, con progreso `índice/total/veredicto`):

```
Docente ──POST dry-run──→ 202 job_id ──→ worker ──por caso──→ runner
   │──GET job──→ {status, progreso}   Editar casos/límites/comparador/referencia ──→ stale
```

Evaluación (todos los casos, sin detención temprana):

```
Envío ──→ AST/Semgrep ──→ build (fallo = CE) ──→ run ──→ comparador ──→ veredictos + run_metrics
```

## Cambios de archivos

| Archivo | Acción | Descripción |
|---|---|---|
| `backend/migrations/00009..00012_*.sql` | Crear | Ver despliegue |
| `core/domain/exercise.go` | Modificar | Veredictos MLE, VE, CE; resultado por caso |
| `core/services/comparators/*.go` | Crear | Registro + exact, unordered, float, custom |
| `core/services/evaluation_service.go` | Modificar | Dos fases, sin detención temprana |
| `infrastructure/docker/.../base_runner.go` | Modificar | Build/run, OOM→MLE, métricas por caso |
| `infrastructure/docker/.../lang_*.go` | Modificar | Digests fijados |
| `core/services/language_profile_service.go` | Crear | Perfiles + auditoría |
| `delivery/http/` (handlers, dto) | Modificar | Payloads, dry-run 202, códigos máquina |
| `exercise-editor/` y pasos | Modificar | Parámetros del comparador, progreso dry-run, badge stale; sale `forceRecursion` |

## Contratos

`comparator_spec` JSONB (`{"id":"exact"}`), override por caso en `config`; `reference_solution`; `run_metrics(...)`; trabajos con progreso. Códigos: comparador desconocido, rango inválido, referencia ausente, publicación bloqueada, checker inválido.

```sql
SELECT percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ms) FROM run_metrics
WHERE language = $1 AND verdict = 'AC' AND created_at > now() - $2::interval;
```

La ventana `$2` se lee de `language_profiles.p95_window_days` (default 30 días). La imagen sidecar del checker (`checker_sidecar_image`, digest fijado) es propiedad de administración vía el endpoint de perfiles y todo cambio queda auditado.

## Pruebas

| Capa | Qué |
|---|---|
| Unitaria | Comparadores con fixtures congelados; rangos, alias, `template_id` ausente |
| Integración | Sin detención temprana, `TrimSpace` solo en exact, p95 sobre AC (`go test ./...`) |
| Frontend | Parámetros visibles, progreso, stale, claves EJ-*; compilación |
| Pinning | Cero `:latest` o sin digest en runners y perfiles |

## Amenazas

| Límite | Estado |
|---|---|
| Checker docente en sidecar sin red, solo lectura, con timeout (fallo = VE; imagen curada con digest, dueño admin vía endpoint de perfiles con auditoría) | Aplicable; probar timeout |
| Contenedores efímeros de envíos | Aplicable heredado, sin cambio |
| Enrutado, shell en host, VCS, ejecutables | No aplicable |

## Migración y despliegue

`00007`/`00008` existen; la serie comienza en `00009` (ajuste verificado):

- `00009`: `comparator_spec` JSONB NOT NULL (`{"id":"exact"}`); override en `config`.
- `00010`: `reference_solution`, `stale` (falso), `last_valid_dry_run_at`.
- `00011`: `language_profiles` (incluye `p95_window_days INT NOT NULL DEFAULT 30` y `checker_sidecar_image TEXT NOT NULL` pineada por digest) + `language_profile_audits`; semilla v0 (python/javascript 2000/256, cpp/c 1000/128, csharp 2500/256, java 3000/512). La ventana del p95 es campo configurable, nunca constante de código.
- `00012`: `run_metrics` y `dry_run_jobs`.

Orden: migraciones, re-pin, semilla. Sin desvíos nuevos (DESVÍO-01..03 se corrigen aquí).

## Evidencia

`ValidateRamAgainstHost` (`domain/admin_governance.go:99`) invocado en `admin_governance_service.go:188,250`, `workspace_service.go:185`, `template_audit_worker.go:174`, `env_test_service.go:168`; estrategias `factory.go:48-53`; `lang_c/cpp` (`gcc:latest`), `lang_csharp` (`mono:latest`); AST `ast_analyzer.go` + `rules/*/forbidden.yaml`; editor `exercise-editor-modal.component.ts:152-154`, `step-rules.component.ts:24-26`; detención temprana `evaluation_service.go:180-197`; migraciones `00001..00008`; sin `run_metrics`, `language_profiles`, `dry_run_jobs` ni `comparator_spec`.

## Preguntas abiertas

Ninguna bloqueante. Riesgos cerrados el 2026-10-04: ventana del p95 como campo `p95_window_days` (default 30 días) en `language_profiles`; imagen sidecar del checker con dueño admin vía endpoint de perfiles, pineada por digest y auditada.
