# Feature: Aula en Vivo por Sesión de Laboratorio

Objetivo: que el docente pueda entrar a una práctica mientras ocurre, ver el estado real de sus alumnos y actuar sobre lo que pasa ahora, sin mezclarlo con el trabajo pendiente del curso.
Branch: `feat/teacher-live-session` (desde `main` tras `feat/teacher-review-and-gradebook`) | Delivery strategy: ask-on-risk | Test runner: `go test ./...` & `npm run test:ci`

Forecast de líneas autoradas: ~1800 (backend ~900, frontend ~800, docs ~100). Supera el presupuesto de 400 → entrega en cadena por spec, no en un solo cambio.

Diseño de referencia: `docs/UI/DOCENTE/REDISENO_PANEL_DOCENTE.md` (§2.2, §3.2, §5).

## Decisiones cerradas

- **La sesión es la unidad que faltaba.** `lab_sessions` + `session_participants`. Una sesión abierta por materia (índice único parcial). Motivo: hoy no existe nada que signifique "esta clase está pasando", así que lo "en vivo" se aproxima con heartbeats globales y dos widgets dan cifras distintas de activos.
- **La sesión se ata al alumno, no al contenedor.** El workspace se enlaza si ya existe; no se crea ni se migra ninguno. Motivo: no tocar el ciclo de vida de Docker que hoy funciona.
- **Estar en la sesión y tener entorno vivo son dos cosas.** El read model expone `in_session` y `has_live_environment` por separado. Motivo: en un juez efímero no hay contenedor y aun así el alumno está en clase; si se mezclan, se reproduce la contradicción actual.
- **Un entorno abierto fuera de sesión no aparece en el aula.** Aparece en Materia.
- **Solo las excepciones con dueño docente entran al aula.** OOM y contenedor caído son de admin; la regla de código bloqueada dentro de sesión es del docente; fuera de sesión es bitácora.
- **Presencia con ventana.** Participante con `last_seen_at` dentro de 120 s. Sin heartbeat de workspace, la fila sigue mostrando al alumno como presente pero sin entorno.
- **Ver código en vivo exige consentimiento.** El código entregado es visible siempre; el editor sin entregar requiere que el alumno acepte por sesión, y cada acceso queda auditado con actor y hora.
- **Asistencia registrada, sin módulo de asistencia.** Se guarda `joined_at` y `left_at`, con exportación CSV y política de retención documentada. No hay pantalla de pasar lista.
- **La lista tiene un orden que sirve para dar clase.** Excepciones del docente primero (trabados, sin empezar), luego quienes entregaron, luego el resto; alfabético dentro de cada grupo.
- **El aula es consola, no grid.** Lista maestra a la izquierda, detalle a la derecha. Treinta tarjetas no se escanean; treinta filas de 44 px sí.

## Specs

- **S1 — Esquema de sesión.**
  - Migración `backend/migrations/00035_lab_sessions.sql` (Up/Down):
    - `lab_sessions(id, tenant_id, subject_id, exercise_id NULL, module_id NULL, opened_by, status CHECK ('scheduled','open','closed'), opens_at, closes_at, instructions TEXT DEFAULT '', created_at, updated_at)`.
    - Índice único parcial `(subject_id) WHERE status='open'`.
    - `session_participants(session_id, student_id, joined_at, last_seen_at, left_at, workspace_id NULL, code_view_consented_at NULL)` con PK `(session_id, student_id)`.
  - Criterio: abrir una segunda sesión en la misma materia falla con conflicto de dominio, no con error de base de datos.

- **S2 — Dominio y ciclo de vida.**
  - `domain/lab_session.go` con transiciones `scheduled → open → closed`, apertura y cierre idempotentes, y validación de que `exercise_id` pertenece a la materia.
  - Unirse con sesión abierta o programada; si está cerrada, se rechaza salvo configuración de la sesión (`allow_late_join`, por defecto falso).
  - Criterio: cerrar dos veces devuelve el mismo estado sin error; unirse a una sesión cerrada sin permiso devuelve 409.

- **S3 — Presencia y tablero.**
  - `TeacherSessionStore.Board(ctx, tenantID, sessionID)` con una sola consulta agregada (sin N+1): por alumno `{student_id, name, joined_at, last_seen_at, in_session, has_live_environment, submitted_at, exercise_state, outcome, exceptions[]}`.
  - Estado del alumno: `sin_empezar | empezando | trabado | entregado | ausente`.
  - Criterio: con 30 participantes y 30 workspaces, el tablero se resuelve en una consulta y responde por debajo de 150 ms en local.

- **S4 — Excepciones con dueño.**
  - Read model de excepciones con `owner ∈ {docente, alumno, sistema, admin}` y filtro fijo `owner='docente'` en el tablero.
  - Agrupación por causa: mismo error de compilación en N alumnos, misma regla bloqueada, mismo caso de prueba fallido.
  - Criterio: 8 alumnos con el mismo error aparecen como una fila de grupo con "8", no como 8 filas.

- **S5 — Acciones del aula.**
  - `POST /api/v1/teacher/sessions` (abrir), `PATCH /{id}/close`, `POST /{id}/join` (alumno), `PATCH /{id}/heartbeat`, `GET /{id}/board`.
  - `POST /{id}/messages` con destinatario individual o grupo (`student_ids`), que crea notificación `in_app` con `event_type` y `metadata`.
  - `POST /{id}/participants/{student_id}/consent` para habilitar la vista de código en vivo; `GET /{id}/participants/{student_id}/code` exige consentimiento vigente y escribe auditoría.
  - Modalidad IDE: `POST /api/v1/teacher/workspaces/{id}/extend-memory` y `/restart`, reutilizando el servicio de workspaces existente.
  - Criterio: sin consentimiento, el endpoint de código devuelve 403 y deja rastro en auditoría; la consigna al grupo crea una notificación por alumno.

- **S6 — Frontend: consola del aula.**
  - `teacher/session/:id` con `session-header`, `student-row` y `exception-row`; filtros con `status-tabs`, búsqueda con `search-bar`, detalle sin modal.
  - Un solo punto de color por fila según estado; todo lo demás con `var(--text-*)` y `var(--border-subtle)`; valores técnicos con `[machineData]`; fechas con `dateText` dentro de `<time>`.
  - Estado silencioso "Todo en orden: 24 de 28 entregaron, 0 trabados" cuando no hay excepciones.
  - Teclado: flechas para moverse, Enter para abrir, foco visible; `aria-live="polite"` en el contador de excepciones.
  - Modo proyección: oculta el sidebar, sube la tipografía y muestra solo el avance del grupo.
  - Criterio: la pantalla cabe sin scroll horizontal a 1100×640 útiles; con datos de 30 alumnos se ve el estado del grupo sin desplazarse.

- **S7 — Actualización en vivo.**
  - Tablero por SSE (`GET /api/v1/teacher/sessions/{id}/stream`) con evento cada cambio de estado; reconexión con `EventSource` y fallback a sondeo de 5 s.
  - Indicador de frescura: "actualizado hace N s" en ámbar cuando la conexión cae, sin borrar la lista.
  - Criterio: cortar la red deja la lista visible con el indicador; al volver, se recupera sin recargar.

- **S8 — Verificación y documentación indivisible.**
  - `go vet ./...` + `go test ./...`; integración nueva sobre `solv_test` con una sesión de dos alumnos, cierre y reintento de unión.
  - `npm run lint:styles && npm run lint:ts && npm run test:ci && CI=true npm run build`.
  - ADR nuevo (sesión de laboratorio, presencia y propiedad de excepciones, con alternativas descartadas: atar workspace a sesión, presencia por heartbeat global) y entrada en `INDEX.md`.
  - `docs/API_CONTRACTS.md` con el bloque de sesiones; `docs/UI/DOCENTE/` con la vista del aula y su política de retención de asistencia.

## Tasks

- [ ] **S1**: migración de sesiones y participantes con su índice único parcial.
- [ ] **S2**: ciclo de vida de la sesión y reglas de unión, con tests de transición.
- [ ] **S3**: read model del tablero en una sola consulta.
- [ ] **S4**: excepciones con dueño y agrupación por causa.
- [ ] **S5**: endpoints de apertura, tablero, consigna, consentimiento y acciones de entorno.
- [ ] **S6**: consola del aula con orden por excepción, estado silencioso, teclado y modo proyección.
- [ ] **S7**: SSE con fallback y comunicación de frescura.
- [ ] **S8**: matriz de verificación, ADR, contratos y documentación de la vista.

## Log

- **L1 (verbatim):** "Tengo planeado que para un laboratorio, se pueda 'entrar', el docente pueda entrar, y en allí si pueda tener difentes acciones que pueda realizar, algo como si fuera una clase virtual, ver los estuidantes conectados, ver eso de 'telemetria' que ahora mismo esta como en la pantalla principal."
- **L2 (verbatim):** "Solo lo entregado, y en vivo solo con consentimiento (Recomendado)" — respuesta a la visibilidad del código del alumno.
- **L3 (verbatim):** "Sí, registro exportable (Recomendado)" — respuesta sobre asistencia.
- **L4 (verbatim):** "ahora mimso el sistema no es para poner asistencias ni nada parecido, si hay paralelos, la idea tambien es que un docente con un curso pueda tener muchos laboratorios, asi esta en la bd" — de ahí el supuesto de un paralelo por materia y la ausencia de módulo de asistencia.
- **Hallazgo verificado:** no existe tabla de sesión ni de participantes; lo "en vivo" sale de `workspaces.last_heartbeat_at`, y la telemetría cuenta `running|active|pending`, de ahí las dos cifras distintas de activos.
- **Hallazgo verificado:** `notifications` guarda `event_type`, `metadata` y `link`, suficiente para que la consigna se componga con el vocabulario del cliente sin prosa en el backend.
