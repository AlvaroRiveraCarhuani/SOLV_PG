# Tasks: Separar el entorno de práctica del entorno de examen

## Phase 1: Dominio y migración

- [x] 1.1 Agregar `EnvironmentProfile` (`practice`|`exam`), `EnvironmentProfileSpec` y `IsExam()` en `backend/internal/core/domain/practice_environment.go`, sin fijar valores numéricos.
- [x] 1.2 Crear la migración idempotente `environment_profile` y `exam_attempt` con default y el índice por `(student_id, subject_id, exercise_id, exam_attempt)`. Queda no único: puede haber filas previas repetidas y la unicidad se agrega con el alta de intentos (tarea 3.1).
- [x] 1.3 Extender el repositorio Postgres para leer y escribir las dos columnas.
- [x] 1.4 Escribir pruebas unitarias del dominio: perfil por propósito, propósito vacío como `practice`, `IsExam()`.

## Phase 2: Resolución del perfil en servicios

- [x] 2.1 Implementar `resolveEnvironmentProfile(exercise)` en `practice_environment_service.go` leyendo `exercise.purpose`.
- [x] 2.2 Aplicar el perfil en `StartPracticeEnvironmentForExercise` y en el arranque del docente: ambos pasan por `startPracticeEnvironment`, y el DTO no admite un perfil del cliente.
- [x] 2.3 Interruptor `ENV_PROFILE_SPLIT`: descartado con motivo en ADR-083. No restaura los volúmenes anteriores: el perfil y el volumen por intento ya quedaron escritos al aplicar la migración, así que apagarlo no devuelve el comportamiento previo y sí agrega una rama que nadie prueba. Volver atrás exige mover archivos, no cambiar una bandera.
- [x] 2.4 Pruebas unitarias: `class` → `practice`, `exam` → `exam`, sin ejercicio → `practice`, propósito vacío → `practice`. El caso `off` queda con la tarea 2.3.

## Phase 3: Espacio de examen e intentos

- [x] 3.1 Implementar el volumen por intento (`_exam_<n>`, sin sufijo en práctica) y el intento inicial al crear: 0 en práctica y 1 en examen. El resto del ciclo de vida ya usa `VolumeName()`, así que siembra, congelado, entrega y avances quedan atados al intento. El alta del intento 2 o superior depende de la tarea 3.2.
- [x] 3.2 Abrir el recuperatorio sobre un intento nuevo: con el intento cerrado y una convocatoria del mismo ejercicio abierta después de ese cierre, la fila avanza a `exam_attempt + 1`, estrena volumen sembrado desde la versión base y reinicia la siembra. La base admite una sola fila por práctica (`idx_workspaces_practice_scope`), así que el intento se avanza en esa fila y el volumen anterior queda intacto. Sin convocatoria nueva, o si la convocatoria todavía no empezó, el entorno sigue cerrado.
- [x] 3.3 Estado `preparing`/`ready`/`closed`/`failed` calculado del registro (`PracticeEnvironment.State()`), expuesto en cada respuesta del entorno por `MarshalJSON`, sin columna nueva. Una siembra en cuarentena, fallida o con el volumen perdido cuenta como `failed`, no como `preparing`: así el alumno no espera un entorno que no va a abrir. Pruebas de la tabla de estados y del JSON.
- [x] 3.4 Rechazar apertura y entrega sobre un intento congelado. Ya estaba: el alta del entorno devuelve el error de congelado y `SubmitPracticeEnvironmentProject` lo rechaza antes de leer el volumen, con pruebas en `practice_environment_submission_test.go`.
- [x] 3.5 Pruebas con repo falso: primer intento (`practice_environment_exam_creation_test.go`), apertura previa sin contaminación, intento cerrado sin reapertura y recuperatorio con volumen `_exam_2` (`practice_environment_retake_test.go`). Integración con PostgreSQL: convocatoria más nueva solo de exámenes y reinicio del intento en la misma fila (`tests/integration/exam_retake_test.go`).

- [x] 3.6 Corregir el barrido de congelado para que cierre solo el entorno de la práctica del aula: antes tomaba todos los entornos de la materia y bloqueaba prácticas ajenas al examen. Los entornos anteriores al alcance por práctica se siguen incluyendo porque no hay otro dato que los ubique.

## Phase 4: Ciclo de vida y captura de tecleo

- [x] 4.1 Eximir del hibernado por inactividad a los entornos con perfil `exam` en `qos_worker.go`. La hibernación por presión de host no existe en el código: la inactividad es la única vía de apagado automático, así que no hubo nada que conservar.
- [x] 4.2 Rechazar la hibernación explícita de un entorno de examen en `practice_environment_lifecycle.go`: `PausePracticeEnvironment` y `HibernatePracticeEnvironment` devuelven `ErrExamPauseForbidden`, el transporte responde 409 y hay pruebas de pausa, hibernación y del caso de práctica que sigue funcionando.
- [x] 4.3 Rechazar `SaveKeystrokeEvents` para entregas de entornos con perfil `practice`.
- [x] 4.4 Pruebas unitarias: práctica hiberna y examen no, captura aceptada en examen y rechazada en práctica, y registro conservado cuando el catálogo no responde.

## Phase 5: Entrega y contrato HTTP

- [x] 5.1 Hacer accionable el error de copia excedida. La revisión encontró que el tope no daba error: el lector del volumen recortaba en silencio y el alumno entregaba medio proyecto. Ahora la entrega usa `SnapshotPracticeEnvironmentVolumeStrict`, que falla con `ErrPracticeEnvironmentSnapshotTooLarge` nombrando ruta y tamaño; los avisos de clase siguen recortando. Los mensajes de archivos y de archivo suelto dicen cantidad y bytes.
- [x] 5.2 Confirmado: la entrega guarda el snapshot del volumen en `submissions.code` y la corrección lee esa copia. Prueba `TestSubmitPracticeEnvironmentProject_LaEntregaCongelaLaCopia`: después de entregar, cambiar el volumen no cambia lo guardado.
- [x] 5.3 El entorno ya expone `environment_profile` y `exam_attempt` al cliente por las etiquetas JSON del dominio. El `state` de preparación depende de la tarea 3.3 y del aviso en pantalla (6.1): no se agrega un campo que nadie calcula.
- [x] 5.4 Bloquear el cambio de `purpose` con entornos `running` y exigir confirmación auditada con entornos hibernados: `UpdateExerciseWithPurposeConfirmation` cuenta los entornos del ejercicio, responde 409 con el conteo, pide `confirm_purpose_change` y `reason` cuando solo hay entornos cerrados, y deja `EXERCISE_PURPOSE_CHANGED` en la auditoría. La regla vive en `decidePurposeChange`, con tabla de casos.

## Phase 6: Interfaz del alumno

- [x] 6.1 Mostrar los estados "Preparando tu examen", "Estás rindiendo el examen" y "Examen cerrado" en `student-exercise-solver`, con el caso de arranque fallido.
- [x] 6.2 Aviso en el examen: "Tu práctica de esta materia sigue intacta: el examen usa un espacio aparte".
- [x] 6.3 Pruebas de componente con Signals: los tres estados del examen, el aviso de práctica intacta y la ausencia del aviso en una práctica.
- [x] 6.4 Aviso de examen cerrado al entrar: si el alta del entorno devuelve `environment_frozen`, la pantalla marca el examen como cerrado aunque no haya entorno cargado, en vez de mostrar solo un error. El transporte devuelve ese código estable en el alta y en el reinicio.

## Phase 7: Verificación

- [x] 7.1 Prueba de integración con Docker local (`tests/integration/exam_retake_docker_e2e_test.go`, se activa con `EXAM_RETAKE_E2E=1`): el intento nuevo estrena volumen, la siembra escribe solo ahí y el intento anterior conserva su trabajo. El congelado al entregar ya lo cubre `exam_freeze_docker_e2e_test.go`.
- [x] 7.2 La misma prueba de Docker confirma que el volumen de práctica conserva su contenido y que su nombre no lleva sufijo de intento.
- [ ] 7.3 Ejecutar el plan de impacto de `tools/test-impact` sobre las suites afectadas y adjuntar el informe. Primera corrida (nivel L3): `backend-unit`, `backend-integration`, `backend-docker` y `frontend-unit` en verde (1134 tests de frontend); `frontend-e2e` en rojo porque los flujos crean ejercicios sobre una base sin catálogo aprobado ("only approved templates can be promoted to models"), y `build` y `security` quedaron pendientes al cortar el runner en el primer fallo. No cuenta como validado.
- [x] 7.4 ADR-083 escrito e indexado, y `docs/OPERATIONS/capacidad.md` con la sección de volúmenes de examen e intentos.
- [x] 7.5 Trazabilidad actualizada en `docs/00-MAPA_SLICES.md`: fila del cambio con 28/29 y movimiento en el historial.
