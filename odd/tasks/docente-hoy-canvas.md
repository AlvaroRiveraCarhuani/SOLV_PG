# Feature: Hoy como Canvas de Materias y Columna de Pendientes

Objetivo: cerrar la etapa 1 del área docente. La pantalla `Hoy` deja la grilla de widgets y pasa a ser un canvas de materias con su columna de pendientes, alimentada por el resumen que ya existe.

Diseño de referencia: `docs/UI/DOCENTE/REDISENO_PANEL_DOCENTE.md` §5.1, §6.1 y §9.
Decisión: `docs/ARQUITECTURA/ADR/ADR-057-hoy-canvas-de-materias-y-columna-de-pendientes.md`.
Delivery strategy: encadenado. Son más de 400 líneas autoradas, así que se parte en tres tramos revisables: contrato, pantalla, retiro.

## Estado de partida, antes de esta etapa (2026-10-30)

- `GET /api/v1/teacher/overview` existe, compila y resuelve período, sesión, cola, próximo vencimiento, entornos y materias en cuatro consultas agregadas (`teacher_overview_repository.go`, `teacher_overview.go`).
- `frontend/src/app/features/teacher/models/teacher.models.ts` tipa `TeacherOverview`, `TeacherOverviewNextDue` y `TeacherOverviewEnvironments`.
- `TeacherDashboardService.loadOverview()` existe y **no tiene consumidor**.
- La pantalla `Hoy` sigue montada sobre la grilla `cdkDropList` con los bloques `courses` y `attention`, el banner de atención y `Personalizar`. No lee el resumen.

## Decisiones cerradas

- **Canvas, no tabla.** Orden por código ascendente desde el servidor, sin reordenar por urgencia y sin arrastrar: la posición es la memoria del docente.
- **Una sola columna lateral**, con tres bloques de orden fijo y ocultamiento por contenido. Es la única admitida en el área; cualquier otra necesita su ADR.
- **Nada de ejecución en Hoy.** `environments` y `open_environments` salen del contrato; la telemetría se queda en Aula.
- **La vista no elige ni recorta.** `due_soon` llega ordenado y limitado, con `due_soon_total` para declarar el corte.
- **Prioridad en el DOM.** La columna va antes que el canvas en el documento; la grilla las ubica sin cambiar ese orden.
- **Sin acento por materia.** El canvas se reconoce por código, nombre y posición.

## Delta de contrato

| Campo | Acción | Detalle |
| --- | --- | --- |
| `session.subject_code` | Agregar | La franja nombra la materia por su código |
| `session.ends_at` | Agregar | La franja dice a qué hora cierra el aula |
| `due_soon[]` | Agregar | Hasta 5 prácticas publicadas con vencimiento dentro de 7 días: `subject_id`, `subject_code`, `subject_name`, `exercise_id`, `exercise_title`, `due_at` |
| `due_soon_total` | Agregar | Total antes del límite, para que la columna diga "y N más" sin contar |
| `due_soon_hidden` | Agregar | Cuántos quedaron fuera del límite, restado por el servidor: la vista no resta filas |
| `next_due` | Retirar | Sustituido por `due_soon` |
| `subjects[].next_due_title` | Agregar | La tarjeta nombra la práctica que vence |
| `environments` | Retirar | Cifra de contenedores sin acción para el docente |
| `subjects[].open_environments` | Retirar | Igual que el anterior; la presencia de la materia sale de la sesión abierta |

## Specs

- **S1 — Contrato del resumen.** Dominio, repositorio, handler y modelo de TypeScript con el delta de la tabla. `due_soon` se resuelve en la misma consulta de materias, sin sumar viajes.
  - Criterio: la respuesta no contiene `environments` ni `open_environments`; `due_soon` viene ordenado por fecha y con su total; un estudiante sigue recibiendo 403 y un `period_id` inválido, 400.
  - Verificación: `TestSlice18_TeacherOverview_ReadModel` extendido con `due_soon`, el total, `ends_at` y el nombre de la práctica que vence.

- **S2 — Rutas con materia preseleccionada.** `/teacher/aula` y `/teacher/por-calificar` leen `?subject_id=` y lo aplican al formulario de apertura y al filtro de la cola; sin parámetro se comportan como hoy.
  - Criterio: abrir `/teacher/aula?subject_id=<id>` deja la materia elegida; con un identificador que no es del docente, cae al comportamiento sin preselección y no muestra error de permiso.

- **S3 — `subject-card`.** Componente de presentación sin estado propio: código en `<code>`, nombre a dos líneas con `title`, cifras que se omiten en cero, vencimiento con `dateText:'daymonth'` en `<time>`, hasta dos acciones y enlace estirado al nombre.
  - Criterio: sin cola ni aula, la tarjeta no renderiza ninguna acción y dice "Al día"; con `without_submission` nulo no aparece la cifra de faltantes; ningún estilo usa hex ni literales de tipografía (`npm run lint:styles` en verde).

- **S4 — `pending-column`.** Tres bloques con `section` y `h2`, visibles solo con contenido, con `[Ir al aula]` en `Aula ahora`, el detalle del vencimiento en `Vencimientos` y el resumen de la cola con `[Calificar]`.
  - Criterio: sin sesión no aparece el bloque `Aula ahora`; sin vencimientos ni cola no se dibuja la columna; con más de cinco vencimientos el bloque cierra con el total.

- **S5 — Pantalla Hoy.** Cabecera con fecha, selector de período y `[Abrir aula]`; franja de sesión condicional; grilla `repeat(auto-fill, minmax(260px, 320px))` con `justify-content: start` y columna de 320 px; refresco cada 60 s sin mover el foco; estados de carga, período vacío, sin pendientes y error con reintento.
  - Criterio: la plantilla no contiene `reduce`, sumas ni `slice`; el primer contenido queda a 88 px sin aula y a 132 px con aula; a 1280 px la columna sube sobre el canvas y a 768 px queda una tarjeta por fila.

- **S6 — Retiro.** Se quitan la grilla `cdkDropList`, el banner de atención y `Personalizar` del panel; `dashboard-courses-widget` y `dashboard-attention-widget` se eliminan; el uso docente de `dashboard-layout.service.ts` se retira y el servicio queda para el panel del estudiante.
  - Criterio: `grep` no encuentra referencias a los widgets retirados ni a la clave de almacenamiento del panel docente; la pantalla del estudiante sigue cargando su diseño.

- **S7 — Documentación.** §5.1, §6.1, §9 y §12 de `REDISENO_PANEL_DOCENTE.md`, la ficha 4.1 de `MAPA_DE_VISTAS.md` y el ADR-057 ya están escritos; al cerrar la etapa se actualiza el estado de §12 y se anota su resultado en este archivo.
  - Criterio: ninguna sección declara hecho lo que no está en el árbol.

## Orden de trabajo

1. Contrato y su test de integración (S1), con el frontend todavía sobre los tipos viejos.
2. Rutas con preselección (S2) y componentes de presentación (S3, S4), probados de forma aislada.
3. Pantalla (S5) sobre el servicio existente, y recién entonces el retiro (S6).

## Matriz de verificación

| Comando | Criterio |
| --- | --- |
| `cd backend && go build ./... && go vet ./...` | Sin errores ni avisos |
| `cd backend && go test ./tests/integration -run TestSlice18_TeacherOverview_ReadModel -v` | Verde contra base real, con `due_soon` y sin `environments` |
| `cd frontend && npx ng build --configuration development` | Compila sin errores |
| `cd frontend && npm run test:ci` | Specs de la pantalla y de los dos componentes en verde, más los gates |
| `cd frontend && npm run lint:styles` | Estilos inline, formato de fechas, datos técnicos y stylelint en verde |
| `grep -rn "dashboard-attention-widget\|dashboard-courses-widget\|solv_teacher_dashboard_bento" frontend/src` | Sin resultados |
| Carga manual con datos sembrados (2026-10-30) | Una sola petición de resumen; canvas en dos columnas, columna con sus tres bloques y el corte "y 1 más" |

## Riesgos

- **Contrato en uso.** `next_due` y `environments` pueden tener otro consumidor: antes de retirarlos, `grep` sobre `frontend/src` y sobre los tests de integración.
- **`due_soon` con costo.** Va en la consulta de materias; si se resuelve con una consulta aparte, se pierde el presupuesto de §6.3.
- **Pocas materias.** Con una materia el canvas se ve vacío; no se compensa con bloques decorativos. El estado "Nada pendiente" es un estado real.
- **`?subject_id=` sin filtrar.** La preselección de Por calificar debe salir del alcance del docente, no de lo que llega en la URL.

## Resultado (2026-10-30)

Cerrada la etapa 1. Lo que quedó en el árbol:

- **Contrato.** `GET /api/v1/teacher/overview` publica `due_soon` (hasta cinco, ordenados por fecha), `due_soon_total`, `due_soon_hidden`, `subjects[].next_due_title`, `session.ends_at` y `session.subject_code`, y ya no publica `environments`, `open_environments` ni `next_due`. Sigue resolviéndose en cuatro consultas agregadas, sin bucle por materia.
- **Pantalla.** `teacher-dashboard` es el canvas de materias con la columna de pendientes, refresco de 60 s, franja de sesión condicional y estados de carga, período vacío, sin pendientes y error con reintento que conserva lo cargado.
- **Componentes.** `subject-card` y `pending-column`, ambos de presentación y sin estado propio. La franja de sesión se pinta en la plantilla de Hoy.
- **Rutas.** `/teacher/aula` y `/teacher/por-calificar` leen `?subject_id=`; un identificador ajeno a la URL cae al comportamiento sin preselección.
- **Retiro.** Salieron la grilla `cdkDropList`, el banner de atención, `Personalizar`, `TEACHER_DEFAULT_WIDGETS`, la clave de almacenamiento del panel y los widgets `dashboard-courses-widget` y `dashboard-attention-widget`. `dashboard-layout.service.ts` queda para el panel del estudiante.

Verificación ejecutada:

| Comando | Resultado |
| --- | --- |
| `cd backend && go build ./... && go vet ./...` | Sin errores ni avisos |
| `cd backend && go test ./tests/integration -run TestSlice18_TeacherOverview_ReadModel -v` | Verde, incluidos `due_soon`, su total, el corte en cero y el aula con `ends_at` |
| `cd frontend && npx ng build --configuration development` | Compila sin errores |
| `cd frontend && npx ng test --watch=false --filter="TeacherDashboardComponent\|SubjectCardComponent\|PendingColumnComponent"` | 14 tests en verde |
| `cd frontend && npm run lint:styles` | Estilos inline, fechas, datos técnicos y stylelint en verde |
| `grep` de widgets y clave retirados sobre `frontend/src` | Sin referencias fuera de su retiro |

Verificación en el navegador (2026-10-30, con sesión de docente real y datos sembrados):

- Estructura medida a 1364 px: cabecera 56, franja de sesión 44, contenido a 124 px del borde superior, canvas de 718 y columna de 320, tarjetas de 320 uniformes y sin desborde horizontal.
- A 1242 px la columna sube sobre el canvas y el canvas deja dos columnas; a 691 px queda una tarjeta por fila de 320, sin estirarse.
- Las acciones generan `/teacher/por-calificar?subject_id=…` y `/teacher/aula?subject_id=…`; la cola filtrada muestra solo la materia elegida y "Ver todas" limpia el filtro; el formulario de aula abre con la materia preseleccionada y sus prácticas publicadas.
- La presencia del aula sigue la definición: al vencer los 120 s de la última señal, la franja pasó de "2 de 2 dentro" a "0 de 2 dentro".

Dos correcciones salieron de mirarla: la tarjeta ya no dice "Al día" cuando tiene alumnos sin entregar, y las fechas de la columna van sin segundos.

Observado y fuera de alcance: por debajo de ~900 px la barra lateral fija del shell deja el área de contenido más angosta que una tarjeta, y ninguna pantalla del área docente se ve bien. Es del shell, no de esta pantalla, y necesita su propio cambio.

Sin hacer: la pantalla que lista los vencimientos más allá de los cinco de la columna, que llega con la materia en la etapa 3.
