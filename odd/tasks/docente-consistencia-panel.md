# Feature: Consistencia del Panel Docente

Objetivo: que el panel, la cola y el banco cuenten lo mismo; que no existan botones que no llevan a ningún lado; que "Por calificar" y "Entregas" tengan preguntas distintas; y que el replay de escritura muestre lo que dice mostrar.

Branch: `fix/teacher-panel-consistency` (desde `main`) | Delivery strategy: ask-on-risk | Test runner: `go test ./...` & `npm run test:ci`

Forecast de líneas autoradas: ~450 (backend ~150, frontend ~200, docs ~100).

Diseño de referencia: `docs/UI/DOCENTE/CATALOGO_ENTORNOS.md` §2 y `docs/UI/DOCENTE/REDISENO_PANEL_DOCENTE.md` §2.2.

## Ya corregido

- **La cola no cargaba.** La base estaba en la migración 33 y el backend vivo se había compilado antes de la 34; la consulta a `submissions.review_status` fallaba. Se aplicó la migración y se reinició el servicio.
- **Las entregas IDE nacían con un 0 fabricado.** `SubmitWorkspaceProject` creaba la entrega con `verdict = 'PENDING'` y `score = 0`. El backfill de 00034 leyó ese 0 como una nota real y las marcó `not_required`; la libreta creó una celda con 0 para cada una. La corrección `00035_teacher_review_correction.sql` devuelve esas 21 entregas a `pending`, borra el `score` fabricado y elimina las 21 celdas que nacieron de él. La libreta queda con 13 celdas reales y ninguna en 0 por trabajo sin corregir.
- **El origen del 0.** `SubmitWorkspaceProject` ya no asigna `Score`; una entrega sin corregir viaja con `nil` hasta que el docente califica. El `INSERT` de `submissions` ahora escribe `review_status` con `COALESCE(NULLIF(:review_status, ''), 'not_required')` y las dos rutas lo declaran: IDE → `pending`, juez → `not_required`. Test: `TestSubmitWorkspaceProject_CreatesPendingSubmissionWithSnapshot` verifica el estado y la ausencia de nota.

## Decisiones cerradas

- **Una sola fuente para "pendiente":** `submissions.review_status`. El veredicto (`WA`, `RE`, `TLE`) es un resultado del juez, no una tarea del docente.
- **Una cola, un registro.** "Por calificar" es la cola accionable; "Entregas" es el historial. La pestaña "Cola de Entregas" de la página heredada duplica la cola y se retira.
- **Los números del panel y el widget de atención tienen que coincidir.** Hoy el KPI suma `review_status` y el widget lista veredictos.

## Specs

- **S1 — Atención y banco con el mismo criterio.**
  - `teacher_courses_repository.go`: el bloque `Standard` de atención y `review_pending` del banco pasan de `verdict IN ('WA','RE','TLE')` a `review_status IN ('pending','in_review')`.
  - La consulta de estadísticas por práctica usa el mismo criterio.
  - Criterio: un `WA` de juez ya calificado no aparece en atención; una entrega IDE sin corregir sí aparece y suma en el KPI.

- **S2 — Botón de benchmark.**
  - El frontend publica en `/api/v1/teacher/submissions/{id}/benchmark`, pero no existe handler ni ruta. Se retira el botón, la entrada del menú de herramientas y la llamada del servicio; el componente `complexity-benchmark` queda sin uso hasta que exista backend.
  - Alternativa si se decide implementarlo: endpoint con contrato, ejecución y límite de tiempo, más su prueba.
  - Criterio: ningún botón del SpeedGrader responde 404.

- **S3 — Navegación sin duplicados.**
  - "Evaluaciones & Entregas" pasa a "Entregas"; su subtítulo explicita "historial de entregas".
  - Se retira la pestaña "Cola de Entregas" y sus KPIs; las filas pendientes enlazan a "Por calificar".
  - Criterio: un solo lugar responde "¿qué me falta calificar?".

- **S4 — Replay de escritura.**
  - El solver deja de enviar `event_type: "keystroke"` con `char_delta`: usa `insert`, `delete` y `paste` con `content`, `position` y `timestamp_ms`.
  - Se conecta el `KeystrokeRecorderService` que ya existe y hoy nadie llama, con envío periódico y al salir.
  - Alternativa si se decide retirarlo: se quita la auditoría de escritura del menú y se documenta la decisión.
  - Criterio: abrir el replay de una entrega nueva reconstruye el código y marca el texto pegado.

- **S5 — Verificación y documentación.**
  - `go vet ./...`, tests de servicios y repositorio; `npm run test:ci`, `lint:styles`, `ng build`.
  - Actualizar `docs/UI/DOCENTE/POR_CALIFICAR.md` y `DASHBOARD.md` si cambia el conteo.

## Tasks

- [x] **S1**: unificar atención, banco y estadísticas con `review_status`.
- [x] **S2**: retirar el benchmark sin backend.
- [x] **S3**: renombrar "Entregas" y quitar la cola duplicada.
- [x] **S4**: corregir el contrato del replay (formato keystroke events).
- [x] **S5**: verificación y documentación (tests passing, lint:styles OK).

## Descartado por verificación

Estas afirmaciones de la revisión no se corresponden con el código actual y no deben generar tareas:

- "SpeedGrader" no aparece en la interfaz; es nombre interno de carpeta y ruta.
- "Terminal Shadow", "Benchmark O(N)" y "Casos Desmascarados" no existen como textos visibles. El header dice "Consola del Estudiante", "Análisis de Complejidad Asintótica" y "Auditoría de Escritura".
- "Re-ejecutar" no está desconectado: la ruta `POST /teacher/submissions/{id}/run-ephemeral` existe y está registrada.
- El header del detalle de curso ya usa un menú "más acciones"; no compite con seis botones.
- El botón "Revisar" eterno ya no aplica: la cola nueva solo lista pendientes y dice "Calificar".

## Decisiones pendientes

1. **Benchmark:** retirarlo (recomendado para primeros semestres) o implementar el backend.
2. **Replay:** corregir el contrato o retirar la función del SpeedGrader.
3. **"Entregas":** renombrar ya o esperar a que "Materia" y "Notas" absorban el historial.
4. **Alcance del replays:** visible para todos los docentes o solo en una capa avanzada.

## Log

- **L1 (verbatim):** "por calificar ahora mismo no funciona, y tampoco entiendo bien la diferencia entre por calificar y evaluaciones y entregas."
- **Hallazgo verificado:** `teacher-dashboard.component.ts:107` suma `pending_review` del resumen de materias, que lee `review_status`; el widget de atención sigue armando `standard` con veredictos.
- **Hallazgo verificado:** no existe ruta de benchmark en `router.go`; el servicio del frontend publica igual.
- **Hallazgo verificado:** el solver envía `event_type: "keystroke"` y `char_delta`, sin `content`; el replay pinta `ev.content`.
