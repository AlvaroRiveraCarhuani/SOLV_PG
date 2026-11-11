# Área Docente: Arquitectura de Información y Panel de Trabajo

> **Especificación de arquitectura de información y diseño objetivo**
> **Rol:** Docente (con impacto en Estudiante)
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System / [UX-PRINCIPLES](../UX-PRINCIPLES.md)
> **ADRs:** [055](../../ARQUITECTURA/ADR/ADR-055-panel-docente-read-model-por-pregunta.md), [056](../../ARQUITECTURA/ADR/ADR-056-sesion-de-aula-y-presencia-acotada.md), [057](../../ARQUITECTURA/ADR/ADR-057-hoy-canvas-de-materias-y-columna-de-pendientes.md)
> **Estado:** especificación cerrada, implementación por etapas (§12)
> **Sustituye a:** [DASHBOARD.md](DASHBOARD.md), congelado

---

## Qué cambia, en una frase

El panel deja de ser un tablero de widgets y pasa a ser una pantalla de trabajo con una sola pregunta; las cifras las calcula el servidor una vez, y ningún aviso que el docente no pueda accionar llega a su pantalla.

---

## 1. Problema medido

Los defectos se midieron sobre la pantalla en uso, no sobre una idea de pantalla.

| # | Hecho observado | Causa |
| --- | --- | --- |
| 1.1 | La franja de KPIs dice 18 / 0 / 21 / 0 y las tarjetas dicen 15 + 6 | Los cuatro indicadores son la suma exacta de las tarjetas que están debajo |
| 1.2 | "Activos en vivo" marca 0 mientras la telemetría marca 4 activos | Dos definiciones de activo: heartbeat menor a 2 minutos contra cualquier entorno devuelto por la vista en vivo |
| 1.3 | Tres números distintos para el mismo hecho: 21 en el KPI, 4 en la cabecera de tareas, 0 en la tarjeta "Por calificar" | Tres agregaciones en el cliente sobre endpoints con criterios distintos |
| 1.4 | El banner anuncia "2 estudiantes" y las dos tarjetas son el mismo estudiante | Cuenta alertas (una por entorno) y las presenta como personas |
| 1.5 | El primer contenido real empieza a los 410 px de un alto útil de 640 px | Tres bloques de chrome: cabecera con subtítulo, banner, barra de control con KPIs |
| 1.6 | Con 30 materias, una carga pide 4N + 1 consultas | El resumen de materias ejecuta cuatro consultas por materia dentro de un bucle |
| 1.7 | Cuatro destinos de calificación con tres definiciones de "pendiente" | `/teacher/evaluaciones` y la pestaña del curso filtran por veredicto; `/teacher/por-calificar` lee `review_status` |
| 1.8 | El aviso de memoria insiste con el mismo entorno cada 24 horas y ofrece "Ver entregas", que no lleva al alumno | No hay cierre del aviso ni acción propia del docente |

El modelo ya guarda más de lo que la interfaz muestra. El problema no es falta de datos, es que la pantalla responde tres preguntas con refrescos distintos: el estado académico cambia en días, el trabajo pendiente cambia en horas y la telemetría cambia cada tres segundos.

---

## 2. Las tres reglas del área docente

Estas reglas son la parte normativa del documento. Cualquier pantalla nueva del área docente las cumple o no se construye.

### R1. Un aviso sin acción en el rol actual no se renderiza

El dueño de una excepción es un dato del dominio, no una decisión de la vista. El read model del docente devuelve solo las excepciones cuyo dueño es el docente; el resto ni viaja.

Qué evita: la pregunta "¿esto va en el panel?" y las tarjetas con la acción equivocada.

Cómo se verifica: ninguna excepción del área docente llega sin una acción declarada que el docente tenga permiso de ejecutar.

### R2. Una cifra sin definición escrita no se publica

Cada número visible sale de una consulta y tiene su definición en este documento (§6). Si no se puede definir, no se muestra.

Qué evita: "En riesgo", que exigía vencimiento en 24 horas y sin heartbeat en 24 horas, y por eso marcaba 0 de forma estructural; y "Activos", que tenía dos definiciones incompatibles.

Cómo se verifica: cada cifra de la pantalla aparece en la tabla de §6 con su definición y su origen.

### R3. El cliente no suma, no agrupa y no recorta

Los conteos, las agrupaciones y los límites se resuelven en el read model. La vista solo pinta lo que recibe.

Qué evita: los tres números distintos del mismo hecho, y los cortes silenciosos tipo `slice(0, 3)` que esconden trabajo real sin decirlo.

Cómo se verifica: ninguna plantilla del área docente contiene sumas, agrupaciones ni cortes de listas.

---

## 3. Presupuesto vertical

Referencia de trabajo: portátil de 1366 x 768 con la barra del navegador, que deja 640 px de alto útil bajo la barra superior de la aplicación.

| Bloque | Hoy | Objetivo |
| --- | --- | --- |
| Cabecera con título y subtítulo | 60 px | 56 px, título en una línea con el período y la acción a la derecha |
| Banner de alerta | 56 px | 0 |
| Barra de control | 72 px | 0, el selector sube a la cabecera |
| Franja de KPIs | 100 px | 0 |
| Franja de sesión | — | 44 px, solo con aula abierta |
| **Chrome antes del contenido** | **320 px** | **88 px sin aula, 132 px con aula** |

Regla: **dos filas de chrome arriba del pliegue como máximo**, la cabecera y una fila de acción. Nada de banners.

Con 88 px de chrome quedan 552 px para el contenido y, con el aula abierta, 508 px: alcanzan para la primera fila de tarjetas completa y los tres bloques de la columna de pendientes, que es lo que el docente necesita ver sin desplazar (§5.1).

---

## 4. Dueño de cada excepción

| Excepción | Dueño | Superficie | Acción del dueño |
| --- | --- | --- | --- |
| Memoria agotada, contenedor caído, entorno huérfano | Admin | Panel de incidencias del admin | Ver salida del contenedor, reiniciar, ajustar recursos |
| Regla de código bloqueada durante una clase | Docente | Aula, como "se trabó" | Ver el código, dejar una consigna al alumno o al grupo |
| Errores de compilación repetidos en clase | Docente | Aula, como "se trabó" | Igual que el anterior |
| Regla bloqueada o error repetido fuera de clase | Docente | Materia, como resumen del día siguiente | Revisar la práctica con el alumno |
| Entrega esperando calificación | Docente | Cola de corrección | Calificar en SpeedGrader |
| Práctica sin corregir | Docente | Libreta de la materia | Calificar o declarar la entrega que cuenta |

La memoria agotada **no llega al docente**. El panel de incidencias del admin ya existe y muestra el contenedor con su salida. Si el límite de memoria de una práctica está mal calibrado, el docente lo descubre por los resultados de la práctica, no por una alerta de infraestructura.

Cuando una misma persona aparece con más de un aviso, la superficie muestra un solo registro con la cuenta: "se quedó sin memoria 2 veces", no dos tarjetas.

---

## 5. Las cuatro pantallas

### 5.1 Hoy

Ruta `/teacher/dashboard`. Pregunta: **¿qué me espera hoy?** Se refresca cada 60 segundos.

La pantalla tiene dos superficies y una sola jerarquía: el canvas de materias, que se reconoce por posición, y la columna de pendientes, que es la única lista accionable. Ninguna de las dos muestra datos de ejecución: los contenedores viven en [Aula](#52-aula), donde hay una acción posible para el docente.

```
Hoy   30 oct 2026              Gestión 2026 · 2026-II (activo) ▾      [Abrir aula]   56 px
──────────────────────────────────────────────────────────────────────────────────────────
 SIS-211 · 10:00–11:30 · Práctica 3       20 de 28 dentro      [Ir al aula]        44 px
──────────────────────────────────────────────────────────────────────────────────────────
┌ Canvas de materias ──────────────────────────────────┐  ┌ Pendientes ──────────┐
│ ┌────────────┐ ┌────────────┐ ┌────────────┐         │  │ Aula ahora           │
│ │ SIS-211    │ │ SIS-420    │ │ SIS-310    │         │  │ SIS-211 · 20 dentro  │
│ │ Estructuras│ │ Desarrollo │ │ Bases de   │         │  │ cierra 11:30         │
│ │ de Datos   │ │ Web        │ │ Datos      │         │  │ [Ir al aula]         │
│ │ 28 insc    │ │ 9 insc     │ │ 12 insc    │         │  │                      │
│ │ 6 faltan   │ │ 15 cola    │ │ al día     │         │  │ Vencimientos         │
│ │ Vence 31/10│ │ Vence 31/10│ │            │         │  │ SIS-420 · Práctica 3 │
│ │ [Calificar]│ │ [Calificar]│ │ [Abrir aula│         │  │ vence 31/10 · 23:59  │
│ └────────────┘ └────────────┘ └────────────┘         │  │                      │
└──────────────────────────────────────────────────────┘  │ Por calificar        │
                                                          │ 21 en 4 prácticas    │
                                                          │ [Calificar]          │
                                                          └──────────────────────┘
```

**Composición.** Canvas a la izquierda con `repeat(auto-fill, minmax(260px, 320px))` y `justify-content: start`: la tarjeta no crece más de 320 px, así que con una o dos materias no se estira hasta ocupar media pantalla. La columna mide 320 px fijos y la separación es de 24 px. Arriba va la franja de sesión, que solo existe con aula abierta; sin franja, el canvas empieza a 88 px del borde superior.

**Tarjeta de materia.** Un solo componente para todas, sin variantes por estado. De arriba hacia abajo: código, nombre, línea de cifras, línea de vencimiento y acciones.

- **Código** en monocromo y con semántica de dato técnico (`<code>`), porque se compara contra el sílabo y contra la pantalla del alumno.
- **Nombre** en hasta dos líneas; si excede, el nombre completo viaja en `title` y la tarjeta no crece. La altura de la tarjeta es uniforme por fila.
- **Cifras**: inscritos, por calificar y sin entregar. Se dibujan solo las que existen: **una cifra en cero no aparece** y `without_submission` nulo se omite, porque significa que la materia no tiene práctica por vencer y no hay conjunto que contar. Sin cola y sin faltantes, la tarjeta dice "Al día" en el verde semántico de éxito de los tokens de estado; con alumnos sin entregar no lo afirma, porque las dos cosas juntas se leen como una contradicción.
- **Vencimiento**: solo con `next_due_at`, como `Vence 31/10 · Práctica 3`, con `dateText:'daymonth'` dentro de un `<time>`.
- **Acciones**: como máximo dos. `Calificar` cuando hay entregas en la cola, hacia `/teacher/por-calificar?subject_id=`; `Abrir aula` hacia `/teacher/aula?subject_id=` cuando la materia no tiene aula abierta y tiene cola que mover; `Ir al aula` cuando la sesión abierta es la suya. Sin cola y sin aula la tarjeta no lleva acción: su acción es abrir la materia, y abrir una clase sin trabajo pendiente queda en la cabecera.
- **Superficie**: el nombre es un enlace estirado sobre la tarjeta (`::after`), de modo que la tarjeta entera abre `/teacher/cursos/:id` con un solo tab stop; las acciones quedan por encima y conservan el suyo.
- **Sin color propio por materia.** Se descarta el acento cromático por curso: seis acentos sin regla semántica no se distinguen entre sí y compiten con el rojo de "se trabó". El canvas se reconoce por código, nombre y posición. Si más adelante hace falta, entra por tokens de estado, no por estilos de la tarjeta.

**Columna de pendientes.** Tres bloques, en orden fijo.

1. `Aula ahora`: solo con sesión abierta. Materia, "N de M dentro", hora de cierre y `[Ir al aula]`. Encabeza la columna porque es lo único que cambia mientras el docente mira.
2. `Vencimientos`: prácticas publicadas vencidas o con vencimiento próximo, ordenadas por urgencia y luego por fecha. Implementa paginación compacta de a 4 ítems y enlace directo a la pestaña de entregas de la materia (`/teacher/cursos/:id?tab=submissions&labId=:exercise_id`). Si el total excede el tope de 50 del backend, declara "y N más".
3. `Por calificar`: entregas en `pending` o `in_review`, prácticas distintas y la más antigua, con `[Calificar]`.

Reglas de la columna: cada bloque se oculta sin contenido; el orden no cambia entre refrescos ni con el cursor encima; sin nada pendiente la columna no se dibuja, el canvas ocupa el ancho y queda una línea de estado bajo la cabecera. La columna no repite cifras del canvas: aquí solo entra lo que tiene una acción a un clic.

**Orden y estabilidad.** El servidor ordena las materias por código ascendente y la vista no reordena por urgencia. Mover tarjetas entre cargas rompe la memoria de posición, que es lo único que un canvas aporta sobre una tabla. Sin arrastrar, sin personalizar y sin ocultar bloques. La urgencia se expresa en la columna, no moviendo el mapa.

**Regla de sustitución de la franja.** Con aula abierta, la franja muestra la sesión con sus presentes. Sin aula, no se dibuja: la cabecera lleva `[Abrir aula]` y el canvas empieza arriba. Una práctica por vencer se anuncia en la columna, no como franja propia.

**Responsive.** Por encima de 1280 px, canvas y columna conviven. Entre 1024 y 1279 px la columna sube sobre el canvas y ocupa el ancho completo con sus bloques en fila. Por debajo de 768 px, una tarjeta por fila y la columna queda primero, porque es lo accionable. El canvas nunca cambia su composición interna.

**Accesibilidad.** El orden del documento es columna y después canvas, porque es el orden de prioridad; la grilla coloca el canvas a la izquierda sin alterar ese orden. Cada bloque de la columna es un `section` con `h2`. Cada tarjeta anuncia su enlace como "SIS-211, Estructuras de Datos, 15 por calificar" para no obligar a recorrer las cifras. Ningún estado se comunica solo por color. La franja de sesión usa `aria-live="polite"` y el refresco no mueve el foco.

**Estados.** Cargando: esqueleto con la forma de tres tarjetas y un bloque, sin salto de diseño. Período sin materias: estado vacío con el selector de período. Sin pendientes: una línea "Nada pendiente en el período" y el canvas a todo el ancho. Error de red: panel con reintento que conserva lo ya cargado.

**Lo que no está en esta pantalla.** Sin ejecución, sin entornos, sin telemetría, sin KPIs, sin banner de alertas, sin grilla reordenable. Ninguna cifra aquí se puede sumar con otra: cada una responde a un hecho distinto.

**Qué dejó la etapa 1.** Construida el 2026-10-30 sobre el resumen que ya existía:

| Cambio | Dónde | Resultado |
| --- | --- | --- |
| `session.ends_at` y `session.subject_code` | read model | La franja dice la materia y a qué hora cierra |
| `due_soon[]`, `due_soon_total` y `due_soon_hidden` | read model | La columna recibe la lista ordenada y el corte ya restado |
| `subjects[].next_due_title` | read model | La tarjeta nombra la práctica que vence |
| `environments` y `subjects[].open_environments` | read model | Retirados: ninguna cifra de contenedores se dibuja en Hoy |
| Pantalla Hoy | `features/teacher/dashboard` | Canvas y columna sobre el resumen; salieron la grilla, el banner y `Personalizar` |
| `subject-card` y `pending-column` | `dashboard/subject-card`, `dashboard/pending-column` | Componentes de presentación, sin estado propio |
| `?subject_id=` en `/teacher/aula` y `/teacher/por-calificar` | frontend | Las acciones abren el destino con la materia ya elegida |

`due_soon_hidden` es la diferencia entre el total y lo entregado, restada por el servidor: la columna dice "y N más" sin contar ni restar filas.

El archivo de trabajo, con su matriz de verificación y lo que quedó fuera, está en [docente-hoy-canvas.md](../../../odd/tasks/docente-hoy-canvas.md).

### 5.2 Aula

Ruta `/teacher/aula`. Pregunta: **¿qué pasa ahora en esta clase?** Se refresca cada 5 segundos mientras haya aula abierta.

**Con aula abierta**

```
SIS-211 · 10:00–11:30 · Práctica 3               20 de 28 dentro        [Cerrar aula]
────────────────────────────────────────────────────────────────────────────────────────
 Todo en orden: 24 de 28 entregaron, 0 trabados.
────────────────────────────────────────────────────────────────────────────────────────
 Buscar alumno  [Todos 28] [Trabados 2] [Sin empezar 3] [Salieron 1] [Entregaron 22]

 Alumno            Estado         Entrega        Intentos  Entorno
 Luis Mamani       Se trabó       10:14 · RE            3   [Consola]
 Ana Quispe        Entregó        10:12 · AC            1   [Consola]
 Jorge Cruz        Sin empezar    —                     0   Sin entorno
```

- La cabecera lleva la materia, la hora de apertura y de fin, la práctica de la sesión, `presentes de inscritos` y `Cerrar aula`.
- **Examen.** Si la práctica de la sesión es un examen, la cabecera lleva la marca «Examen» y el cierre pasa a `[Cerrar examen]`, con un aviso de que los entornos de los alumnos quedan en solo lectura (ADR-005 §5). Cerrado, la pantalla suma el panel **Entornos del examen**: cuántos volúmenes quedaron en solo lectura, el resultado del cierre y la lista de alumnos con entorno, con `[Reabrir]` para la excepción individual —motivo obligatorio, registrado en el historial—. Si algún volumen no se pudo cerrar, `[Reintentar congelamiento]` repite la operación sin reabrir la clase; el servidor también lo reintenta cada 15 minutos.
- El aula se abre con una duración (1 h, 1 h 30, 2 h o 3 h; por defecto 1 h 30) y **se cierra sola al vencer**. El docente puede cerrarla antes; si la clase se alarga, cierra y abre otra. Al vencer, la pantalla deja el modo en vivo en el siguiente refresco y avisa a qué hora se cerró.
- Marcador: sin excepciones de dueño docente escribe "Todo en orden: N de M entregaron, 0 trabados" en una línea, con `aria-live="polite"`. Con excepciones, cada una es un chip con su cuenta: trabados, sin entregar, sin empezar, salieron.
- Filtros: `Todos`, `Trabados`, `Sin empezar`, `Salieron`, `Entregaron`, `Trabajando`, más búsqueda por nombre o correo.
- Orden que llega del servidor: trabados, sin empezar, salieron, entregaron, trabajando; alfabético dentro de cada grupo. El cliente no reordena, no suma y no recorta.
- Color: un solo punto por fila según estado (rojo trabado, ámbar trabajando, verde entregó, neutro sin empezar o salió). Es el único color de la pantalla.
- La telemetría se muestra debajo, acotada a los entornos de la materia de la sesión: una clase por vez.
- El botón de la fila abre la consola de sombra sobre el entorno del alumno.

**Sin aula abierta**

La misma pantalla muestra el formulario de apertura (materia, práctica publicada opcional y duración) y, si hay entornos encendidos, la telemetría de todos ellos. Sin aula no hay lista de faltantes: no existe el conjunto contra el cual contar. "Abrir aula" también aparece en la cabecera de [Hoy](#51-hoy).

**Pendiente de esta pantalla**

Modo proyección, agrupación de excepciones por causa ("2 alumnos con el mismo error"), vista del código sin entregar con consentimiento auditado y actualización por SSE con marca de frescura. Están especificados abajo y en el [ADR-056](../../ARQUITECTURA/ADR/ADR-056-sesion-de-aula-y-presencia-acotada.md), y no forman parte de lo entregado en la etapa 2.

### 5.3 Por calificar

Ruta `/teacher/por-calificar`. Pregunta: **¿qué trabajo manual me falta?** Sin refresco automático: la cola cambia cuando el docente actúa.

Lista de prácticas a la izquierda, entregas de la práctica seleccionada a la derecha, de la más antigua a la más nueva. La acción abre el SpeedGrader existente. Se conserva lo especificado en [POR_CALIFICAR.md](POR_CALIFICAR.md), que ya respeta el estado de revisión declarado por el [ADR-053](../../ARQUITECTURA/ADR/ADR-053-revision-docente-y-libreta-de-notas.md).

### 5.4 Materia

Ruta `/teacher/cursos/:id`. Pregunta: **¿qué contiene y cómo va?** Cuatro pestañas:

| Pestaña | Contenido | Origen |
| --- | --- | --- |
| Prácticas | Módulos como secciones y prácticas como filas con estado y fecha | Existente |
| Entregas | La misma cola de §5.3, filtrada por la materia y en modo lectura | `GET /teacher/reviews?subject_id=` |
| Alumnos | Matriculados, último acceso y prácticas sin entregar | Nuevo |
| Libreta | Tabla de alumnos por prácticas, con `—` en lo no corregido | Implementada en la pestaña de la materia (`course-gradebook`) sobre `GET /teacher/courses/{id}/grades/matrix` |

### 5.5 Qué se retira

- La franja de KPIs y el banner de alertas del panel.
- La grilla de widgets reordenables, el botón "Personalizar Panel" y su almacenamiento de diseño.
- La telemetría como widget: pasa a ser la pantalla de §5.2.
- La tabla de materias del panel: el canvas de tarjetas la sustituye ([ADR-057](../../ARQUITECTURA/ADR/ADR-057-hoy-canvas-de-materias-y-columna-de-pendientes.md)).
- La cifra de entornos abiertos en el panel: el docente no opera contenedores y no tiene acción sobre ellos.
- El indicador "En riesgo" en todo el área docente.
- `/teacher/evaluaciones` completa: su planilla se mudó a la pestaña Libreta de la materia y la ruta redirige a Por calificar.

---

## 6. Contratos de lectura

Un read model por pregunta. Ninguno devuelve listas crudas para que el cliente las agregue.

| Endpoint | Pregunta | Refresco | Estado |
| --- | --- | --- | --- |
| `GET /api/v1/teacher/overview?period_id=` | ¿qué me espera hoy? | 60 s | Implementado |
| `GET /api/v1/teacher/sessions/open` | ¿tengo un aula abierta? | 5 s | Implementado |
| `GET /api/v1/teacher/sessions/{id}/board` | ¿qué pasa ahora en esta clase? | 5 s | Implementado |
| `POST /api/v1/teacher/sessions` (`subject_id`, `exercise_id?`, `duration_minutes?`) · `PATCH /api/v1/teacher/sessions/{id}/close` | abrir y cerrar aula | por acción | Implementado |
| `GET /api/v1/student/sessions/active` · `POST /api/v1/student/sessions/{id}/heartbeat` | ¿hay clase y estoy en ella? | 60 s | Implementado |
| `GET /api/v1/teacher/reviews?subject_id=&exercise_id=` | ¿qué me falta corregir? | manual | Existente |
| `GET /api/v1/teacher/courses/{id}/submissions` | la cola filtrada por materia | manual | Existente |

### 6.1 Contrato del resumen

```json
{
  "period": { "id": "uuid", "name": "Gestión 2026", "code": "2026-II", "is_active": true },
  "session": {
    "id": "uuid", "subject_id": "uuid", "subject_code": "SIS-211",
    "subject_name": "Estructuras de Datos", "exercise_title": "Práctica 3",
    "opens_at": "2026-10-30T10:00:00Z", "ends_at": "2026-10-30T11:30:00Z",
    "closes_at": null, "enrolled": 28, "present": 20
  },
  "queue": { "pending": 21, "practices": 4, "oldest_submitted_at": "2026-10-28T10:00:00Z" },
  "due_soon": [
    {
      "subject_id": "uuid", "subject_code": "SIS-420", "subject_name": "Desarrollo Web Fullstack",
      "exercise_id": "uuid", "exercise_title": "Práctica 3", "due_at": "2026-10-31T23:59:00Z"
    }
  ],
  "due_soon_total": 6,
  "due_soon_hidden": 1,
  "subjects": [
    {
      "id": "uuid", "code": "SIS-420", "name": "Desarrollo Web Fullstack",
      "enrolled": 9, "pending_review": 15, "without_submission": 4,
      "next_due_at": "2026-10-31T23:59:00Z", "next_due_title": "Práctica 3"
    }
  ],
  "signals": []
}
```

`session` es `null` cuando no hay aula abierta; el resumen toma la sesión del mismo almacén que la pantalla de aula, así que la franja y el aula no pueden discrepar. `due_soon` viene ordenado y limitado por el servidor, `due_soon_total` declara el corte y `due_soon_hidden` cuántos quedaron fuera, ya restado: la columna dice "y N más" sin contar nada. `subjects` viene ordenado por código; la vista no reordena. `signals` sigue vacío: la agrupación de excepciones por causa no está implementada.

**Diferencia con el contrato en uso.** Lo entregado en la etapa 1 devuelve `next_due` en singular, `environments.open_now`, `subjects[].open_environments` y una sesión sin `ends_at`. Este contrato los sustituye: `due_soon` con su total, `next_due_title` por materia, la hora de fin y el código de la materia en la sesión, y ninguna cifra de contenedores. El motivo está en el [ADR-057](../../ARQUITECTURA/ADR/ADR-057-hoy-canvas-de-materias-y-columna-de-pendientes.md).

### 6.1.1 Contrato del tablero

```json
{
  "session": {
    "id": "uuid", "subject_id": "uuid", "subject_code": "SIS-211",
    "subject_name": "Estructuras de Datos", "exercise_title": "Práctica 3",
    "status": "open", "opens_at": "2026-10-30T10:00:00Z",
    "ends_at": "2026-10-30T11:30:00Z", "closes_at": null,
    "enrolled": 28, "present": 20
  },
  "summary": {
    "enrolled": 28, "present": 20, "not_started": 3, "working": 2,
    "submitted": 22, "stuck": 1, "absent": 1, "without_submission": 6
  },
  "students": [
    {
      "student_id": "uuid", "name": "Luis Mamani", "email": "luis@dominio",
      "state": "trabado", "present": true,
      "joined_at": "2026-10-30T10:02:00Z", "last_seen_at": "2026-10-30T10:14:40Z",
      "environment_id": "uuid", "environment_status": "running",
      "submitted_at": "2026-10-30T10:14:00Z", "verdict": "RE",
      "review_status": "pending", "attempts": 3
    }
  ]
}
```

`state` sale del servidor en el orden en que se muestra: `trabado`, `sin_empezar`, `ausente`, `entregado`, `trabajando`. `summary` se calcula sobre la misma lista, así que la cabecera y las filas no pueden discrepar.

### 6.2 Definición de cada cifra

| Cifra | Definición |
| --- | --- |
| Inscritos | Alumnos matriculados en la materia |
| Vence pronto | Prácticas publicadas de las materias del docente con vencimiento futuro, sin ventana de fechas. La columna muestra hasta cinco y declara el resto con `due_soon_hidden`. |
| Presentes (aula) | Alumnos con `last_seen_at` dentro de 120 s en `class_session_participants` de esa sesión. La señal la emite el cliente del alumno cada minuto mientras la pestaña está visible. En Hoy es la presencia de la materia con aula abierta, y sin aula no se dibuja: no hay conjunto que contar. |
| Fin del aula | `ends_at`, fijado al abrir con la duración elegida (90 min por defecto, entre 15 y 480). Al vencer, la sesión pasa a `closed` con `closes_at = ends_at` en la siguiente lectura o apertura; no hay proceso periódico. |
| Entregaron | Alumnos con al menos una entrega de la práctica de la sesión desde que se abrió el aula |
| Trabado | Última entrega de la práctica en la sesión con veredicto `RE`, `TLE` o `AST_BLOCKED` en los últimos 10 minutos |
| Sin empezar | Inscritos sin fila en `class_session_participants` para esa sesión |
| Salieron | Participantes cuya última señal tiene más de 120 s. No es lo mismo que no haber entrado nunca |
| Por calificar | Entregas con `review_status` en `pending` o `in_review` |
| Prácticas con pendientes | Prácticas distintas que tienen al menos una entrega por calificar |
| Sin entregar | Inscritos menos alumnos distintos con al menos una entrega en esa práctica |
| Próximo vencimiento | Fecha futura más cercana entre las prácticas publicadas de la materia |

"En riesgo" no aparece: dependía de una regla de 24 horas que no describe el trabajo del alumno. En su lugar se muestra "Sin entregar", que es un hecho contable.

### 6.3 Rendimiento

El resumen se resuelve en tres consultas agregadas —período, materias y cola—, sin recorrer materias en un bucle. `due_soon` sale de la consulta de materias, no de una cuarta. Con 30 materias, la carga deja de pedir 121 viajes a la base.

---

## 7. La sesión como envoltorio opcional

La clase es sincrónica: el docente y los alumnos están, y el sistema lo sabe. También hay trabajo fuera de clase, y el diseño lo cubre sin duplicar pantallas.

`class_sessions` no es un requisito de la práctica: es una envoltura opcional.

| Situación | Qué muestra la pantalla de aula |
| --- | --- |
| Hay aula abierta | "20 de 28 dentro" y el tablero de faltantes y trabados |
| Hay práctica abierta sin aula | "4 entornos abiertos ahora", sin lista de faltantes |
| Nadie conectado | Nada. No hay un "0 activos" ocupando lugar |

El alumno que se traba fuera de clase no queda sin dueño: al día siguiente aparece en la materia como resumen de la práctica ("3 alumnos se trabaron desde tu última visita"). Es una señal para revisar, no una tarea en la cola. Ese resumen todavía no está implementado.

El modelo y la definición de presencia están en el [ADR-056](../../ARQUITECTURA/ADR/ADR-056-sesion-de-aula-y-presencia-acotada.md).

---

## 8. Navegación

```
HOY          Inicio
DOCENCIA     Aula · Por calificar · Materias
CONTENIDOS   Banco de ejercicios · Plantillas de entorno
SUPERVISIÓN  Control de plagio
CUENTA       Mi inteligencia artificial
```

Cinco grupos y ocho destinos. Desaparece el acceso "Entregas", que apuntaba a una ruta inexistente y devolvía al inicio de sesión. Desaparece "Análisis · Estudiantes", que nunca tuvo pantalla especificada.

La planilla de notas sigue accesible durante la transición a través del destino que hoy la sirve, y se muda a la pestaña Libreta de la materia en la etapa 3.

---

## 9. Estados, densidad y accesibilidad

| Estado | Presentación |
| --- | --- |
| Cargando | `skeleton-loader` con la forma del contenido, sin salto de diseño |
| Sin cola | La fila se oculta; no muestra "0 pendientes" |
| Sin materias en el período | Estado vacío con la acción de cambiar de período |
| Error de red | Panel con el mensaje y botón de reintento, sin borrar lo ya cargado |
| Dato con más de un minuto | Marca de frescura en ámbar, según la Ley 5 de [UX-PRINCIPLES](../UX-PRINCIPLES.md) |

Densidad: filas de 44 px en la pantalla de aula, 40 px en tablas de registro, cabecera de tabla en `var(--text-xs)` en mayúsculas, tarjetas de 260 a 320 px de ancho en el canvas. Sin tarjetas anidadas. Una sola columna lateral admitida: la de pendientes de Hoy, que existe solo con trabajo pendiente y cuyo contenido tiene acción y dueño declarados ([ADR-057](../../ARQUITECTURA/ADR/ADR-057-hoy-canvas-de-materias-y-columna-de-pendientes.md)); cualquier otra necesita su propio ADR. El espacio se gana con acciones, no con simetría.

Anchos: por encima de 1280 px el canvas y la columna conviven; entre 1024 y 1279 px la columna sube sobre el canvas con sus bloques en fila; por debajo de 768 px, una tarjeta por fila y la columna primero. El canvas no cambia su composición interna en ningún ancho.

---

## 10. Fuera de alcance de esta etapa

Chat o videollamada dentro del aula; edición colaborativa del código; analítica histórica comparada; pase de módulo y prerrequisitos; calificación desde móvil; ficha individual del alumno como pantalla propia.

---

## 11. Supuestos documentados

- **Paralelos:** `subjects` no modela secciones; cada paralelo es una materia propia. La sesión es única por materia.
- **Asistencia:** la sesión registra quién entró y cuándo, y ese registro es exportable. No se construye un módulo de pasar lista.
- **Nota de la celda:** la elige el docente, con la mejor entrega como sugerencia, según el ADR-053.
- **Alertas de infraestructura:** pertenecen al panel de incidencias del admin, que ya existe. El docente no administra contenedores.
- **Cierre de excepciones:** toda excepción con dueño docente se cierra al resolverse o al marcarse como revisada; ninguna vuelve a aparecer sin un hecho nuevo.

---

## 12. Etapas y estado

| Etapa | Alcance | Retira | Estado |
| --- | --- | --- | --- |
| 0 | El panel deja de mentir: sin KPIs, sin banner, sin personalización; la cola lee el estado de revisión; el acceso "Entregas" apunta a un destino real | Franja de KPIs, banner, grilla reordenable | Hecho (2026-10-30): la franja de KPIs, el banner de atención y `Personalizar` salieron del panel |
| 1 | Pantalla "Hoy" con el resumen agregado: canvas de materias y columna de pendientes | Tabla de materias, columna de tareas anterior, telemetría como widget, "En riesgo" | Hecho (2026-10-30) |
| 2 | Aula con `class_sessions`, presencia acotada y tablero de faltantes y trabados | Definición doble de "activo" | Hecho (2026-10-30), con pendientes en §12.1 |
| 3 | Materia con sus cuatro pestañas y la libreta; se retira `/teacher/evaluaciones` | Pestaña de entregas del centro de evaluaciones | Hecho (2026-11-11): libreta en la materia y `/teacher/evaluaciones` redirige a Por calificar |

Estado verificado en el árbol al 2026-10-30:

- `GET /api/v1/teacher/overview` resuelve período, materias, cola y vencimientos en cuatro consultas agregadas, sin bucle por materia, y ya no publica cifras de contenedores.
- La pantalla `Hoy` pinta el canvas y la columna desde ese resumen, con refresco de 60 s; la grilla reordenable, el banner de atención y `Personalizar` salieron del panel.
- Los widgets `dashboard-courses-widget` y `dashboard-attention-widget` se retiraron con la grilla; el panel del estudiante conserva su propio uso de `dashboard-layout.service.ts`.
- `GET /api/v1/teacher/attention` se retiró. Ninguna pantalla leía su resultado; `loadDashboardData` lo pedía y lo descartaba. Sus alertas quedan cubiertas por `signals` del resumen de `overview`.

### 12.1 Qué dejó la etapa 2

`class_sessions` con un aula abierta por materia (índice parcial único), hora de fin (`ends_at`) y cierre por vencimiento, participantes con `joined_at`/`last_seen_at`/`left_at` y `environments.session_id`. El docente abre y cierra desde `/teacher/aula`; el alumno consulta sus aulas abiertas cada minuto y late mientras la pestaña está visible; el panel toma la fila de sesión del mismo almacén que el aula. La navegación y los contratos están en §5.2 y §6.

Queda pendiente, y no se declara hecho:

- **Modo proyección** (sidebar oculto, tipografía mayor, avance del grupo con nombres grandes).
- **Agrupación de excepciones por causa** y `signals` en el resumen: hoy cada alumno trabado es una fila y el resumen no trae el bloque.
- **Código sin entregar con consentimiento auditado**: la fila abre la consola de sombra existente, que es otra superficie.
- **Actualización por SSE** con marca de frescura; hoy el refresco es sondeo de 5 s.
- **Resumen de trabados para el trabajo fuera de clase** ("3 alumnos se trabaron desde tu última visita").
- **Exportación de asistencia** en CSV.

Cada etapa se abre como archivo de trabajo en `odd/tasks/` antes de escribir código, con su matriz de verificación y su actualización de documentación.

---

## 13. Componentes

- **Reutilizados:** `empty-state`, `search-bar`, `status-tabs`, `pagination-bar`, `resource-meter`, `skeleton-loader`, `combobox`, `dateText`, `verdict-badge` y `status-badge` en el aula.
- **Nuevos:** `subject-card` (tarjeta del canvas), `pending-column` (columna con sus tres bloques), `student-row` y `exception-row` (aula). La franja de sesión de Hoy se resuelve en su plantilla, sin componente propio.
- **Retirados del área docente:** `dashboard-kpi-widget`, `dashboard-attention-widget`, `dashboard-courses-widget`, la grilla `cdkDropList` del panel y el uso docente de `dashboard-layout.service.ts`, que sigue vivo para el panel del estudiante.
- **Mudado:** `dashboard-telemetry-widget` pasa a `aula/aula-telemetry-widget`, con la terminal de sombra y la pantalla `aula` que las aloja.
- **Nombres:** kebab-case de dominio, sin prefijo `solv-` (`subject-card`, no `solv-subject-card`).
