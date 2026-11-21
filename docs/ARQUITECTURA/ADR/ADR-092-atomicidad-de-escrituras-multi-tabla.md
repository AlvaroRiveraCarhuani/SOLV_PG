# ADR-092: Atomicidad de las Escrituras que Tocan Varias Tablas

- **Estado:** Aceptado
- **Fecha:** 2026-11-21
- **Ámbito:** Core (plantillas, periodos académicos, aulas, materias)
- **Decisores:** Arquitectura SOLV

## Contexto

Varias operaciones de escritura de los repositorios ejecutaban dos sentencias sueltas. Cuando la segunda fallaba, la primera ya estaba confirmada y el sistema quedaba en un estado que ninguna pantalla puede explicar:

- El alta de una plantilla insertaba la plantilla y después sus ejecutores de lenguaje, compensando con un `DELETE` que podía fallar en silencio.
- El duplicado insertaba la copia y después copiaba los ejecutores, sin compensación.
- La promoción a modelo insertaba el modelo y después vinculaba la plantilla, descartando el error del vínculo.
- Activar un periodo desactivaba los demás y después guardaba el elegido; un fallo dejaba la institución sin periodo activo.
- El archivado de periodos vencidos sellaba los periodos y después sus materias, ignorando el error del sellado.
- Cerrar un aula marcaba la sesión cerrada y después la salida de los participantes; el reintento ya no sellaba salidas porque la sesión figuraba cerrada.
- Vincular un entorno a un aula guardaba el vínculo y después la presencia del alumno.
- El alta de una materia registraba la entrada del catálogo y después la materia.

## Decisión

Toda operación que escribe en más de una tabla abre una transacción en el repositorio, ejecuta todas sus sentencias con ese ejecutor y confirma al final. El error de cualquier sentencia se devuelve; ninguno se descarta. La transacción pertenece a la operación de negocio, no a una función auxiliar, para que no queden dos transacciones encadenadas.

Cuando la segunda escritura puede fallar por una restricción real de la base, la prueba de integración instala un disparador temporal que la hace fallar y comprueba que la primera no quedó guardada. El disparador se elimina al terminar la prueba y la base es descartable.

Queda fuera de esta decisión la libreta derivada de la revisión docente (`UpdateSubmissionReview` y `syncReviewAfterOverride`). El servicio decidió que un fallo al anotar la nota no invalide la corrección de la entrega; convertirlo en atómico cambiaría ese contrato y exige una decisión de producto sobre qué debe ver el docente cuando la libreta falla.

## Consecuencias

- Un fallo a mitad de estas operaciones deja la base como estaba, sin plantillas sin ejecutores, periodos sin materias selladas, aulas cerradas con alumnos dentro, entornos sin presencia ni catálogo sin materia.
- Los reintentos encuentran el estado anterior y pueden completar la operación.
- Las transacciones abarcan solo las sentencias de escritura; las lecturas previas y la relectura de la respuesta quedan fuera para no alargar el bloqueo.
- Las pruebas de integración usan disparadores temporales para forzar el fallo; son DDL de la base descartable y se eliminan en el mismo test.
- La revisión y la nota del docente siguen siendo dos escrituras tolerantes a fallos, documentadas como pendiente de decisión.
