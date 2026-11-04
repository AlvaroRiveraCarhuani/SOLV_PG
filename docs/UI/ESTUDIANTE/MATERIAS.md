# Vistas 2 y 3: Mis materias y ficha de materia

> **Especificación de interfaz y contratos**
> **Rol:** Estudiante
> **Rutas:** `/student/courses` y `/student/courses/:subjectId`
> **Gobernanza:** SOLV Design System · ADR-059 (entorno por práctica)

---

## 1. Modelo mental

**Mis materias** es el índice: qué cursás, cuánto avanzaste y qué se entrega pronto.
**La ficha de materia** es el cuaderno: el mapa curricular del docente, las prácticas
que podés resolver y el refuerzo opcional.

La materia agrupa; la práctica es la unidad de trabajo. El entorno pertenece a la
práctica (ADR-059), así que ninguna de las dos vistas abre contenedores: enlazan a
`/student/exercises/:exerciseId`.

Los totales del semestre viven en el inicio (`/student`). Esta vista no los repite:
una tarjeta de KPI arriba de un listado que ya muestra lo mismo sólo agrega ruido.

---

## 2. Mis materias (`/student/courses`)

### 2.1. Encabezado

- `Mis materias` como título de página y una línea de subtítulo que explica la
  acción: entrar a una materia para ver prácticas, avance y entregas.
- Sin banner informativo y sin tarjetas de KPI.

### 2.2. Barra de filtros

Sólo aparece cuando hay materias inscritas:

- `status-tabs` con tres pestañas y su conteo: **Todas**, **En curso**, **Completadas**.
  Completada = `progress_percent = 100` con `total_exercises > 0`.
- `search-bar` por nombre o código, sin distinguir mayúsculas.

### 2.3. Fila de materia

Una fila por materia, con borde izquierdo del color asignado a la materia:

| Elemento | Contenido |
| :--- | :--- |
| Código | `machineData`, con el punto de color; el botón abre el selector de color |
| Nombre | Título de la fila |
| Próxima entrega | Etiqueta `Entrega`, título de la práctica y fecha con `dateText:'daymonth'` en un `<time>`; sin entregas, `Sin entregas próximas` |
| Avance | `completadas` de `total` prácticas y porcentaje, ambos con `machineData`, sobre una barra con `role="progressbar"` |
| Acción | `Ver prácticas` hacia `/student/courses/:subjectId` |

El color de la materia se guarda en `CourseColorService` y se expone como
`--course-accent`, `--course-accent-bg` y `--course-accent-border`.

### 2.4. Estados

- **Cargando:** `skeleton-loader` de lista.
- **Sin materias inscritas:** estado vacío con título y explicación de la matrícula.
  No se dibuja una lista vacía ni un mensaje de filtro.
- **Sin coincidencias:** estado vacío propio con `Quitar filtros`, que limpia la
  búsqueda y vuelve a la pestaña **Todas**.

---

## 3. Ficha de materia (`/student/courses/:subjectId`)

### 3.1. Encabezado

- `Mis materias` como enlace de vuelta.
- Nombre de la materia, código en `machineData` y subtítulo con el avance real:
  `N de M prácticas completadas · X% del curso`. Sin prácticas publicadas, el
  subtítulo lo dice en lugar de mostrar ceros.
- `Actualizar` es la única acción de refresco: incrementa un token que relee el
  mapa curricular y sus prácticas. No hay un botón de actualizar por panel.

### 3.2. Columna principal

Dos paneles independientes, en este orden:

1. **Mapa curricular** — sólo si el docente definió módulos. Muestra cuántos
   módulos están completados y el porcentaje, una barra de avance y una tarjeta
   por módulo con: orden, título, estado (`Completado`, `En curso`, `Disponible`,
   `Bloqueado`), descripción, puntaje de aprobación, prácticas superadas y, si
   está bloqueado, el motivo del bloqueo con el candado.
2. **Prácticas de la materia** o **Ejercicios generales** — siempre que existan:
   - Sin módulos, el panel se titula **Prácticas de la materia** y aclara que el
     docente no definió módulos: todas las prácticas están disponibles.
   - Con módulos, las prácticas sueltas aparecen como **Ejercicios generales**,
     fuera del mapa y siempre disponibles.

Cada práctica muestra título, insignia `Examen` cuando corresponde, dificultad
(Fácil / Medio / Difícil), el mejor puntaje (`Sin intentos` si no hay entregas) y
`Resolver` hacia el solver. Una práctica no entregable dice `Bloqueado` con su
candado, sin botón deshabilitado.

Estados: cargando con esqueleto pulsante; error con `Reintentar`; materia sin
prácticas publicadas con un estado vacío propio.

### 3.3. Columna lateral: práctica de refuerzo

- Panel colapsable, marcado como **Opcional**; no afecta notas ni desbloqueos.
- Sin datos suficientes, una línea explica que hace falta completar ejercicios.
  Con datos, muestra las áreas de enfoque, las prácticas sugeridas con su motivo y
  el enlace `Practicar`.
- El estado vacío es una línea con icono, no una tarjeta anidada.

---

## 4. Contratos

| Pedido | Uso |
| :--- | :--- |
| `GET /api/v1/student/dashboard` | Materias con totales y avance; nombre y código en la ficha |
| `GET /api/v1/student/assignments/due` | Próxima entrega por materia en el índice |
| `GET /api/v1/student/courses/{id}/map` | Módulos, prerrequisitos y prácticas generales |
| `GET /api/v1/student/courses/{id}/recommendations` | Refuerzo opcional |

Un fallo del mapa no borra la identidad de la materia: el encabezado se dibuja con
el resumen del dashboard y el panel ofrece reintentar.

---

## 5. Reglas de interfaz

1. **Un solo camino a la práctica.** Ninguna vista abre entornos por materia.
2. **Fechas.** Sólo `dateText` dentro de `<time>`.
3. **Datos técnicos.** Códigos, conteos y porcentajes con `machineData`; la prosa
   y las etiquetas quedan fuera del nodo mono.
4. **Tokens.** Superficies y textos con `--bg-surface*`, `--text-*`, `--border-*`;
   primario con `--tenant-primary*`; estados con `--status-*` y `--color-*`. No se
   usan tokens inexistentes (`--tenant-text-*`, `--tenant-bg-surface*`,
   `--text-main`, `--border-color`): resolvían a valores heredados y dejaban la
   ficha sin contraste.
5. **Espaciado.** El cascarón ya aplica `padding: var(--space-6)`; la ficha no
   agrega padding propio.
6. **Movimiento.** Sin `@keyframes` locales: los esqueletos usan
   `animate-pulse-subtle` y las transiciones `--transition-fast` /
   `--transition-normal`.
7. **Encabezados.** Todas las vistas del alumno usan `.page-header`, `.page-title`
   y `.page-subtitle`. Los banners informativos y las tarjetas de KPI se retiraron
   de **Mis materias**, **Evaluaciones y prácticas** e **Historial de entregas**;
   los conteos por estado quedan en las pestañas del filtro.

---

## 6. Vistas hermanas ajustadas

- **Evaluaciones y prácticas** (`/student/evaluations`): encabezado con subtítulo,
  sin banner ni KPI; pestañas con conteo y filtros de materia, tipo y búsqueda.
- **Historial de entregas** (`/student/history`): encabezado con subtítulo y sin
  banner; tabla y modal de código sin cambios.

---

## 7. Verificación

- `student-courses.component.spec.ts`: filas por materia, ausencia de KPI, filtros
  por pestaña y búsqueda, enlace a la ficha y fecha semántica de la próxima entrega.
- `student-course-detail.component.spec.ts`: resolución del id y de la materia.
- `curricular-map.component.spec.ts`: progreso por módulos, bloqueo con motivo y
  insignia de examen.
- `student-evaluations.component.spec.ts` y `student-history.component.spec.ts`:
  listados, filtros, veredictos y modal.
- `npm run lint:styles` (gates de estilos, fechas, datos técnicos y movimiento) y
  `ng build --configuration development`.
