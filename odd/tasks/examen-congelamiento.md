# Feature: Congelamiento del Examen por Volumen

Objetivo: que al terminar un examen el trabajo de cada alumno quede en solo lectura de verdad —desde el sistema de archivos, no desde la pantalla— y que el docente conserve la lectura y una excepción individual auditada.

Estado: **circuito completo (2026-10-31)**. El volumen se congela al cerrar el aula, al vencer `ends_at`, desde el barrido y cuando un alumno entrega antes —solo su volumen—; el alumno ve el aviso, la pantalla de entrega pide confirmación y el docente cierra el examen, ve cuántos volúmenes quedaron en solo lectura y reabre a un alumno con motivo desde `/teacher/aula`. Probado contra Docker y el volumen reales.
Decisión de referencia: [ADR-005 §5](../../docs/ARQUITECTURA/ADR/ADR-005-sistema-evaluacion-juez-virtual-dual.md#5-bloqueo-de-edición-al-cerrar-un-examen-mecanismo-de-la-fase-b) · relacionado con [ADR-058](../../docs/ARQUITECTURA/ADR/ADR-058-avance-voluntario-del-docente-en-aula.md) (examen sin avances).

## Decisiones cerradas

- **Montaje `:ro`, no bloqueo del editor.** El editor se sirve por su propia URL y trae terminal: un cerrojo de pantalla se abre copiando una dirección. El kernel es el único que aplica el bloqueo a todo proceso. Comparación completa en ADR-005 §5.3–5.4.
- **Estado en la base, no pausa.** `workspaces.frozen_at` y `status = 'frozen'`. La pausa se descartó porque la reanudación del laboratorio despausa antes de mirar otra cosa, así que el primer alumno que volviera a entrar recuperaba la escritura.
- **Recrear con la configuración de siempre.** Congelar y descongelar reutilizan el alta del contenedor con `ReadOnly` en `true` o `false`; no hay un segundo camino de montaje que pueda quedar desincronizado.
- **El cierre reporta, no supone.** La respuesta del cierre y la acción explícita devuelven `{ frozen, skipped, failed }`. Un fallo no aborta el resto y el barrido lo reintenta.
- **Solo exámenes.** Un aula de práctica nunca congela volúmenes. La sesión de clase congelada de ADR-058 es otra cosa: bloquea avances y no toca el volumen de nadie.
- **La excepción es individual y con motivo.** Reabrir un alumno exige texto, queda en `audit_logs` con actor, alumno y motivo, y se puede repetir sin efecto.
- **El barrido cubre lo que el cierre no pudo.** Corre al arrancar y cada 15 minutos, así que un examen vencido con el backend detenido se congela igual.

## Specs entregadas

- **E1 — Esquema.** `backend/migrations/00041_workspace_exam_freeze.sql` (idempotente): `workspaces.frozen_at TIMESTAMPTZ` con índice parcial para el barrido. `status` ya es texto, así que `frozen` no necesita cambiar la restricción; queda documentado como estado válido y neutro: no libera cupo ni se trata como fallo.
- **E2 — Docker.** `WorkspaceContainerConfig.ReadOnly` agrega el sufijo `:ro` al bind del volumen; la recreación detiene, quita, crea y arranca el contenedor con la misma configuración. El arranque no bloquea: la primera escritura espera a que el proceso esté en ejecución. `workspaceVolumeBind` queda cubierto por `internal/infrastructure/docker/workspace_bind_test.go`.
- **E3 — Servicio y ganchos.** `core/services/exam_freeze.go`: congelar por aula, por alumno y barrido; `SetWorkspaceMount` en `workspace_freeze.go` reconstruye el entorno. El cierre del aula (`CloseAula`) congela y devuelve el reporte; un aula de clase responde "no es examen" y no cambia el resultado.
- **E4 — Reanudación.** Con `frozen_at` no nulo, abrir o reiniciar responde `409` con `ErrWorkspaceFrozen`; el alumno no recupera la escritura recargando ni pidiendo reanudar.
- **E5 — API.** `POST /api/v1/teacher/sessions/{id}/exam-freeze` devuelve el conteo; `POST /api/v1/teacher/sessions/{id}/students/{studentID}/unfreeze` exige `reason`. El cierre incluye `freeze_report`. Roles: el alumno recibe `403`; un aula ajena, `404`; un examen abierto, `409`.
- **E6 — Worker.** `ExamFreezeWorker` arranca con el backend y barre cada 15 minutos (`ClosedExamSessions`, tope de 50 por ciclo).
- **E7 — Pantalla del alumno.** Con estado `frozen` el laboratorio muestra «El examen terminó: tu entorno quedó en solo lectura», con la hora del cierre en `<time>`, y no ofrece iniciar el entorno. El estado `frozen` se pinta como neutro en `status-badge`.
- **E8 — Documentación.** ADR-005 §5.9, `docs/UI/ESTUDIANTE/LABORATORIO_ACTIVO.md`, `docs/UI/MAPA_DE_VISTAS.md` y `docs/API_CONTRACTS.md`.
- **E9 — Pantalla del docente.** La sesión trae `purpose` (`class` o `exam`) desde la práctica. En `/teacher/aula` el examen se marca en la cabecera y el cierre pasa a `[Cerrar examen]`, con un aviso de que los entornos quedan en solo lectura. Cerrado, un panel muestra el conteo real (`frozen`, `skipped`, `failed`) y la lista de alumnos con entorno; cada uno en solo lectura ofrece `[Reabrir]`, que abre el modal del motivo, y el panel permite repetir el congelamiento sin reabrir la clase.
- **E10 — Entrega anticipada por alumno.** La entrega del proyecto (`POST /api/v1/workspaces/{id}/submit`) revisa el `purpose` del ejercicio: si es examen, después de registrar la entrega cierra solo ese volumen (montaje `:ro` + `frozen`), audita el laboratorio como recurso y deja la clase abierta. Una entrega posterior sobre un volumen congelado responde `409`, para que la vía de escritura del submit no reabra la edición. La pantalla pide confirmación antes de entregar y deshabilita el botón después.

## Tasks

- [x] **E1**: migración de `frozen_at` y estado `frozen`.
- [x] **E2**: `ReadOnly` en el contenedor y recreación del montaje.
- [x] **E3**: servicio de congelamiento con conteo, auditoría y gancho al cerrar.
- [x] **E4**: bloqueo de la reanudación.
- [x] **E5**: endpoints de congelar y de excepción individual.
- [x] **E6**: worker de recuperación.
- [x] **E7**: aviso en el laboratorio del alumno.
- [x] **E8**: documentación y contratos.
- [x] **E9**: controles del docente en `/teacher/aula` (marca de examen, cierre, conteo, reintento y excepción con motivo).
- [x] **E10**: cierre anticipado por alumno al entregar el examen, con confirmación en la pantalla y entrega bloqueada después.

## Pruebas

- `backend/internal/core/services/exam_freeze_test.go`: monta en solo lectura y salta los ya congelados; rechaza un examen abierto y un aula de clase; un fallo de montaje no frena al resto; reabrir exige motivo y vuelve a `:rw`; el barrido recorre cada aula.
- `backend/internal/core/services/workspace_freeze_test.go`: un entorno congelado no se reanuda ni se reactiva.
- `backend/internal/core/services/class_session_close_test.go`: el cierre devuelve el conteo y no congela si el aula no se cerró.
- `backend/tests/integration/exam_freeze_test.go`: contra PostgreSQL, qué aulas son examen, qué laboratorios participan, qué queda pendiente de congelar, la marca en la fila y la reapertura.
- `frontend/src/app/features/student/workspaces/active-lab/student-active-lab.component.spec.ts`: con el examen cerrado se muestra el aviso, no hay iframe ni botón de inicio.
- `frontend/src/app/features/teacher/services/class-session.service.spec.ts`: el cierre conserva `freeze_report`, el reintento lo actualiza, la excepción individual envía el motivo y el tablero cerrado se lee sin reabrir el aula.
- `frontend/src/app/features/teacher/aula/unfreeze-student-modal/unfreeze-student-modal.component.spec.ts`: el motivo es obligatorio, la excepción se envía con sesión y alumno, y el error del servidor se muestra sin cerrar el modal.
- `backend/internal/core/services/exam_freeze_workspace_test.go`: congelar un volumen suelto monta en solo lectura, marca la fila y audita; un volumen ya congelado no se reconstruye; un fallo de montaje no marca la fila.
- `backend/internal/core/services/workspace_submission_test.go`: entregar un examen congela solo ese volumen, una práctica de clase no congela nada y un volumen congelado rechaza la entrega sin leerlo.
- `frontend/src/app/features/student/exercises/solver/student-exercise-solver.component.spec.ts`: el primer clic no entrega un examen y pide confirmación; al confirmar, la entrega se registra y no queda habilitada otra.
- `backend/tests/integration/exam_freeze_docker_e2e_test.go`: contra Docker y el volumen reales, el alumno escribe, entrega el examen y la escritura falla con `Read-only file system` desde el editor y desde un contenedor efímero en `:ro`; la lectura sigue disponible, una segunda entrega responde `409` sin escribir y reabrir devuelve la escritura. Apagada por defecto: `EXAM_FREEZE_E2E=1 go test ./tests/integration -run TestExamFreezeDockerE2E`.

## Pendientes

- Ninguno.

## Log

- **L1 (verbatim):** "Definí cómo se bloquea la edición durante un examen: compará solo lectura del entorno contra bloqueo del editor, recomendá uno y dejalo escrito en la ADR."
- **L2:** se eligió el montaje `:ro` porque es el único que impide escribir desde el terminal y desde cualquier proceso. El bloqueo del editor y el corte por proxy se descartaron por saltables; la pausa, porque la reanudación del laboratorio la deshace sola.
- **L3:** el congelamiento no reutiliza la recreación de la pausa. Tiene su propio camino, que reconstruye el contenedor con la misma configuración; si falla, el cierre ya ocurrió y el barrido reintenta.
- **L4:** el reporte del cierre viaja en la misma respuesta de `PATCH /close` como `freeze_report`, en lugar de una segunda llamada: el docente cierra y ya sabe cuántos quedaron. Los clientes que solo leen la sesión ignoran el campo.
- **L5:** el aviso del alumno se apoya en `frozen_at` que ya devuelve la lectura de la materia; no hizo falta un endpoint nuevo. El estado `frozen` se pintó neutro, como la pausa, porque es un estado buscado y no un error.
- **L6:** la pantalla no deduce el examen del título de la práctica: la sesión devuelve `purpose` desde `exercises.purpose`, con `class` cuando el aula no tiene práctica. Se expuso en el mismo SELECT de la sesión para que el panel y el cierre compartan el dato. Después del cierre, la lista de entornos se lee del tablero por identificador en lugar de inventar un endpoint nuevo: el tablero ya trae `workspace_status` y funciona con la sesión cerrada. El reporte del cierre no se pisa al reabrir a un alumno; el conteo se refresca desde el tablero, y el reintento del congelamiento sí actualiza el reporte.
- **L7:** la entrega anticipada no creó un endpoint nuevo: se engancha en la entrega que ya existe, porque es la única vía por la que el alumno puede escribir en el volumen después de entregar. Si el submit siguiera aceptando un volumen congelado, el payload de archivos sería un bypass del cierre; por eso una segunda entrega responde `409`. El cierre corre después de registrar la entrega: si falla, la entrega ya quedó y el barrido del aula lo reintenta. La confirmación en la pantalla es parte del mecanismo: entregar un examen no se deshace.
- **L8 (verbatim):** "Escribí la prueba de punta a punta con Docker: escribir en el volumen, congelar por entrega y comprobar que la escritura falla desde el editor y desde un contenedor efímero."
- **L9:** la prueba dispara el congelamiento por la entrega, no llamando al servicio: arma el servicio de entornos con Docker real y le inyecta el congelador real, así que el `:ro` lo aplica el mismo camino que usa la aplicación. Las escrituras de prueba corren como root dentro del contenedor para que el fallo sea del montaje y no de un permiso de directorio. Al escribirla apareció que la recreación no espera a que el editor arranque: la prueba espera el estado `running` antes de ejecutar, y la documentación quedó corregida.
