# ADR-083: Perfil y Estado del Entorno según el Propósito del Ejercicio

## Estado

Aceptado

## Contexto

Hasta este cambio, el entorno de un alumno no sabía si estaba en una práctica o en
un examen. La única diferencia era el cierre del aula, que dejaba el volumen en solo
lectura. Todo lo demás era igual: el mismo volumen, los mismos permisos, la misma
política de apagado y el mismo registro de tecleo.

Eso dejaba cuatro huecos visibles en un examen:

1. Un examen podía arrancar sobre archivos que el alumno había dejado en una
   apertura previa del mismo ejercicio.
2. El recuperatorio se resolvía reabriendo el mismo intento, con los archivos del
   examen anterior dentro.
3. El entorno se apagaba por inactividad mientras el alumno pensaba.
4. Se registraba el tecleo también en práctica, donde no hay nada que vigilar.

La base ya tiene un índice único (`idx_workspaces_practice_scope`) que admite una
sola fila por persona, materia y ejercicio. Cualquier diseño de intentos tiene que
convivir con esa restricción.

## Decisión

1. **El perfil se deriva del ejercicio.** `exercise.purpose = "exam"` produce el
   perfil `exam`; cualquier otro valor, o la ausencia de ejercicio, produce
   `practice`. Ninguna petición declara el perfil: no hay campo de cliente que lo
   fije. El perfil queda guardado en `environments.environment_profile` para auditar
   con qué reglas se ejecutó el entorno.

2. **Cada intento tiene su volumen.** La práctica usa
   `solv_workspace_<alumno>_<materia>_<práctica>`; el examen agrega `_exam_<n>`. El
   intento inicial de un examen es 1 y el de práctica es 0, así que un examen nunca
   monta el volumen de la práctica ni el de un intento anterior.

3. **El recuperatorio es un intento nuevo, no una reapertura.** Si el intento está
   cerrado y el docente abrió una convocatoria del mismo ejercicio **después** del
   cierre, el entorno avanza a `exam_attempt + 1`, estrena volumen y lo siembra con
   la versión publicada. El volumen del intento anterior no se toca. Sin
   convocatoria nueva, o si la convocatoria todavía no empezó, el entorno sigue
   cerrado: recargar la página no reabre nada.

   La fila de la práctica se reutiliza porque la base no admite una segunda. El
   número de intento es la trazabilidad y el nombre del volumen preserva el
   trabajo de cada intento.

4. **Un examen no se apaga a mano.** `PausePracticeEnvironment` y
   `HibernatePracticeEnvironment` rechazan el perfil de examen con
   `ErrExamPauseForbidden` (409 en el transporte). El cierre del examen es la
   operación del aula, no una pausa. La práctica se sigue pausando.

5. **El propósito no cambia con entornos abiertos.** Un entorno abierto conserva
   las reglas con las que nació, así que cambiar el propósito dejaría al alumno en
   un examen sin reglas de examen, o al revés. Con entornos abiertos la edición
   devuelve 409 con el conteo. Con entornos cerrados pide `confirm_purpose_change`
   y `reason`, y el cambio queda en `audit_logs` como `EXERCISE_PURPOSE_CHANGED`.

6. **La entrega guarda una copia, y la corrección lee esa copia.** El snapshot del
   volumen se escribe en `submissions.code` al entregar. Lo que el alumno escriba
   después no cambia lo corregido. Cuando el proyecto excede el tope de 4 MB la
   entrega **falla** con `ErrPracticeEnvironmentSnapshotTooLarge` y el detalle del
   archivo; antes se recortaba en silencio y se guardaba medio proyecto.

7. **El estado que ve el alumno se calcula, no se guarda.**
   `PracticeEnvironment.State()` devuelve `preparing`, `ready`, `closed` o
   `failed`, y `MarshalJSON` lo agrega a cada respuesta. Un campo guardado se
   quedaría viejo en cuanto el contenedor cambie sin pasar por el servicio.

## Consecuencias

- Un examen nuevo no hereda nada de la práctica ni de un intento anterior.
- El recuperatorio entrega con el proyecto base, y el intento anterior queda como
  evidencia en su propio volumen.
- El docente conserva la excepción individual (`/students/{studentID}/unfreeze`)
  para el caso "se le cortó el entorno", que trabaja sobre el mismo intento.
- El consumo de disco crece con cada intento de examen: el volumen anterior no se
  borra solo. Está anotado en `docs/OPERATIONS/capacidad.md`.
- Un examen no se duerme por inactividad, y como no existe apagado por presión de
  memoria, un examen con toda la RAM se libera solo cuando el aula se cierra o
  cuando alguien cierra el entorno.

## Alternativas descartadas

- **Reutilizar el volumen del ejercicio.** Es lo que había y es el problema: el
  examen arrancaría con los archivos de la práctica.
- **Vaciar el volumen existente.** Es destructivo y no deja evidencia del intento
  anterior.
- **Una fila por intento.** Choca con `idx_workspaces_practice_scope`. Cambiar el
  índice obligaría a revisar todas las consultas que asumen un entorno por
  práctica.
- **Un campo `state` guardado.** Se desincroniza del estado real del contenedor.
- **Interruptor `ENV_PROFILE_SPLIT` para volver atrás.** No restaura los volúmenes
  viejos: al aplicarse la migración, los entornos nuevos ya nacen con perfil y
  volumen por intento. Volver atrás exigiría mover archivos, no cambiar una bandera.

## Migración y rollback

La migración `00060_environment_profile.sql` es aditiva: agrega
`environment_profile` con default `practice` y `exam_attempt` con default 0. Las
filas anteriores quedan como práctica, que es lo que eran. El rollback borra las dos
columnas y el índice, y los volúmenes nombrados siguen en el disco.

## Verificación

- Dominio: `TestPracticeEnvironmentState`, `TestNextExamAttempt`,
  `TestPracticeEnvironmentAttemptVolumeName`.
- Servicios: creación con perfil e intento, recuperatorio con volumen `_exam_2`,
  recarga y convocatoria futura sin reapertura, pausa de examen rechazada, regla
  del cambio de propósito, copia congelada en la entrega.
- Integración con PostgreSQL: congelado por práctica, convocatoria más nueva solo
  de exámenes y reinicio del intento en la misma fila.
- Docker: lectura estricta del volumen que falla en vez de recortar.
