# ADR 058: Avance Voluntario del Docente en el Aula

> **Estado:** Implementado
> **Fecha:** 2026-10-30
> **Actualizado:** 2026-10-31 — ventana de disponibilidad, modo tarea acotado, retención de snapshots y respaldos, y límite de aplicaciones simultáneas
> **Ámbito:** Docencia, Core, Workspaces, Aula
> **Reemplaza:** ADR-007 (sincronización forzada por instantáneas de volumen)
> **Relacionado:** ADR-056 (sesión de aula y presencia acotada), ADR-005 (examen y congelación de clase), ADR-041 (examen con semilla por estudiante), ADR-027 (audit logs), ADR-011 (ciclo de vida y hibernación)

---

## Contexto

En una práctica universitaria el docente suele mostrar cómo resolver un paso y quiere que el alumno lo pruebe en su propio entorno. ADR-007 propuso que el backend pausara el contenedor del alumno y sobrescribiera su volumen. Ese diseño quita al alumno la decisión sobre su trabajo y puede borrar código que ya escribió, sin aviso.

Los documentos de interfaz (`docs/UI/DOCENTE/DASHBOARD.md` y `docs/UI/ESTUDIANTE/LABORATORIO_ACTIVO.md`) describen un flujo distinto: el docente emite, el alumno recibe un aviso y decide si aplica el cambio. Ese flujo es el que se adopta aquí.

En un examen, cualquier copia de código desde el docente sería una vía para entregar soluciones durante la prueba. El proyecto ya trata el examen como modo aparte (`purpose = exam`, ADR-041), y la sincronización debe respetar ese límite.

El código actual no tiene nada de esto: no hay endpoint de emisión, ni botón en el aula, ni aviso en el laboratorio del alumno. Tampoco existe la acción de congelar clase que ADR-005 menciona como pendiente de interfaz.

---

## Decisión

### 1. El avance es voluntario

El alumno decide si aplica un avance. El sistema nunca sobrescribe su código sin confirmación. Un alumno puede descartar el aviso y seguir con su trabajo.

### 2. El avance pertenece a una sesión de aula abierta

Solo el docente de la materia puede emitir un avance, y solo mientras la sesión de aula (ADR-056) esté abierta. Así el avance tiene dueño, fin y destinatarios conocidos: los presentes de la sesión.

Fuera de una sesión no hay emisión. Esto evita que un avance quede pendiente días después de la clase.

### 3. Destinatarios: toda la clase o alumnos seleccionados

El docente elige en cada emisión:

| Alcance | Destinatarios |
| --- | --- |
| `class` | Todos los alumnos que tengan participación en la sesión (`session` con latido) |
| `selected` | Solo los alumnos marcados por el docente, de entre los participantes de la sesión |

Un alumno que no participa de la sesión no recibe avisos, aunque esté matriculado.

### 4. Congelación, ventana de disponibilidad y examen

**Examen (`purpose = exam`).** No se emiten ni aplican avances. Durante el examen el alumno programa: el bloqueo de edición no se activa al empezar, sino al cerrar, cuando su volumen pasa a solo lectura (ADR-005 §5). No hay ventana de disponibilidad en examen.

**Congelar clase.** Es una acción manual del docente sobre la sesión abierta, sin temporizador. Mientras la clase esté congelada no se emiten avances nuevos. La congelación no bloquea la edición del alumno: puede seguir programando.

**Cierre de la sesión.** Al cerrar el aula (a mano o al vencer `ends_at`, ADR-056) se bloquean las emisiones nuevas. Los avances ya emitidos quedan aplicables durante la ventana de disponibilidad de la sesión, que arranca en 30 minutos después del cierre. El valor es de 30 minutos para todas las materias; el valor por materia queda para una iteración posterior.

**Ventana de disponibilidad.** Es una sola ventana por sesión y cubre todos sus avances. El docente la ve como "Disponible hasta ...". Tiene tres acciones:

| Acción | Efecto |
| --- | --- |
| Sumar minutos | Agrega minutos al final vigente. Si la ventana ya venció, cuenta desde ahora. El campo es libre: 1 a 480 minutos por suma. |
| Fijar fecha límite | Para una tarea, el docente elige día y hora. Reemplaza el final vigente. El máximo es de 7 días desde el cierre de la clase. |
| Cerrar ahora | Fija el final en el momento y los avances de la sesión dejan de aplicarse. |

No hay tope de cantidad de extensiones: el control es el máximo de tiempo y el registro de cada cambio. Una ventana no puede superar el máximo institucional.

**Modo tarea.** Es la misma ventana con una fecha límite larga, no un mecanismo aparte. Sirve para que el alumno aplique la versión del docente fuera del horario de clase. No mantiene abierta la sesión, no enciende el entorno del docente y no cambia el horario ni el acceso al laboratorio del alumno. El snapshot ya está guardado; el alumno lo aplica en su propio entorno cuando lo abra, dentro del plazo.

**Vencimiento de la ventana.** Al vencer, el destinatario pendiente pasa a `expired`. El aviso deja de mostrarse y no puede aplicarse. `expired` es un estado del sistema, no una decisión del alumno: si el docente reabre la ventana (sumando minutos desde ahora o fijando una fecha futura), esos destinatarios vuelven a `pending`. `applied`, `dismissed` y `failed` no se revierten. El vencimiento se refleja sin depender de que el alumno abra el laboratorio: la consulta del alumno y la lectura del docente cierran los pendientes vencidos.

**Estado de implementación.** La ventana automática, la suma de minutos, la fecha límite y el cierre manual están implementados. El cierre a mano y el cierre por `ends_at` fijan `grace_until`; `PATCH /grace` acepta `add_minutes`, `until` o `close`, y al reabrir una ventana vencida los destinatarios `expired` vuelven a `pending`.

### 5. Qué se copia

Se copian los archivos de código del ejercicio del docente. Se excluyen las rutas siguientes:

| Ruta excluida | Motivo |
| --- | --- |
| `.vscode/` | Ajustes del editor del docente; no deben cambiar el entorno del alumno |
| `.solv_*` (incluye `.solv_initialized`, `.solv_setup.log`, `.solv_backups/`) | Archivos internos de la plataforma |
| `.git/` | Historial del docente; copiarlo mezcla dos repositorios |
| `node_modules/` | Dependencias instaladas; se reconstruyen en el entorno del alumno |
| `.cache/` | Estado y telemetría del servidor de editor; no es código de la práctica |
| `.openvscode-server/` | Binarios y datos del servidor de editor que corre dentro del volumen |

La lista de exclusión vive en configuración del backend, no en código fijo. Las dos últimas rutas importan porque el IDE escribe su runtime dentro del propio volumen del ejercicio: sin excluirlas, el avance reemplazaría el identificador de dispositivo y la configuración del editor del alumno con los del docente.

**Origen del código.** El snapshot sale del entorno de clase del docente de la práctica de la sesión (`solv_workspace_<docente>_<materia>_<práctica>`, ADR-059), que es donde programa en vivo. No se copia desde una carpeta aparte ni desde la base: lo que se emite es lo que el docente tiene abierto. Sin ese entorno la emisión responde 409, porque no hay código que mostrar. El entorno se abre desde la ficha de la práctica (**Abrir IDE**), que reutiliza el contenedor si ya existe; la lectura por materia (`GET /api/v1/workspaces?subject_id=`) sigue disponible para consultar el entorno vigente sin levantar contenedores. Una sesión sin práctica conserva el respaldo del entorno por materia.

### 6. Snapshot al emitir

Al emitir, el backend empaqueta el código del ejercicio del docente (tar.gz) y lo guarda como objeto en **SeaweedFS** mediante su API S3 compatible. La clave del objeto lleva el identificador del avance, y la base guarda también el hash SHA-256 y el tamaño. El alumno recibe siempre la versión emitida, aunque el docente siga editando después.

El almacén de snapshots va detrás de la interfaz `AdvanceSnapshotStore` en el dominio. La implementación inicial es el adaptador S3 de SeaweedFS (`infrastructure/storage/seaweedfs`); el servicio, su configuración y su aprovisionamiento en Compose se describen en [STORAGE_STRATEGY.md](../STORAGE_STRATEGY.md#almacenamiento-de-objetos-seaweedfs). El snapshot no se guarda en la base (un `bytea` inflaría el `pg_dump` y cada respaldo remoto) ni en el disco del backend (la fila y el archivo podían desincronizarse). Al leer, el backend verifica el hash antes de copiar al alumno.

Se fijan topes configurables de tamaño (bytes) y de número de archivos por avance. Un snapshot que supera el tope se rechaza al emitir con 422.

**Retención.** El snapshot no se borra al cerrar el aula: la ventana puede reabrirse hasta 7 días después del cierre. Se conserva **8 días desde `closes_at`** (7 del máximo de la ventana más 1 de margen) y después se elimina el objeto. Nunca se borra con el aula abierta, sin cierre o con `grace_until` futuro. La fila de `class_advances`, los destinatarios, el historial y la auditoría se conservan: el código es lo único que se va. El barrido corre al arrancar el backend y después cada 24 horas (`AdvanceRetentionWorker`). Borra el objeto, marca `snapshot_purged_at` en la fila y, si el objeto ya no existe, la marca igual. Un fallo se reintenta en el ciclo siguiente. También borra objetos huérfanos de más de 24 horas que ninguna fila referencie. La migración `00040_class_advance_snapshot_retention.sql` agrega la marca y el índice parcial de pendientes. La política completa, con la poda de respaldos, vive en [retencion-snapshots-y-respaldos.md](../../../proyecto-grado/avances/retencion-snapshots-y-respaldos.md).

### 7. Aplicar con confirmación y respaldo

Cuando el alumno aplica un avance:

1. El frontend muestra cuántos archivos cambiarán y cuáles se sobrescriben o se agregan. El alumno confirma o cancela.
2. Antes de copiar, el backend guarda el estado actual de los archivos que se van a sobrescribir en `.solv_backups/<avance_id>/`.
3. El backend copia el snapshot al volumen del alumno con su contenedor en ejecución. No se pausa ni se reinicia el contenedor. El volumen es el de la práctica que el alumno abrió (`solv_workspace_<alumno>_<materia>_<práctica>`, ADR-059); si el alumno todavía no tiene entorno de esa práctica, se usa el de la materia.
4. El resultado queda registrado (aplicado o fallido). Si falla la copia, el respaldo permite recuperar el estado previo.

El contenedor debe estar en ejecución para aplicar. El alumno aplica desde su laboratorio, así que ya lo tiene activo; si lo intenta desde el solver con el entorno de la práctica detenido, el aviso sigue pendiente y el modal lo explica: la misma pantalla ofrece `Iniciar IDE`, y al abrirlo puede aplicar sin perder el aviso.

**Confirmación previa.** El paso 1 necesita nombres de archivo, y `/pending` solo conoce la cantidad. `GET /api/v1/class-advances/{id}/files` devuelve `advance_id`, `exercise_id`, `file_count`, `added[]` y `overwritten[]`: lee el snapshot y lo compara con el volumen vigente del alumno, sin escribir nada. Es la lectura que alimenta la confirmación. Un aviso ya decidido responde 409; en examen, 409; con la ventana vencida, 409 y el destinatario pasa a `expired`. Sin almacén de snapshots responde 503 y el aviso sigue visible.

**Resultado de la aplicación.** `POST /api/v1/class-advances/{id}/apply` responde con `advance_id`, `files_written`, `overwritten[]`, `added[]` y `backup_path`. Las dos listas salen de comparar el snapshot con el volumen vigente: el alumno ve qué se reemplazó y qué se agregó. `backup_path` es `.solv_backups/<avance_id>` y solo aparece cuando hubo archivos sobrescritos.

**Fallos.** Si no se puede leer el snapshot o el estado previo del volumen, no se escribe nada y el destinatario sigue `pending`: el alumno puede reintentar. Si falla la copia de los archivos del avance, el destinatario pasa a `failed` y el respaldo queda en el volumen para recuperar el estado anterior. Si el snapshot no coincide con su hash o no se puede desempaquetar, no se copia nada.

**Alcance del respaldo.** El respaldo guarda el contenido de los archivos de texto que se van a sobrescribir. Un binario del alumno en la misma ruta que un archivo del avance no se respalda; el avance solo transporta archivos regulares del ejercicio.

**Retención de respaldos.** Cada volumen conserva los **5 respaldos más recientes** de `.solv_backups/`. Al aplicar el sexto, se borra la carpeta más vieja. La poda corre después de escribir el respaldo nuevo y es best-effort: si falla, la aplicación termina y los respaldos viejos se reintentan en la próxima. Solo se borran hijas directas de `.solv_backups`; una ruta con `../` o con más segmentos se descarta. El respaldo es una red de seguridad para deshacer una aplicación, no un historial de versiones.

**Sin almacén de snapshots.** Si el backend arranca sin la configuración de SeaweedFS, aplicar responde 503 y el aviso sigue visible; emitir y congelar no dependen de ese almacén.

### 8. Aviso por consulta periódica

El cliente del alumno consulta cada minuto sus avances pendientes, con el mismo ciclo que el latido de sesión de ADR-056. No se usa WebSocket ni SSE para este flujo. Un aviso puede tardar hasta un minuto en aparecer, y así se documenta en la interfaz.

La consulta solo entrega avisos que todavía se pueden aplicar. Uno cuya ventana de disponibilidad venció se cierra como `expired` en ese momento, de modo que deja de mostrarse sin esperar a que el alumno lo intente. Un aviso de examen se oculta pero no se marca vencido: es una prohibición, no un plazo.

El aviso aparece en la pantalla donde el alumno trabaja: el solver de la práctica muestra solo los avisos de su `exercise_id` y los aplica en el volumen de esa práctica (ADR-059). La pantalla de laboratorio por materia conserva la franja general de la materia. Ambas superficies comparten la consulta periódica y el mismo modal de confirmación.

### 9. Modelo de datos

**Cambio en `class_sessions`:**

| Campo | Tipo | Nota |
| --- | --- | --- |
| `frozen_at` | timestamptz, nullable | Momento de congelación manual; nulo si la clase no está congelada |
| `grace_until` | timestamptz, nullable | Fin de la ventana de disponibilidad de los avances de la sesión; se fija al cerrar con 30 minutos y puede sumarse, fijarse con fecha límite o cerrarse antes |

**Nueva tabla `class_advances`:**

| Campo | Tipo | Nota |
| --- | --- | --- |
| `id` | uuid | PK |
| `tenant_id` | uuid | Aislamiento multi-tenant |
| `session_id` | uuid | FK a `class_sessions` |
| `exercise_id` | uuid | Ejercicio cuyo código se emite |
| `scope` | varchar(10) | `class` o `selected` |
| `snapshot_key` | text | Clave del objeto en SeaweedFS (`class-advances/<tenant_id>/<avance_id>.tar.gz`) |
| `snapshot_sha256` | char(64) | Hash del tar.gz; se verifica antes de copiar |
| `snapshot_size_bytes` | bigint | Tamaño del tar.gz, para el tope por avance |
| `file_count` | integer | Archivos incluidos en el snapshot |
| `created_by` | uuid | Docente que emitió |
| `created_at` | timestamptz | Emisión |
| `snapshot_purged_at` | timestamptz, nullable | Cuando el barrido borró el objeto; nulo mientras el código siga en SeaweedFS |

**Nueva tabla `class_advance_recipients`:**

| Campo | Tipo | Nota |
| --- | --- | --- |
| `advance_id` | uuid | FK a `class_advances` |
| `student_id` | uuid | Alumno destinatario |
| `status` | varchar(10) | `pending`, `applied`, `dismissed`, `failed`, `expired` |
| `decided_at` | timestamptz, nullable | Cuando el alumno aplicó o descartó |
| `backup_path` | text, nullable | Respaldo creado al aplicar |

Clave primaria compuesta `(advance_id, student_id)`. Las migraciones viven en `backend/migrations` y son idempotentes.

### 10. Endpoints

| Método | Ruta | Rol | Efecto |
| --- | --- | --- | --- |
| `POST` | `/api/v1/teacher/sessions/{id}/advances` | docente | Emite un avance (`exercise_id`, `scope`, `student_ids` si es `selected`) |
| `GET` | `/api/v1/teacher/sessions/{id}/advances` | docente | Emisiones del aula con el estado de cada destinatario |
| `GET` | `/api/v1/teacher/sessions/{id}/history` | docente | Historial de la clase: emisiones, congelaciones, ventana y decisiones de los alumnos |
| `PATCH` | `/api/v1/teacher/sessions/{id}/freeze` | docente | Congela la sesión; responde con la sesión y su `frozen_at` |
| `PATCH` | `/api/v1/teacher/sessions/{id}/unfreeze` | docente | Descongela la sesión; `frozen_at` vuelve a nulo |
| `PATCH` | `/api/v1/teacher/sessions/{id}/grace` | docente | Ajusta la ventana con el aula cerrada: `{"add_minutes": 15}` suma al final vigente, `{"until": "2026-11-01T22:00:00Z"}` fija la fecha límite de una tarea y `{"close": true}` la cierra en el momento |
| `GET` | `/api/v1/class-advances/pending` | alumno | Avances pendientes para el alumno autenticado |
| `GET` | `/api/v1/class-advances/{id}/files` | alumno | Archivos del aviso: los que se agregan y los que se reemplazan, sin escribir |
| `POST` | `/api/v1/class-advances/{id}/apply` | alumno | Aplica el avance con respaldo previo |
| `POST` | `/api/v1/class-advances/{id}/dismiss` | alumno | Marca el aviso como descartado |

Los diez endpoints están montados. La lectura de archivos es la que hace posible el paso 1 de §7; sin ella la confirmación solo podría anunciar una cantidad. La lectura por sesión es la que responde "quién lo aplicó": devuelve cada emisión con sus destinatarios ya resueltos (nombre, correo, estado y hora de decisión) y el conteo por estado, del más nuevo al más viejo. El filtro es el docente dueño de la materia; un `teacherID` vacío (administrador) no filtra. Congelar y descongelar comparten camino: el rol `student` recibe 403 antes de llegar al servicio, una sesión inexistente o de otro docente recibe 404 y el éxito devuelve 200 con la sesión. Ajustar la ventana exige el aula cerrada (409 si sigue abierta): `add_minutes` suma al final vigente (1 a 480 minutos), `until` fija una fecha futura dentro de los 7 días desde el cierre y `close` la termina en el momento. El historial de la clase se lee en `GET /history`; ver §12.

**Pantalla docente.** `/teacher/aula` usa esas rutas en la misma vista: el modal de emisión elige la práctica publicada y el alcance entre quienes pasaron por el aula; la cabecera congela o descongela; y, con el aula cerrada, el bloque de la ventana muestra "Disponible hasta ...", un campo libre de minutos para sumar, un campo de fecha y hora para la tarea y `[Cerrar ventana ahora]`. La pantalla conserva la sesión cerrada al cerrar el aula, porque ajustar la ventana necesita su identificador. Ver `proyecto-grado/avances/emision-avances-aula.md`.

**Estado de los avances en el aula.** `/teacher/aula` muestra cada emisión de la sesión con la práctica, la hora, la cantidad de archivos y los conteos de aplicaron, descartaron y pendientes; `[Ver detalle]` abre la lista de alumnos con su estado y la hora en que decidieron. La lista se refresca al emitir y cuando el docente lo pide: contar destinatarios no es el ritmo de la clase, así que no se consulta en el bucle de 5 segundos.

**Pantalla del alumno.** Su laboratorio activo muestra una franja ámbar por aviso pendiente, con la cantidad de archivos, la hora de emisión y la aclaración de que puede tardar hasta un minuto. `[Ver archivos]` abre la confirmación con las dos listas; `[Descartar]` cierra el aviso sin tocar el entorno. Ver `proyecto-grado/avances/aviso-avance-alumno.md`.

Reglas de autorización:

- Solo el docente que es dueño de la materia puede congelar, descongelar, extender o emitir. Para otro docente el aula no existe: 404.
- Emitir con un snapshot que supera el tope de tamaño o de archivos devuelve 422.
- Un alumno solo ve y aplica avances donde es destinatario. Cualquier otro identificador devuelve 404.
- Emitir a un alumno que no participa de la sesión devuelve 422.
- Emitir o aplicar en examen, con la clase congelada o con la ventana de disponibilidad vencida devuelve 409. Ajustar la ventana solo es válido con la sesión cerrada; puede hacerse con la ventana activa o ya vencida.
- Todas las consultas filtran por `tenant_id`.

### 11. Concurrencia

La emisión guarda un solo snapshot; no copia el código a cada volumen en ese momento. La copia al volumen del alumno ocurre cuando cada alumno aplica el avance. Cuando muchos alumnos aplican a la vez, el backend limita cuántas copias corren en paralelo, para no saturar la red ni el disco del servidor On-Premise. El valor vive en configuración, no en código. Valor por defecto: 4.

El límite no bloquea a nadie: los pedidos esperan turno y cada alumno recibe su copia. Es una decisión de capacidad del servidor, no una opción del docente ni un plazo para el alumno. El valor se configura con `ADVANCE_APPLY_CONCURRENCY` (4 por defecto). El semáforo se toma después de validar el aviso y antes de leer el snapshot, así que una aplicación en espera no toca el volumen del alumno.

### 12. Auditoría

Emitir, congelar, descongelar, sumar minutos, fijar la fecha límite, cerrar la ventana, aplicar y descartar generan entradas en `audit_logs` (ADR-027), con el docente o el alumno como actor, la sesión como recurso y el avance en la metadata. Cada cambio de ventana guarda el valor anterior y el nuevo.

| Acción | Detalle que guarda |
| --- | --- |
| `CLASS_ADVANCE_EMITTED` | Avance, práctica, alcance, destinatarios y archivos |
| `CLASS_FROZEN` / `CLASS_UNFROZEN` | Docente y estado de congelación |
| `CLASS_ADVANCE_GRACE_EXTENDED` | Minutos sumados, ventana anterior y nueva |
| `CLASS_ADVANCE_GRACE_DEADLINE_SET` | Ventana anterior y fecha límite |
| `CLASS_ADVANCE_GRACE_CLOSED` | Ventana anterior y hora de cierre |
| `CLASS_ADVANCE_APPLIED` | Alumno, avance y archivos escritos |
| `CLASS_ADVANCE_DISMISSED` | Alumno y avance |

El docente lee el historial de su clase con `GET /api/v1/teacher/sessions/{id}/history`. La respuesta resuelve nombre, correo y rol del actor, y viene del más nuevo al más viejo, con un máximo de 200 movimientos. En `/teacher/aula` el bloque **Historial de esta clase** aparece con el aula abierta y con la última cerrada. El alumno recibe `403`; un aula ajena, `404`; el administrador puede leer cualquier aula de su institución y ve el registro completo en la pantalla de auditoría.

El registro es de solo lectura. Si la escritura de una entrada falla, la acción del aula no se cae: el movimiento falta en el historial y la pantalla conserva la última lista que leyó. La lectura usa el índice parcial de `00039_class_session_history_index.sql`, así que no recorre todo el registro de la institución.

---

## Alternativas descartadas

| Alternativa | Motivo del descarte |
| --- | --- |
| Sincronización forzada con pausa y sobrescritura (ADR-007) | Quita la decisión al alumno y puede borrar su trabajo sin aviso |
| Emitir sin sesión de aula | Deja avances pendientes sin fin ni destinatarios claros |
| Avance permitido en examen | Permite entregar soluciones durante la prueba |
| Copiar todo el volumen, incluida configuración del IDE | Cambia el entorno del alumno y puede romper la inicialización de la plataforma |
| Aviso por WebSocket o SSE | Agrega una conexión que mantener y probar, sin beneficio frente a un aviso por minuto |
| Aplicar sin respaldo | Sin copia previa, un error de copia no se puede deshacer |

---

## Consecuencias

**Positivas**

- El alumno conserva el control de su código.
- El docente puede mostrar una solución sin apagar entornos ni reiniciar contenedores.
- La regla de examen queda explícita en el modelo, no en la interfaz.

**Costos y cuidados**

- La ventana se fija sola al cerrar y cada cambio queda en el historial de la clase; el administrador conserva el registro completo en la auditoría.
- La ventana puede durar hasta el máximo institucional (7 días desde el cierre, por defecto). El máximo y el registro de cada cambio evitan que un avance quede abierto sin control. El modo tarea no agrega un segundo mecanismo: es la misma ventana con una fecha límite.
- Un avance puede seguir disponible después de que la clase termine. La interfaz debe mostrar la fecha al docente y al alumno, y el docente debe poder cerrarla antes.
- El aviso puede tardar hasta un minuto. La interfaz debe decirlo.
- Aplicar requiere que el contenedor del alumno esté en ejecución.
- El snapshot se conserva 8 días desde el cierre del aula y después se borra. El plazo cubre la última reapertura posible de la ventana.
- Los respaldos en `.solv_backups/` se limitan a los 5 más recientes por volumen. El respaldo es una red de seguridad, no un historial de versiones.
- SeaweedFS pasa a ser un servicio más de On-Premise, con su volumen, credenciales y respaldo propios. Su imagen se fija a una versión concreta, sin `:latest`.

---

## Documentos afectados

- ADR-007: pasa a estado **Reemplazado por ADR-058**.
- `docs/ARQUITECTURA/ADR/INDEX.md`: se agrega la fila 058 y el nuevo estado de 007.
- `docs/UI/DOCENTE/DASHBOARD.md` (sección 4.2) y `docs/UI/ESTUDIANTE/LABORATORIO_ACTIVO.md`: describen el flujo implementado, incluida la ventana automática y sus tres acciones.
- `docs/UI/MAPA_DE_VISTAS.md` §4.4 y §4.5, y `docs/API_CONTRACTS.md`: fichas del aula (emisión, congelación y ventana), del entorno de la práctica y de su lectura por materia.
- `proyecto-grado/avances/retencion-snapshots-y-respaldos.md`: política de retención de snapshots y respaldos, con plazos, poda y criterios de implementación.

---

## Verificación

- Emitir en una sesión abierta con `scope = class` crea un destinatario por cada participante de la sesión.
- Emitir con `scope = selected` a un alumno fuera de la sesión devuelve 422.
- Al cerrar el aula, la ventana queda en 30 minutos por defecto.
- Sumar minutos alarga el final vigente; si la ventana ya venció, cuenta desde ahora.
- Fijar una fecha límite para una tarea mantiene el avance aplicable fuera del horario de clase.
- Cerrar la ventana impide aplicar de inmediato.
- Un destinatario `expired` vuelve a `pending` si la ventana se reabre antes de que decida; `applied`, `dismissed` y `failed` no se revierten.
- El vencimiento se refleja sin depender de que el alumno abra el laboratorio.
- El docente ve en el historial la emisión, la congelación, la descongelación, los cambios de ventana y las decisiones de los alumnos.
- Un alumno recibe 403 al pedir el historial; un docente de otra materia recibe 404.
- El historial de una sesión no mezcla peticiones HTTP ni movimientos de otra aula.
- Emitir sobre un ejercicio `purpose = exam` devuelve 409.
- Emitir con la sesión congelada devuelve 409; descongelar permite emitir de nuevo.
- Un docente de otra materia recibe 404 al emitir, congelar o leer los avances del aula.
- Un alumno no ve avances ajenos y recibe 404 si intenta aplicar uno que no es suyo.
- Aplicar crea `.solv_backups/<avance_id>/` con los archivos previos antes de copiar.
- Las rutas excluidas (`.vscode/`, `.solv_*`, `.git/`, `node_modules/`, `.cache/`, `.openvscode-server/`) no cambian en el volumen del alumno.
- Un avance emitido no cambia cuando el docente edita su código después.

Estos casos se cubren en `backend/tests/integration/class_advance_edges_test.go` (recorrido HTTP completo contra PostgreSQL) y en los subcasos de `class_advance_repository_test.go`. Las exclusiones de rutas y las rutas inseguras del snapshot se prueban en `backend/internal/infrastructure/storage/seaweedfs/archive_test.go`.

La retención tiene criterios propios en [retencion-snapshots-y-respaldos.md](../../../proyecto-grado/avances/retencion-snapshots-y-respaldos.md). `class_advance_retention_test.go` cubre el borrado de vencidos y huérfanos, el fallo de borrado que deja la fila sin marcar, una clave de otro tenant que no se toca, la falta de almacén y la poda de respaldos; la prueba de integración homónima cubre la consulta de candidatos y la marca contra PostgreSQL, incluido el borde de 7/9 días.

### Recorrido real

`backend/tests/integration/class_advance_docker_e2e_test.go` recorre el circuito sin dobles: dos volúmenes de Docker sembrados con código, un snapshot real en SeaweedFS, el aviso del alumno, la previsualización, la aplicación con respaldo y el vencimiento de la ventana. Está apagado por defecto porque necesita el daemon de Docker, el servicio `seaweedfs` de Compose y credenciales válidas; se activa con `ADVANCE_E2E=1` y `SEAWEEDFS_*` apuntando al host:

```bash
ADVANCE_E2E=1 SEAWEEDFS_ENDPOINT=http://127.0.0.1:8333 SEAWEEDFS_BUCKET=solv-advances \
  go test ./tests/integration -run TestClassAdvanceDockerE2E -v
```

Comprueba que el snapshot excluye `.vscode/` y `node_modules/`, que leerlo con otro hash falla, que el volumen del alumno recibe los archivos emitidos, que `.solv_backups/<avance_id>/` conserva el archivo previo y que, al vencer la ventana, el destinatario queda `expired` sin tocar el volumen. El objeto sigue en SeaweedFS después del vencimiento: la retención lo borra recién a los 8 días del cierre. El bucket de la prueba debe ser de desarrollo, porque crea y borra objetos.

La comprobación de la retención contra SeaweedFS real (objeto a los 7 días, aula abierta, aislamiento entre instituciones) sigue pendiente.
