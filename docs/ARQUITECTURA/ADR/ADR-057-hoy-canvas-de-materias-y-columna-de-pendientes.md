# ADR 057: Hoy como Canvas de Materias y Columna de Pendientes

> **Estado:** Aprobado
> **Fecha:** 2026-10-30
> **Ámbito:** Docencia, contratos de lectura, interfaz
> **Relacionado:** ADR-055 (read model por pregunta), ADR-056 (sesión de aula y presencia acotada), ADR-053 (estado de revisión)
> **Enmienda:** el contrato de `GET /api/v1/teacher/overview` del ADR-055

---

## Contexto

La etapa 1 del panel docente cerró el contrato de lectura pero no la pantalla. Al revisar la superficie quedaron cuatro hechos:

1. **La tabla exigía leer, no reconocer.** Con dos materias alcanza; con treinta, el docente recorre filas de números para encontrar la suya y vuelve a leer el código cada vez. Un canvas de tarjetas se reconoce por posición antes de leer.
2. **La urgencia no tenía lugar.** La tabla ordena por código y todas las filas pesan lo mismo; nada decía qué vence esta semana ni qué aula está abierta ahora.
3. **Los contenedores seguían en la pantalla del docente.** El resumen devuelve `environments.open_now` y `subjects[].open_environments`, y la pantalla los dibujaba como "Ahora" y "Entornos". El docente no puede reiniciar ni ajustar un entorno: es una cifra sin acción, justo lo que la regla R1 del área prohíbe. Los contenedores se operan en Aula, donde hay una consola y un tablero.
4. **Un solo vencimiento obligaba a recortar.** El contrato entrega `next_due` en singular. Un bloque de vencimientos con una sola fecha no sirve, y resolverlo en la vista habría exigido elegir y cortar, contra R3.

---

## Decisión

### 1. El canvas es la superficie de materias

Cada materia es una tarjeta con código, nombre, cifras y acciones, en una grilla `repeat(auto-fill, minmax(260px, 320px))` con `justify-content: start`, para que dos materias no se estiren hasta ocupar media pantalla. El orden llega del servidor por código ascendente y **no cambia**: mover tarjetas entre cargas rompe la memoria de posición, que es lo único que el canvas aporta sobre una tabla. Sin arrastrar, sin personalizar y sin ocultar.

Cifras en la tarjeta: inscritos, por calificar y sin entregar. Una cifra en cero no se dibuja y `without_submission` nulo se omite, porque significa que no hay práctica por vencer y no existe conjunto que contar. Sin cola, la tarjeta dice "Al día".

### 2. Los pendientes son una sola columna, con dueño y acción

Tres bloques en orden fijo: `Aula ahora`, `Vencimientos` y `Por calificar`. Cada bloque se oculta sin contenido; sin nada pendiente la columna no se dibuja y el canvas ocupa el ancho. El contenido de la columna es lo único de la pantalla con una acción a un clic, y esa es la condición para estar ahí.

La columna no repite cifras del canvas y no se reordena entre refrescos. Es la única columna lateral admitida en el área docente; cualquier otra necesita su propio ADR.

### 3. La ejecución sale de Hoy y del contrato del resumen

`environments` y `subjects[].open_environments` se retiran del read model del panel. La telemetría, los entornos y la terminal de sombra viven en `/teacher/aula`, que ya los resuelve con datos de la sesión. Una cifra de contenedores sin acción para el rol no viaja.

### 4. `due_soon` sustituye a `next_due` y es accionable con paginación compacta

`due_soon[]` entrega las prácticas publicadas próximas a vencer o con entregas pendientes (tope técnico de 50 registros en backend). La columna implementa paginación minimalista de 4 ítems por página y enlace directo a la pestaña de entregas de la materia (`/teacher/cursos/:id?tab=submissions&labId=:exercise_id`). `due_soon_total` declara el total y `due_soon_hidden` los que exceden el tope, sin restar filas en el cliente.

La sesión del resumen suma `subject_code` y `ends_at`, y cada materia suma `next_due_title`, porque la franja y la tarjeta nombran la materia y la práctica que vencen.

### 5. Prioridad del documento por encima de la posición visual

El orden del documento es la columna y después el canvas; la grilla coloca el canvas a la izquierda sin alterar ese orden. Un lector de pantalla recorre primero lo accionable, y el orden de tabulación coincide con la prioridad del trabajo.

---

## Alternativas descartadas

| Alternativa | Motivo del descarte |
| --- | --- |
| Mantener la tabla de materias de la etapa 1 | Ordena por código y todas las filas pesan igual: no hay urgencia y con treinta materias obliga a leer, no a reconocer |
| Volver a la grilla de widgets reordenable | Cuatro bloques iguales comunican que todo importa lo mismo y desplazan al docente la decisión de qué mirar |
| Canvas con una franja de KPIs arriba | Vuelve la suma de las tarjetas que el ADR-055 retiró, y roba la altura que necesita la primera fila |
| Acento cromático por materia | Seis acentos sin regla semántica no se distinguen entre sí y compiten con el rojo de "se trabó" |
| Columna de pendientes siempre visible | Una columna vacía ocupa 320 px y sugiere que falta algo que no existe |
| Reordenar las tarjetas por urgencia | El mapa cambia debajo del cursor y el docente pierde la referencia de posición, que es su razón de ser |
| Resolver los vencimientos en la vista a partir de `next_due` | La vista tendría que elegir y recortar, contra R3 |

---

## Consecuencias

**Positivas**

- La pantalla responde qué vence, qué se corrige y qué aula está abierta sin leer una rejilla de números.
- Desaparece la última cifra de infraestructura del área docente sin acción para el rol.
- El contrato deja de tener un campo que obliga a recortar en el cliente (`next_due`).
- El retiro de la personalización elimina un mecanismo de estado en el navegador y una fuente de capturas que no coinciden.

**Costos y cuidados**

- El contrato del resumen cambia: `next_due`, `environments` y `open_environments` salen; `due_soon`, `due_soon_total`, `due_soon_hidden`, `session.ends_at`, `session.subject_code` y `subjects[].next_due_title` entran. La pantalla no se publica contra el contrato viejo.
- Las acciones de la tarjeta necesitan `?subject_id=` en `/teacher/aula` y `/teacher/por-calificar`; sin el parámetro, la tarjeta abre el destino sin preselección.
- Con una sola materia el canvas se ve vacío; el ancho máximo de 320 px lo evita a medias, y el estado "Nada pendiente" puede dejar la pantalla casi sin contenido. Es un estado real con pocas materias, no un defecto a tapar con bloques decorativos.
- El bloque `Vencimientos` pagina de a 4 prácticas con controles compactos (`< N/M >`) y enlaza directo a las entregas de cada práctica en el detalle de la materia (`/teacher/cursos/:id?tab=submissions&labId=:exercise_id`).

---

## Verificación

- Ninguna tarjeta ni bloque de Hoy muestra entornos, contenedores ni telemetría.
- Una materia sin entregas por calificar no muestra un cero: muestra "Al día".
- Sin pendientes, la columna no se dibuja y el canvas ocupa el ancho completo.
- Con treinta materias, la pantalla se resuelve en una carga y el orden de las tarjetas es el mismo entre refrescos.
- El orden de tabulación empieza en la columna de pendientes y sigue en el canvas.
- La respuesta del resumen no contiene `environments` ni `open_environments`, y sí `due_soon` con su total y su cantidad oculta.
- La pantalla no contiene sumas, agrupaciones ni cortes de listas; solo compara identidad para saber qué materia tiene el aula abierta.
