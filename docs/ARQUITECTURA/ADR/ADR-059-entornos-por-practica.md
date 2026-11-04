# ADR 059: Entornos Acotados a una Práctica (Docente y Alumno)

> **Estado:** Implementado
> **Fecha:** 2026-11-01
> **Ámbito:** Docencia, Core, Workspaces (docente y alumno)
> **Relacionado:** ADR-001 (volumen nombrado), ADR-005 (examen y congelación), ADR-011 (ciclo de vida e hibernación), ADR-050 (servicios satélite), ADR-052 (discriminador de modalidad por entorno), ADR-058 (avance del docente en el aula)

---

## Contexto

La ficha de una práctica (`/teacher/courses/:courseId/exercises/:id`) mostraba la plantilla, la base
satélite, la política de entregas y la rúbrica, y ofrecía *Probar Entorno*, que valida el entorno en
un contenedor efímero. No había forma de abrir el repositorio de esa práctica: el entorno del docente
era el de la materia, con un volumen por `(docente, materia)`.

Eso produce dos problemas concretos:

1. El docente no puede preparar el repositorio de una práctica desde su ficha. Tiene que abrir el
   entorno de la materia, que es otra pantalla y otro trabajo.
2. El volumen por materia mezcla el código de todas las prácticas de la materia. Abrir la práctica 3
   después de la práctica 1 deja el repositorio de la primera y el docente no sabe qué está mirando.

El esquema de `workspaces` tampoco distinguía entornos dentro de una materia: la clave de reutilización
era `(student_id, subject_id)` y la tabla no tenía forma de acotar un entorno a un ejercicio.

## Decisión

### 1. El entorno se acota por práctica

Un entorno (de docente o de alumno) puede declarar el ejercicio al que pertenece
(`workspaces.exercise_id`). Con ejercicio declarado:

- el volumen es `solv_workspace_<docente>_<materia>_<práctica>`, distinto del volumen por materia;
- la clave de reutilización es `(tenant_id, student_id, subject_id, exercise_id)`, con índice único
  parcial `idx_workspaces_practice_scope` para que una persona no tenga dos entornos de la misma
  práctica;
- la plantilla y el `db_init_script` salen del ejercicio, igual que en el camino del alumno.

Los entornos por materia (el del docente y los de los alumnos) mantienen `exercise_id` nulo y el
volumen histórico `solv_workspace_<usuario>_<materia>`. La migración `00042_workspace_exercise_scope`
es idempotente y no reescribe filas existentes; el índice parcial cubre a docentes y alumnos.

### 2. La ficha abre ese entorno en una pestaña nueva

`POST /api/v1/teacher/courses/{courseId}/exercises/{exerciseId}/ide` resuelve la práctica, verifica
que la materia sea del docente (o que el rol sea `admin`), y devuelve `workspace_id`, `template_id`,
`status`, `volume_name` y `access_url`.

- Solo prácticas `IDE_PERSISTENTE`: una práctica del juez responde `400 not_project_practice`, porque
  no tiene repositorio persistente que abrir.
- Un entorno ya corriendo se devuelve tal cual; uno pausado se reanuda. No se duplican contenedores
  ni puertos.
- Un volumen congelado (examen cerrado, ADR-005) responde `409`: abrir el IDE no puede saltarse el
  cierre.
- El entorno por práctica del docente no se inscribe en el aula de la materia: ese tablero es de
  alumnos.

En la interfaz, *Abrir IDE* es la acción principal de la cabecera y *Probar Entorno* queda como acción
secundaria: la primera abre el repositorio real, la segunda valida el entorno sin crear workspace. La
pestaña se abre antes de esperar la respuesta para no quedar sujeto al bloqueador de ventanas
emergentes; si la respuesta falla, la pestaña se cierra y el error se informa en la misma ficha. Si
el navegador bloqueó la ventana, el entorno se abre en la pestaña actual en lugar de no hacer nada.

### 3. El alumno también abre su entorno por práctica

`StartWorkspaceForExercise` acota el entorno al ejercicio cuando llega `exercise_id`. La clave de
reutilización es `(alumno, materia, práctica)` y el volumen `solv_workspace_<alumno>_<materia>_<práctica>`.
Abrir la práctica 2 ya no devuelve el entorno de la materia ni escribe en su volumen: cada práctica
tiene su propio repositorio, igual que el del docente. El chequeo de presencia y el enlace con el
aula se saltan solo en el camino del docente: el alumno sí es participante.

El avance del docente (ADR-058) se aplica en el volumen de la práctica que el alumno abrió, que es el
que tiene delante en el laboratorio.

**Actualización (2026-11-04).** El entorno por materia se retiró del todo. En el área del alumno se
eliminaron `/student/workspaces` y `/student/workspaces/:subjectId` con sus componentes, y el aviso
de avance llega al solver de la práctica, que filtra por `exercise_id`. En el backend se eliminaron
`GET /api/v1/workspaces`, `WorkspaceForSubject`, `GetByStudentAndSubject` y el arranque sin práctica:
`POST /api/v1/workspaces/start` exige `subject_id` y `exercise_id`, y `WorkspaceVolumeName` ya no
compone el nombre por materia. El módulo docente dejó de cargar y pausar entornos por materia: solo
abre el IDE de una práctica. El avance del aula usa `session.ExerciseID` como única fuente; el modal
no ofrece selector y un aula sin práctica responde `409` con `ErrAdvancePracticeRequired`. Se
eliminaron los volúmenes `solv_workspace_<alumno>_<materia>` y sus filas en `workspaces`, sin
migrarlos: eran datos de prueba.

## Consecuencias

- El docente prepara cada práctica en su propio repositorio, y el código de una práctica no aparece en
  otra de la misma materia.
- Un docente con varias prácticas activas ocupa un contenedor por práctica: el cupo `max_containers`,
  la admisión por RAM del host y la hibernación por inactividad siguen aplicando igual. El alumno abre
  solo el entorno de la práctica que está resolviendo.
- Los volúmenes nuevos se crean al abrir cada práctica. No queda ningún volumen por materia del
  alumno: los que existían se eliminaron y el inicio ya no puede crearlos.
- El inicio del alumno se rehízo sobre ese modelo: muestra la clase abierta, las entregas próximas y
  el avance por materia, sin tarjetas de entorno ni personalización de widgets. `StudentSubjectItem`
  perdió `active_workspace`; la lista y el detalle de materia dejaron de anunciar un entorno propio y
  remiten a cada práctica.
- `GetByIDAndTenant` de ejercicios tolera `description` nula (`COALESCE`), porque el camino del IDE
  lee el ejercicio completo y una práctica sin descripción dejaba de resolverse.

## Alternativas descartadas

| Alternativa | Motivo |
| --- | --- |
| Reutilizar el volumen de la materia para todas las prácticas | Mezcla repositorios y hace imposible saber qué código pertenece a qué práctica |
| Abrir el IDE dentro del entorno de la materia | No cumple el flujo de la ficha: obliga a salir de la práctica y pierde el contexto |
| Un contenedor nuevo por cada apertura | Duplica contenedores y puertos con el mismo volumen; la reutilización por `(docente, materia, práctica)` ya da la idempotencia |
| Permitir abrir el IDE de una práctica del juez | No existe repositorio persistente que abrir; la validación de esa práctica es *Probar Solución* |
