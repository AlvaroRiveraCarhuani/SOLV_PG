# ADR 071: Alcance del Script de Inicialización de Base de Datos y Tope de Tamaño

> **Estado:** Aceptado
> **Fecha:** 2026-11-12
> **Ámbito:** Docencia, Core, Servicios satélite
> **Relacionado:** ADR-046 (servicios satélite y `db_init_script` multi-motor), ADR-050 (servicios satélite por rol), ADR-051 (PostgreSQL de aula como satélite dedicado), ADR-059 (entornos acotados a una práctica), ADR-063 (solicitud de entorno del docente), ADR-065 (vocabulario de entorno de práctica), ADR-072 (ejecución por entrada estándar)

---

## Contexto

El editor de ejercicios permite declarar `config.ide.db_init_script`: sentencias SQL que se aplican
en la base del alumno antes de comenzar la práctica. El campo nació como un `<textarea>` de ocho
líneas, sin carga de archivos ni medición del contenido.

Eso deja tres problemas concretos:

1. Un docente con un script de 2.000 líneas tiene que abrir el archivo en su equipo, copiar todo el
   texto y pegarlo a mano. Pegar un volumen así en un `textarea` sin virtualización degrada la
   pestaña.
2. El título del campo no declara su alcance. Un docente puede interpretar que configura la base de
   toda la materia y no solo la de la práctica en curso.
3. No hay ningún número visible que indique a partir de qué tamaño el script deja de aplicarse con
   seguridad, y el motivo de ese número no está escrito en ningún lado.

El punto 2 se apoya en una confusión razonable sobre el modelo de datos: la base del alumno es
`solv_std_<alumno>_<materia>` (`backend/internal/core/services/practice_environment_satellite.go`),
una sola base por materia y alumno. Los scripts de las prácticas se aplican sobre esa misma base y se
sellan por `exercise_id` en `_solv_exercise_migrations`. La separación real es de **volumen de
trabajo** por práctica (ADR-059), no de base de datos.

## Decisión

### 1. El script es exclusivo de la práctica, no de la materia

`db_init_script` pertenece al ejercicio. No existe un script de materia, y el docente no necesita
aprobación del administrador para declararlo: la plantilla del curso ya está homologada y el script
solo agrega datos de la consigna.

La interfaz lo declara de forma explícita en tres lugares del componente
`ide-db-init-script`:

- la insignia de alcance junto al título;
- la descripción de la sección, que indica que se ejecuta una sola vez por alumno al iniciar la
  actividad;
- la ayuda bajo el editor, que acota el contenido al ejercicio.

El componente además declara que la base es una sola durante toda la materia, para que el docente no
interprete que cada práctica abre una base distinta ni que este campo configura el curso completo.

El alcance no es una decisión de diseño nueva: es cómo funciona el motor desde ADR-046. La decisión
es hacerlo visible en la interfaz en lugar de dejarlo implícito en el backend.

### 2. Los datos de una práctica persisten en las prácticas siguientes de la misma materia

La base no se recrea entre prácticas. Un `CREATE TABLE` de la práctica 1 sigue disponible en la
práctica 2 del mismo alumno y la misma materia. Lo que no se repite es la aplicación del script: el
sello en `_solv_exercise_migrations` lo aplica una sola vez por `exercise_id`.

Por lo tanto, un docente que quiere una sola base durante todo el semestre **ya lo tiene** sin
configuración adicional: declara el esquema en la primera práctica y las posteriores lo encuentran
creado. No hace falta un campo de materia para eso.

### 3. El tope visible es 2 MiB y es una advertencia de costo, no una cuota

Cuando se escribió este ADR el script viajaba como argumento de `psql`, y Linux limita el tamaño
total de `argv` más `envp` de un proceso a 2 MiB (`ARG_MAX`). Un script cercano a ese total podía
fallar con `E2BIG` antes de ejecutar una sola sentencia. ADR-072 eliminó ese techo para SQL: el
script se entrega por entrada estándar y no ocupa lugar en la línea de comandos.

El umbral de 2 MiB se mantiene en la interfaz (`MAX_RECOMMENDED_SQL_BYTES`) como recomendación de
costo y no como límite del sistema. Superarlo muestra un aviso que explica la demora al aprovisionar
y no bloquea el guardado: un `ALTER` bloqueante también puede ser lento con un script pequeño, y el
tamaño no es el único factor de costo.

Para dimensionar: una sentencia `INSERT` promedio pesa entre 60 y 90 bytes. Un script de 2.000 líneas
ronda los 150 a 200 KB, es decir una décima parte del umbral. Un esquema de referencia como Northwind
o Sakila entra sin acercarse al aviso.

### 4. La base precargada por materia no se implementa en este corte

Una base de cientos de megabytes precargada una sola vez y clonada por alumno resolvería un caso
distinto al de este ADR: no la persistencia, que ya existe, sino el costo de insertar el mismo
volumen una vez por alumno al inicio de la clase.

No se implementa ahora por tres razones:

1. **No hay caso bloqueado.** Ninguna práctica actual queda sin poder dictarse por su ausencia. El
   tamaño que un docente produce al pegar 2.000 líneas está muy por debajo del umbral de 2 MiB.
2. **Duplica un concepto existente.** El sistema ya tiene una entidad por materia (`subjects`) con
   plantilla homologada y un flujo de solicitud de entorno (ADR-063). Un artefacto nuevo de base
   precargada tiene que colgar de esa entidad y pasar por ese flujo, no abrir un segundo camino de
   aprobación.
3. **No es portable entre motores.** PostgreSQL resuelve el clonado con
   `CREATE DATABASE ... TEMPLATE`, que exige que la plantilla no tenga conexiones activas y que el rol
   tenga `CREATEDB`; el satélite lo cumple porque el aprovisionamiento usa el rol raíz. MySQL y
   MariaDB no tienen un equivalente soportado, así que el mecanismo sería exclusivo de PostgreSQL y el
   resto de los motores seguiría aplicando SQL lógico. Eso obliga a que el modelo declare el mecanismo
   por motor en lugar de asumir uno solo.

El día que se implemente, el orden de trabajo es: primero mover el script de `exercises.config`
(JSONB) a un artefacto con almacenamiento propio, porque el texto sigue viviendo en la fila del
ejercicio; después agregar la base precargada por materia con cuota, validación en una base temporal
y clonado por plantilla. La ejecución por entrada estándar ya está resuelta en ADR-072, así que el
techo de `ARG_MAX` no es un motivo para diferirlo.

## Consecuencias

- El componente acepta archivos `.sql`, `.txt` y `.js`, muestra nombre, tamaño, líneas y sentencias
  estimadas, y permite descargar o limpiar el contenido.
- El texto de la sección declara que el script pertenece a la práctica, así que un docente no puede
  confundirlo con la configuración de la materia.
- El aviso de tamaño no bloquea el guardado. La validación de tamaño en el servidor queda pendiente
  y se resuelve junto con el pasaje a entrada estándar.
- La ejecución SQL entrega el script por entrada estándar (ADR-072), así que su tamaño ya no está
  acotado por el límite de argumentos del kernel. Un script que falle al aplicarse aborta el arranque,
  sin dejar un entorno a medio inicializar.
- La sección declara en pantalla que la base es una sola por materia y que los datos de una práctica
  quedan disponibles en las siguientes, de modo que el docente no confunda el script de la práctica
  con la configuración del curso.

## Verificación

- Tests del componente en
  `frontend/src/app/features/teacher/exercise-editor/steps/step-ide-environment/components/ide-db-init-script/ide-db-init-script.component.spec.ts`,
  que cubren carga por archivo, rechazo de formato, métricas, aviso de tamaño, descarga y limpieza.
- `npm run lint:styles` en `frontend/`, sin errores de tokens, tipografía, fechas o terminología.
- El registro histórico del comportamiento del motor está en ADR-046, ADR-050 y ADR-051, que no
  cambian con este ADR.
