# Feature: Aula en Vivo por Sesión de Clase

Objetivo: que el docente pueda entrar a una práctica mientras ocurre, ver el estado real de sus alumnos y actuar sobre lo que pasa ahora, sin mezclarlo con el trabajo pendiente del curso.

Estado: **etapa 2 entregada (2026-10-30)**. Alcance cerrado: `class_sessions` con abrir y cerrar, presencia acotada a la sesión y la pantalla de aula con faltantes y trabados. Lo que no entró está en §Pendientes.
Diseño de referencia: `docs/UI/DOCENTE/REDISENO_PANEL_DOCENTE.md` (§5.2, §6, §12.1) · [ADR-056](../../docs/ARQUITECTURA/ADR/ADR-056-sesion-de-aula-y-presencia-acotada.md).

## Decisiones cerradas

- **La sesión es la unidad que faltaba.** Se llama `class_sessions`, no `lab_sessions`: la clase es la que tiene hora, no la práctica. Una sesión abierta por materia, garantizada por índice parcial único. Motivo: sin esa entidad nada significa "esta clase está pasando" y lo "en vivo" se aproxima con heartbeats globales.
- **La sesión es opcional.** Una práctica se trabaja sin aula abierta y el panel no se rompe: sin sesión, la pantalla muestra los entornos encendidos y ninguna lista de faltantes, porque no hay conjunto contra el cual contar.
- **La sesión se ata al alumno, no al contenedor.** `class_session_participants` es la presencia; `workspaces.session_id` solo deja constancia de en qué aula nació un entorno. Un fallo al enlazar nunca impide que el alumno trabaje.
- **Presencia con ventana.** Participante con `last_seen_at` dentro de 120 s. El alumno consulta sus aulas abiertas cada minuto y late solo con la pestaña visible; un 409 (aula cerrada) o un 403 (dejó de estar matriculado) corta el latido.
- **Estar en el aula y tener entorno vivo son dos cosas.** La fila muestra las dos por separado: un juez efímero no tiene contenedor y el alumno sigue en clase.
- **Solo las excepciones con dueño docente entran al aula.** OOM y contenedor caído son del admin; el bloqueo de código o el error repetido dentro de la clase son del docente.
- **El aula es lista, no grid.** Una fila por alumno con un solo punto de color por estado.
- **El cliente no suma ni recorta.** El estado de cada fila y los conteos del marcador se resuelven en SQL, sobre las mismas filas que la pantalla pinta.

## Specs entregadas

- **S1 — Esquema de sesión.** `backend/migrations/00036_class_sessions.sql` (Up/Down): `class_sessions(id, tenant_id, subject_id, exercise_id NULL, status CHECK ('open','closed'), opens_at, closes_at NULL, created_by, instructions, created_at, updated_at)`, índice parcial único `(tenant_id, subject_id) WHERE status='open'`, `class_session_participants` con PK `(session_id, student_id)` y `workspaces.session_id` nullable. `backend/migrations/00037_class_session_ends_at.sql` agrega `ends_at` (fin previsto), rellena las filas existentes con su cierre o 90 minutos desde la apertura, y lo declara `NOT NULL`.
- **S2 — Dominio y ciclo de vida.** `core/domain/class_session.go` y `core/services/class_session_service.go`: abrir dos veces la misma materia devuelve `ErrSessionAlreadyOpen` (409), cerrar dos veces devuelve el mismo estado, y un docente ajeno no ve ni cierra el aula. La práctica de la sesión debe pertenecer a la materia. La duración se elige al abrir (90 min por defecto, 15–480); fuera de rango es `ErrSessionDurationInvalid` (400).
- **S2.1 — Cierre por vencimiento.** El repositorio cierra las aulas vencidas de la institución antes de abrir, leer el aula actual, armar el tablero, listar las aulas del alumno o registrar un latido: `UPDATE ... WHERE status='open' AND ends_at <= NOW()` más el `left_at` de sus participantes, en una sola sentencia con CTE. No hay proceso periódico que operar.
- **S3 — Presencia y tablero.** `PostgresClassSessionRepository.Board` resuelve sesión y alumnos en dos consultas fijas, sin bucle por alumno. Estado por fila: `trabado | sin_empezar | ausente | entregado | trabajando`, calculado en SQL y devuelto ya ordenado por urgencia docente.
- **S4 — Endpoints.** `POST /api/v1/teacher/sessions` (`subject_id`, `exercise_id?`, `duration_minutes?`), `PATCH /api/v1/teacher/sessions/{id}/close`, `GET /api/v1/teacher/sessions/open`, `GET /api/v1/teacher/sessions/{id}/board`, `GET /api/v1/student/sessions/active`, `POST /api/v1/student/sessions/{id}/heartbeat`.
- **S5 — Frontend.** `/teacher/aula` con cabecera de sesión (apertura y fin), marcador silencioso ("Todo en orden: N de M entregaron, 0 trabados"), filtros por estado, búsqueda, tabla de alumnos y telemetría acotada a la materia de la sesión. Sin aula, formulario de apertura con duración y entornos abiertos; si el aula anterior venció, lo dice con su hora. La cabecera de "Hoy" lleva `[Abrir aula]` cuando no hay sesión y la fila de sesión cuando la hay.
- **S6 — Presencia del alumno.** `ClassPresenceService` arranca con el layout del estudiante y late cada minuto.
- **S7 — Verificación.** `go build ./...`, `TestSlice19_ClassSession_Aula` (apertura, conflicto, docente ajeno, presencia por señal, alumno no matriculado, orden del tablero, cierre idempotente, duración fuera de rango y aula vencida que deja abrir otra), `ng build`, `npm run lint:styles`, `eslint` y las pruebas del panel.

## Tasks

- [x] **S1**: migración de sesiones y participantes con su índice parcial único.
- [x] **S2**: ciclo de vida de la sesión con errores de dominio y no de base de datos.
- [x] **S3**: tablero en dos consultas fijas, con el estado calculado en SQL.
- [x] **S4**: endpoints de apertura, cierre, sesión actual, tablero y latido del alumno.
- [x] **S5**: consola del aula con orden por excepción, estado silencioso y filtros.
- [x] **S6**: señal de presencia desde el cliente del alumno.
- [x] **S7**: matriz de verificación, ADR-056, contratos y documentación de la vista.
- [x] **S8**: hora de fin al abrir y cierre por vencimiento, sin proceso periódico.

## Pendientes

- Modo proyección (sidebar oculto, tipografía mayor, avance del grupo con nombres grandes).
- Agrupación de excepciones por causa y bloque `signals` en el resumen del panel.
- Código sin entregar con consentimiento del alumno y acceso auditado.
- Actualización por SSE con marca de frescura; hoy es sondeo de 5 s.
- Resumen de trabados para el trabajo fuera de clase ("3 alumnos se trabaron desde tu última visita").
- Prórroga de una clase en curso: hoy, si la clase se extiende, el docente cierra y abre otra aula.
- Exportación de asistencia en CSV.

## Log

- **L1 (verbatim):** "Tengo planeado que para un laboratorio, se pueda 'entrar', el docente pueda entrar, y en allí si pueda tener difentes acciones que pueda realizar, algo como si fuera una clase virtual, ver los estuidantes conectados, ver eso de 'telemetria' que ahora mismo esta como en la pantalla principal."
- **L2 (verbatim):** "Solo lo entregado, y en vivo solo con consentimiento (Recomendado)" — respuesta a la visibilidad del código del alumno.
- **L3 (verbatim):** "Sí, registro exportable (Recomendado)" — respuesta sobre asistencia.
- **L4 (verbatim):** "ahora mimso el sistema no es para poner asistencias ni nada parecido, si hay paralelos, la idea tambien es que un docente con un curso pueda tener muchos laboratorios, asi esta en la bd" — de ahí el supuesto de un paralelo por materia y la ausencia de módulo de asistencia.
- **Hallazgo verificado:** no existía tabla de sesión ni de participantes; lo "en vivo" salía de `workspaces.last_heartbeat_at`, y la telemetría contaba `running|active|pending`, de ahí las dos cifras distintas de activos.
- **Hallazgo verificado:** un mismo docente puede tener varias materias con aula abierta, así que la pantalla toma una sesión por vez (la más reciente) y la fila del panel muestra esa misma.
