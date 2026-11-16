# Tasks: Separar el entorno de práctica del entorno de examen

## Phase 1: Dominio y migración

- [x] 1.1 Agregar `EnvironmentProfile` (`practice`|`exam`), `EnvironmentProfileSpec` y `IsExam()` en `backend/internal/core/domain/practice_environment.go`, sin fijar valores numéricos.
- [x] 1.2 Crear la migración idempotente `environment_profile` y `exam_attempt` con default y el índice por `(student_id, subject_id, exercise_id, exam_attempt)`. Queda no único: puede haber filas previas repetidas y la unicidad se agrega con el alta de intentos (tarea 3.1).
- [x] 1.3 Extender el repositorio Postgres para leer y escribir las dos columnas.
- [x] 1.4 Escribir pruebas unitarias del dominio: perfil por propósito, propósito vacío como `practice`, `IsExam()`.

## Phase 2: Resolución del perfil en servicios

- [x] 2.1 Implementar `resolveEnvironmentProfile(exercise)` en `practice_environment_service.go` leyendo `exercise.purpose`.
- [x] 2.2 Aplicar el perfil en `StartPracticeEnvironmentForExercise` y en el arranque del docente: ambos pasan por `startPracticeEnvironment`, y el DTO no admite un perfil del cliente.
- [ ] 2.3 Agregar el interruptor `ENV_PROFILE_SPLIT` y su rama de comportamiento único.
- [x] 2.4 Pruebas unitarias: `class` → `practice`, `exam` → `exam`, sin ejercicio → `practice`, propósito vacío → `practice`. El caso `off` queda con la tarea 2.3.

## Phase 3: Espacio de examen e intentos

- [x] 3.1 Implementar el volumen por intento (`_exam_<n>`, sin sufijo en práctica) y el intento inicial al crear: 0 en práctica y 1 en examen. El resto del ciclo de vida ya usa `VolumeName()`, así que siembra, congelado, entrega y avances quedan atados al intento. El alta del intento 2 o superior depende de la tarea 3.2.
- [ ] 3.2 Bloquear la reutilización de un volumen previo del mismo ejercicio cuando el perfil es `exam`.
- [ ] 3.3 Exponer el estado `preparing`/`ready` desde el servicio y calcularlo a partir del arranque real del contenedor.
- [ ] 3.4 Rechazar apertura y entrega sobre un intento congelado, con el error de entorno congelado existente.
- [ ] 3.5 Pruebas unitarias con repo falso: primer intento, apertura previa sin contaminación, intento cerrado sin reapertura.

- [x] 3.6 Corregir el barrido de congelado para que cierre solo el entorno de la práctica del aula: antes tomaba todos los entornos de la materia y bloqueaba prácticas ajenas al examen. Los entornos anteriores al alcance por práctica se siguen incluyendo porque no hay otro dato que los ubique.

## Phase 4: Ciclo de vida y captura de tecleo

- [ ] 4.1 Eximir del hibernado por inactividad a los entornos con perfil `exam` en `qos_worker.go`, conservando la hibernación por presión de host con auditoría.
- [ ] 4.2 Rechazar la hibernación explícita de un entorno de examen en `practice_environment_lifecycle.go`.
- [ ] 4.3 Rechazar `SaveKeystrokeEvents` para entregas de entornos con perfil `practice`.
- [ ] 4.4 Pruebas unitarias: QoS con reloj falso (práctica hiberna, examen no), captura aceptada/rechazada.

## Phase 5: Entrega y contrato HTTP

- [ ] 5.1 Hacer accionable el error de copia excedida en `practice_environment_submission.go`, indicando ruta y tamaño.
- [ ] 5.2 Confirmar que la corrección usa la copia entregada y no el volumen vivo, y cubrirlo con una prueba.
- [ ] 5.3 Agregar `profile` y `state` al DTO del entorno en `delivery/http`.
- [ ] 5.4 Bloquear el cambio de `purpose` con entornos `running` y exigir confirmación auditada con entornos hibernados.

## Phase 6: Interfaz del alumno

- [ ] 6.1 Mostrar los estados "preparando tu examen", "examen listo" y "examen cerrado" en `student-exercise-solver`.
- [ ] 6.2 Agregar el aviso de que el espacio de práctica del alumno sigue intacto.
- [ ] 6.3 Pruebas de componente con Signals para los tres estados.

## Phase 7: Verificación

- [ ] 7.1 Prueba de integración con Docker local: volumen de intento nuevo, siembra desde la versión base, congelado al entregar.
- [ ] 7.2 Verificar que un entorno de práctica existente conserva su volumen y su contenido tras el cambio.
- [ ] 7.3 Ejecutar el plan de impacto de `tools/test-impact` sobre las suites afectadas y adjuntar el informe.
- [ ] 7.4 Actualizar ADR e `INDEX.md`, la trazabilidad de slices y `docs/OPERATIONS/capacidad.md`.
