# ADR 055: Panel Docente como Read Model por Pregunta y Dueño Declarado de Excepciones

> **Estado:** Aprobado
> **Fecha:** 2026-10-30
> **Ámbito:** Docencia, Core, Contratos de lectura
> **Relacionado:** ADR-053 (estado de revisión y libreta), ADR-052 (read model del banco de ejercicios), ADR-049 (telemetría persistida)

---

## Contexto

El panel del docente mostraba cifras que no cerraban entre sí y avisos que el docente no podía accionar. La medición sobre la pantalla en uso dejó cinco hechos:

1. **La franja de indicadores era la suma de las tarjetas.** Total de estudiantes, activos, entregas pendientes y alumnos en riesgo se calculaban en el navegador sumando los campos de las tarjetas que aparecen debajo. No agregaban información.
2. **"Activo" tenía dos definiciones.** El indicador contaba entornos con heartbeat menor a 2 minutos; el widget de telemetría mostraba cualquier entorno devuelto por la vista en vivo. En la misma pantalla se leían 0 y 4 activos.
3. **El mismo hecho tenía tres cifras.** La cola aparecía como 21 en el indicador, 4 en la cabecera de tareas y 0 en la tarjeta "Por calificar". La última nacía del cruce de `review_status` en la vista con un endpoint que no envía ese campo: seguía deduciendo la cola del veredicto, en contra de lo que el ADR-053 había declarado.
4. **La columna de tareas mezclaba dueños.** Un bloqueo por memoria del contenedor entraba como tarea del docente, duplicado por entorno, sin acción en la propia tarjeta y sin forma de cerrarse: la consulta volvía a traerlo durante 24 horas en cada carga.
5. **Los conteos no escalaban.** El resumen de materias ejecutaba cuatro consultas por materia dentro de un bucle: con 30 materias, 121 viajes a la base por carga.

El problema de fondo es que la pantalla agregaba datos de tres dominios distintos (académico, trabajo pendiente, infraestructura) con refrescos y dueños diferentes, y lo hacía en el cliente.

---

## Decisión

### 1. Un read model por pregunta

Cada endpoint responde una pregunta y devuelve los datos ya resueltos:

| Endpoint | Pregunta |
| --- | --- |
| `GET /api/v1/teacher/overview?period_id=` | ¿qué me espera hoy? |
| `GET /api/v1/teacher/sessions/{id}` | ¿qué pasa ahora en esta clase? |
| `GET /api/v1/teacher/reviews` | ¿qué me falta corregir? |
| `GET /api/v1/teacher/courses/{id}/submissions` | la cola filtrada por materia |

Abrir y cerrar aula son escrituras: `POST /api/v1/teacher/sessions` y `PATCH /api/v1/teacher/sessions/{id}`, con el estado `open` o `closed`.

El resumen devuelve conteos (`pending`, `practices`, `open_now`, `enrolled`, `present_now`, `without_submission`) y no listas crudas para agregar. Se resuelve con dos consultas agregadas, sin recorrer materias en un bucle.

### 2. El cliente no suma, no agrupa y no recorta

Las plantillas del área docente no contienen sumas ni agrupaciones sobre colecciones, y ningún corte de lista es silencioso. Cuando una lista se limita, el read model lo declara con el total y la vista ofrece la salida completa.

Un corte del tipo `slice(0, 3)` sobre una cola de trabajo esconde trabajo real; se reemplaza por un total explícito o por una lista sin límite dentro de su propia pantalla.

### 3. El dueño de la excepción es un dato del dominio

Cada excepción se clasifica por rol responsable antes de salir del backend:

| Excepción | Dueño |
| --- | --- |
| Memoria agotada, contenedor caído, entorno huérfano | Admin |
| Regla de código bloqueada o error repetido en clase | Docente |
| Regla bloqueada o error repetido fuera de clase | Docente, como resumen en la materia |
| Entrega esperando calificación | Docente |
| Práctica sin corregir | Docente |

El read model del docente devuelve solo las excepciones con dueño docente. Las de infraestructura no se envían: viven en el panel de incidencias del admin, que ya existe con la salida del contenedor.

Regla derivada: **un aviso sin acción disponible para el rol actual no se renderiza.** Una tarjeta con la acción equivocada es peor que su ausencia.

### 4. Una sola cola, varias puertas

La cola de corrección se define por `review_status` en `pending` o `in_review`, según el ADR-053. "Por calificar", la pestaña de entregas de la materia y el acceso desde el panel leen el mismo read model y muestran los mismos números. Se retira la pestaña de entregas del centro de evaluaciones porque duplicaba la cola con otro criterio.

Estar en varios lugares no es el problema; tener varias implementaciones del mismo cálculo, sí.

### 5. Presupuesto vertical como restricción de diseño

Arriba del pliegue, dos filas de chrome como máximo: la cabecera y una fila de acción. Sin banners de alerta.

La medición de partida dejaba el primer contenido a 410 px de un alto útil de 640 px. El objetivo es 88 px. Una alerta que no se puede cerrar ni ejecutar no ocupa el espacio de una tarea.

---

## Alternativas descartadas

| Alternativa | Motivo del descarte |
| --- | --- |
| Mantener los indicadores y quitar solo el duplicado visual | Siguen siendo sumas de las tarjetas; el espacio se gasta en repetir, no en informar |
| Corregir la definición de "activo" en el cliente | El dato sigue viajando sin definición; cualquier vista nueva vuelve a inventarla |
| Dejar la cola de infraestructura en el panel con un botón de descartar | El descarte sin resolución oculta el problema en lugar de cerrarlo, y el docente no puede resolverlo |
| Enviar todas las excepciones y ocultarlas por rol en la vista | La regla queda en la plantilla; cualquier cliente nuevo la incumple |
| Un solo endpoint por entidad, con la agregación en la vista | Es el estado actual: tres cifras del mismo hecho y 4N consultas |
| Sumar la columna de tareas al panel sin dueño declarado | Obliga a decidir pantalla por pantalla qué entra, que es el problema que se quiere eliminar |

---

## Consecuencias

**Positivas**

- Una cifra por hecho, con definición escrita y un solo origen.
- El docente deja de recibir avisos que no puede accionar y se elimina la pregunta "¿esto va en el panel?".
- La carga del panel deja de crecer con el número de materias.
- El retiro de la grilla reordenable y de la personalización elimina dos mecanismos de estado en el cliente.

**Costos y cuidados**

- `GET /teacher/overview` es un contrato nuevo: la etapa 1 depende de él, y la pantalla no se publica antes.
- `AttentionStandardAlert` y el widget de tareas se retiran del área docente; los consumidores que queden deben migrar al endpoint de revisiones.
- El indicador "En riesgo" desaparece del producto. Cualquier reporte o captura que lo mencione queda obsoleto.
- Los conteos pasan a ser responsabilidad del backend: un cambio de criterio se hace en el read model, no en la plantilla.
- El presupuesto vertical exige revisar cada pantalla nueva contra la medición de §5, no solo contra el diseño.

---

## Verificación

- Ninguna plantilla del área docente contiene `reduce`, sumas sobre colecciones ni cortes de lista.
- El resumen se resuelve sin consultas dentro de un bucle: se verifica con el conteo de consultas de la carga sobre una base con 30 materias.
- Un aviso de memoria agotada no aparece en ninguna respuesta del área docente y sí en el panel de incidentes del admin.
- La cola de "Por calificar", la de la materia y el contador del panel coinciden en la misma base.
