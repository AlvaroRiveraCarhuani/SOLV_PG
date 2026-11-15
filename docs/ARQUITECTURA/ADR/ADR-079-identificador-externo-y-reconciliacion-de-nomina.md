# ADR-079: Identificador Externo y Reconciliación de la Nómina

## Estado
Aprobado

## Contexto

La importación de matrícula por planilla (`POST /api/v1/admin/subjects/{id}/roster`) sólo sabía **agregar**. Un administrador subía la nómina de la semana 1 y volvía a subir la de la semana 8 cuando el sistema de gestión académica de la institución reflejaba tres bajas y dos altas. El resultado era acumulativo: los tres alumnos que abandonaron seguían matriculados, con entorno de práctica asignado y ocupando cupo de concurrencia, y nadie se enteraba porque la respuesta sólo informaba cuántos se habían matriculado.

El segundo problema era la identidad. La planilla se resolvía por correo electrónico, que es único por cuenta en toda la instalación (`users.email UNIQUE`). Cuando la institución cambia la dirección de un estudiante (`j.perez@u.edu.bo` → `juan.perez@u.edu.bo`), el sistema no puede saber que es la misma persona: crea una cuenta nueva, y la anterior queda con entregas y veredictos que nadie vuelve a mirar. El identificador estable de una persona es el que le asigna el sistema académico de origen, no su correo.

ADR-033 declaró `users.student_code`, pero esa columna nunca se creó en una migración: quedó como intención en el diseño.

## Decisión

1. **Un identificador externo por entidad espejada.** `users`, `subjects` y `enrollments` ganan `external_id` (la clave en el sistema de origen), `source_system` (quién manda el dato) y `last_synced_at` (cuándo se leyó). Las tres columnas son opcionales: una instalación administrada a mano sigue funcionando sin llenarlas.

2. **`external_id` reemplaza a `student_code`.** Dos nombres para el mismo concepto es la clase de divergencia que este ADR evita; `student_code` nunca existió en la base y no se crea ahora.

3. **La resolución de identidad es externo primero, correo después.** Si la fila trae `external_id` y existe una cuenta con ese identificador, se usa. Si no, se busca por correo; si la cuenta existe y no tiene identificador, se le completa (adopción) en vez de crear una segunda cuenta. Si el correo pertenece a una cuenta que no es de estudiante, la fila se descarta y se cuenta: cambiarle el rol a un docente porque aparece en una planilla sería peor que saltarse la fila.

4. **La reconciliación se acota por origen.** Una sincronización con `reconcile: true` quita de la materia las matrículas cuyo `source_system` coincide con el origen de esa misma sincronización y cuyo estudiante ya no figura en la lista. Las matrículas manuales (`source_system = 'manual'`) sobreviven. Una nómina de origen `manual` no puede reconciliar: sería una forma de vaciar un curso por accidente.

5. **Una nómina se aplica en una transacción.** Resolver cuentas, matricular y quitar ocurre en la misma transacción, con la materia validada contra el tenant primero. Aplicada a medias, la nómina deja el curso en un estado que nadie pidió, y reconciliar contra una lista parcial borra a quien sí correspondía.

6. **El contrato se mantiene.** `{emails: [...]}` sigue funcionando; `entries` con `external_id` es la forma rica. La capacidad de sincronización se resuelve por type-assertion sobre `RosterSyncRepository`, así que un repositorio que no la implemente conserva el camino anterior sin bajas.

## Consecuencias

- **A favor**: la nómina se puede reaplicar todas las semanas con el mismo archivo y converge al estado del sistema de origen. Un cambio de correo no crea una cuenta nueva. La respuesta informa `removed_students`, `matched_by_external_id` y `skipped_not_found`, así que una baja inesperada se ve.
- **En contra**: la matrícula manual de alguien que además figura en la planilla pasa a ser propiedad de la planilla (el `ON CONFLICT` actualiza el origen). Es intencional: si la persona está en el origen, el origen manda. Pero es un comportamiento que hay que conocer antes de marcarlo.
- **En contra**: reconciliar con una planilla incompleta quita alumnos. Por eso la casilla está apagada por defecto y el panel informa cuántos se dieron de baja en lugar de hacerlo en silencio.
- El parser de planillas sigue extrayendo sólo correos: un archivo subido no aporta identificador externo. Para aprovecharlo hay que enviar `entries` por JSON o extender el parser, que es un cambio aparte.
- `subjects.external_id` queda declarado pero ninguna ruta lo llena todavía; es la base para importar la oferta académica sin duplicar materias por código.

## Referencias

- `backend/migrations/00058_roster_source_sync.sql`
- `backend/internal/core/services/admin_governance_students.go` (`RosterEnrollFromSource`)
- `backend/internal/infrastructure/storage/postgres/admin_roster_sync_repository.go`
- ADR-033 (gestión de estudiantes), ADR-024 (esquema académico multitenant)
