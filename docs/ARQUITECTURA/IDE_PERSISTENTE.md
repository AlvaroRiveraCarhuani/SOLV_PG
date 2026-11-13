# IDE Persistente — Flujo as-built

Documenta el camino completo de un ejercicio `IDE_PERSISTENTE`: configuración docente,
aprovisionamiento del entorno, entrega del proyecto y calificación por rúbrica.

## 1. Configuración docente

El editor de ejercicios (`/teacher/cursos/:id/ejercicios/:exerciseId/editar`) usa el paso
`step-ide-environment` cuando `environment_type === 'IDE_PERSISTENTE'`:

- Plantilla aprobada: se detecta servicio de base de datos con cualquiera de las dos formas
  que el backend puede serializar en `services_config`:
  `{ database: { enabled, engine } }` o `{ services: [{ category: 'database', engine: 'postgres' }] }`.
  Sólo con esa detección se habilita el editor de `db_init_script`.
- Script inicial de base de datos (`ide-db-init-script`): el docente carga un archivo `.sql`, `.txt`
  o `.js` por arrastre o por selector, y también puede escribir las sentencias en el editor. Con
  contenido cargado, la sección muestra nombre del archivo, tamaño, cantidad de líneas y sentencias
  estimadas, y ofrece reemplazar, descargar y limpiar. El campo es exclusivo de la práctica: no
  configura la base de la materia (ADR-071).
- Aviso de tamaño: por encima de 2 MiB se muestra una advertencia, sin bloquear el guardado. El
  script se entrega al cliente por entrada estándar (ADR-072), así que el umbral es una recomendación
  de costo de aprovisionamiento y no un límite del sistema (ADR-071).
- Aclaración de alcance: la sección indica que la base es una sola durante toda la materia y que los
  datos de una práctica quedan disponibles en las siguientes, además de que el script es exclusivo de
  la práctica (ADR-071).
- Persistencia del script: el texto vive en la tabla `exercise_db_init_scripts`, no en
  `exercises.config`. La lectura por id lo reconstruye y la lista por materia no lo carga (ADR-073).
- Parámetros de entrega: `due_date`, `allow_resubmit`, `auto_checkpoint`.
- Rúbrica por criterios y niveles.

Guardar borrador y publicar esperan (`await`) el `PUT /api/v1/exercises/{id}/rubric` antes de
cerrar el flujo: sin esa espera la publicación podía quedar sin rúbrica y SpeedGrader no tenía
criterios para calificar la entrega del proyecto. En borrador un fallo de rúbrica no bloquea el
guardado; en publicación sí aborta con mensaje explícito.

## 2. Aprovisionamiento del entorno

`POST /api/v1/environments/start` con `{ subject_id, exercise_id? }`.

- La identidad sale del JWT (`resolveRequestStudentID`). El `student_id` del body sólo puede
  sobrescribirla con rol `admin` o `teacher`; un `student` que envía otro `student_id` recibe `403`.
- `StartEnvironmentForExercise` resuelve la plantilla priorizando `exercise.TemplateID` sobre la
  plantilla de la materia.
- `ensureExerciseSatelliteDB(ctx, studentID, subjectID, template, exercise)` se ejecuta en los tres
  caminos: entorno nuevo, entorno existente `running` y reactivación/restart. Garantiza la base
  satélite `solv_std_<student>_<subject>` cuando la plantilla declara PostgreSQL **o** el ejercicio
  define `config.ide.db_init_script`, y aplica el script una sola vez por `exercise_id` usando
  `_solv_exercise_migrations`.
- La base es una por alumno y materia, no por práctica: los datos de una práctica siguen disponibles
  en las siguientes de la misma materia y solo la aplicación del script se sella por `exercise_id`
  (ADR-071).
- El `db_init_script` es **hard-fail**: si falla, el inicio o la reactivación se aborta y el alumno
  ve el error; no queda un entorno a medio inicializar.
- Sin `BASE_DOMAIN` el entorno publica `3000/tcp` en `127.0.0.1:<puerto dinámico>` y
  `access_url = http://localhost:<puerto>`. No se emiten labels Traefik ni dominios `.local` en ese
  camino.
- El environment es único por `(student_id, subject_id, exercise_id)` (ADR-059): el modelo por materia
  se retiró. Por eso el `db_init_script` se re-verifica en cada arranque en lugar de depender de que
  el contenedor sea nuevo.

### 2.1 Entorno por práctica

`POST /api/v1/environments/start` con `subject_id` y `exercise_id` (obligatorios) abre el entorno del
alumno acotado a la práctica: volumen `solv_workspace_<alumno>_<materia>_<práctica>` y reutilización
por `(alumno, materia, práctica)`. Abrir otra práctica no reutiliza entorno ni volumen. No hay
entorno sin práctica: el modelo por materia se retiró.

`POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/ide` (ADR-059) abre el entorno del
docente acotado a una práctica, desde su ficha:

- `environments.exercise_id` acota el entorno; el volumen pasa a
  `solv_workspace_<docente>_<materia>_<práctica>` y la clave de reutilización es
  `(tenant_id, student_id, subject_id, exercise_id)`.
- Plantilla y `db_init_script` salen del ejercicio. El entorno existente se devuelve tal cual o se
  reanuda; no se crean contenedores duplicados.
- Solo prácticas `IDE_PERSISTENTE`; un volumen congelado responde `409` y la materia de otro docente
  `403`. El entorno por práctica no se inscribe en el aula.
- El docente abre la URL en una pestaña nueva; *Probar Entorno* sigue siendo la validación efímera
  que no crea environment ni entrega.

### 2.2 Inicialización del proyecto en el volumen

Al crear un entorno nuevo, el servicio lee `workspace_projects.current_published_version_id` y
persiste la versión en `environments.workspace_project_version_id` antes de crear el volumen. La
versión se valida contra tenant, ejercicio y proyecto. Un entorno existente reutiliza siempre su
versión asignada: no se consulta la publicación actual para decidir qué contenido recibió.

- **Sembrado:** tras `EnsureVolumeExists` y antes de arrancar OpenVSCode, un contenedor Alpine
  efímero sin red monta solo el volumen nombrado en `/environment` y escribe el snapshot textual
  normalizado. Conserva directorios, nombres relativos y contenido, con uid/gid 1000. No toca la
  versión publicada, la imagen, la plantilla ni el volumen de otro alumno.
- **Diario y marcador:** `.solv_workspace/pending.json` identifica versión y hash antes de escribir;
  `.solv_workspace/init.json` con estado `verified` se escribe solo después de comparar todo el
  árbol. El diario y el marcador viven dentro del volumen, nunca en el host.
- **No sobrescritura:** un volumen con archivos ajenos, una modificación, un symlink, una ruta
  inesperada o un diario de otra versión se rechaza sin borrar ni reemplazar nada. No existe una
  re-siembra destructiva. Un entorno `legacy`, `none`, en cuarentena o con un estado ilegible no se
  siembra.
- **Recuperación:** un diario `pending` de la misma versión permite completar únicamente archivos
  faltantes; luego se verifica y se promueve a `setup_done`. Un fallo transitorio queda `failed` y
  se reintenta en la siguiente apertura. Una reapertura de `setup_done` verifica el marcador: si el
  volumen lo perdió, pasa a cuarentena en lugar de asumir que está vacío.
- **Concurrencia:** las aperturas del mismo alumno/materia/práctica se serializan con un lock en
  proceso; la unicidad de `(tenant_id, student_id, subject_id, exercise_id)` sigue protegiendo la
  base. El lock es local al nodo On-Premise.
  El alcance es un backend único: el repositorio no declara réplicas ni balanceo del proceso
  (`compose.yml` levanta Traefik, PostgreSQL, SeaweedFS y observabilidad, y el backend corre como
  un binario). Si alguna vez se ejecutan dos instancias contra la misma base, este lock deja de
  alcanzar y la serialización tiene que pasar a PostgreSQL.
- **Estado sin proyecto:** `not_required` no es un estado interno: `environments.workspace_init_status`
  lo persiste cuando el ejercicio no recibe proyecto (entornos `none`, `JUEZ_EFIMERO` o una
  práctica sin versión publicada) y el `CHECK` de `00053_workspace_projects.sql` lo admite junto a
  `legacy`, `pending`, `volume_created`, `setup_done`, `failed`, `quarantined` y `missing`.
- **Errores:** el backend responde `409 workspace_seed_volume_not_empty`,
  `workspace_seed_conflict`, `workspace_seed_quarantined` o `workspace_seed_version_invalid`; un
  problema operativo responde `503 workspace_seed_unavailable`. El solver muestra el mensaje y no
  abre un IDE vacío como si fuera un éxito.

La entrega sigue tomando la versión del entorno, por lo que un alumno sembrado con V1 conserva V1
aunque el docente publique V2; un alumno nuevo recibe V2. `JUEZ_EFIMERO` no recorre este camino.

## 3. Entrega del proyecto

`POST /api/v1/environments/{id}/submit` con `{ exercise_id, files? }`.

- Valida pertenencia del entorno al alumno autenticado (`ErrEnvironmentForbidden`).
- Si llegan `files`, se escriben en el volumen del entorno; el snapshot que se guarda es
  determinista (orden alfabético, cabeceras `# === FILE: <ruta> ===`, ignora `.git`,
  `node_modules`, `__pycache__`, `.venv`, binarios, tope 4 MiB, 800 archivos y 1 MiB por archivo).
- Crea una `submission` con `verdict: "PENDING"`, `review_status: "pending"` y sin `score`.
  `base_status` clasifica la base: `version` cuando el entorno tiene una versión
  publicada exacta, `no_base` cuando nació sin proyecto y `legacy_unknown` para el
  historial anterior a Fase 1. La calificación por rúbrica se hace después en SpeedGrader.
- Respuesta: `{ submission_id, environment_id, exercise_id, verdict, score }`.
- La carpeta del IDE viaja como `multipart/form-data` (cuerpo máximo 8 MiB; 4 MiB de
  contenido, 800 archivos y 1 MiB por archivo). El JSON anterior se conserva solo para
  cuerpos de hasta 512 KiB. El snapshot del volumen tiene el mismo techo funcional de
  4 MiB/800/1 MiB.

### 3.1 Lectura para SpeedGrader

`GET /api/v1/teacher/submissions/{id}/review` arma el read model de la revisión sin tocar la base:

- `ParseProjectSnapshot` separa `submissions.code` en `files[]` (`path`, `language`, `content`,
  `start_line`, `line_count`) usando las cabeceras `# === FILE: <ruta> ===`. El arreglo se ordena por
  ruta; `start_line` es la línea 1-based de la primera línea de contenido dentro de `code`.
- `files_available` queda en `false` si el snapshot no tiene cabeceras reconocibles, excede 4 MiB o
  800 archivos, repite rutas, tiene rutas absolutas, con `..`, con `\` o con caracteres de control.
  En ese caso la vista muestra el visor plano de `code`.
- En esta modalidad `test_cases` va vacío y `metrics_available` es `false`: no hay veredicto de
  ejecución ni tiempos que mostrar.
- El proyecto base sale de `workspace_project_versions.text_snapshot` de la versión exacta que
  referencia la entrega; `exercises.boilerplate` queda como espejo de compatibilidad del ejercicio
  publicado.
- El read model separa esa base en `base_files[]` con el mismo parser y la expone junto a
  `base_files_available`; el diff compara cada archivo entregado contra el homónimo. Solo
  `base_status = version` habilita el diff. `no_base` y `legacy_unknown` no se comparan contra el
  espejo actual, porque el alumno nunca recibió ese texto.
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

### 3.3 Proyectos y versiones publicadas

- `workspace_projects` es el proyecto estable del ejercicio; `workspace_project_drafts` es el
  único borrador editable con revisión optimista. Cada publicación crea una fila inmutable en
  `workspace_project_versions` y mueve `current_published_version_id` en la misma transacción.
- `GET /api/v1/teacher/exercises/{id}/project` devuelve proyecto, borrador, versiones y la
  referencia legacy si existe. `PUT .../project/draft` guarda el borrador y `POST .../project/publish`
  publica una versión.
- `POST .../project/import/folder` recibe `multipart/form-data` con un `manifest`
  (`{path, field, size}`) y un archivo por campo; `POST .../project/import/zip` recibe un único
  `archive`. Ambas rutas producen el mismo snapshot normalizado y solo crean/actualizan el
  borrador. Los topes son 1 MiB de contenido, 300 archivos, 512 KiB por archivo para el proyecto;
  el ZIP está limitado a 10 MiB comprimidos, 1000 entradas, 20 MiB expandidos y 2 MiB por entrada.
- `POST .../project/generate` recibe `{ description, purpose?, technology?, requirements? }` y
  devuelve `{ files: [{ path, content }], summary }` validado. Usa el proveedor de IA ya
  configurado por la institución (`CompleteJSON`, acción `generate-workspace-project`) y no crea
  proyecto, borrador ni versión. Los archivos generados pasan por el mismo normalizador que una
  carpeta o un ZIP y nunca se ejecutan, compilan ni escriben en un volumen.
- El manifiesto y el ZIP se validan antes de persistir: rutas relativas, sin `..`, sin `\\`,
  sin duplicados ni colisiones Unicode, texto UTF-8, sin binarios, symlinks, cifrado, Zip Slip ni
  tamaños inconsistentes. El extractor no escribe en el sistema de archivos.
- `WORKSPACE_PROJECTS_PUBLISH_ENABLED` debe valer exactamente `true` para publicar. Por defecto
  queda apagado: el endpoint de publicación responde `403 workspace_publishing_disabled` y la
  interfaz solo muestra el estado informativo.
- La publicación no escribe el proyecto en el volumen del alumno. La siembra ocurre al abrir el
  entorno, desde la versión exacta asignada; una publicación posterior no modifica entornos
  existentes. Ver la sección 2.2 y el ADR-068.
- Antiplagio y timeline usan la misma versión exacta. El juez `JUEZ_EFIMERO` conserva
  `base_status = NULL` y `exercises.boilerplate`.
- `workspace_legacy_project_references` conserva el `boilerplate` histórico sin convertirlo en
  versión ni enlazarlo a entregas. La migración `00053_workspace_projects.sql` hace el backfill y
  agrega las FKs compuestas; el rollback está documentado en `ADR-068` y es destructivo por diseño.

### 3.4 Experiencia de proyecto inicial en el editor

`ide-project-initial` vive dentro de `step-ide-environment` y solo se muestra para
`IDE_PERSISTENTE`. La vista consulta el proyecto al cargar el ejercicio y ofrece:

- selección de carpeta con File System Access API, alternativa `webkitdirectory`, ZIP o arrastre;
- árbol de archivos con rutas anidadas, editor de contenido, alta, renombrado/movimiento y borrado;
- guardado con la revisión leída del servidor; un `409` muestra conflicto y exige recargar sin
  sobrescribir;
- estado del borrador, versión publicada, historial y snapshot de solo lectura;
- confirmación de publicación con versión a crear y resumen de agregados/modificados/eliminados.

La acción **Generar con IA** abre un panel dentro de la misma tarjeta con un formulario corto
(qué se quiere crear, qué debe hacer, lenguaje y requisitos de archivos). El panel muestra el
progreso, permite cancelar la espera y, al recibir la propuesta, lista los archivos con una vista
de solo lectura. Descartarla no toca el editor; usarla pide confirmación si hay cambios sin guardar
y queda bloqueada ante un conflicto de revisión. La propuesta entra como archivos editables: nada
se guarda ni se publica hasta que el docente guarda el borrador y decide publicar.

El frontend pre-valida rutas, límites y texto, pero el backend repite todas las reglas. La
selección de carpeta lee los archivos con UTF-8 fatal para que un binario no se convierta en texto
por el reemplazo U+FFFD del navegador. Si `publishing_enabled` es falso no se renderiza el botón
de publicar; solo se informa el estado.

## 4. Experiencia del alumno

`StudentExerciseSolverComponent` detecta el ejercicio IDE con:

```
type === 'project' | 'environment' | environment_type === 'IDE_PERSISTENTE' | 'environment'
```

Cuando es IDE:

- No invoca `/api/v1/evaluations` ni muestra el panel del juez.
- Inicia/reconecta el entorno con `startEnvironment(subject_id, exercise_id)` y muestra `status`,
  `access_url` y `last_activity_at`; botón para abrir el IDE en otra pestaña.
- La entrega es el volumen de la práctica: el alumno trabaja en el IDE y el solver no le pide
  copiar archivos en el navegador. `POST /environments/{id}/submit` fotografía el volumen tal como
  está; `files` sigue siendo opcional en la API para herramientas, pero el solver no lo envía.
- "Entregar Proyecto" registra la entrega y muestra `verdict: PENDING` con `submission_id` y fecha.
  Si `allow_resubmit` es `false`, la entrega queda bloqueada tras la primera.

## 5. Metadatos públicos del ejercicio

`ExercisePublicResponse` expone `allow_resubmit` y `auto_checkpoint` cuando existen en
`config.ide`. `db_init_script` nunca se serializa al rol `student`.

## 6. Verificación

- Backend: `go vet ./...` y `go test ./internal/...`.
- Frontend: `npm run lint:styles` y `CI=true npm run build`.
- Integración de environment (`tests/integration/slice04_environment_lifecycle_test.go`) requiere
  daemon Docker: valida `access_url` local y el binding `3000/tcp` en `127.0.0.1`.
- La inicialización del proyecto se prueba en
  `tests/integration/workspace_projects_phase3_test.go` (PostgreSQL descartable) y
  `tests/integration/workspace_projects_phase3_docker_test.go` (volúmenes Docker aislados).
  Cubren asignación V1/V2, ausencia de proyecto, legacy, reapertura sin re-siembra, entrega atada a
  la versión del entorno, aislamiento, cuarentena, archivos ajenos, modificación del alumno y
  reanudación del diario.
- La generación con IA se prueba con proveedores simulados en
  `workspace_project_generate_test.go` (servicio) y `workspace_project_generate_handler_test.go`
  (HTTP). No se usan claves reales ni se generan costos externos.
