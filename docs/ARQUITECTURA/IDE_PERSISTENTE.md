# IDE Persistente — Flujo as-built

Documenta el camino completo de un ejercicio `IDE_PERSISTENTE`: configuración docente,
aprovisionamiento del workspace, entrega del proyecto y calificación por rúbrica.

## 1. Configuración docente

El editor de ejercicios (`/teacher/courses/:id/exercises/:exerciseId/edit`) usa el paso
`step-ide-environment` cuando `environment_type === 'IDE_PERSISTENTE'`:

- Plantilla aprobada: se detecta servicio de base de datos con cualquiera de las dos formas
  que el backend puede serializar en `services_config`:
  `{ database: { enabled, engine } }` o `{ services: [{ category: 'database', engine: 'postgres' }] }`.
  Sólo con esa detección se habilita el editor de `db_init_script`.
- Parámetros de entrega: `due_date`, `allow_resubmit`, `auto_checkpoint`.
- Rúbrica por criterios y niveles.

Guardar borrador y publicar esperan (`await`) el `PUT /api/v1/exercises/{id}/rubric` antes de
cerrar el flujo: sin esa espera la publicación podía quedar sin rúbrica y SpeedGrader no tenía
criterios para calificar la entrega del proyecto. En borrador un fallo de rúbrica no bloquea el
guardado; en publicación sí aborta con mensaje explícito.

## 2. Aprovisionamiento del workspace

`POST /api/v1/workspaces/start` con `{ subject_id, exercise_id? }`.

- La identidad sale del JWT (`resolveRequestStudentID`). El `student_id` del body sólo puede
  sobrescribirla con rol `admin` o `teacher`; un `student` que envía otro `student_id` recibe `403`.
- `StartWorkspaceForExercise` resuelve la plantilla priorizando `exercise.TemplateID` sobre la
  plantilla de la materia.
- `ensureExerciseSatelliteDB(ctx, studentID, subjectID, template, exercise)` se ejecuta en los tres
  caminos: workspace nuevo, workspace existente `running` y reactivación/restart. Garantiza la base
  satélite `solv_std_<student>_<subject>` cuando la plantilla declara PostgreSQL **o** el ejercicio
  define `config.ide.db_init_script`, y aplica el script una sola vez por `exercise_id` usando
  `_solv_exercise_migrations`.
- El `db_init_script` es **hard-fail**: si falla, el inicio o la reactivación se aborta y el alumno
  ve el error; no queda un workspace a medio inicializar.
- Sin `BASE_DOMAIN` el workspace publica `3000/tcp` en `127.0.0.1:<puerto dinámico>` y
  `access_url = http://localhost:<puerto>`. No se emiten labels Traefik ni dominios `.local` en ese
  camino.
- El workspace es único por `(student_id, subject_id)`, por eso el `db_init_script` se re-verifica
  en cada arranque en lugar de depender de que el contenedor sea nuevo.

## 3. Entrega del proyecto

`POST /api/v1/workspaces/{id}/submit` con `{ exercise_id, files? }`.

- Valida pertenencia del workspace al alumno autenticado (`ErrWorkspaceForbidden`).
- Si llegan `files`, se escriben en el volumen del workspace; el snapshot que se guarda es
  determinista (orden alfabético, cabeceras `# === FILE: <ruta> ===`, ignora `.git`,
  `node_modules`, `__pycache__`, `.venv`, binarios, tope 512 KB).
- Crea una `submission` con `verdict: "PENDING"` y `score: 0`, sin migración nueva. La calificación
  por rúbrica se hace después en SpeedGrader.
- Respuesta: `{ submission_id, workspace_id, exercise_id, verdict, score }`.

## 4. Experiencia del alumno

`StudentExerciseSolverComponent` detecta el ejercicio IDE con:

```
type === 'project' | 'workspace' | environment_type === 'IDE_PERSISTENTE' | 'workspace'
```

Cuando es IDE:

- No invoca `/api/v1/evaluations` ni muestra el panel del juez.
- Inicia/reconecta el workspace con `startWorkspace(subject_id, exercise_id)` y muestra `status`,
  `access_url` y `last_activity_at`; botón para abrir el IDE en otra pestaña.
- Editor por archivo del proyecto (alta/baja de archivos, lenguaje derivado de la extensión) que se
  envía como `files` al entregar.
- "Entregar Proyecto" registra la entrega y muestra `verdict: PENDING` con `submission_id` y fecha.
  Si `allow_resubmit` es `false`, la entrega queda bloqueada tras la primera.

## 5. Metadatos públicos del ejercicio

`ExercisePublicResponse` expone `allow_resubmit` y `auto_checkpoint` cuando existen en
`config.ide`. `db_init_script` nunca se serializa al rol `student`.

## 6. Verificación

- Backend: `go vet ./...` y `go test ./internal/...`.
- Frontend: `npm run lint:styles` y `CI=true npm run build`.
- Integración de workspace (`tests/integration/slice04_workspace_lifecycle_test.go`) requiere
  daemon Docker: valida `access_url` local y el binding `3000/tcp` en `127.0.0.1`.
