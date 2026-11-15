# Alcance por institución en consultas y rutas

Regla corta: **toda lectura o escritura que reciba un id desde la petición filtra
por la institución del contexto autenticado**. No hay ids "de confianza" porque
un uuid sea difícil de adivinar.

## De dónde sale la institución

`TenantMiddleware` valida el token y escribe `domain.TenantIDKey` en el contexto
del request. La capa de storage lo lee con `domain.GetTenantID(ctx)` y lo usa
como parámetro de la consulta. Los repositorios que reciben `tenantID` por
parámetro lo hacen cuando el llamador ya lo resolvió (jobs, CLI, servicios).

Si el contexto no trae institución, `tenantIDFromContext` y el repositorio de
entornos caen en `domain.DefaultTenantID`, que corresponde a la institución
única creada por el instalador. Sin ese respaldo, los procesos internos que no
pasan por HTTP dejarían de ver sus propias filas.

## Tablas que no tienen `tenant_id`

`course_modules` y `module_prerequisites` cuelgan de `subjects`, y la materia sí
tiene `tenant_id`. Por eso el filtro se escribe como un `JOIN`:

```sql
SELECT cm.id, cm.title
FROM course_modules cm
JOIN subjects s ON s.id = cm.subject_id
WHERE cm.id = $1 AND s.tenant_id = $2
```

Lo mismo aplica a `AssignExerciseModule`: un ejercicio sólo puede apuntar a un
módulo de su institución, o el módulo desaparece del mapa curricular y el dato
queda cruzado.

Las rúbricas tampoco tienen `tenant_id`: cuelgan de un ejercicio, que sí lo
tiene. `GetByExerciseID` y `CreateOrUpdate` filtran con un `JOIN` a `exercises`,
para que un id de ejercicio ajeno no devuelva ni permita escribir una rúbrica.

## Rutas

Además del filtro por institución, la escritura exige rol. Las rutas de
`/api/v1/teacher/modules` y compañía usan `tm(am(staffOnly(...)))`: docentes y
administradores. Que la ruta empiece con `/teacher` no autorizaba nada por sí
sola; el estudiante autenticado pasaba igual.

`PUT /api/v1/exercises/{id}/rubric` también exige `staffOnly`: la rúbrica es
criterio de calificación. Cuando el ejercicio no existe en la institución del
contexto, el servicio responde `404` y no `500`; el `INSERT` de la rúbrica nunca
llega a ejecutarse.

## Alcance por docente en escrituras

El filtro por institución no distingue entre dos docentes de la misma facultad.
Cuando una institución tiene más de uno, ese es el límite que importa: cada
docente escribe solo en las materias que dicta.

El alcance se aplica con `domain.TeacherWriteScope` (rol de docente → su id;
administrador o llamador interno → sin restricción; cualquier otro rol → no
escribe) y el guardia `assertActorWritesSubject`, que consulta
`TeacherOwnsSubject` sobre `subjects`. Una materia ajena responde `404` con el
mismo error que una inexistente: no confirma que exista.

Escrituras acotadas:

- `POST /api/v1/exercises` y `PUT /api/v1/exercises/{id}`: crear un ejercicio en
  la materia de un colega responde `404`, igual que mover uno hacia ella. La
  edición revisa la materia actual y la de destino, así que cambiar el
  `subject_id` no es una puerta lateral. Un ejercicio sin materia sigue
  permitido: no tiene pertenencia que violar, y el alta inicial todavía no elige
  curso.
- `PUT /api/v1/exercises/{id}/rubric`: la rúbrica califica a los alumnos de su
  materia, así que un docente no la edita en la de un colega.
- `POST /api/v1/teacher/courses/{id}/modules`, `PUT`/`DELETE` de
  `/api/v1/teacher/modules/{id}`, sus prerrequisitos y
  `PUT /api/v1/teacher/exercises/{id}/module`: todo el mapa curricular de la
  materia.

Las lecturas ya estaban acotadas por docente en la cola de corrección, las
analíticas, las sesiones de aula y los cursos. La calificación con rúbrica
(`EvaluateSubmission`) y la revisión de entregas ya exigían que la materia fuera
del docente; lo que faltaba era la escritura de criterios y del mapa curricular.

## Cómo se comprueba

- `backend/tests/integration/course_module_tenant_test.go`: con dos
  instituciones sembradas, la institución A no puede leer, listar, actualizar,
  borrar ni crear módulos dentro de la materia de B, y la fila de B queda
  intacta.
- `backend/internal/delivery/http/course_module_authorization_test.go`: las seis
  rutas de escritura responden 403 a un alumno antes de llegar al handler.
- `backend/tests/integration/rubric_tenant_test.go`: la rúbrica de un ejercicio
  de la institución B no se lee ni se sobrescribe desde la institución A.
- `backend/tests/integration/cross_tenant_staff_writes_test.go`: por HTTP, un
  docente de A recibe `404` al editar o borrar un módulo de B y al guardar la
  rúbrica de un ejercicio de B; las filas de B quedan intactas. El docente de A
  y el de B sí escriben lo propio, así que el rechazo viene del alcance y no de
  una ruta rota. El mismo archivo cubre el caso dentro de una institución
  (`NoCrossTeacherSameTenant`): el docente 1 no edita ni borra módulos del
docente 2 ni escribe su rúbrica, cada uno sí escribe lo suyo y el administrador
  escribe ambas materias.
- `backend/internal/core/services/teacher_scope_test.go`: el alcance en los
  servicios, sin base de datos (docente, administrador, docente sin lector de
  pertenencia y estudiante).
- `backend/internal/core/domain/actor_scope_test.go`: la tabla de roles de
  `TeacherWriteScope`.
- `backend/tests/integration/exercise_teacher_scope_test.go`: por HTTP, un
  docente crea en su materia, recibe `404` al crear en la del colega (y no queda
  fila), edita lo propio, recibe `404` al mover su ejercicio hacia la materia
  ajena, y un administrador sí crea en cualquiera.

## Revisión pendiente

Los informes de auditoría de plantillas (`UpdateAuditResults`,
`UpdateLifecycleStatus`) se escriben por id de plantilla; hoy los invocan jobs
sobre plantillas que el administrador ya seleccionó. Si en el futuro se expone
una ruta que reciba ese id desde el cliente, hay que agregar el filtro por
`tenant_id` antes de llamarlos.

- `POST /api/v1/exercises/{id}/publish`, `/unpublish`,
  `/test-cases/bulk`, `/dry-run` y
  `/api/v1/teacher/exercises/{exerciseId}/generate-from-script`: cambian el
  estado o el contenido del ejercicio, así que aplican el mismo alcance. La
  carga masiva de casos expone `404` en vez de `500`, y el script de generación
  también exige rol docente o administrador (antes solo comprobaba la
  institución).

No necesitan alcance por materia las rutas que no reciben un ejercicio:
`/exercises/calculate-outputs`, `/exercises/validate-input` y
`/exercises/generate-cases` (propuesta de casos con IA) trabajan con el cuerpo
enviado y no escriben borrador ni publicación.

- `POST /api/v1/teacher/courses/{courseId}/exercises/import`: la importación
  masiva escribe hasta 50 ejercicios en la materia del path, así que también
  exige que el docente la dicte, incluso con `dry_run=true` (si no, la ruta
  serviría para descubrir materias). Además exige rol docente o administrador:
  antes cualquier usuario autenticado llegaba al servicio.

Con esto, ninguna ruta de escritura de ejercicios se resuelve solo por
institución.
