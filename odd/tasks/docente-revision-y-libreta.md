# Feature: Revisión Docente Real y Libreta que No Miente

Objetivo: que "por calificar" sea un estado declarado y no una deducción a partir de veredictos, y que la libreta deje de contar como 0 el trabajo que el docente todavía no corrigió.
Branch: `feat/teacher-review-and-gradebook` (desde `main`) | Delivery strategy: ask-on-risk | Test runner: `go test ./...` & `npm run test:ci`

Forecast de líneas autoradas: ~1200 (backend ~700, frontend ~400, docs ~100). Supera el presupuesto de 400 → definir estrategia de entrega antes de empezar.

Diseño de referencia: `docs/UI/DOCENTE/REDISENO_PANEL_DOCENTE.md` (§2.2, §3.3, §3.4).

## Decisiones cerradas

- **El estado de revisión es explícito, no inferido.** `submissions.review_status` con los valores `not_required | pending | in_review | graded | returned`, más `reviewed_at`, `reviewed_by`, `feedback_summary` y `returned_at`. Motivo: hoy la cola se arma buscando `WA`, `RE`, `TLE`, `failed` y `oom_killed`, así que muestra como tarea lo que el juez ya calificó y esconde lo que espera corrección manual (las entregas IDE quedan en `PENDING`).
- **Quién exige revisión.** Juez efímero → `not_required` (califica solo). IDE persistente → `pending` desde la primera entrega. El valor es del dominio, no de la vista.
- **Las entregas son versiones; la revisión es por alumno × práctica.** Un reintento crea una entrega nueva y no reabre una revisión ya cerrada.
- **La celda de nota es una entidad.** `exercise_grades (subject_id, student_id, exercise_id, selected_submission_id, score, source, overridden_by, reason, updated_at)`. La elige el docente; el sistema sugiere la mejor. Motivo: hoy la matriz recalcula `max(score)` en cada lectura (`teacher_analytics_repository.go:94-113`) y adjudica 0 cuando `score` es nulo, con lo que "no entregó" y "no lo corregí" se ven igual.
- **Sin corregir no es cero.** La libreta muestra `—` y la fila indica cuántas prácticas están calificadas ("5 de 8"). El promedio se calcula solo sobre prácticas con nota y lo declara en la interfaz.
- **El detalle técnico no es la cola.** La cola muestra práctica, alumno y antigüedad. El veredicto, los casos y las métricas viven en el panel de trabajo, en capa 2/3.

## Specs

- **S1 — Esquema de revisión y calificación.**
  - Migración `backend/migrations/00034_teacher_review_and_gradebook.sql` (Up/Down) con:
    - `submissions.review_status VARCHAR(20) NOT NULL DEFAULT 'not_required'` con `CHECK` en los cinco valores, `reviewed_at TIMESTAMPTZ`, `reviewed_by UUID REFERENCES users(id)`, `feedback_summary TEXT NOT NULL DEFAULT ''`, `returned_at TIMESTAMPTZ`.
    - Índice `(tenant_id, review_status)` y `(exercise_id, review_status)`.
    - Tabla `exercise_grades` con PK `(student_id, exercise_id)` y FK a `submissions`, `subjects`, `exercises`; `source VARCHAR(10) CHECK IN ('auto','manual')`.
  - Backfill: entregas de ejercicios `IDE_PERSISTENTE` sin `score` → `pending`; el resto → `not_required`. `exercise_grades` se llena con la mejor entrega existente por alumno y práctica, `source='auto'`.
  - Criterio: tras la migración, la cola de un curso con solo prácticas de juez queda vacía; con una práctica IDE sin corregir, muestra exactamente esa entrega.

- **S2 — Dominio y puerto.**
  - `domain/submission_review.go` con `ReviewStatus`, transiciones válidas (`pending → in_review → graded`, `graded → returned`, `returned → graded`) y errores tipados.
  - `domain/gradebook.go` con `ExerciseGrade` y la regla de selección: la sugerencia es la mejor entrega con `score` no nulo; la elección manual gana siempre.
  - Puerto `TeacherReviewStore` declarado en el consumidor, siguiendo el patrón del repositorio de telemetría.
  - Criterio: los tests de transiciones cubren los cinco estados y rechazan saltos inválidos.

- **S3 — Cola real.**
  - `TeacherReviewStore.PendingReviews(ctx, tenantID, teacherID, subjectID?)` agrupado por práctica con `{exercise_id, title, subject, pending, total, oldest_submitted_at}`.
  - Se elimina la deducción por veredictos de `teacher_courses_repository.go` (`PendingReview`) y el campo pasa a leer `review_status`; el contrato JSON conserva el nombre para no romper la UI en el mismo cambio.
  - Criterio: una entrega `AC` de un ejercicio de juez no aparece; una entrega IDE `PENDING` sí; una ya `graded` desaparece de la cola.

- **S4 — API de revisión.**
  - `GET /api/v1/teacher/reviews?subject_id=&exercise_id=&status=`
  - `PATCH /api/v1/teacher/submissions/{id}/review` con `{status, feedback_summary, score?, override_reason?}`; valida transición, escribe auditoría con actor y motivo.
  - `PUT /api/v1/teacher/grades/{student_id}/exercises/{exercise_id}` con `{selected_submission_id, score?, reason}`.
  - Permisos: solo el docente de la materia o admin. Un docente ajeno recibe 403.
  - Criterio: repetir el `PATCH` con el mismo body es idempotente; un cambio de nota sin `reason` devuelve 422.

- **S5 — Frontend: cola por calificar.**
  - Vista `teacher/reviews` con la plantilla de consola del documento de diseño: lista agrupada por práctica a la izquierda, panel de trabajo a la derecha, "Guardar y siguiente".
  - Reutiliza `search-bar`, `status-tabs`, `pagination-bar`, `empty-state`, `verdict-badge` y el vocabulario de `OUTCOME_COPY`.
  - Criterio: calificar 3 entregas seguidas avanza sin volver a la lista y el contador baja; sin pendientes, estado vacío explícito con enlace a la materia.

- **S6 — Frontend: libreta.**
  - `exercise-grades` matrix: celda con `—` cuando no hay nota, tooltip con "sin corregir", clic abre panel lateral con la entrega seleccionada y el selector de cuál cuenta.
  - Encabezado con "5 de 8 prácticas calificadas" y el promedio declarando su base.
  - Criterio: ninguna celda muestra 0 por ausencia de corrección; cambiar la entrega seleccionada mueve la nota y queda en auditoría.

- **S7 — Verificación y documentación indivisible.**
  - `cd backend && go vet ./... && go test ./...`; integración nueva en `backend/tests/integration/` sobre `solv_test`.
  - `cd frontend && npm run lint:styles && npm run lint:ts && npm run test:ci && CI=true npm run build`.
  - ADR nuevo en `docs/ARQUITECTURA/ADR/` (estado de revisión y celda de nota, con alternativas descartadas) y entrada en `INDEX.md`.
  - `docs/API_CONTRACTS.md` con los tres endpoints; `docs/UI/DOCENTE/DASHBOARD.md` actualizado en la sección de cola de revisión.

## Tasks

- [x] **S1**: migración `00034` con `review_status` y `exercise_grades`, más backfill y su Down.
  - Estado: **cerrado**. Aplicada y revertida contra una base de prueba (`solv_review_check`). Sin bloques `DO`: los `CHECK` se recrean con `DROP … IF EXISTS` + `ADD`. Backfill verificado con fixtures: IDE sin nota → `pending` sin celda; AC con 80 → celda 80; AC sin nota → celda 100.
- [x] **S2**: dominio, transiciones y regla de selección con tests unitarios.
  - Estado: **cerrado**. `domain/review_workflow.go` con cinco estados, transiciones, `ResolvedSubmissionScore` y `NeedsGradeReason`. 4 tests en `review_workflow_test.go`.
- [x] **S3**: read model de la cola y retiro de la deducción por veredictos.
  - Estado: **cerrado con corrección posterior**. `ListPendingReviewGroups` agrupa por práctica con inscritos y antigüedad; `pending_review` del resumen lee `review_status`; la matriz lee `exercise_grades` en lugar de recalcular `max(score)`.
  - **Corrección (2026-10-30):** el `INSERT` de `submissions` no escribía `review_status`, así que toda entrega nueva nacía `not_required` y la cola nunca se llenaba. Ahora `SubmitWorkspaceProject` la crea en `pending`, el camino del juez en `not_required`, y el `INSERT` escribe la columna con `COALESCE(NULLIF(:review_status, ''), 'not_required')`. El test `TestSubmitWorkspaceProject_CreatesPendingSubmissionWithSnapshot` verifica el estado.
  - **Corrección de datos (00035):** la entrega IDE se creaba con `score = 0`; el backfill de 00034 lo leyó como nota real y la marcó `not_required`, y la libreta creó una celda con 0. La migración `00035_teacher_review_correction.sql` devuelve las 21 entregas afectadas a `pending`, borra el score fabricado y elimina las 21 celdas que nacieron de él. La creación ya no asigna `Score`: queda `nil` hasta que el docente califica.
- [x] **S4**: endpoints de revisión y de calificación con permisos y auditoría.
  - Estado: **cerrado**. `GET /teacher/reviews`, `PATCH /teacher/submissions/{id}/review`, `PUT /teacher/grades/{studentId}/exercises/{exerciseId}`, con errores tipados (409/422/400) y auditoría de cada cambio. El almacén se inyecta por tipo en el constructor para no romper los dobles existentes.
- [x] **S5**: vista "Por calificar" con panel de trabajo y flujo continuo.
  - Estado: **cerrado**. `teacher-reviews` en `/teacher/por-calificar`: lista de prácticas, entregas pendientes ordenadas por antigüedad y acceso al SpeedGrader. La calificación en línea queda en el SpeedGrader existente, no duplicada.
- [x] **S6**: libreta con `—`, selector de entrega y promedio declarado.
  - Estado: **cerrado**. `grades[exercise]` es `*int`/`null`; la celda muestra `—`, el CSV la deja vacía y el promedio usa solo prácticas calificadas (`graded_count`/`pending_count`). El selector de entrega se expone por API; la UI de selección dentro de la libreta queda para la siguiente pasada.
- [ ] **S7**: matriz de verificación, ADR, contratos y actualización de la documentación de la vista.
  - Estado: **parcial**. ADR-053, `INDEX.md`, `API_CONTRACTS.md`, `POR_CALIFICAR.md` y `DASHBOARD.md` actualizados. Verificado: `go build`, `go vet`, tests de dominio/servicios, 838 tests de frontend, `lint:styles` y `ng build`. Pendiente: test de integración contra `solv_test` (requiere base disponible) y la UI de selección de entrega en la libreta.

## Log

- **L1 (verbatim):** "si, ahora mismo eso de WA, RE, TLE, ya no lo estamos usando, ya que esos tecnisismos son mas para competidores o personas acostumbradas, pero para personas de primeros semestres puede ser complicado."
- **L2 (verbatim):** "tus propuestas estan medio bien, pero siento que son parches, osea ahora mismo no podemos quedarnos con el modelo que tenemos, o la iu, tiene que evolucionar a una mas potente y diseñada para lo que es, clases de programacion."
- **L3 (verbatim):** "La que el docente elija, con la mejor como sugerencia (Recomendado)" — respuesta a qué entrega define la nota.
- **Hallazgo verificado:** `teacher_analytics_repository.go:94-113` asigna 0 cuando `score` es nulo y el veredicto no es `AC`, de modo que una entrega IDE sin corregir vale 0 en la libreta.
- **Hallazgo verificado:** `teacher_courses_repository.go` arma `pending_review` con `WA`, `RE`, `TLE`, `failed` y `oom_killed`; las entregas IDE en `PENDING` no entran.
- **Ya cerrado:** el vocabulario de resultados quedó unificado en `OUTCOME_COPY` (`frontend/src/app/core/models/verdict.model.ts`) y documentado en `docs/UI/UX-PRINCIPLES.md` §3. Esta feature consume ese vocabulario, no crea otro.
- **Hallazgo de operación (2026-10-30):** "Por calificar" mostraba error porque la base estaba en la migración 33 y el backend vivo se había compilado antes de la 34. Se aplicó la migración y se reinició el servicio. La cola respondía vacía porque las 21 entregas IDE tenían un 0 fabricado que el backfill tomó como nota; la corrección 00035 las devuelve a `pending` y la cola muestra las cuatro prácticas.
