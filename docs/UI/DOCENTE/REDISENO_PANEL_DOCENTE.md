# Área Docente: Arquitectura de Información y Panel de Trabajo

> **Especificación de arquitectura de información y diseño objetivo**
> **Rol:** Docente (con impacto en Estudiante)
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System / [UX-PRINCIPLES](../UX-PRINCIPLES.md)
> **ADRs:** [055](../../ARQUITECTURA/ADR/ADR-055-panel-docente-read-model-por-pregunta.md), [056](../../ARQUITECTURA/ADR/ADR-056-sesion-de-aula-y-presencia-acotada.md)
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
| Cabecera de la tabla de materias | 32 px | 32 px |
| **Chrome antes del contenido** | **320 px** | **88 px** |

Regla: **dos filas de chrome arriba del pliegue como máximo**, la cabecera y una fila de acción. Nada de banners.

Con 88 px de chrome quedan 552 px para el contenido: la línea de sesión (44 px), dos filas de trabajo (44 px cada una) y alrededor de nueve filas de tabla visibles sin desplazar.

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

```
Hoy   30 oct 2026                       Gestión 2026 · 2026-II (activo) ▾          44 px
────────────────────────────────────────────────────────────────────────────────────────
 Vence  SIS-420 · Práctica 3 · 31/10/2026 23:59                  [Ver práctica >]   44 px
 Por calificar  21 entregas en 4 prácticas · la más antigua       [Calificar >]    44 px
 Entornos  4 abiertos ahora                                    [Ver entornos >]   44 px
────────────────────────────────────────────────────────────────────────────────────────
 Materias (2)                                                                      32 px

 Código    Materia                          Inscritos  Ahora  Por calificar  Sin entregar
 SIS-420   Desarrollo Web Fullstack              9       0         15              —
 SIS-211   Estructuras de Datos y Algoritmos     9       0          6              —
```

Regla de sustitución de la primera fila: si hay aula abierta, muestra la sesión con sus presentes; si no, la práctica abierta más próxima a vencer; si tampoco hay, la fila no se dibuja. La cabecera lleva la acción `[Abrir aula]` cuando no hay sesión abierta; con el aula abierta, la acción es la fila de sesión ("Ir al aula").

Las filas de trabajo se ocultan cuando no tienen contenido. Una pantalla sin cola ni entornos abiertos no muestra ceros.

La tabla nunca se recorta: muestra todas las materias del período, así que no lleva "Ver todas" mientras no exista una pantalla propia de Materias (etapa 3).

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
| Libreta | Tabla de alumnos por prácticas, con `—` en lo no corregido | Existente, migra desde `/teacher/evaluaciones` |

### 5.5 Qué se retira

- La franja de KPIs y el banner de alertas del panel.
- La columna lateral de tareas pendientes.
- La grilla de widgets reordenables, el botón "Personalizar Panel" y su almacenamiento de diseño.
- La telemetría como widget: pasa a ser la pantalla de §5.2.
- Las tarjetas de materia en el panel: quedan solo en la tabla.
- El indicador "En riesgo" en todo el área docente.
- La pestaña de entregas de `/teacher/evaluaciones`; su planilla se muda a la pestaña Libreta de la materia.

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
  "environments": { "open_now": 4 },
  "subjects": [
    {
      "id": "uuid", "code": "SIS-420", "name": "Desarrollo Web Fullstack",
      "enrolled": 9, "present_now": 0, "pending_review": 15,
      "without_submission": 0, "next_due_at": "2026-10-31T23:59:00Z"
    }
  ],
  "signals": []
}
```

`session` es `null` cuando no hay aula abierta; el resumen toma la sesión del mismo almacén que la pantalla de aula, así que la fila y el aula no pueden discrepar. `signals` sigue vacío: la agrupación de excepciones por causa no está implementada.

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
      "workspace_id": "uuid", "workspace_status": "running",
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
| Ahora | Entornos de la materia en estado `running`, `active` o `pending`: la misma condición que usa la vista en vivo. Es una cifra de contenedores, no de personas, y por eso la columna se llama "Ahora" y su encabezado dice qué cuenta. |
| Presentes (aula) | Alumnos con `last_seen_at` dentro de 120 s en `class_session_participants`. La señal la emite el cliente del alumno cada minuto mientras la pestaña está visible. |
| Fin del aula | `ends_at`, fijado al abrir con la duración elegida (90 min por defecto, entre 15 y 480). Al vencer, la sesión pasa a `closed` con `closes_at = ends_at` en la siguiente lectura o apertura; no hay proceso periódico. |
| Entregaron | Alumnos con al menos una entrega de la práctica de la sesión desde que se abrió el aula |
| Trabado | Última entrega de la práctica en la sesión con veredicto `RE`, `TLE` o `AST_BLOCKED` en los últimos 10 minutos |
| Sin empezar | Inscritos sin fila en `class_session_participants` para esa sesión |
| Salieron | Participantes cuya última señal tiene más de 120 s. No es lo mismo que no haber entrado nunca |
| Entornos abiertos | Suma de "Ahora" sobre las materias del docente. El resumen y la tabla no pueden discrepar. |
| Por calificar | Entregas con `review_status` en `pending` o `in_review` |
| Prácticas con pendientes | Prácticas distintas que tienen al menos una entrega por calificar |
| Sin entregar | Inscritos menos alumnos distintos con al menos una entrega en esa práctica |
| Próximo vencimiento | Fecha futura más cercana entre las prácticas publicadas de la materia |

"En riesgo" no aparece: dependía de una regla de 24 horas que no describe el trabajo del alumno. En su lugar se muestra "Sin entregar", que es un hecho contable.

### 6.3 Rendimiento

El resumen se resuelve en dos consultas agregadas, sin recorrer materias en un bucle. Con 30 materias, la carga deja de pedir 121 viajes a la base.

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

Densidad: filas de 44 px en la pantalla de aula, 40 px en tablas de registro, cabecera de tabla en `var(--text-xs)` en mayúsculas. Sin tarjetas anidadas ni columnas laterales permanentes. El espacio se gana con acciones, no con simetría.

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
| 0 | El panel deja de mentir: sin KPIs, sin banner, sin personalización; la cola lee el estado de revisión; el acceso "Entregas" apunta a un destino real | Franja de KPIs, banner, grilla reordenable | Hecho (2026-10-30) |
| 1 | Pantalla "Hoy" con el resumen agregado; la telemetría se muda a `/teacher/aula` | Columna de tareas pendientes, tarjetas de materia, telemetría como widget, "En riesgo" | Hecho (2026-10-30) |
| 2 | Aula con `class_sessions`, presencia acotada y tablero de faltantes y trabados | Definición doble de "activo" | Hecho (2026-10-30), con pendientes en §12.1 |
| 3 | Materia con sus cuatro pestañas y la libreta; se retira `/teacher/evaluaciones` | Pestaña de entregas del centro de evaluaciones | Pendiente |

Lo entregado en las etapas 0 y 1: `GET /api/v1/teacher/overview` resuelve período, fila de trabajo y materias en cuatro consultas agregadas —sin bucle por materia— y la pantalla `Hoy` solo pinta esa respuesta. La telemetría vive en `/teacher/aula`, con destino propio en la navegación, y refresca cada 5 segundos; el panel refresca cada 60.

### 12.1 Qué dejó la etapa 2

`class_sessions` con un aula abierta por materia (índice parcial único), hora de fin (`ends_at`) y cierre por vencimiento, participantes con `joined_at`/`last_seen_at`/`left_at` y `workspaces.session_id`. El docente abre y cierra desde `/teacher/aula`; el alumno consulta sus aulas abiertas cada minuto y late mientras la pestaña está visible; el panel toma la fila de sesión del mismo almacén que el aula. La navegación y los contratos están en §5.2 y §6.

Queda pendiente, y no se declara hecho:

- **Modo proyección** (sidebar oculto, tipografía mayor, avance del grupo con nombres grandes).
- **Agrupación de excepciones por causa** y `signals` en el resumen: hoy cada alumno trabado es una fila y el resumen no trae el bloque.
- **Código sin entregar con consentimiento auditado**: la fila abre la consola de sombra existente, que es otra superficie.
- **Actualización por SSE** con marca de frescura; hoy el refresco es sondeo de 5 s.
- **Resumen de trabados para el trabajo fuera de clase** ("3 alumnos se trabaron desde tu última visita").
- **Exportación de asistencia** en CSV.

`GET /api/v1/teacher/attention` sigue en pie, sin consumidor en el área docente; se retira cuando `signals` tome el relevo de las excepciones.

Cada etapa se abre como archivo de trabajo en `odd/tasks/` antes de escribir código, con su matriz de verificación y su actualización de documentación.

---

## 13. Componentes

- **Reutilizados:** `status-badge`, `verdict-badge`, `empty-state`, `search-bar`, `status-tabs`, `pagination-bar`, `resource-meter`, `skeleton-loader`, `combobox`, `dateText`.
- **Nuevos:** `session-header`, `student-row`, `exception-row`. La tabla de materias se resuelve dentro de la pantalla `Hoy`, porque tiene un solo consumidor.
- **Retirados del área docente:** `dashboard-kpi-widget`, `dashboard-attention-widget`, `dashboard-courses-widget` en su modo tarjeta, la grilla `cdkDropList` del panel y el uso docente del servicio de diseño de panel, que sigue vivo para el panel del estudiante.
- **Mudado:** `dashboard-telemetry-widget` pasa a `aula/aula-telemetry-widget`, con la terminal de sombra y la pantalla `aula` que las aloja.
