# Rediseño del Área Docente: de Panel a Consola de Clases

> **Especificación de arquitectura de información y diseño objetivo**
> **Rol:** Docente (con impacto en Estudiante)
> **Gobernanza:** SDD / Docs-as-Code / SOLV Design System / [UX-PRINCIPLES](../UX-PRINCIPLES.md)
> **Estado:** objetivo aprobado, implementación por etapas
> **Sustituye progresivamente a:** [DASHBOARD.md](DASHBOARD.md) (se actualiza pantalla por pantalla a medida que cada etapa cierra)

---

## 1. Problema

El panel actual es un agregado de entregas por prácticas por materias. No existe en el sistema nada que signifique *dar clase*, así que la pantalla solo puede ser un collage:

1. **No hay tiempo.** `due_date` es un vencimiento, no una sesión. Lo "en vivo" se aproxima con heartbeats de todos los workspaces del docente, sin acotar. Por eso dos widgets dan cifras distintas de "activos" sobre el mismo hecho.
2. **No hay estado de revisión.** `score` y `graded_by` son nulos hasta que alguien corrige, y la matriz de notas resuelve con máximo y 0 por defecto: una entrega sin corregir **vale 0 en la libreta**.
3. **No hay señal del grupo.** Se guardan resultados por caso de prueba, complejidad, AST y keystrokes por entrega, y ningún read model los agrega.
4. **La interfaz no cumple sus propias leyes.** El panel separa estado académico y técnico (Ley 1), muestra jerga de infraestructura sin acción (Ley 2), expone el feed crudo (Ley 3) y solo comunica frescura en la telemetría (Ley 5).

El modelo ya es más rico que la interfaz: `purpose`, `module_id`, `difficulty`, `tags`, `expected_complexity`, `environment_type` y `rubrics` existen y ninguna vista los usa. El rediseño no inventa datos: los muestra.

---

## 2. Reglas de diseño

### 2.1 Dos vocabularios, una fuente

El código del juez (`AC`, `WA`, `TLE`, ...) es valor técnico interno; nunca se muestra ni se escribe en templates. Lo visible es el resultado pedagógico, definido una sola vez en `OUTCOME_COPY` ([UX-PRINCIPLES §3](../UX-PRINCIPLES.md)). La etiqueta nombra la consecuencia; el detalle explica sin jerga.

### 2.2 Toda excepción tiene dueño

Solo las excepciones con dueño docente entran a la cola del docente. Esto elimina la pregunta "¿esto va en el panel?".

| Excepción | Dueño | Dónde vive |
|---|---|---|
| OOM, contenedor caído, huérfano | Admin | Panel de incidencias |
| Regla de código bloqueada durante la sesión | Docente | Aula, como "se trabó" |
| Regla de código bloqueada fuera de sesión | Sistema | Bitácora, sin tarea |
| Errores de compilación repetidos | Docente | Aula, como alumno trabado |
| Entrega esperando calificación | Docente | Cola de calificación |
| Entrega sin corregir en la libreta | Docente | Libreta, como pendiente |

### 2.3 Una pantalla, una pregunta

| Pantalla | Pregunta | Refresco |
|---|---|---|
| Hoy | ¿qué clase tengo y qué me espera? | 60 s |
| Aula | ¿qué pasa ahora en esta práctica? | 3–5 s |
| Por calificar | ¿qué trabajo manual me falta? | manual |
| Notas | ¿cómo va el curso? | manual |
| Materia | ¿qué contiene y cómo va? | manual |

### 2.4 La densidad la decide la tarea

- **Consola** (aula): lista maestra + detalle. Filas de 44 px, un solo punto de color por fila, sin tarjetas anidadas.
- **Registro** (notas, prácticas): tabla con columnas fijas y cabecera `var(--text-xs)` en mayúsculas.
- **Prohibido:** grid de widgets reordenables, bloques de KPIs como decoración, columnas laterales permanentes de acciones.

El espacio se gana con acciones, no con simetría.

---

## 3. Pantallas objetivo

### 3.1 Hoy

```
Hoy                                        Jueves 30 de octubre
────────────────────────────────────────────────────────────────
 10:00  Programación I · Práctica 3        [ Abrir aula ]
        En 25 minutos · 28 inscritos

 Por calificar                            12 entregas · 2 prácticas  ›
────────────────────────────────────────────────────────────────
```

Regla de sustitución: si hay sesión hoy, se muestra; si no, la cola sube; si tampoco hay cola, un estado vacío con la acción de abrir aula. Nunca una pantalla vacía.

### 3.2 Aula

```
Práctica 3 · Lista enlazada      ● En vivo · 42 min restantes  [Cerrar aula]
────────────────────────────────────────────────────────────────────────
 Buscar alumno   [Todos ▾] [Trabados 4] [Sin empezar 3] [Sin entregar 9]
────────────────────────┬───────────────────────────────────────────────
 ● Ana Quispe      entregó │  Ana Quispe            [Ver código] [Consigna]
 ● Luis Mamani     trabado │  No compila · 3 intentos en 6 min
 ○ Rosa Flores     empezó  │  línea 12: índice fuera de rango
 ○ Jorge Cruz      sin empezar
                           │  ── 4 alumnos con el mismo error ──
                           │  [ Ver los 4 ]   [ Consigna al grupo ]
```

- Color: gris sin empezar, ámbar empezó sin entregar, verde entregó, rojo trabado. Es el único color de la pantalla.
- Orden: excepciones del docente primero (trabados, sin empezar), luego entregaron, luego el resto; alfabético dentro de cada grupo.
- Estado silencioso obligatorio: "Todo en orden: 24 de 28 entregaron, 0 trabados".
- Presencia: participante de la sesión + heartbeat acotado a la sesión. Un entorno abierto fuera de sesión no aparece aquí, aparece en Materia.
- Acciones: abrir/cerrar, ver la entrega, dejar consigna al alumno o al grupo, ampliar memoria o reiniciar el entorno (solo modalidad IDE). Ver el código en vivo exige consentimiento del alumno y queda auditado.
- Accesibilidad: navegación con flechas, Enter para abrir el detalle, foco visible, `aria-live` en el contador de excepciones.

### 3.3 Por calificar

Lista agrupada por práctica ("7 de 12") y panel de trabajo a la derecha: código, criterios de rúbrica y, solo a demanda, el detalle técnico. Acción principal "Guardar y siguiente".

### 3.4 Notas

Tabla con alumno fijo y prácticas en columnas. Sin corregir se muestra `—`, nunca 0. La celda indica qué entrega cuenta y permite cambiarla con justificación auditada. Al hacer clic se abre un panel lateral, sin perder la tabla.

### 3.5 Materia

Módulos como secciones y prácticas como filas con estado y fecha. Sin barra de avance ni prerrequisitos hasta que exista el cálculo de pase.

### 3.6 Señales

Dentro de la práctica, no como destino de navegación: distribución de resultados por caso de prueba y patrones de error repetidos.

---

## 4. Navegación

```
HOY        Inicio
DOCENCIA   Mis materias · Por calificar · Notas
CONTENIDOS Prácticas · Banco de ejercicios · Plantillas
ANÁLISIS   Estudiantes
CUENTA     Configuración
```

Cuatro destinos de trabajo. "Señales" no es uno: vive dentro de la práctica.

---

## 5. Condiciones reales de uso

- Objetivo de pantalla: portátil de 1366×768 o proyector; ~1100×640 útiles con el sidebar colapsado.
- Sin scroll horizontal en el aula; cabecera pegajosa de 56 px; acciones en la fila.
- **Modo proyección:** oculta el sidebar, sube la tipografía y muestra solo el avance del grupo con nombres grandes.
- Red intermitente: no se borra la lista; se muestra "última actualización hace 12 s" en ámbar (Ley 5).

---

## 6. Vocabulario visible

- Nada de "AC", "WA", "TLE", "cgroup", "OOM", "contenedor", "workspace" en la capa 1.
- Cada etiqueta técnica lleva su acción al lado ("Se quedó sin memoria · Ampliar a 512 MB").
- Botones con verbo y cantidad ("Calificar 12 entregas", "Abrir aula"), nunca "Gestión de...".
- Siglas solo si están expandidas en el mismo lugar.

---

## 7. Fuera de alcance de esta etapa

Chat o videollamada en el aula; edición colaborativa del código; analítica histórica comparada; pase de módulo y prerrequisitos; calificación desde móvil.

---

## 8. Supuestos documentados

- **Paralelos:** `subjects` no modela secciones; cada paralelo es una materia propia. La sesión es única por materia.
- **Asistencia:** la sesión registra quién entró y cuándo, y ese registro es exportable. No se construye un módulo de toma de asistencia ni una pantalla de pasar lista.
- **Nota de la celda:** la elige el docente, con la mejor entrega como sugerencia.

---

## 9. Componentes

Se reutilizan `status-badge`, `verdict-badge`, `empty-state`, `search-bar`, `status-tabs`, `pagination-bar`, `help-drawer`, `resource-meter` y `skeleton`.

Nacen tres: `session-header`, `student-row` y `exception-row`. Ninguno se puede resolver con lo existente.

Se retiran del área docente: el grid reordenable, el widget de KPIs, el widget de atención y la telemetría global.
