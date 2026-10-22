# IDE Persistente — OA-1 Backend

Objetivo: reparar R1–R6 del flujo de IDE Persistente (identidad JWT, plantilla por ejercicio,
arranque de contenedor por tipo de imagen, acceso local sin `.local`, entrega de proyecto y
`author_id` en rúbricas). Branch: `fix/persistent-ide-backend`. Test runner: `go test ./internal/...`.

## Specs

- S1 (R1): `StartWorkspaceRequest.StudentID` deja de ser obligatorio; si llega vacío se resuelve desde
  `domain.GetUserID(ctx)` (fallback `X-User-Id`). Se acepta `exercise_id` opcional.
- S2 (R4/#8): `StartWorkspaceForExercise(ctx, studentID, subjectID, exerciseID)` prioriza
  `exercise.TemplateID` sobre `subject.TemplateID`. `StartWorkspace(ctx, studentID, subjectID)` delega
  con `exerciseID = ""` sin cambiar firma.
- S3 (R4): si el ejercicio define `config.ide.db_init_script` y hay base satélite, se crea
  `_solv_exercise_migrations (exercise_id TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`
  y el script se ejecuta solo si `exercise_id` no fue aplicado.
- S4 (R2): imágenes que contienen `openvscode-server` arrancan OpenVSCode en 3000 con
  `--without-connection-token --host 0.0.0.0 --port 3000`; imágenes de runtime puro arrancan con
  `/bin/sh -c "mkdir -p /home/workspace && chmod 0777 /home/workspace && if [ -x ... ]; then exec ...; else exec sleep infinity; fi"`
  sin forzar `User: 1000:1000`.
- S5 (R3): el puerto 3000 del workspace se publica en `127.0.0.1:<hostPort>` dinámico; `AccessURL` es
  `http://localhost:<hostPort>` cuando `BASE_DOMAIN` no está configurado (nunca `.local`).
- S6 (R5): `POST /api/v1/workspaces/{id}/submit` acepta `{exercise_id, files?}`, escribe `files` en el
  volumen y arma un snapshot determinista (orden alfabético, cabeceras `# === FILE: <ruta> ===`,
  ignora `.git`, `node_modules`, `__pycache__`, `.venv`, binarios, tope 512 KB) creando una submission
  con `verdict: "PENDING"`, `score: 0` y sin migración nueva.
- S7 (#4): `EvaluateSubmission` inserta el comentario general con `author_id` real desde
  `domain.GetUserID(ctx)`.

## Tasks

- T1 — S1/S2/S3/S5: `workspace_service.go` (inyección fluida, resolución por ejercicio, puerto, `AccessURL`, DBInitScript). inline — done
- T2 — S2/S5: `workspace_lifecycle.go` (mismo criterio en reactivación) + `UpdateAccessURL` opcional en repo. inline — done
- T3 — S4/S5: `docker_container.go` (`Cmd` por imagen, `ExposedPorts`, `PortBindings`). inline — done
- T4 — S6: `docker_workspace_volume.go` (contenedor efímero snapshot/escritura). inline — done
- T5 — S6: servicio + handler + router + wiring en `main.go`. inline — done
- T6 — S7: `rubric_service.go`. inline — done
- T7 — Verificación: `go vet ./... && go test ./internal/...`. inline — done (verde)

## Evidence

- `go vet ./...` → exit 0.
- `go test ./internal/...` → todas las suites `ok`, sin fallos.
- Tests nuevos: `workspace_submission_test.go` (snapshot determinista, submission PENDING/score 0,
  ownership) y `docker_workspace_volume_test.go` (filtrado de excluidos/binarios y presupuesto de bytes).
- `tests/integration/slice04_workspace_lifecycle_test.go:174,205` sigue afirmando `.solv.local` y las
  labels Traefik incondicionales: requiere actualización en la próxima orden (necesita daemon Docker).


## Log

- L1 (orden del usuario): "Reevaluamos tus 5 puntos contra `ports.go`, `workspace_service.go`,
  `workspace_lifecycle.go` y los mocks de test. Tus hallazgos son exactos. Acá tenés las decisiones
  cerradas más 3 salvaguardas para que no se rompa ningún test ni quede desfasado PostgreSQL".
  Decisiones: sin migración `language`; sin cambio de firmas existentes; interfaz opcional para el
  volumen; opción (c) para R2 con autodetección; `AccessURL = http://localhost:<hostPort>` y reserva
  de puerto antes de `repo.Create`; `template-studio.utils.ts` queda fuera de esta orden.

## Fase OA-1.1 — Cierre backend

Specs:

- S8 (R1/IDOR): `StartWorkspace` resuelve siempre la identidad autenticada; el `student_id` del body
  sólo sobrescribe con rol `admin`/`teacher`. Un `student` con otro `student_id` recibe `403`.
- S9 (R4): `ensureExerciseSatelliteDB(ctx, studentID, subjectID, template, exercise)` asegura la BD
  satélite cuando la plantilla declara `postgres` **o** el ejercicio define `db_init_script`, y aplica
  el script en modo hard-fail. Se invoca en workspace nuevo, existente `running` y reactivación/restart.
- S10 (R5): `ExercisePublicResponse` expone `allow_resubmit` y `auto_checkpoint`; nunca `db_init_script`.
- S11 (higiene): asserts de `slice04` alineados a `http://localhost:` + binding `3000/tcp`; sin
  literales `.local` en `lab_service.go` ni `manager.go`.

Tasks:

- T8 — S8: `workspace_handler.go`. inline — done
- T9 — S9: `workspace_service.go` (helper + guardas) y `workspace_lifecycle.go` (`reactivateWorkspace`
  recibe el nombre de la BD satélite; `RestartWorkspace` la asegura). inline — done
- T10 — S10: `exercise_dto.go`. inline — done
- T11 — S11: `slice04_workspace_lifecycle_test.go`, `lab_service.go` (`publicBaseDomain`),
  `manager.go`. inline — done
- T12 — Tests: `workspace_satellite_guard_internal_test.go` (detección de postgres y guardas sin
  conexión). inline — done

Evidence:

- `gofmt -l` limpio, `go vet ./...` exit 0, `go test ./internal/...` todas `ok`.
- `slice04_workspace_lifecycle_test.go` compila y pasa `go vet`, pero no se ejecutó localmente:
  necesita daemon Docker.

## Fase OA-2 — Frontend IDE persistente

Specs:

- S12: `hasDatabaseService`/`templateOptions` detectan BD satélite en las dos formas de
  `services_config`; `\u00XX` literales reemplazados por UTF-8 real en el paso y sus subcomponentes.
- S13: guardar/publicar esperan el `PUT /rubric` antes de completar; publicar aborta si la rúbrica falla.
- S14: `StudentService.startWorkspace(subjectId, exerciseId?)` envía `exercise_id`, usa
  `withCredentials` y desenvuelve el envelope; `submitWorkspaceProject` cubre `POST /workspaces/{id}/submit`;
  el DTO público expone `allow_resubmit`/`auto_checkpoint`.
- S15: el solver ramifica por `isIdeExercise`, sin juez; inicia/conecta el workspace, edita archivos,
  entrega el proyecto y muestra `verdict: PENDING` respetando `allow_resubmit`.
- S16: estilos con tokens (cero hex/rem), fechas con `dateText` en `<time>`, datos técnicos con
  `[machineData]`/`font-mono`.

Tasks:

- T13 — S12: `step-ide-environment.component.ts` + escapes en `.html` de los subcomponentes. inline — done
- T14 — S13: `exercise-editor.store.ts` (`persistIdeRubric`, `saveDraft`/`executePublish` async). inline — done
- T15 — S14: `student.service.ts`. inline — done
- T16 — S15/S16: `student-exercise-solver.component.ts/.html/.scss`. inline — done
- T17 — Verificación: `npm run lint:styles` y `CI=true npm run build`. inline — done (verde)

Evidence:

- `npm run lint:styles` → gates de inline styles, fechas y machine-data OK.
- `CI=true npm run build` → "Application bundle generation complete", sin errores.
- `npx eslint` sobre los archivos tocados: 0 findings nuevos; el `no-explicit-any` del store baja
  de 21 a 16 (las nuevas rutas usan `unknown` + `errorMessage`).
- Documentación as-built: `docs/ARQUITECTURA/IDE_PERSISTENTE.md`.

## Fase OA-IDE-3 — Verificación Docker (`slice04`) + E2E Real `J08`

Specs:

- S17: `TestSlice04WorkspaceLifecycle` pasa contra el daemon Docker publicando `3000/tcp` en
  `127.0.0.1:<hostPort>` y devolviendo `http://localhost:<hostPort>`.
- S18: `J08-persistent-ide-lifecycle.spec.mjs` cubre de punta a punta por UI y Docker real:
  homologación de plantilla `IDE_PERSISTENTE` con PostgreSQL, autoría docente con
  `<ide-db-init-script>` y rúbrica al 100%, inicio de contenedor real por el estudiante, edición y
  entrega de archivos (`POST /api/v1/workspaces/{id}/submit` -> `PENDING`), reconexión idempotente
  de `_solv_exercise_migrations`, y calificación con rúbrica y comentario en SpeedGrader.

Tasks:

- T18 — S17: ejecución de `TestSlice04WorkspaceLifecycle` contra Docker real. inline — done (verde)
- T19 — S18: cableado de `RubricHandler` en `main.go`, corrección de inserción en
  `submission_comments` en `rubric_service.go`, normalización de `workspace_id` en
  `student.service.ts`, preservación de estado en `exercise-editor.store.ts` /
  `exercise-editor.component.ts` y desenrollado paginado en `step-ide-environment.component.ts`. inline — done
- T20 — S18: creación y ejecución en verde de `frontend/e2e/flows/02-medium/J08-persistent-ide-lifecycle.spec.mjs` + registro en `frontend/e2e/README.md`. inline — done (verde)

Evidence:

- `cd backend && go test -v -count=1 ./tests/integration -run TestSlice04WorkspaceLifecycle` → PASS (`2.28s`).
- `cd frontend && npx playwright test e2e/flows/02-medium/J08-persistent-ide-lifecycle.spec.mjs --config=playwright.e2e.config.mjs` → `1 passed (1.5m)`.
- Verificación en PostgreSQL (`solv_std_f3c99eba_b9ee92ce`): 1 fila en `_solv_exercise_migrations` y 1 fila en `items (1, 'ok')` tras reconectar el workspace.
