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
- El workspace por materia es único por `(student_id, subject_id)`; el de la práctica, por
  `(student_id, subject_id, exercise_id)` (ADR-059). Por eso el `db_init_script` se re-verifica en cada
  arranque en lugar de depender de que el contenedor sea nuevo.

### 2.1 Entorno por práctica

`POST /api/v1/workspaces/start` con `exercise_id` abre el entorno del alumno acotado a la práctica:
volumen `solv_workspace_<alumno>_<materia>_<práctica>` y reutilización por
`(alumno, materia, práctica)`. Abrir otra práctica no reutiliza el entorno ni el volumen de la
materia. Sin `exercise_id` se mantiene el entorno por materia.

`POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/ide` (ADR-059) abre el entorno del
docente acotado a una práctica, desde su ficha:

- `workspaces.exercise_id` acota el entorno; el volumen pasa a
  `solv_workspace_<docente>_<materia>_<práctica>` y la clave de reutilización es
  `(tenant_id, student_id, subject_id, exercise_id)`.
- Plantilla y `db_init_script` salen del ejercicio. El entorno existente se devuelve tal cual o se
  reanuda; no se crean contenedores duplicados.
- Solo prácticas `IDE_PERSISTENTE`; un volumen congelado responde `409` y la materia de otro docente
  `403`. El entorno por práctica no se inscribe en el aula.
- El docente abre la URL en una pestaña nueva; *Probar Entorno* sigue siendo la validación efímera
  que no crea workspace ni entrega.

## 3. Entrega del proyecto

`POST /api/v1/workspaces/{id}/submit` con `{ exercise_id, files? }`.

- Valida pertenencia del workspace al alumno autenticado (`ErrWorkspaceForbidden`).
- Si llegan `files`, se escriben en el volumen del workspace; el snapshot que se guarda es
  determinista (orden alfabético, cabeceras `# === FILE: <ruta> ===`, ignora `.git`,
  `node_modules`, `__pycache__`, `.venv`, binarios, tope 512 KB).
- Crea una `submission` con `verdict: "PENDING"`, `review_status: "pending"` y sin `score`, sin
  migración nueva. La calificación por rúbrica se hace después en SpeedGrader.
- Respuesta: `{ submission_id, workspace_id, exercise_id, verdict, score }`.

### 3.1 Lectura para SpeedGrader

`GET /api/v1/teacher/submissions/{id}/review` arma el read model de la revisión sin tocar la base:

- `ParseProjectSnapshot` separa `submissions.code` en `files[]` (`path`, `language`, `content`,
  `start_line`, `line_count`) usando las cabeceras `# === FILE: <ruta> ===`. El arreglo se ordena por
  ruta; `start_line` es la línea 1-based de la primera línea de contenido dentro de `code`.
- `files_available` queda en `false` si el snapshot no tiene cabeceras reconocibles, excede 2 MiB o
  500 archivos, repite rutas, tiene rutas absolutas, con `..`, con `\` o con caracteres de control.
  En ese caso la vista muestra el visor plano de `code`.
- En esta modalidad `test_cases` va vacío y `metrics_available` es `false`: no hay veredicto de
  ejecución ni tiempos que mostrar.
- `exercises.boilerplate` guarda el proyecto base en el mismo formato de snapshot. El read model lo
  separa en `base_files[]` con el mismo parser y lo expone junto a `base_files_available`; el diff
  compara cada archivo entregado contra el homónimo. Si la práctica no define una base, la vista
  oculta el diff y no inventa una comparación.
- Los comentarios IDE guardan **archivo y línea local** además de la línea global:
  `submission_comments.file_path` y `submission_comments.local_line_number` (migración
  `00045_submission_comment_anchors.sql`). `line_number` sigue siendo la coordenada global sobre
  `code` y se conserva para retrocompatibilidad y para validar el rango.
- `POST /comments` acepta `file_path` + `local_line_number` (y `local_end_line_number` opcional); el
  backend calcula la global con `start_line`. Si además llega `line_number`, debe coincidir. Si solo
  llega la global, deriva el ancla al leer el snapshot. Un archivo inexistente responde `422
  comment_file_unknown`; un bloque que cruza archivos, `422 line_range_spans_files`; una línea fuera
  de rango, `422 line_out_of_range`.
- La respuesta de revisión omite `code` y `boilerplate` cuando `files_available` es `true`. Si el
  snapshot no se pudo separar, conserva `code` crudo para no perder la entrega. En juez, `files` y
  `base_files` viajan como `null` y `code` se mantiene.
- El ancla por archivo es la que consume SOLV Companion: la extensión abre el comentario con
  `vscode.Uri` y la línea local, sin recalcular offsets del snapshot.

### 3.2 Entorno de prueba del docente

`POST /api/v1/teacher/submissions/{id}/test-environment` ejecuta la entrega en un contenedor efímero
para que el docente compruebe que la aplicación arranca. El servicio copia los archivos de
`submissions.code` a un volumen propio (`solv-review-env-<id>`), nunca al volumen del alumno.

- La receta de arranque vive en `exercises.config.ide`: `app_run_command` (obligatorio) y `app_port`
  (3000 si se omite). No se infiere del `package.json` ni del `Dockerfile`.
- El contenedor escucha en el puerto publicado en `127.0.0.1` y, si hay `BASE_DOMAIN`, recibe una
  ruta Traefik `review-<id>.<dominio>`. La app se abre en una pestaña nueva, nunca en un iframe.
- El QoS Worker renueva el TTL de 15 minutos cuando crece el contador de red del contenedor; sin
  tráfico, detiene el contenedor, borra el volumen y marca el preview como `expired`. No hay botón de
  detener. Si el host no tiene RAM, la admisión responde `503 oom_guard` antes de crear recursos.
- La tabla `review_test_environments` conserva el historial; un índice parcial único permite un solo
  preview activo por entrega. El contenedor lleva `solv.review_environment=true` para que el
  recolector de zombies no lo confunda con un laboratorio huérfano.
- Límite de esta fase: el preview ejecuta la app y el `setup_script` de la plantilla, pero no
  aprovisiona satélites de base de datos del nodo. Una app que dependa de un servicio satélite
  necesita un setup local o queda fuera del preview.

## 4. Experiencia del alumno

`StudentExerciseSolverComponent` detecta el ejercicio IDE con:

```
type === 'project' | 'workspace' | environment_type === 'IDE_PERSISTENTE' | 'workspace'
```

Cuando es IDE:

- No invoca `/api/v1/evaluations` ni muestra el panel del juez.
- Inicia/reconecta el workspace con `startWorkspace(subject_id, exercise_id)` y muestra `status`,
  `access_url` y `last_activity_at`; botón para abrir el IDE en otra pestaña.
- La entrega es el volumen de la práctica: el alumno trabaja en el IDE y el solver no le pide
  copiar archivos en el navegador. `POST /workspaces/{id}/submit` fotografía el volumen tal como
  está; `files` sigue siendo opcional en la API para herramientas, pero el solver no lo envía.
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
