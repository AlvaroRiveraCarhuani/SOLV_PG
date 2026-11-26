# SOLV Design System — Especificación de Ingeniería UI
**Versión:** 1.5 (Fuente única de verdad consolidada — Tokens, Arquetipos 1–3, SpeedGrader, Hub Docente, Editor de Prácticas, Aula del Docente y Portal del Estudiante)  
**Regla para el agente implementador:** NO inventar colores, espaciados, alturas ni patrones alternativos. Cualquier valor no especificado aquí debe resolverse usando los tokens de la Sección 1.

---

## Índice de Contenidos

1. [1. Contrato Exacto de Tokens CSS](#1-contrato-exacto-de-tokens-css)
   - [1.1. `frontend/src/styles/_semantic.scss` (Light Neutro + Dark Workspace)](#11-frontendsrcstyles-semanticscss-light-neutro-dark-workspace)
   - [1.2. `frontend/src/styles/_tenant.scss` (Luminosidad Relativa)](#12-frontendsrcstyles-tenantscss-luminosidad-relativa)
2. [2. Geometría Global y Controles Base (`styles.scss`)](#2-geometría-global-y-controles-base-stylesscss)
3. [3. Arquetipo 1 — Catálogo y Filtros (`/admin/plantillas`) — ✅ CERRADO](#3-arquetipo-1-catálogo-y-filtros-adminplantillas-cerrado)
   - [3.1. Cabecera Compacta (sin breadcrumb redundante ni subtítulos gigantes)](#31-cabecera-compacta-sin-breadcrumb-redundante-ni-subtítulos-gigantes)
   - [3.2. Barra de Herramientas Unificada (1 sola fila, reemplaza las 6 filas actuales)](#32-barra-de-herramientas-unificada-1-sola-fila-reemplaza-las-6-filas-actuales)
   - [3.3. Popover Unificado de `Filtros` (especificación exacta)](#33-popover-unificado-de-filtros-especificación-exacta)
   - [3.4. Anatomía de la Tarjeta de Entorno (Grid: `repeat(auto-fill, minmax(340px, 1fr))`, `gap: 14px`)](#34-anatomía-de-la-tarjeta-de-entorno-grid-repeatauto-fill-minmax340px-1fr-gap-14px)
4. [4. Arquetipo 2 — Workspace de Código (`/student/exercises/:exerciseId`) — ✅ CERRADO](#4-arquetipo-2-workspace-de-código-studentexercisesexerciseid-cerrado)
   - [4.1. Estructura General (*Edge-to-Edge*)](#41-estructura-general-edge-to-edge)
   - [4.2. Barra Superior del Ejercicio (`height: 44px`, 1 sola línea obligatoria)](#42-barra-superior-del-ejercicio-height-44px-1-sola-línea-obligatoria)
   - [4.3. Paneles Divisibles (Resizers)](#43-paneles-divisibles-resizers)
   - [4.4. Panel del Juez y Terminología Humanizada (`terminology-gate`)](#44-panel-del-juez-y-terminología-humanizada-terminology-gate)
5. [5. Arquetipo 3 — Cola Maestro-Detalle, Tabla Densa y Dashboard Técnico — ✅ CERRADO](#5-arquetipo-3-cola-maestro-detalle-tabla-densa-y-dashboard-técnico-cerrado)
   - [5A. Cola Maestro-Detalle y Tabla Densa (`/teacher/por-calificar`)](#5a-cola-maestro-detalle-y-tabla-densa-teacherpor-calificar)
   - [5B. Dashboard Técnico de Infraestructura (`/admin/dashboard`)](#5b-dashboard-técnico-de-infraestructura-admindashboard)
6. [6. SpeedGrader — Revisión Docente (Evaluación y Desarrollo) — ✅ CERRADO](#6-speedgrader-revisión-docente-evaluación-y-desarrollo-cerrado)
   - [6.1. Estructura General y Geometría (*Edge-to-Edge*)](#61-estructura-general-y-geometría-edge-to-edge)
   - [6.2. Cabecera de Revisión (`height: 48px`, 1 sola línea obligatoria)](#62-cabecera-de-revisión-height-48px-1-sola-línea-obligatoria)
   - [6.3. Área de Trabajo, Visor y Comentarios Inline (Estilo Azure DevOps)](#63-área-de-trabajo-visor-y-comentarios-inline-estilo-azure-devops)
   - [6.4. Menú `Herramientas` y Utilidades de Inspección](#64-menú-herramientas-y-utilidades-de-inspección)
   - [6.5. Panel Derecho — Entorno de Evaluación (`width: 380px`, `min-width: 320px`, scroll interno)](#65-panel-derecho-entorno-de-evaluación-width-380px-min-width-320px-scroll-interno)
   - [6.6. Panel Derecho — Entorno de Desarrollo (`width: 380px`, `min-width: 320px`, flex column)](#66-panel-derecho-entorno-de-desarrollo-width-380px-min-width-320px-flex-column)
   - [6.7. Modal `Calificar / Ajuste manual` (`modal-shell`, `width: 480px`, atajo `A`)](#67-modal-calificar-ajuste-manual-modal-shell-width-480px-atajo-a)
7. [7. Hub Docente — `Hoy`, `Detalle del Curso` y `Ficha de la Práctica` — ✅ CERRADO](#7-hub-docente-hoy-detalle-del-curso-y-ficha-de-la-práctica-cerrado)
   - [7.1. Inicio del Docente — `Hoy` (`/teacher`)](#71-inicio-del-docente-hoy-teacher)
   - [7.2. Detalle del Curso (`/teacher/cursos/:courseId`)](#72-detalle-del-curso-teachercursoscourseid)
   - [7.3. Ficha de la Práctica (`/teacher/cursos/:courseId/practicas/:labId`)](#73-ficha-de-la-práctica-teachercursoscourseidpracticaslabid)
8. [8. Editor y Creador de Prácticas / Laboratorios — ✅ CERRADO](#8-editor-y-creador-de-prácticas-laboratorios-cerrado)
   - [8.1. Estructura General, Geometría y Regla Sticky](#81-estructura-general-geometría-y-regla-sticky)
   - [8.2. Cabecera Fija de Edición (`height: 48px`, `position: sticky; top: 48px`)](#82-cabecera-fija-de-edición-height-48px-position-sticky-top-48px)
   - [8.3. Columna Izquierda Sticky (`220px`) — `SECCIONES` + `Verificación (Preflight)` Compacto](#83-columna-izquierda-sticky-220px-secciones-verificación-preflight-compacto)
   - [8.4. Columna Derecha (`minmax(0, 1fr)`) — Contexto Fijo y Hoja de Consulta Rápida](#84-columna-derecha-minmax0-1fr-contexto-fijo-y-hoja-de-consulta-rápida)
   - [8.5. Contenido por Modalidad — Entorno de Desarrollo (`IDE_PERSISTENTE`)](#85-contenido-por-modalidad-entorno-de-desarrollo-ide-persistente)
   - [8.6. Contenido por Modalidad — Entorno de Evaluación (`JUEZ_EFIMERO`)](#86-contenido-por-modalidad-entorno-de-evaluación-juez-efimero)
   - [8.7. Contratos de Modales del Editor (`modal-shell`, `max-height: 85vh`)](#87-contratos-de-modales-del-editor-modal-shell-max-height-85vh)
9. [9. Aula del Docente (`/teacher/aula`) — ✅ CERRADO](#9-aula-del-docente-teacheraula-cerrado)
   - [9.1. Estructura General y Reglas de Dominio](#91-estructura-general-y-reglas-de-dominio)
   - [9.2. Estado «Sin aula abierta» (Layout 2 Columnas: `1fr` + `340px`)](#92-estado-sin-aula-abierta-layout-2-columnas-1fr-340px)
   - [9.3. Estado «Aula en vivo» — Matriz Estricta `Entorno × Propósito`](#93-estado-aula-en-vivo-matriz-estricta-entorno-propósito)
   - [9.4. Tablero Adaptativo y Panel Lateral](#94-tablero-adaptativo-y-panel-lateral)
10. [10. Portal del Estudiante (v1.5) — ✅ CERRADO](#10-portal-del-estudiante-v15-cerrado)
   - [10.1. Reglas Generales del Portal Estudiantil](#101-reglas-generales-del-portal-estudiantil)
   - [10.2. Inicio del Estudiante (`/student`)](#102-inicio-del-estudiante-student)
   - [10.3. Mis Materias (`/student/materias`)](#103-mis-materias-studentmaterias)
   - [10.4. Evaluaciones y Prácticas (`/student/evaluaciones` — Radar Transversal por Prioridad)](#104-evaluaciones-y-prácticas-studentevaluaciones-radar-transversal-por-prioridad)
   - [10.5. Detalle de la Materia (`/student/courses/:id` — Hub de Materia en 2 Columnas)](#105-detalle-de-la-materia-studentcoursesid-hub-de-materia-en-2-columnas)
   - [10.6. Resolución — Entorno de Desarrollo (`/student/exercises/:id` · `IDE_PERSISTENTE`)](#106-resolución-entorno-de-desarrollo-studentexercisesid-ide-persistente)
   - [10.7. Resolución — Entorno de Evaluación (`/student/exercises/:id` · `JUEZ_EFIMERO`)](#107-resolución-entorno-de-evaluación-studentexercisesid-juez-efimero)
   - [10.8. Paneles Deslizables Complementarios (`Drawer`)](#108-paneles-deslizables-complementarios-drawer)

---

## 1. Contrato Exacto de Tokens CSS

### 1.1. `frontend/src/styles/_semantic.scss` (Light Neutro + Dark Workspace)
Se reemplaza la escala Slate (gris azulado) por grises neutros (Zinc) para compatibilidad multi-tenant. Se corrigen los tokens huérfanos detectados (`--color-danger*`, `--color-success-*`).

```scss
:root {
  // Superficies (Light Neutro — cero tinte azulado)
  --bg-canvas:             #FAFAFA;
  --bg-surface:            #FFFFFF;
  --bg-surface-subtle:     #F4F4F5;
  --bg-surface-active:     #E4E4E7;

  // Overlays
  --overlay-backdrop:      rgba(9, 9, 11, 0.50);
  --overlay-backdrop-soft: rgba(9, 9, 11, 0.35);

  // Bordes (Border-first)
  --border-subtle:         #E4E4E7;
  --border-strong:         #D4D4D8;
  --border-focus:          var(--tenant-primary);

  // Tipografía
  --text-primary:          #09090B;
  --text-secondary:        #52525B;
  --text-muted:            #71717A;
  --text-tertiary:         #71717A;
  --text-inverse:          #FFFFFF;

  // Semánticos Base + Alias de tokens huérfanos
  --color-success:         #16A34A;
  --color-success-dark:    #15803D;
  --color-success-bg:      #DCFCE7;
  --color-success-subtle:  rgba(22, 163, 74, 0.10);
  --color-warning:         #D97706;
  --color-error:           #DC2626;
  --color-danger:          var(--color-error);
  --color-danger-bg:       #FEE2E2;
  --color-neutral:         #71717A;

  // Estados de Environments (ADR inamovible)
  --status-running-color:      #16A34A;
  --status-running-bg:         #DCFCE7;
  --status-running-border:     #BBF7D0;

  --status-pending-color:      #D97706;
  --status-pending-bg:         #FEF3C7;
  --status-pending-border:     #FDE68A;

  --status-hibernated-color:   #71717A;
  --status-hibernated-bg:      #F4F4F5;
  --status-hibernated-border:  #E4E4E7;

  --status-failed-color:       #DC2626;
  --status-failed-bg:          #FEE2E2;
  --status-failed-border:      #FECACA;

  --status-oom-color:          #DC2626;
  --status-oom-bg:             #FEE2E2;
  --status-oom-border:         #FECACA;

  // Veredictos del Juez (ADR-012 / ADR-037)
  --verdict-ac-color:          #16A34A;
  --verdict-ac-bg:             #DCFCE7;
  --verdict-ac-border:         #86EFAC;

  --verdict-wa-color:          #DC2626;
  --verdict-wa-bg:             #FEE2E2;
  --verdict-wa-border:         #FCA5A5;

  --verdict-tle-color:         #D97706;
  --verdict-tle-bg:            #FEF3C7;
  --verdict-tle-border:        #FCD34D;

  --verdict-re-color:          #EA580C;
  --verdict-re-bg:             #FFEDD5;
  --verdict-re-border:         #FDBA74;

  --verdict-ast-color:         #991B1B;
  --verdict-ast-bg:            #FEE2E2;
  --verdict-ast-border:        #F87171;

  --verdict-ac:  var(--verdict-ac-color);
  --verdict-wa:  var(--verdict-wa-color);
  --verdict-tle: var(--verdict-tle-color);
  --verdict-re:  var(--verdict-re-color);
  --verdict-ast: var(--verdict-ast-color);

  // Sombras discretas
  --shadow-subtle:   0 1px 2px rgba(0, 0, 0, 0.04);
  --shadow-card:     0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-dropdown: 0 8px 24px -4px rgba(0, 0, 0, 0.12), 0 2px 6px -1px rgba(0, 0, 0, 0.06);
}

[data-theme="dark"] {
  // Superficies carbón escalonadas (nunca negro puro uniforme)
  --bg-canvas:             #09090B;
  --bg-surface:            #111113;
  --bg-surface-subtle:     #18181B;
  --bg-surface-active:     #27272A;

  --overlay-backdrop:      rgba(0, 0, 0, 0.70);
  --overlay-backdrop-soft: rgba(0, 0, 0, 0.50);

  --border-subtle:         #27272A;
  --border-strong:         #3F3F46;

  --text-primary:          #FAFAFA;
  --text-secondary:        #A1A1AA;
  --text-muted:            #71717A;
  --text-tertiary:         #71717A;
  --text-inverse:          #09090B;

  --color-success:         #4ADE80;
  --color-success-dark:    #4ADE80;
  --color-success-bg:      rgba(74, 222, 128, 0.12);
  --color-success-subtle:  rgba(74, 222, 128, 0.10);
  --color-warning:         #FBBF24;
  --color-error:           #F87171;
  --color-danger:          var(--color-error);
  --color-danger-bg:       rgba(248, 113, 113, 0.12);
  --color-neutral:         #A1A1AA;

  // Estados en Dark (tintes translúcidos, contraste AA)
  --status-running-color:      #4ADE80;
  --status-running-bg:         rgba(74, 222, 128, 0.12);
  --status-running-border:     rgba(74, 222, 128, 0.28);

  --status-pending-color:      #FBBF24;
  --status-pending-bg:         rgba(251, 191, 36, 0.12);
  --status-pending-border:     rgba(251, 191, 36, 0.28);

  --status-hibernated-color:   #A1A1AA;
  --status-hibernated-bg:      rgba(161, 161, 170, 0.10);
  --status-hibernated-border:  rgba(161, 161, 170, 0.22);

  --status-failed-color:       #F87171;
  --status-failed-bg:          rgba(248, 113, 113, 0.12);
  --status-failed-border:      rgba(248, 113, 113, 0.28);

  --status-oom-color:          #F87171;
  --status-oom-bg:             rgba(248, 113, 113, 0.12);
  --status-oom-border:         rgba(248, 113, 113, 0.28);

  // Veredictos en Dark
  --verdict-ac-color:          #4ADE80;
  --verdict-ac-bg:             rgba(74, 222, 128, 0.12);
  --verdict-ac-border:         rgba(74, 222, 128, 0.28);

  --verdict-wa-color:          #F87171;
  --verdict-wa-bg:             rgba(248, 113, 113, 0.12);
  --verdict-wa-border:         rgba(248, 113, 113, 0.28);

  --verdict-tle-color:         #FBBF24;
  --verdict-tle-bg:            rgba(251, 191, 36, 0.12);
  --verdict-tle-border:        rgba(251, 191, 36, 0.28);

  --verdict-re-color:          #FB923C;
  --verdict-re-bg:             rgba(251, 146, 60, 0.12);
  --verdict-re-border:         rgba(251, 146, 60, 0.28);

  --verdict-ast-color:         #FCA5A5;
  --verdict-ast-bg:            rgba(239, 68, 68, 0.16);
  --verdict-ast-border:        rgba(248, 113, 113, 0.35);

  --shadow-subtle:   0 1px 2px rgba(0, 0, 0, 0.30);
  --shadow-card:     0 1px 3px rgba(0, 0, 0, 0.35);
  --shadow-dropdown: 0 12px 32px -4px rgba(0, 0, 0, 0.65), 0 0 0 1px var(--border-strong);
}

```

### 1.2. `frontend/src/styles/_tenant.scss` (Luminosidad Relativa)

```scss
:root {
  --tenant-primary-h: 221;
  --tenant-primary-s: 83%;
  --tenant-primary-l: 53%;

  --tenant-primary:        hsl(var(--tenant-primary-h), var(--tenant-primary-s), var(--tenant-primary-l));
  --tenant-primary-hover:  hsl(var(--tenant-primary-h), var(--tenant-primary-s), clamp(24%, calc(var(--tenant-primary-l) - 6%), 75%));
  --tenant-primary-active: hsl(var(--tenant-primary-h), var(--tenant-primary-s), clamp(18%, calc(var(--tenant-primary-l) - 12%), 68%));
  --tenant-primary-subtle: hsla(var(--tenant-primary-h), var(--tenant-primary-s), var(--tenant-primary-l), 0.09);
  --tenant-primary-border: hsla(var(--tenant-primary-h), var(--tenant-primary-s), var(--tenant-primary-l), 0.28);
  --tenant-primary-text:   #FFFFFF; // Calculado en TenantService por luminosidad AA (#09090B en colores claros como Ámbar)
}

[data-theme="dark"] {
  --tenant-primary-subtle: hsla(var(--tenant-primary-h), var(--tenant-primary-s), var(--tenant-primary-l), 0.15);
  --tenant-primary-border: hsla(var(--tenant-primary-h), var(--tenant-primary-s), var(--tenant-primary-l), 0.35);
}

```

---

## 2. Geometría Global y Controles Base (`styles.scss`)

| Elemento | Especificación exacta |
| --- | --- |
| **Topbar** | `height: 48px`, `border-bottom: 1px solid var(--border-subtle)`, `background: var(--bg-surface)`. |
| **Sidebar** | Expandido: `width: 224px`. Colapsado (modo rail): `width: 52px`. `border-right: 1px solid var(--border-subtle)`. Ítem activo: `background: var(--tenant-primary-subtle)`, texto e ícono en `var(--tenant-primary)`, `border-radius: 6px`. |
| **Rutas con rail por defecto (`52px`)** | Inician colapsadas en modo rail (`52px`) para maximizar el área de trabajo (expandibles a `224px` bajo demanda, sin perfil duplicado al pie):<br>• `/student` (`Inicio` del Estudiante)<br>• `/student/materias` (`Mis materias`)<br>• `/student/evaluaciones` (`Evaluaciones y prácticas` — radar transversal)<br>• `/student/courses/:id` (`Detalle de la materia`)<br>• `/student/exercises/:exerciseId` (Solver / Workspace de resolución en Desarrollo y Evaluación)<br>• `/teacher` (`Hoy` / Hub Docente)<br>• `/teacher/cursos/:courseId` (`Detalle del Curso`)<br>• `/teacher/cursos/:courseId/practicas/:labId` (`Ficha de la Práctica`)<br>• `/teacher/cursos/:courseId/ejercicios/nuevo` (`Editor de Prácticas` — nuevo)<br>• `/teacher/cursos/:courseId/ejercicios/:id/editar` (`Editor de Prácticas` — edición)<br>• `/teacher/aula` (`Aula del Docente` en vivo)<br>• `/teacher/por-calificar` (`Cola Maestro-Detalle`)<br>• `/teacher/submissions/:id/review` (`SpeedGrader` / Mesa de revisión) |
| **Altura de controles** | Botones, inputs, selects y toggles en barra de herramientas: `height: 32px` (`28px` en barras compactas de IDE/tarjetas). |
| **Radios** | Controles/inputs/botones: `6px`. Tarjetas/popovers/modales: `8px`. Puntos de estado: `9999px` (`6px × 6px`). |
| **Jerarquía de botones** | `.btn-primary`: fondo `var(--tenant-primary)`, texto `var(--tenant-primary-text)`, **MÁXIMO 1 por pantalla**. `.btn-secondary`: fondo `var(--bg-surface)`, borde `1px solid var(--border-subtle)`, hover `var(--bg-surface-subtle)`. `.btn-ghost`: sin borde, hover `var(--bg-surface-active)`. |
| **CodeMirror Diff** | **Prohibido** `text-decoration: underline` en líneas agregadas. Usar `background: rgba(74, 222, 128, 0.12)` + `border-left: 2px solid var(--verdict-ac-color)`. |

---

## 3. Arquetipo 1 — Catálogo y Filtros (`/admin/plantillas`) — ✅ CERRADO

*Aplica a: `/admin/plantillas`, `/admin/tecnologias`, `/student/courses`, `/teacher/exercises`.*

### 3.1. Cabecera Compacta (sin breadcrumb redundante ni subtítulos gigantes)

* **Layout:** Flex horizontal (`justify-content: space-between; align-items: flex-end; padding: 20px 24px 16px`).
* **Izquierda:**
* `h1`: `20px`, `font-weight: 600`, `letter-spacing: -0.02em`, `line-height: 1.2`.
* Línea única de metadatos debajo (`margin-top: 4px`, `12px`, `--text-muted`): descripción breve o conteo en `JetBrains Mono` (`7 activos · 2 rechazadas · RAM prom. 585 MB`).


* **Derecha (`gap: 8px`):** `Preparar entornos` (`.btn-secondary`, `32px`) + `+ Nuevo Entorno` (`.btn-primary`, `32px`).

### 3.2. Barra de Herramientas Unificada (1 sola fila, reemplaza las 6 filas actuales)

* **Contenedor:** `display: flex; justify-content: space-between; align-items: center; padding: 0 24px; height: 44px; border-bottom: 1px solid var(--border-subtle)`.
* **Izquierda (Tabs):** `Activos [7]` y `Rechazadas [2]`. Tab activo con `border-bottom: 2px solid var(--tenant-primary)` y contador en `JetBrains Mono` (`11px`, fondo `--bg-surface-subtle`, `padding: 1px 6px`, `border-radius: 4px`).
* **Derecha (`gap: 8px`):**
1. **Buscador:** `width: 260px`, `height: 32px`, ícono lupa izquierda, badge `/` en mono a la derecha (atajo de teclado que enfoca el input).
2. **Botón `Filtros`:** `.btn-secondary` (`height: 32px`), ícono `SlidersHorizontal` (`14px`) + texto `Filtros`. Si hay filtros activos, muestra contador circular en `--tenant-primary` (ej. `2`).
3. **Toggle vista:** Segmented control (`height: 32px`, `border: 1px solid var(--border-subtle)`, `border-radius: 6px`) con íconos `LayoutGrid` (Tarjetas) y `Table` (Tabla).



### 3.3. Popover Unificado de `Filtros` (especificación exacta)

* **Anclaje:** `position: absolute; right: 0; top: calc(100% + 6px); width: 440px; z-index: 40`.
* **Contenedor:** `background: var(--bg-surface); border: 1px solid var(--border-strong); border-radius: 8px; box-shadow: var(--shadow-dropdown)`.
* **Cabecera (`height: 36px`, `padding: 0 14px`, `border-bottom: 1px solid var(--border-subtle)`):** Izquierda: `FILTROS` (`11px`, `font-weight: 600`, `letter-spacing: 0.06em`, `--text-muted`). Derecha: botón `.btn-ghost` `Limpiar` (`12px`).
* **Cuerpo (`display: grid; grid-template-columns: 1fr 1fr`):**
* **Columna 1 (`MODALIDAD`, `border-right: 1px solid var(--border-subtle)`, `padding: 10px`):** `Todas`, `Entorno de Desarrollo`, `Entorno de Evaluación`, `Desarrollo y Evaluación`.
* **Columna 2 (`TECNOLOGÍA PRINCIPAL`, `padding: 10px`):** `Todas` + lista de tecnologías con su mini-logo a color (`14px`).
* **Fila de opción:** `height: 30px; padding: 0 8px; border-radius: 5px; display: flex; align-items: center; justify-content: space-between; font-size: 12px`. Derecha: contador en `JetBrains Mono` (`11px`, `--text-muted`) + ícono `Check` (`13px`, `--tenant-primary`) solo en la opción activa.


* **Comportamiento:** Cierra con clic fuera o tecla `Escape`. Si hay filtros activos, muestra chips removibles (`×`) discretos en una sub-barra de `32px` solo mientras estén activos.

### 3.4. Anatomía de la Tarjeta de Entorno (Grid: `repeat(auto-fill, minmax(340px, 1fr))`, `gap: 14px`)

* **Contenedor:** `background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 8px; padding: 16px; display: flex; flex-direction: column; gap: 10px`. Hover: `border-color: var(--border-strong)`.
* **Orden exacto de filas:**
1. **Título + Estado:** Izquierda: título (`14px`, `font-weight: 600`, `--text-primary`). Derecha: punto `6px × 6px` + texto `Activa` / `Rechazada` (`12px`, `font-weight: 500`, color `--status-running-color` / `--status-failed-color`, **sin fondo ni borde de píldora**).
2. **Modalidad y Categoría:** Línea única (`11px`, `JetBrains Mono`, `--text-muted`): ícono `12px` + `DESARROLLO` o `EVALUACIÓN` + `·` + nombre de categoría. (**Prohibido** encerrarlo en pill azul/amarilla).
3. **Stack de Tecnologías:** `display: flex; flex-wrap: wrap; gap: 6px 12px`. Cada tecnología = **mini-logo SVG a color real (`15px × 15px`) + texto (`12px`, `--text-secondary`)**, apoyado directo sobre la tarjeta **sin borde ni fondo de píldora**.
4. **Descripción:** `13px`, `--text-secondary`, `line-height: 1.45`, máximo 2 líneas (`-webkit-line-clamp: 2`).
5. **Acordeón `Detalles técnicos`:** Trigger `12px`, `--text-muted`, chevron `13px` que rota `180deg`. Al abrir: bloque `--bg-surface-subtle` (`padding: 8px 10px`, `border-radius: 6px`, `font-family: var(--font-mono)`, `11px`) con imagen Docker y herramientas.
6. **Pie (`border-top: 1px solid var(--border-subtle); padding-top: 10px; margin-top: auto`):**
* Izquierda: `RAM 1024 MB · 05/11/2026` (`11px`, `JetBrains Mono`, `--text-muted`) + estado discreto (`✓ Entorno preparado`, `11px`, `--status-running-color`, **sin pill verde**).
* Derecha: botón `Editar` (`.btn-ghost`, `height: 28px`, `12px`) + botón `⋮` (`28px × 28px`, `.btn-ghost`).




* **Prohibiciones explícitas:** Cero numeración decorativa (`01`, `02`, `03`); cero píldoras de color compitiendo entre sí; cero botones primarios dentro de tarjetas.

---

## 4. Arquetipo 2 — Workspace de Código (`/student/exercises/:exerciseId`) — ✅ CERRADO

*Aplica a: Solver, SpeedGrader, Vista de Similitud, Probar Ejercicio, Detalle de Entrega.*

> **Precedencia normativa para el Solver (`/student/exercises/:exerciseId`):** La presente Sección 4 establece el marco geométrico, de resizers, topbar y contratos de veredictos humanizados compartidos. Las reglas funcionales, de flujo y de interacción definitivas del estudiante en el solver (modelo de 2 primitivas `Probar código` vs. `Enviar a evaluación` / `Entregar examen`, consola inferior plegable de 2 pestañas y rúbrica visible en web para Entorno de Desarrollo) se rigen con **precedencia estricta por la Sección 10 (v1.5)**.

### 4.1. Estructura General (*Edge-to-Edge*)

* **Contenedor:** `height: calc(100vh - 48px); padding: 0; margin: 0; overflow: hidden`. **Cero tarjetas flotantes** alrededor del enunciado, editor o juez.
* **Sidebar:** Inicia colapsado en **modo rail (`52px`)** para maximizar el ancho del código.

### 4.2. Barra Superior del Ejercicio (`height: 44px`, 1 sola línea obligatoria)

* **Regla anti-wrap:** Todos los hijos tienen `white-space: nowrap; flex-shrink: 0` (excepto el título, que lleva `min-width: 0; overflow: hidden; text-overflow: ellipsis`). **Nunca se parte en 2 líneas.**
* **Izquierda (`gap: 10px`):** `← Volver` (`.btn-ghost`, `height: 28px`) + divisor `1px × 16px` + Título (`14px`, `600`) + metadatos planos (`Medio · Límite 2 s`, `12px`, `JetBrains Mono`, `--text-muted`).
* **Derecha (`gap: 8px`):** Selector de lenguaje (`height: 28px`, `12px`) + `Reiniciar` (`.btn-secondary`, `height: 28px`) + `Enviar a Evaluación` (`.btn-primary`, `height: 28px`, único botón primario de la vista).

### 4.3. Paneles Divisibles (Resizers)

* **Split Horizontal:** Panel Izquierdo (Enunciado, default `38%`, `min-width: 280px`) | **Resizer Vertical** (`width: 4px`, línea central visible `1px var(--border-subtle)`, hover/drag `var(--tenant-primary)`) | Columna Derecha (`flex: 1`).
* **Split Vertical (Columna Derecha):**
* **Editor CodeMirror 6 (`flex: 1`, `min-height: 220px`):** Barra de archivo superior (`height: 34px`, `border-bottom: 1px solid var(--border-subtle)`) con `solucion.py` a la izquierda y `UTF-8 · Ln X, Col Y` en mono a la derecha. Fondo `var(--bg-surface)`.
* **Resizer Horizontal** (`height: 4px`, misma regla de color).
* **Panel de Resultados del Juez (default `240px`, colapsable a `36px`):** Cabecera fija `36px` con título `Resultados` y botón colapsar.



### 4.4. Panel del Juez y Terminología Humanizada (`terminology-gate`)

* **Prohibición estricta:** Nunca mostrar siglas técnicas (`AC`, `WA`, `TLE`, `RE`, `AST`) en la UI del estudiante.
* **Tabla de mapeo obligatoria:**

| Código interno | Etiqueta visible | Subtítulo en lenguaje natural | Token de color |
| --- | --- | --- | --- |
| `AC` | **Cumple** | Su solución produjo la salida esperada en todos los casos. | `--verdict-ac-*` |
| `WA` | **Con diferencias** | La salida obtenida no coincide con la esperada en uno o más casos. | `--verdict-wa-*` |
| `TLE` | **Tiempo excedido** | Su solución superó el tiempo límite permitido para esta práctica. | `--verdict-tle-*` |
| `RE` | **Error de ejecución** | El programa se detuvo inesperadamente durante la prueba. | `--verdict-re-*` |
| `CE` | **Error de compilación** | El código presentó fallos de compilación o sintaxis durante la prueba. | `--color-error` / `--verdict-wa-*` |
| `PENDING` | **Pendiente de revisión** | La entrega fue registrada y aguarda calificación o revisión docente. | `--status-pending-*` |
| `AST` | **No evaluado** | Se detectó una instrucción no permitida por las reglas de la práctica. | `--verdict-ast-*` |

* **Disposición interna del Panel de Resultados:**
1. **Banner compacto (`padding: 10px 14px`):** Fondo `var(--verdict-*-bg)`, borde `1px solid var(--verdict-*-border)`, `border-radius: 6px`. Muestra ícono + etiqueta humanizada + subtítulo a la izquierda, y tiempo total / memoria en `JetBrains Mono` a la derecha.
2. **Detalle de casos (2 columnas `200px / 1fr`):** Lista vertical seleccionable (`Caso 1` … `Caso N`, cada fila con punto semántico + etiqueta + tiempo en mono) a la izquierda; a la derecha bloques de `Entrada`, `Salida esperada` y `Salida obtenida` en `JetBrains Mono` (`12px`) sobre `--bg-surface-subtle`.

---

## 5. Arquetipo 3 — Cola Maestro-Detalle, Tabla Densa y Dashboard Técnico — ✅ CERRADO

### 5A. Cola Maestro-Detalle y Tabla Densa (`/teacher/por-calificar`)
*Aplica a: `/teacher/por-calificar`, `/admin/auditoria`, vistas tabulares de `/teacher/aula` y CRUDs densos.*

#### Cabecera Compacta (1 sola línea)
- **Layout:** `display: flex; justify-content: space-between; align-items: center; padding: 16px 24px 12px`.
- **Izquierda (`display: flex; align-items: baseline; gap: 12px`):**
  - `h1`: `20px`, `font-weight: 600`, `letter-spacing: -0.02em`, `--text-primary`.
  - Metadatos en línea (`12px`, `JetBrains Mono`, `--text-muted`): `21 entregas en espera · 4 prácticas · espera máx. 29 d`.
- **Derecha (`gap: 8px`):** `Actualizar` (`.btn-secondary`, `height: 32px`) + `Calificar siguiente →` (`.btn-primary`, `height: 32px`, **único botón primario sólido de la pantalla**; abre la primera entrega pendiente de la práctica seleccionada).
- **Prohibiciones:** Cero breadcrumb redundante, cero etiqueta *eyebrow* superior, cero franja separada de KPIs que repita los números ya visibles en la cabecera.

#### Barra de Filtros Unificada (arriba del contenedor, 1 sola fila)
- **Contenedor:** `display: flex; align-items: center; gap: 8px; padding: 0 24px 12px`.
- **Controles exactos (`height: 32px`, `border-radius: 6px`):**
  1. Select de materia (`Todas las materias`, `min-width: 190px`).
  2. Buscador de estudiante (`width: 260px`, ícono lupa izquierda + badge `/` en `JetBrains Mono` a la derecha).
  3. Select de ordenamiento (`Más antiguas primero`, `Más recientes`, `Nombre A–Z`).
- **Prohibición:** No colocar atajos de teclado en esta barra (viven únicamente en el pie de la tabla).

#### Contenedor Maestro-Detalle Unificado (1 sola caja *border-first*)
- **Contenedor:** `margin: 0 24px 20px; background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 8px; overflow: hidden; display: grid; grid-template-columns: 300px 1fr; min-height: calc(100vh - 172px)`.

#### Panel Izquierdo — Cola de Prácticas (`border-right: 1px solid var(--border-subtle)`)
- **Cabecera interna (`height: 36px; padding: 0 14px; border-bottom: 1px solid var(--border-subtle)`):** Izquierda: `PRÁCTICAS` (`11px`, `600`, uppercase, `letter-spacing: 0.06em`, `--text-muted`). Derecha: `21 pendientes` (`11px`, `JetBrains Mono`, `--text-muted`).
- **Ítem de práctica (`padding: 12px 14px; border-bottom: 1px solid var(--border-subtle); border-left: 2px solid transparent; cursor: pointer`):**
  - Estado activo: `background: var(--tenant-primary-subtle); border-left-color: var(--tenant-primary)`.
  - Fila 1: Nombre de materia (`10px`, `600`, uppercase, `letter-spacing: 0.05em`, `--text-muted`).
  - Fila 2: Título de práctica (`13px`, `600`, `--text-primary`, máx. 2 líneas).
  - Fila 3: Progreso y fecha (`11px`, `JetBrains Mono`, `--text-muted`, ej. `6 de 9 · 21/10 · 23:47:08`).
  - Fila 4: Barra de progreso fina (`height: 3px; margin-top: 8px; border-radius: 2px; background: var(--bg-surface-active)`, relleno `var(--tenant-primary)`).

#### Panel Derecho — Tabla Densa de Entregas (`display: flex; flex-direction: column`)
- **Cabecera de práctica (`padding: 14px 20px; border-bottom: 1px solid var(--border-subtle); display: flex; justify-content: space-between; align-items: center`):**
  - Izquierda: Etiqueta de materia (`10px`, uppercase, `--text-muted`) + título (`15px`, `600`) + resumen mono (`6 por calificar · 3 de 9 calificadas · Más antiguas primero`, `12px`, `--text-muted`).
  - Derecha: `Abrir práctica` (`.btn-secondary`, `height: 30px`, ícono `ExternalLink`).
- **Tabla de exactamente 4 columnas (prohibida columna redundante `ESTADO`):**
  1. `ESTUDIANTE` (`1fr`): Avatar `28px × 28px` (`border-radius: 6px`, `--bg-surface-subtle`, iniciales en `JetBrains Mono` `11px`) + Nombre (`13px`, `500`, `--text-primary`) + correo (`11px`, `--text-muted`).
  2. `ESPERA` (`width: 120px`): Punto `6px × 6px` (verde `< 3 d`, ámbar `3–14 d`, rojo `> 14 d`) + tiempo (`hace 29 d`, `12px`, `JetBrains Mono`).
  3. `ENTREGA` (`width: 150px`): `21/10 · 23:47:08` (`12px`, `JetBrains Mono`, `--text-secondary`).
  4. `ACCIÓN` (`width: 110px; text-align: right`): Botón `Calificar →` (`.btn-ghost`, `height: 28px`, `12px`, `white-space: nowrap`). En hover de fila o foco de teclado cambia a `color: var(--tenant-primary); background: var(--tenant-primary-subtle)`.
- **Navegación por teclado:** `↑` / `↓` resalta fila (`background: var(--bg-surface-subtle); box-shadow: inset 2px 0 0 var(--tenant-primary)`), `Enter` abre la calificación, `/` enfoca el buscador.
- **Pie de tabla (`margin-top: auto; height: 34px; padding: 0 20px; border-top: 1px solid var(--border-subtle); background: var(--bg-surface-subtle); display: flex; justify-content: space-between; align-items: center`):**
  - Izquierda: Atajos discretos con `<kbd>` (`↑ ↓ mover · Enter calificar · / buscar`).
  - Derecha: `Actualizado HH:mm` (`11px`, `JetBrains Mono`, `--text-muted`).
- **Prohibiciones explícitas:** Cero botones primarios sólidos apilados por fila; cero duplicación del perfil de usuario en el sidebar (vive solo en el topbar).

---

### 5B. Dashboard Técnico de Infraestructura (`/admin/dashboard`)
*Aplica a: `/admin/dashboard` y vistas de telemetría o analítica (`/teacher/cursos/:courseId/analytics`).*

#### Cabecera Compacta (sin breadcrumb ni *eyebrow*)
- **Layout:** `display: flex; justify-content: space-between; align-items: flex-end; padding: 18px 24px 14px`.
- **Izquierda:**
  - Fila 1 (`display: flex; align-items: center; gap: 10px`): `Salud del Servidor y Recursos` (`20px`, `600`, `letter-spacing: -0.02em`) + badge de estado operativo inline (`• Operación Normal` / `• Rendimiento Degradado` / `• Umbral Crítico`, con tokens `--status-*` inamovibles).
  - Fila 2 (`margin-top: 4px`, `12px`, `JetBrains Mono`, `--text-muted`): `Docker Engine v29.3.1 (overlayfs) · Uptime: 2 días 1 h · Host: <hostname>`.
- **Derecha (`gap: 8px`):**
  - `Personalizar`, `Auto-refresco` (con punto verde activo `6px × 6px`) y `Actualizar`: **todos `.btn-secondary` (`height: 32px`)**.
  - `Hibernar Todo`: `.btn-secondary` (`height: 32px`), texto e ícono con tinte de peligro sutil; hover `background: var(--color-danger-bg); color: var(--color-error); border-color: var(--status-failed-border)`. Siempre dispara `confirm-modal`.
- **Prohibición:** No usar `.btn-primary` sólido en `Actualizar` ni en acciones de monitoreo pasivo.

#### Tira Unificada de 4 KPIs (margen exterior suave, sin tarjetas centradas)
- **Contenedor:** `margin: 0 24px 16px; display: grid; grid-template-columns: repeat(4, 1fr); background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 8px; overflow: hidden`.
- **Cada celda (`padding: 14px 16px; border-right: 1px solid var(--border-subtle); display: flex; flex-direction: column; gap: 8px`, alineación estricta a la izquierda):**
  1. **Label + Ícono (`display: flex; justify-content: space-between; align-items: center`):** Label (`11px`, `600`, uppercase, `letter-spacing: 0.05em`, `--text-muted`) a la izquierda; ícono Lucide (`14px`, `--text-muted`) a la derecha (**nunca pegado al texto**).
  2. **Valor (`display: flex; justify-content: space-between; align-items: baseline`):** Valor principal (`20px`, `600`, `JetBrains Mono`, `--text-primary`) a la izquierda; porcentaje (`12px`, `JetBrains Mono`, color semántico según umbral) a la derecha.
  3. **Barra de progreso (`position: relative; height: 4px; border-radius: 2px; background: var(--bg-surface-active)`):**
     - Relleno: `var(--tenant-primary)` en carga normal, `var(--color-warning)` `>= 75%`, `var(--color-error)` `>= 85%`.
     - **En KPI de RAM:** incluir marca vertical de umbral (`position: absolute; left: 85%; top: -2px; bottom: -2px; width: 1.5px; background: var(--color-error)`).
  4. **Pie a dos extremos (`display: flex; justify-content: space-between; align-items: center; font-size: 11px; color: var(--text-muted)`):**
     - Izquierda: contexto fijo (`Umbral QoS: 85%`, `Carga media`, `/var/lib/docker`, `Techo max_containers`).
     - Derecha en `JetBrains Mono`: dato complementario (`3.8 GB libres`, `1.42 load`, `overlayfs`, `6 hibernados`).

#### Panel de Telemetría en Tiempo Real
- **Contenedor:** `margin: 0 24px 16px; background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 8px; padding: 16px 20px`.
- **Cabecera (`display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px`):**
  - Izquierda (`display: flex; align-items: center; gap: 16px`): título `Consumo de Recursos en Tiempo Real` (`14px`, `600`) + leyenda inline (`Memoria RAM 67%`, `Carga vCPU 50%`, `Límite QoS 85%`) con valores actuales en `JetBrains Mono`.
  - Derecha: segmented control de ventana temporal (`15m` / `60m` / `6h`, `height: 28px`, `font-family: var(--font-mono)`, `11px`).
- **Gráfica:** Grid horizontal sutil (`--border-subtle`), línea punteada fija en `85%` (`--color-error`) con etiqueta discreta `Umbral Crítico (85%)`, serie RAM en `var(--tenant-primary)` con área translúcida (`0.08–0.12`), serie vCPU en `var(--text-secondary)`, *crosshair* vertical y tooltip oscuro/neutro en `JetBrains Mono` al pasar el cursor.

#### Sección Inferior en 2 Columnas (`grid-template-columns: minmax(0, 1.45fr) minmax(300px, 1fr); gap: 16px; margin: 0 24px 16px`)
- **Izquierda — Distribución de Carga por Materia:**
  - Contenedor: `background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 8px; overflow: hidden`.
  - Cabecera (`padding: 12px 16px; border-bottom: 1px solid var(--border-subtle); display: flex; justify-content: space-between; align-items: center; gap: 8px`): Título + conteo en mono a la izquierda; buscador compacto (`height: 30px`, con badge `/`) + toggle `Solo activas` (`.btn-secondary`, `height: 30px`) a la derecha.
  - Tabla densa (`MATERIA / CURSO`, `DOCENTE A CARGO`, `ALUMNOS ACTIVOS`, `RAM TOTAL` con mini-barra proporcional, `ACCIÓN`). El botón `Detalle →` usa `.btn-ghost` con `white-space: nowrap` obligatorio (nunca se parte en 2 líneas).
- **Derecha — Incidencias de Infraestructura:**
  - Contenedor del mismo estilo (`padding: 14px 16px; display: flex; flex-direction: column; gap: 12px`).
  - Cabecera con título + contador semántico (`0 alertas` / `N alertas`).
  - Lista de alertas OOM/QoS (o estado vacío limpio con borde punteado `--border-subtle`) + nota inferior fija **Protección Automática de Memoria** (`background: var(--bg-surface-subtle); border: 1px solid var(--border-subtle); border-radius: 6px; padding: 10px 12px; font-size: 12px`).

#### Pie Desplegable — Instancias Activas del Clúster
- **Contenedor:** `margin: 0 24px 24px; background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 8px; overflow: hidden`.
- **Trigger (`height: 40px; padding: 0 16px; display: flex; justify-content: space-between; align-items: center`):** `Ver todas las instancias activas del clúster` + contador en `JetBrains Mono` + chevron. Al desplegar: tabla densa de contenedores activos con acción individual `Hibernar` (`.btn-ghost`, `white-space: nowrap`).

- **Prohibiciones globales de 5B:** Cero duplicación del perfil de usuario en el pie del sidebar; cero tarjetas flotantes con sombras pesadas; cero saltos de línea en botones de tabla.

---

## 6. SpeedGrader — Revisión Docente (Evaluación y Desarrollo) — ✅ CERRADO
*Aplica a: mesa de revisión de entregas (`/teacher/submissions/:id/review` o equivalente) en ambas modalidades. Hereda la base edge-to-edge del Arquetipo 2 (Sección 4).*

### 6.1. Estructura General y Geometría (*Edge-to-Edge*)
- **Contenedor:** `height: calc(100vh - 48px); padding: 0; margin: 0; overflow: hidden; display: flex; flex-direction: column`. **Cero tarjetas flotantes** alrededor del árbol, el editor o el panel derecho.
- **Sidebar global:** Inicia colapsado en **modo rail (`52px`)**, expandible bajo demanda.
- **Topbar global:** Mantiene estrictamente el contrato de la Sección 2 (`48px`, **prohibido** insertar breadcrumbs de ruta en el topbar global).
- **Prohibiciones estructurales:** No duplicar la cola de `/teacher/por-calificar` como panel fijo dentro de SpeedGrader; no incrustar terminales de ejecución local (en Desarrollo, *Probar aplicación viva* abre una pestaña nueva); no agregar paletas de comandos (`⌘K`), barras de estado inferiores tipo VS Code ni autoguardado de borradores no soportados por el backend.

### 6.2. Cabecera de Revisión (`height: 48px`, 1 sola línea obligatoria)
- **Regla anti-wrap:** Contenedor `display: grid; grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr); align-items: center; padding: 0 16px; border-bottom: 1px solid var(--border-subtle); background: var(--bg-surface)`. Todos los controles llevan `white-space: nowrap; flex-shrink: 0`.
- **Izquierda (`display: flex; align-items: center; gap: 10px; min-width: 0`):**
  - `← Curso` (`.btn-ghost`, `height: 28px`).
  - Divisor vertical `1px × 16px` (`--border-subtle`).
  - Bloque compacto de contexto (`min-width: 0`): Título de la práctica (`14px`, `600`, `overflow: hidden; text-overflow: ellipsis`) + metadato inline o segunda línea ultra compacta (`11px`, `--text-muted`: `Materia · Entorno de Evaluación` o `Entorno de Desarrollo`, sin píldora pesada).
- **Centro (Paginador de entregas — atajos `J` / `K`):**
  - Contenedor `display: flex; align-items: center; gap: 6px; padding: 2px 6px; border: 1px solid var(--border-subtle); border-radius: 6px; background: var(--bg-surface-subtle)`.
  - Botón `<` (`24px × 24px`, `.btn-ghost`, atajo `K`) + Avatar `22px` + Nombre del estudiante (`13px`, `600`) + correo y posición en mono (`11px`, `--text-muted`, ej. `valeria.flores@uab.edu.bo · 1 de 3`) + botón `>` (`24px × 24px`, `.btn-ghost`, atajo `J`).
- **Derecha (`display: flex; justify-content: flex-end; align-items: center; gap: 8px`) — Regla estricta de 1 solo `.btn-primary`:**
  - **En Entorno de Evaluación:**
    - `Re-ejecutar` (`.btn-secondary`, `height: 30px`; muestra spinner y cambia a `Ejecutando…` mientras corre).
    - `Herramientas ▾` (`.btn-secondary`, `height: 30px`).
    - **`Calificar`** con `<kbd>A</kbd>` (`.btn-primary`, `height: 30px`, **único botón primario sólido de la vista**).
    - Botón guía de atajos (`28px × 28px`, `.btn-ghost`, ícono `Keyboard`).
  - **En Entorno de Desarrollo:**
    - `Probar aplicación viva` (`.btn-secondary`, `height: 30px`, ícono `ExternalLink`; estado de carga al iniciar copia temporal).
    - `Herramientas ▾` (`.btn-secondary`, `height: 30px`; incluye aquí `Ajuste manual [A]` como acción excepcional para no competir con la rúbrica).
    - Botón guía de atajos (`28px × 28px`, `.btn-ghost`).
    - **Ningún `.btn-primary` en la cabecera** (el único botón primario sólido vive fijo al pie del panel derecho: `Guardar Calificación`).

### 6.3. Área de Trabajo, Visor y Comentarios Inline (Estilo Azure DevOps)
- **Layout horizontal (`flex: 1; min-height: 0; display: flex`):**
  - **Árbol de archivos (solo Desarrollo cuando existen múltiples archivos):** `width: 220px` (`min-width: 180px`, colapsable), `border-right: 1px solid var(--border-subtle); background: var(--bg-canvas)`. Cabecera `height: 34px` (`ARCHIVOS` + total en mono + botón colapsar). Archivo seleccionado con `background: var(--tenant-primary-subtle); border-left: 2px solid var(--tenant-primary); color: var(--text-primary)`.
  - **Resizer vertical (`4px`):** Línea central visible `1px var(--border-subtle)`; en hover/arrastre pasa a `var(--tenant-primary)`.
  - **Visor de código (`flex: 1; min-width: 0; display: flex; flex-direction: column; background: var(--bg-surface)`):**
    - **Barra de archivo (`height: 34px; padding: 0 14px; border-bottom: 1px solid var(--border-subtle); display: flex; justify-content: space-between; align-items: center`):**
      - Izquierda: ícono + nombre de archivo (`buscar_indice.py` / `reservas/service.py`, `12px`, `600`, mono) + metadatos (`Python 3 · UTF-8 · N líneas`, `11px`, mono, `--text-muted`).
      - Derecha (`gap: 8px`): fecha de entrega (`11px`, mono, `--text-muted`) + contador interactivo de comentarios (`💬 N`, `.btn-ghost`, `height: 24px`, salta entre hilos del archivo) + etiqueta discreta `Solo lectura` (`11px`, `JetBrains Mono`, `--bg-surface-subtle`, `border: 1px solid var(--border-subtle)`, `border-radius: 4px`, `padding: 2px 6px`).
      - **Prohibición:** No colocar el texto largo `"Haga clic o arrastre para comentar"` en la barra de archivo.
    - **Comentarios pedagógicos inline (patrón Azure DevOps / GitHub PR):**
      - **Disparador en *gutter*:** Botón `[+]` (`18px × 18px`, `border-radius: 4px`, `background: var(--tenant-primary)`, `color: var(--tenant-primary-text)`) visible únicamente al hacer *hover* sobre el número de línea, al seleccionar un rango o al pulsar `C`, sin desplazar horizontalmente el código.
      - **Hilo inline compacto:** Anclado inmediatamente debajo de la línea/rango (`margin: 4px 14px 6px 48px; padding: 8px 12px; background: var(--bg-surface-subtle); border: 1px solid var(--border-subtle); border-left: 2px solid var(--tenant-primary); border-radius: 0 6px 6px 0`).
      - **Anatomía del hilo:** Fila superior única (`display: flex; justify-content: space-between; align-items: center; font-size: 11px`): Autor + ancla `L9` (o `L9–L12` en mono) + marca de tiempo relativa a la izquierda; botón `Plegar` y acciones discretas (`Editar`/`Eliminar`) a la derecha. Cuerpo del comentario pegado debajo (`margin-top: 4px; font-size: 13px; line-height: 1.45; color: var(--text-primary)`).
      - **Estado colapsado:** Pastilla compacta de una sola línea (`height: 26px; padding: 0 10px; font-size: 12px; display: flex; align-items: center; gap: 8px`): `💬 L9 · 1 comentario — "Extracto truncado..."`, desplegable con 1 clic para leer el código sin obstrucciones.
      - **Formulario inline de comentario:** Misma geometría que el hilo, cabecera `Comentando L9`, textarea compacto (`min-height: 56px`), pie con `Cancelar` (`.btn-ghost`, `26px`) y `Comentar` (`Ctrl + Enter`, `.btn-secondary` con acento `--tenant-primary`, `26px`).

### 6.4. Menú `Herramientas` y Utilidades de Inspección
- **Opciones permitidas en el menú desplegable (`width: 260px`):**
  1. `Comparar con plantilla / proyecto base` (`<kbd>D</kbd>`) — visible solo si existe base disponible.
  2. `Evolución temporal del código` (`<kbd>T</kbd>`) — alterna la barra integrada sobre el editor.
  3. `Auditoría de escritura (Replay)` — abre modal descriptivo.
  4. `Ajuste manual` (`<kbd>A</kbd>`) — visible únicamente en **Entorno de Desarrollo** como salida excepcional.
- **Prohibición explícita:** Eliminar la opción desconectada `Análisis de Complejidad Asintótica [B]` del menú.
- **Comparador con base (`D`):** Solo lectura, consume `code-diff-viewer` de la Sección 2 (**prohibido** subrayado verde en líneas agregadas; usar únicamente tinte de fondo suave + `border-left: 2px solid var(--verdict-ac-color)`).
- **Barra integrada de Evolución Temporal (`T`):**
  - Se monta entre la barra de archivo y el código (`padding: 8px 14px; border-bottom: 1px solid var(--border-subtle); background: var(--bg-surface-subtle)`).
  - **Naming limpio:** Título **`Evolución temporal`** (en español; **prohibido** usar `"Time-Travel Replay & AST Timeline"` y **prohibida** la sigla `"AST"`).
  - Controles: `Reproducir / Pausar`, paso anterior/siguiente, reiniciar, slider de línea temporal y selector de velocidad (`0.5x`, `1x`, `2x`, `5x`), todos en `.btn-secondary` / `.btn-ghost` (`height: 26px`, **prohibido** usar `.btn-primary` azul sólido en `Reproducir`).
  - Metadatos en mono (`Duración · Pulsaciones · Pegados · Snapshot X / Y`) y botón cerrar (`×`). Los tooltips deben respetar los límites del contenedor sin invadir el panel derecho.
- **Modal `Auditoría de escritura` (`modal-shell`, `width: 720px`):**
  - **Tono estrictamente descriptivo y neutral:** Prohibido emplear lenguaje acusatorio (*"análisis forense"*) o banners rojos punitivos. Presentar los eventos de pegado como contexto de actividad registrado.
  - Tira compacta de 4 métricas en mono (`Tiempo total`, `Eventos de pegado`, `% código pegado`, `Caracteres tipeados`), controles de reproducción paso a paso y visor de código donde los bloques pegados se destacan con tinte ámbar/neutro discreto (`--status-pending-bg`). Si no existen eventos, mostrar estado vacío limpio.

### 6.5. Panel Derecho — Entorno de Evaluación (`width: 380px`, `min-width: 320px`, scroll interno)
Superficie `var(--bg-surface)`, `border-left: 1px solid var(--border-subtle)`. Cabecera fija `height: 34px` (`REVISIÓN` izquierda + estado de entrega en mono derecha). Cuatro secciones apiladas divididas por `1px solid var(--border-subtle)`:

1. **Veredicto Oficial (`padding: 14px 16px`):**
   - Fila superior: etiqueta de estado con tokens `--verdict-*` (`• Cumple`, `• Con diferencias`, etc.) izquierda + puntaje en `JetBrains Mono` (`18px`, `600`, ej. `100 / 100` o `—`) derecha.
   - Fila de métricas en mono (`12px`, `--text-secondary`): `3 ms · 9 MB · Python 3` y fecha de registro debajo (`11px`, `--text-muted`).
   - Si hubo ajuste manual, bloque sutil debajo con el motivo registrado.
2. **Prueba Rápida (condicional — solo visible tras pulsar `Re-ejecutar`):**
   - **Prohibido** renderizar este bloque vacío antes de ejecutar una prueba.
   - Contenedor diferenciado (`margin: 12px 16px 0; padding: 10px 12px; border: 1px dashed var(--border-strong); border-radius: 6px; background: var(--bg-surface-subtle)`) con etiqueta `PRUEBA RÁPIDA · TEMPORAL`, resultado + tiempo en mono y nota explícita de que no modifica el Veredicto Oficial.
3. **Análisis de Complejidad (solo si la entrega incluye datos):**
   - **Superficie 100 % neutra:** `padding: 14px 16px; background: transparent` (**prohibido** borde rojo de error y **prohibido** el subtítulo `"Espejo Forense"`).
   - Cabecera: `Análisis de complejidad` + etiqueta `Informativo` (`11px`, `--text-muted`).
   - Valores en mono (`15px`, `600`): `O(log N)` tiempo · `O(1)` espacio + línea de referencia `✓ Esperado: O(log N) · O(1)`.
   - **Tabla alineada de muestras (`margin-top: 10px`):** Una fila por muestra (`N = 1 000` izquierda / `0,02 ms` derecha en `JetBrains Mono`, `12px`, separadas por bordes sutiles).
   - Nota al pie (`11px`, `--text-muted`): aclara que es información complementaria y no modifica la nota. Usar tinte ámbar (`--status-pending-*`) **únicamente** ante discrepancia real con la complejidad esperada o baja confianza.
4. **Casos de Prueba (compactos, colapsables y honestos):**
   - Cabecera (`padding: 12px 16px 8px`): `Casos de prueba` izquierda + `4 casos · 2 ocultos` en mono derecha.
   - **Regla de honestidad visual:** Cuando no exista resultado individual confirmado por caso en los datos de la entrega, mostrar una única nota neutra arriba (*"El resultado por caso no está registrado en esta entrega; solo se conserva el veredicto oficial"*) y **nunca** pintar checks ni bordes verdes individuales falsos.
   - **Lista de filas colapsables (`padding: 0 16px 16px`):** Cada caso arranca colapsado en 1 fila (`height: 34px`, `border-bottom: 1px solid var(--border-subtle)`): `#1` en mono + badge `Público` (neutro `--bg-surface-subtle`) u `Oculto` (ámbar `--status-pending-*`) + estado a la derecha (`Sin salida registrada` en `--text-muted` o estado verificado si existe medición individual) + chevron.
   - Al expandir: bloques compactos en `JetBrains Mono` (`12px`) sobre `--bg-surface-subtle` para `Entrada` (con botón `Copiar` `.btn-ghost`), `Salida esperada` y `Salida obtenida` (solo si está disponible).

### 6.6. Panel Derecho — Entorno de Desarrollo (`width: 380px`, `min-width: 320px`, flex column)
- **Cabecera fija (`height: 34px; padding: 0 16px; border-bottom: 1px solid var(--border-subtle)`):** `CALIFICACIÓN` izquierda + segmented control `Rúbrica` / `Nota directa` derecha (cuando aplique).
- **Cuerpo con scroll interno (`flex: 1; overflow-y: auto; padding: 14px 16px; display: flex; flex-direction: column; gap: 16px`):**
  - **Modo Rúbrica (prohibidos los `<select>` truncados):**
    - Cada criterio muestra: fila superior con nombre (`13px`, `600`), estado (`Sin evaluar` o puntaje en mono) y ponderación (`35%`, mono, `--text-muted`).
    - **Selector de niveles (`display: grid; grid-template-columns: repeat(4, 1fr); gap: 6px; margin-top: 8px`):** Botones compactos (`padding: 6px 4px; border-radius: 6px; border: 1px solid var(--border-subtle)`) con puntaje en mono (`13px`, `600`) arriba y etiqueta corta (`10px`: `Insuficiente`, `Básico`, `Logrado`, `Destacado`) abajo. Activo: `border-color: var(--tenant-primary); background: var(--tenant-primary-subtle); color: var(--tenant-primary)`.
    - **Estado inicial neutro:** Ningún nivel preseleccionado al abrir una entrega sin calificar (`Sin evaluar`).
    - **Descripción completa sin truncar (`margin-top: 6px; font-size: 12px; line-height: 1.4; color: var(--text-secondary)`):** Muestra íntegro el texto del nivel elegido (o *"Seleccione un nivel para ver el criterio completo"* mientras esté sin evaluar, respetando contraste AA en ambos temas).
  - **Modo Calificación Directa:** Input numérico `0–100` (`height: 34px`, `JetBrains Mono`, `width: 110px`) + nota sobre escala `/ 100`.
  - **Devolución general:** Textarea (`min-height: 76px`) con etiqueta `Comentarios generales`.
- **Pie fijo (`border-top: 1px solid var(--border-subtle); padding: 12px 16px; background: var(--bg-surface-subtle)`):**
  - Fila resumen (`display: flex; justify-content: space-between; align-items: baseline; margin-bottom: 10px`): etiqueta **`Calificación ponderada`** (en español; **prohibido** `"Score Ponderado"`) izquierda + valor en `JetBrains Mono` (`18px`, `600`: `— / 100` si falta evaluar o puntaje calculado en vivo) derecha.
  - Botón **`Guardar Calificación`** (`.btn-primary`, `width: 100%`, `height: 36px`, **único botón primario sólido de la vista**).

### 6.7. Modal `Calificar / Ajuste manual` (`modal-shell`, `width: 480px`, atajo `A`)
- **En Evaluación:** Selector de veredicto oficial, puntaje (`0–100` en mono) y campo **Justificación del ajuste** (obligatorio, mínimo descriptivo para auditoría backend).
- **En Desarrollo:** Puntaje manual (`0–100`), nota explícita de que sobrescribe la rúbrica, y justificación obligatoria.
- **Pie:** `Cancelar` (`.btn-secondary`) + `Guardar ajuste` (`Ctrl + Enter`, `.btn-primary` dentro del modal).

---

## 7. Hub Docente — `Hoy`, `Detalle del Curso` y `Ficha de la Práctica` — ✅ CERRADO
*Aplica al circuito diario del docente: `/teacher` (`Hoy`), `/teacher/cursos/:courseId` (`Detalle del Curso`) y `/teacher/cursos/:courseId/practicas/:labId` (`Ficha de la Práctica` en Desarrollo y Evaluación).*

### 7.1. Inicio del Docente — `Hoy` (`/teacher`)
- **Sidebar global:** Inicia colapsado en **modo rail (`52px`)**, expandible bajo demanda.
- **Cabecera compacta (1 fila, `padding: 16px 24px 12px`):**
  - Izquierda (`display: flex; align-items: baseline; gap: 12px`): `Hoy` (`20px`, `600`, `letter-spacing: -0.02em`) + fecha en mono (`Sáb 21 nov 2026`, `12px`, `--text-muted`) + resumen inline (`3 materias · 12 entregas por calificar · 4 cierres próximos`).
  - Derecha: **`Calificar pendientes (12) →`** (`.btn-primary`, `height: 32px`, **único botón primario sólido de la vista**).
  - **Prohibiciones:** Cero botón `Abrir aula` ambiguo en la cabecera; cero repetición de botones `.btn-primary` dentro de tarjetas; cero tira semanal de 7 días que duplique la lista de prácticas.
- **Layout balanceado de 2 columnas (`display: grid; grid-template-columns: minmax(0, 1.45fr) 340px; gap: 20px; padding: 16px 24px 32px; align-items: start`):**
  - **Columna Izquierda — `Mis materias` + `Prioridades de revisión`:**
    - **Grid de tarjetas de materias (`display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 12px`):** Tarjetas directas sobre `--bg-canvas` (`background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 8px; padding: 14px 16px`). **Prohibido** encerrarlas en un contenedor azul envolvente o usar bordes izquierdos gruesos de acento.
      - Fila 1: Código en `JetBrains Mono` (`INF-101`, `11px`, `--text-muted`) izquierda; indicador de pendientes en ámbar (`--status-pending-*`) derecha **únicamente si `> 0`** (silencio visual si es `0`).
      - Fila 2: Nombre de la materia (`15px`, `600`) + conteo (`8 inscritos · 6 sin entregar`, `12px`, `--text-secondary`).
      - Fila 3: Mini-barra de entregas (`height: 4px`, fondo `--bg-surface-active`, relleno `--tenant-primary`) con porcentaje en mono (`25 % entregado`, `11px`, `--text-muted`).
      - Fila 4: Próximo cierre (`Cierre 26 nov · Resumen de Ventas Diarias`, `12px`, `--text-muted`).
      - Pie (`border-top: 1px solid var(--border-subtle); padding-top: 10px; display: flex; justify-content: space-between; align-items: center`): `Calificar (N)` (`.btn-ghost`, solo si hay cola) y `Abrir curso →` (`.btn-ghost`, `white-space: nowrap`).
    - **Bloque `Prioridades de revisión` (`margin-top: 20px`):** Contenedor unificado *border-first* (`background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 8px; overflow: hidden`) con las prácticas que tienen entregas pendientes, mostrando materia, título, cantidad por calificar, cierre en mono y acción `Calificar →` (`.btn-ghost`, `white-space: nowrap`).
  - **Columna Derecha (`340px`, `display: flex; flex-direction: column; gap: 16px`):**
    - **Bloque `Cierres próximos`:** Lista compacta de prácticas con fecha límite activa definida en SOLV (`22 nov · 15:18`, título, materia y pendientes). **Prohibido** darle formato o nomenclatura de calendario institucional LMS (*"Agenda escolar"*) o repetir la frase `"vence pronto"` en cada fila. Si no existen prácticas con fecha límite, mostrar estado vacío limpio (`Sin cierres próximos configurados`).
    - **Bloque `Acceso rápido al Aula en vivo`:** Contenedor *border-first* con selector de materia, selector de práctica publicada y botón `Abrir aula` (`.btn-secondary`, `width: 100%`, `height: 32px`), resolviendo el contexto explícito antes de abrir `/teacher/aula`. Con aula abierta, la franja `Aula abierta: <código> N de M dentro` es un enlace completo (`/teacher/aula?subject_id=`) que lleva a la sección del aula de esa materia; el botón `Abrir aula` sigue disponible para abrir otra materia.

### 7.2. Detalle del Curso (`/teacher/cursos/:courseId`)
- **Sidebar global:** Inicia en **modo rail (`52px`)** para convivir sin estrechez con la navegación interna de secciones a `1440×900`.
- **Cabecera compacta (`padding: 14px 24px 12px; border-bottom: 1px solid var(--border-subtle); display: flex; justify-content: space-between; align-items: center; gap: 16px`):**
  - Izquierda (`min-width: 0`): `← Hoy` (`.btn-ghost`, `height: 28px`) + divisor `1px × 16px` + código mono (`SIS-12`, `padding: 2px 6px`, `--bg-surface-subtle`, `border: 1px solid var(--border-subtle)`, `border-radius: 4px`) + nombre del curso (`18px`, `600`) + línea de métricas debajo (`10 estudiantes · 8 laboratorios · 2 por revisar` en mono; **prohibido** enlaces subrayados azules; el contador `por revisar` se colorea en ámbar **solo si `> 0`**).
  - Derecha (`gap: 8px; flex-shrink: 0`): `Importar`, `Métricas`, `Más acciones ▾` (todos `.btn-secondary`, `height: 32px`) y **`+ Nuevo Laboratorio` al extremo derecho** (`.btn-primary`, `height: 32px`, **único botón primario sólido de la vista**).
  - **Menú `Más acciones ▾`:** Contiene **`Auditoría de Similitud`** (**prohibido** usar `"Auditoría Anti-Plagio AST"`), `Pausar Entornos de Clase` y `Exportar Notas (CSV)`. El botón `+ Agregar Estudiante` se retira de la cabecera global y vive dentro de la sección `Estudiantes`.
- **Layout interno con navegación lateral `SECCIONES` (`display: grid; grid-template-columns: 188px minmax(0, 1fr); gap: 20px; padding: 18px 24px 32px; align-items: start`):**
  - **Columna Izquierda (`SECCIONES`, `width: 188px`):** Sustituye las pestañas horizontales para eliminar la sensación de barra sobre barra. Contiene **exactamente 4 secciones** con ícono Lucide (`15px`), etiqueta y contador alineado a la derecha en mono:
    1. `Prácticas`
    2. `Entregas`
    3. `Estudiantes`
    4. `Notas` (etiqueta discreta `Auxiliar` — **prohibido** llamarla `Libreta` o usar escala `0–20`; siempre escala `0–100`).
    - Ítem activo: `background: var(--tenant-primary-subtle); color: var(--tenant-primary); font-weight: 600; border-radius: 6px`.
  - **Columna Derecha (`minmax(0, 1fr)`):**
    - **Sección `Prácticas`:**
      - **Sub-barra de controles (1 fila, `display: flex; justify-content: space-between; align-items: center; gap: 8px; margin-bottom: 12px`):**
        - Izquierda: Buscador (`width: 260px`, `height: 32px`, badge `/`) + botón **`Filtros`** (`.btn-secondary`, `height: 32px`, reutiliza el **popover unificado de 2 columnas** del Arquetipo 1: `ESTADO` (`Todos`, `Publicados`, `Borradores`) | `MODALIDAD` (`Todas`, `Entorno de Desarrollo`, `Entorno de Evaluación`), con sub-barra de chips removibles solo cuando existan filtros activos). **Prohibido** apilar dos segmented controls pesados o múltiples `<select>` sueltos.
        - Derecha: Segmented control `Lista | Por módulos` (`height: 32px`, integra la agrupación por módulos sin consumir una sección de primer nivel).
      - **Lista densa unificada (*border-first*, prohibidas tarjetas individuales estiradas al 100 %):** Un único contenedor (`background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 8px; overflow: hidden`) dividido por líneas de `1px`. Cada fila muestra:
        - Izquierda: Ícono de modalidad (`Laptop` para Desarrollo / `Terminal` para Evaluación, `15px`), punto de estado (`Publicado` verde / `Borrador` neutro), título (`13.5px`, `600`, sin hover azul tipo hipervínculo clásico) y fecha límite en mono (`11px`, `--text-muted`).
        - Derecha: Mini-barra + ratio de entregas (`2 / 10` en mono), indicador de pendientes en ámbar **estrictamente si `> 0`** (**silencio visual obligatorio para ceros**: prohibido colorear `0 calificadas auto` en verde o `0 revisión manual` en naranja), y acciones rápidas discretas en *hover* (`Publicar` en borradores, `Revisar`, `Aula`).
    - **Sección `Entregas`:** Soporta pre-filtrado por `labId` desde la Ficha de la Práctica mostrando chip removible (`Filtrado por: <práctica> ×`). Tabla densa con veredictos **siempre humanizados en español** (`Cumple`, `Con diferencias`, `Tiempo excedido`, `Pendiente de revisión` — **prohibido** mostrar `AC`, `WA`, `TLE`, `RE`, `AST` en tablas docentes) y acción `Revisar →` (`.btn-ghost`, `white-space: nowrap`).
    - **Sección `Estudiantes`:** Buscador + botón `+ Agregar Estudiante` (`.btn-secondary`, `height: 32px`) + tabla densa de inscritos, entregas y última actividad.
    - **Sección `Notas`:** Planilla auxiliar en escala `0–100` con nota discreta aclarando que registra únicamente puntajes de laboratorios en SOLV (no sustituye la libreta institucional) y botón `Exportar CSV` (`.btn-secondary`).

### 7.3. Ficha de la Práctica (`/teacher/cursos/:courseId/practicas/:labId`)
*Vista de control de solo lectura; comparte la misma arquitectura asimétrica para Entorno de Desarrollo y Entorno de Evaluación.*

- **Sidebar global:** Inicia en **modo rail (`52px`)**.
- **Cabecera compacta en 1 sola fila (`padding: 14px 24px 12px; border-bottom: 1px solid var(--border-subtle); display: flex; justify-content: space-between; align-items: center; gap: 16px`):**
  - Izquierda (`min-width: 0`): `← Materia` (`.btn-ghost`, `28px`) + divisor `1px × 16px` + Título (`18px`, `600`, `overflow: hidden; text-overflow: ellipsis`) + línea inferior de estado: `• Publicado` / `• Borrador`, ícono + modalidad (`Entorno de Desarrollo` / `Entorno de Evaluación`) y `Vence: <dateText>` (si aplica). **Prohibido** repetir `1000 ms` ni `O(N)` en la cabecera (viven únicamente en `ESPECIFICACIÓN TÉCNICA`).
  - Derecha (`gap: 8px; flex-shrink: 0`): `Más opciones ▾`, `Editar Laboratorio`, `Llevar al Aula` (solo si está publicada, todos `.btn-secondary`, `height: 32px`) y **al extremo derecho el único `.btn-primary` (`height: 32px`)**:
    - En Desarrollo: **`Abrir IDE`** (ícono `Laptop`; pasa a `Abriendo IDE…` con spinner al abrir).
    - En Evaluación: **`Probar Solución`** (ícono `Play`, diferenciado visualmente de `Abrir IDE`).
  - **Menú `Más opciones ▾`:** En Desarrollo incluye `Probar Entorno`; en Evaluación incluye `Casos límite` (abre modal generador); en ambas `Auditoría de Similitud`; si está publicada, separador + `Volver a borrador` en `--color-error` (dispara `confirm-modal`).
- **Layout asimétrico de 2 columnas (`display: grid; grid-template-columns: minmax(0, 1fr) 340px; gap: 20px; padding: 18px 24px 32px; align-items: start`):**
  - **Columna Izquierda (`1fr` — Contenido pedagógico en secciones plegables con `aria-expanded`):**
    - **Consigna (ambas modalidades):** Markdown (+ KaTeX si aplica) renderizado **directamente sobre `--bg-surface`** (**prohibido** anidar un recuadro celeste con borde azul grueso dentro de una tarjeta blanca).
    - **Rúbrica de Calificación (cuando exista):** Encabezado con `Puntaje total: 100 pts`. **Barra horizontal proporcional única de 100 %** arriba (interactiva al *hover* sobre cada segmento/criterio) + lista compacta de criterios con peso en mono y descripción completa (**prohibido** grid de 3 tarjetas que deje criterios huérfanos).
    - **Código Base para el Estudiante (solo Evaluación):** Bloque CodeMirror solo lectura con altura ajustada exactamente a las líneas reales del código (**prohibido** espacio blanco sobrante debajo), cabecera compacta con nombre de archivo, badge `Solo lectura` y botón `Copiar`.
    - **Casos de Prueba (solo Evaluación):** Encabezado con total y puntaje (`Puntaje total: N pts`). Pestañas horizontales estilo *underline* (`Caso 1`, `Caso 2`...) con etiqueta discreta `Público` (`--status-running-*`) u `Oculto` (`--status-pending-*`) — **prohibido** insertar píldoras blancas dentro de botones azules sólidos. Panel del caso activo con descripción, puntaje con pluralización correcta (`1 pt` / `N pts`) y bloques lado a lado `Entrada (STDIN)` y `Salida esperada (STDOUT)` en `JetBrains Mono` sobre `--bg-surface-subtle`.
  - **Columna Derecha Sticky (`340px; position: sticky; top: 18px`) — Separación estricta entre Control en vivo y Referencia técnica:**
    1. **`CONTROL OPERATIVO` (superficie elevada: `background: var(--bg-surface); border: 1px solid var(--border-subtle); border-radius: 8px; padding: 14px 16px`):**
       - **Entregas:** Ratio destacado (`2 / 10 registradas`), barra fina de avance, badge `N pendientes` en ámbar solo si `> 0`, y botón `Ir a Entregas de esta práctica →` (`.btn-secondary`, `width: 100%`).
       - **Prueba de entorno (solo Desarrollo):** Muestra estado actual (`Sin pruebas`, `En curso` con avance % y botón `Cancelar`, `Entorno verificado` o `Error` con detalle) y **botón directo `Probar entorno`** (`.btn-secondary`) en el mismo bloque para evitar callejones sin salida.
    2. **`ESPECIFICACIÓN TÉCNICA` (directamente sobre `--bg-canvas`, SIN tarjeta contenedora — filas clave-valor con divisores `1px solid var(--border-subtle)`):**
       - Solo parámetros útiles para el docente (**prohibido** añadir métricas irrelevantes de infraestructura o reglas de Semgrep aquí).
       - En Desarrollo: `Plantilla` (nombre + mini-logo SVG a color de tamaño explícito fijo `width: 16px; height: 16px` sin borde de píldora), `Base de datos` (si aplica), `Política de entregas` (`Sin configurar` / `Re-entregas habilitadas` / `Una sola entrega`) y `Límites` (si aplica).
       - En Evaluación: **`Lenguaje`** (**prohibido** usar `"Runtime"`, con mini-logo `16px × 16px`), `Tiempo límite` (`1000 ms`), `Complejidad` (`O(N)`) y `Casos` con pluralización exacta (`1 público · 1 oculto` / `2 públicos · 3 ocultos`).
    3. **Acordeón `Bases de Datos de los Alumnos` (solo Desarrollo, cerrado por defecto):** Lista por estudiante con botón `Reiniciar base` (`.btn-ghost`, dispara `confirm-modal` y regenera la base sin tocar el código).

---

## 8. Editor y Creador de Prácticas / Laboratorios — ✅ CERRADO

*Aplica a `/teacher/cursos/:courseId/ejercicios/nuevo` y `/teacher/cursos/:courseId/ejercicios/:id/editar` en ambas modalidades (**Entorno de Desarrollo** `IDE_PERSISTENTE` y **Entorno de Evaluación** `JUEZ_EFIMERO`) y ambos propósitos (**Clase / Formativo** `class` y **Examen / Calificado** `exam`).*

### 8.1. Estructura General, Geometría y Regla Sticky
- **Sidebar global:** Inicia colapsado en **modo rail (`52px`)**, expandible a `224px` bajo demanda. Prohibido duplicar el perfil de usuario en el pie del sidebar.
- **Topbar global:** Mantiene estrictamente el contrato de la Sección 2 (`48px`, prohibido insertar breadcrumbs, buscadores `⌘K` o acciones de la vista en el topbar global).
- **Regla técnica obligatoria para `sticky`:** En `html, body` (y cualquier ancestro del editor) usar `overflow-x: clip` — **prohibido** usar `overflow-x: hidden`, ya que anula `position: sticky` y hace que la barra izquierda de secciones desaparezca al hacer scroll.
- **Layout de 2 columnas (`1440×900` garantizado con cero scroll horizontal):**
  - Contenedor: `display: grid; grid-template-columns: 220px minmax(0, 1fr); gap: 24px; padding: 20px 24px 36px; align-items: start`.
  - **Prohibido** usar un layout de 3 columnas que comprima el editor Markdown dividido o la tabla de Casos de Prueba.
- **Voz y redacción (contrato estricto):** 100 % español neutro formal (**trato de «usted»**). Prohibido mezclar voseo o tuteo (*«Elegí», «usás», «Consultá», «Ingresá», «Presioná», «Arrastrá», «seleccionalos», «cargás», «definís», «calificás», «Guarda primero», «tu biblioteca»*). Usar siempre *«Seleccione», «Guarde primero», «su biblioteca», «Si define criterios... califica manualmente»*. Prohibido usar terminología ajena a ingeniería (*«Esencia», «Capítulos», «SOLV Studio»*).

### 8.2. Cabecera Fija de Edición (`height: 48px`, `position: sticky; top: 48px`)
*Sin tarjeta flotante; superficie plana `--bg-canvas` con `border-bottom: 1px solid var(--border-subtle)` y `padding: 0 24px`.*

- **Izquierda (`display: flex; align-items: center; gap: 10px; min-width: 0; flex: 1`):**
  - `← <Materia>` (`.btn-ghost`, `height: 28px`).
  - Divisor vertical `1px × 16px` (`--border-subtle`).
  - Título en vivo (`15.5px`, `600`, `letter-spacing: -0.02em`, `overflow: hidden; text-overflow: ellipsis; white-space: nowrap`): muestra el título escrito o `Nueva práctica` / `Nuevo examen`.
  - Badge de estado: `Borrador` (neutro) o `Publicado` (`--status-ok-*`).
  - Indicador de persistencia: `• Guardado` (verde) o `• Cambios sin guardar` (ámbar) acompañado del botón discreto `Guardar` (`.btn-ghost`, `height: 24px`) mientras existan cambios o no haya ID guardado.
- **Derecha (`display: flex; align-items: center; gap: 8px; flex-shrink: 0`):**
  - **Píldora viva condicional (solo Desarrollo):** Si `Probar entorno` está en ejecución y el docente navegó a la Sección 1 o 3, mostrar píldora compacta clicable (`Probando entorno… XX%`, `--status-warn-*`, `height: 28px`) que regresa a la Sección 2 al hacer clic. **Prohibido** fijar el botón `Probar entorno` en la cabecera para evitar saturarla.
  - `Vista previa` (`.btn-secondary`, `height: 32px`, abre modal de vista previa sobre el editor).
  - `Generar con IA` (`.btn-secondary`, `height: 32px`, incluye indicador compacto `Institucional` o `BYOK`).
  - Menú `⋮` (`.btn-secondary btn-icon`, `32px × 32px`): protege **`Eliminar borrador`** en `--status-err-color` disparando `confirm-modal`. **Prohibido** colocar `Eliminar borrador` como botón rojo expuesto junto a `Publicar`.
  - **`Publicar`** (o **`Republicar entorno`** si cambió la plantilla en una práctica ya publicada, o **`Actualizar publicación`**): **único `.btn-primary` sólido de toda la vista**, siempre en `var(--tenant-primary)` con texto `var(--tenant-primary-text)` y ubicado al extremo derecho. **Prohibido** pintar `Publicar` en verde sólido.

### 8.3. Columna Izquierda Sticky (`220px`) — `SECCIONES` + `Verificación (Preflight)` Compacto
*`position: sticky; top: calc(48px + 48px + 20px); display: flex; flex-direction: column; gap: 18px`.*

- **Bloque `SECCIONES` (3 bloques conmutables para eliminar el scroll de 5 pantallas):**
  - Cabecera mono `SECCIONES` + contador `N / 3`.
  - *En Desarrollo:* `1. Identidad y consigna` · `2. Entorno y proyecto` · `3. Entrega y rúbrica`.
  - *En Evaluación:* `1. Identidad y enunciado` · `2. Solución y casos` · `3. Reglas y código inicial`.
  - Cada botón (`height: 36px; border-radius: 6px`) muestra su estado en vivo a la derecha: `✓` verde (lista), punto rojo `6px × 6px` (bloqueo obligatorio pendiente en esa sección) o ícono `⚠` ámbar (aviso no bloqueante).
  - Ítem activo: `background: var(--bg-surface); border: 1px solid var(--border-subtle); font-weight: 600; box-shadow: var(--shadow-subtle)`.
- **Tarjeta compacta `Verificación / Preflight` (cero overflow en `1440×900`):**
  - **Prohibido** desplegar la lista completa de bloqueos y avisos permanentemente dentro de la barra izquierda estrecha (y prohibido inventar anillos de porcentaje como «60»).
  - Muestra únicamente: estado resumido en 2 líneas con punto semántico (`Listo para publicar` / `Faltan N requisitos · M avisos`), botón **`Ver detalle`** (`.btn-secondary`, `height: 28px`, `width: 100%`) que abre el Modal de Preflight, y una línea mono de contexto al pie (`RAM 512 MB · PostgreSQL 18` o `1 ej. · 1 púb. · 1 oc.`).

### 8.4. Columna Derecha (`minmax(0, 1fr)`) — Contexto Fijo y Hoja de Consulta Rápida
- **Franja Superior Fija de Contexto (`.surface-box`, `padding: 12px 18px`):**
  - Fila principal con dos controles segmentados (`role="radiogroup"`):
    - `MODALIDAD`: `Entorno de Desarrollo` | `Entorno de Evaluación`.
    - `PROPÓSITO`: `Clase / Formativo` | `Examen / Calificado`.
  - **Sub-franja condicional de Examen (solo si `purpose === 'exam'`):**
    - Cambia la etiqueta del título a `Título del examen *`.
    - Muestra `ACCESO A INTERNET` (`Sin internet` por defecto | `Excepción con motivo`). Recordatorio visible: *«El editor y las ejecuciones locales continúan funcionando sin conexión a internet»*. Al elegir `Excepción con motivo`, despliega textarea obligatorio con contador en vivo y validación real de **mínimo 10 caracteres** (bloqueo en Preflight).
    - **Chip `Anti-copia (semilla por estudiante)`:** Visible **únicamente** cuando `modality === 'JUEZ_EFIMERO'` **y** `purpose === 'exam'`. Al volver a Desarrollo o a Clase se restablece automáticamente a `false`.
- **Hoja de Consulta Rápida (`Ver consigna` / `Ver enunciado`):**
  - Visible únicamente en la **Sección 2** y **Sección 3** (ambas modalidades) en una barra contextual discreta sobre el bloque activo.
  - Al pulsar **`Ver consigna`** (Desarrollo) o **`Ver enunciado`** (Evaluación) (`.btn-secondary`, `height: 28px`, ícono `PanelRightOpen`), se desliza un **drawer lateral derecho (`width: 440px; max-width: 92vw`, solo lectura)** con título, píldoras de metadatos y el Markdown/KaTeX renderizado.
  - Cierra con clic fuera, botón `×`, tecla `Escape` o automáticamente al regresar a la Sección 1. Incluye acceso rápido `Editar en sección 1` al pie.

### 8.5. Contenido por Modalidad — Entorno de Desarrollo (`IDE_PERSISTENTE`)

#### Sección 1 — Identidad y consigna
- **Identidad:** Título (obligatorio, máx. `200` caracteres con contador mono) y campo de etiquetas con chips removibles y botón `Agregar` (`.btn-secondary`).
- **Consigna del proyecto (Markdown + KaTeX):**
  - Subtítulo coherente con proyecto (**prohibido** hablar de «entrada/salida y restricciones» aquí).
  - Vistas en control segmentado neutro (`Editor` | `Dividida` | `Vista previa` — **prohibido** usar botón azul primario sólido en el selector de vista).
  - Barra de formato (`B`, `I`, `</>`, `{ }`, `Σ Fórmula`) + inserciones rápidas **`+ Requisitos`**, **`+ Entregables`** y **`+ Pautas`** (**prohibido** mostrar `Insertar ejemplos de casos` en Desarrollo).
  - Pie con contadores de palabras y caracteres en `JetBrains Mono`.

#### Sección 2 — Entorno y proyecto
1. **Entorno de trabajo (Plantilla homologada):**
   - Combobox buscable filtrado a plantillas `approved` con destino `IDE_PERSISTENTE` o `AMBOS`. Debe mostrar siempre el **nombre legible de la plantilla** (**prohibido** mostrar UUID crudo en el campo de texto) + botón `+ Solicitar nuevo entorno` (`.btn-secondary`, `28px`).
   - Ficha de plantilla seleccionada (`.selected-tpl-card`): nombre, badge `BD <motor>` (si aplica) o `Entorno aprobado`, **RAM asignada en solo lectura** (`RAM asignada: 512 MB (solo lectura)` — gobernada por la plantilla) y stack con **mini-logos SVG a color (`16px × 16px`) apoyados directo sobre la superficie sin borde de píldora**.
   - **Bloque operativo `Prueba de entorno` (integrado en Sección 2):** Estado (`Sin pruebas`, `En curso · XX%` con barra de progreso y `Cancelar`, `Entorno listo` con hora, o `Error` con detalle) + botón `Probar entorno` (`.btn-secondary`, `28px`). El sistema guarda el borrador automáticamente si hay cambios pendientes antes de lanzar el job efímero.
2. **Proyecto inicial (versionado independiente de la práctica):**
   - **Sin ID guardado (`!practiceId`):** Contenedor punteado con aviso claro y botón directo **`Guardar borrador y habilitar`** (`.btn-secondary`) para eliminar callejones sin salida.
   - **Con ID guardado:** Badge `vN · rev. M` y botón **`Generar archivos con IA`** (`.btn-secondary`, diferenciado explícitamente de `Generar con IA` de la cabecera).
   - Controles de carga: `Importar carpeta` (`300 archivos`, `512 KiB/archivo`, `1 MiB total`), `Importar ZIP (10 MiB)` e `Importar tar.gz (32 MiB)` + estado de `Dependencias` (`Preparada` / `En cola` / `Falló`).
   - **Advertencia de compatibilidad de stack:** Si los archivos no coinciden con la plantilla, mostrar banner ámbar con casilla de reconocimiento obligatoria (bloquea publicación en Preflight si no se marca).
   - **IDE editable (`220px` árbol + `1fr` editor UTF-8/LF):** Crear archivo, renombrar y eliminar con `confirm-modal`.
   - **Versionado independiente:** Selector de historial + acciones separadas `Guardar borrador (rev. M)` y `Publicar versión (vN+1)` (deshabilitada mientras haya cambios locales sin guardar). El proyecto inicial es **opcional** y nunca bloquea la publicación por sí solo.
3. **Script inicial de base de datos (condicional):**
   - Visible **únicamente** si la plantilla seleccionada declara motor de BD **o** si existe un script guardado previamente.
   - **Bloqueo por script huérfano:** Si la plantilla actual no declara BD pero existe texto en el script, mostrar banner de error (`--status-err-*`) con botón directo `Limpiar script` y bloquear publicación en Preflight.
   - Acepta `.sql`, `.txt` y `.js` concatenados, botones `Cargar`, `Descargar` y `Limpiar`, editor monospace y métricas en `JetBrains Mono` (`Líneas`, `Sentencias aprox.`, `Tamaño`, `Tiempo est. / alumno`). Si supera `2 MiB`, mostrar aviso ámbar no bloqueante.

#### Sección 3 — Entrega y rúbrica
- **Entrega y sincronización:** Control segmentado unificado `Sin fecha límite | Con fecha límite` (con selector `datetime-local` y atajos `+1 semana` y `Domingo 23:59`), más switches `Permitir reenvíos` y `Puntos de control automáticos`.
- **Rúbrica opcional:**
  - **Regla anti-falsa alarma:** Cuando tiene `0` criterios, **prohibido** mostrar la alarma `Total: 0% · Debe sumar 100%`. Mostrar estado vacío limpio aclarando que sin criterios se califica manualmente sobre 100 puntos.
  - Cuando existe `≥ 1` criterio: mostrar badge `Total: X%` (verde si `100 %`, ámbar si `≠ 100 %`), barra horizontal proporcional única de `100 %`, botón `Equilibrar` (si hay `> 1` criterio) y botón **`+ Agregar criterio`** (`.btn-secondary` — **prohibido** el doble signo `+ +` y prohibido usar `.btn-primary`).
  - Cada criterio inicia con 3 niveles desplegables (`Logrado 100`, `Parcial 50`, `No logrado 0`), nombre, peso `%`, descripción y acción eliminar.

### 8.6. Contenido por Modalidad — Entorno de Evaluación (`JUEZ_EFIMERO`)

#### Sección 1 — Identidad y enunciado
- **Identidad:** Título (obligatorio, máx. `200`).
- **Selector Multilenguaje Escalable (`Lenguajes de programación *`, máximo `12`):**
  - **Prohibido** renderizar todo el catálogo de lenguajes como una nube abierta de píldoras sobre el formulario (crece sin control al agregar nuevas tecnologías).
  - **Control cerrado (`.lang-multi-box`, `min-height: 36px`):** Muestra únicamente los lenguajes seleccionados con un **tope visual de 5 chips removibles (`×`)**; si hay más de 5, los restantes se condensan en una única píldora **`+N más`** (`--tenant-primary-subtle`), junto al disparador `Elegir ▾` a la derecha y contador `X / 12 permitidos` en la etiqueta.
  - **Popover buscable (`.lang-popover`):** Anclado al control, con buscador por texto, atajos `Populares (4)` / `Máximo (12)` / `Limpiar`, lista en **grid de 2 columnas** con scroll interno (`max-height: 236px`, cada fila muestra checkbox + nombre + RAM del ejecutor en mono; bloquea opciones adicionales al llegar a `12`) y pie con contador y botón `Listo`.
  - **Sincronización:** Actualiza automáticamente el lenguaje activo de la Solución de referencia, las pestañas de Código inicial y el selector de lenguaje de la Vista Previa del Estudiante.
- **Parámetros técnicos (grid 2 columnas):** `Complejidad esperada` opcional (`O(1)`, `O(N)`, `O(N log N)`, `O(N²)`, informativa) + `Tiempo límite por caso (ms)` (`100–30 000`, default `1000`, con validación real bloqueante en Preflight).
- Mismo control segmentado unificado de **Fecha límite** que en Desarrollo, etiquetas, y editor Markdown/KaTeX con **`+ Entrada`, `+ Salida`, `+ Restricciones`** + **`Insertar ejemplos de casos (N)`** (deshabilitado si no existen casos con visibilidad `Ejemplo`).

#### Sección 2 — Solución y casos
- **Solución oficial de referencia (plegable a 1 fila):**
  - **Prohibido** mostrar numeración huérfana (`1.`) o píldoras vacías (`CÓDIGO CANÓNICO`).
  - Cabecera colapsable de una sola fila: ícono + `Solución oficial de referencia · <Lenguaje> · N líneas` + badge `Lista` (verde) o `Sin definir · Opcional` (ámbar) + acción `Editar solución` / `Plegar`. Arranca plegada cuando ya tiene código para maximizar el área visible de la tabla de casos.
  - Al expandir: selector de lenguaje de referencia (si hay `> 1` lenguaje permitido), contador de caracteres, aviso ámbar no bloqueante si está vacía y editor monospace con numeración de líneas.
- **Tabla de Casos de Prueba:**
  - Barra de acciones (todos `.btn-secondary`, `height: 28px`, **cero botones azules primarios duplicados en estado vacío**): `Calcular Salidas`, `Equiparar Pesos`, `Generar Casos` y `+ Agregar Caso`.
  - Pestañas *underline*: `Todos`, `Ejemplos Públicos` (agrupa `Ejemplo` y `Público`) y `Casos de Evaluación` (`Ocultos`), cada una con contador mono.
  - Columnas: `#` (con etiqueta pedagógica clicable), `STDIN`, `STDOUT` (con botón calcular por fila), `Visibilidad` (`Ejemplo` / `Público` / `Oculto`), `Peso`, `%` en vivo, `Validez` (`Válido` / `Inválido`) y acciones (`Editar en paralelo`, subir, bajar, duplicar, eliminar).

#### Sección 3 — Reglas y código inicial
- **Bloque plegable 1 — Restricciones didácticas de código:** 3 interruptores (`Exigir algoritmo de ordenamiento propio`, `Forzar solución recursiva pura`, `Restringir módulos del sistema operativo`) + alta de restricción personalizada (`Lenguaje`, `Patrón`, `Mensaje orientador`, botón `+ Prohibir` en `.btn-secondary`) y lista de reglas agregadas con botón quitar.
- **Bloque plegable 2 — Código inicial para el estudiante:** Pestañas *underline* con scroll horizontal sincronizadas con `allowed_languages`, 4 botones de origen rápido (`Plantilla estándar E/S`, `Firma de función`, `Lienzo en blanco`, `Derivar desde solución de referencia`), editor monospace y pie con lenguaje activo, líneas y caracteres.

### 8.7. Contratos de Modales del Editor (`modal-shell`, `max-height: 85vh`)
*Todos los modales abren sobre el editor manteniendo el contexto. **Prohibido** redirigir a pantallas completas separadas para `Vista previa` o `Preflight`.*

1. **Modal `Lista de verificación previa a publicar` (Preflight, `width: 480px`):**
   - Se abre desde `Ver detalle` (barra izquierda) o automáticamente al pulsar `Publicar` si hay bloqueos o avisos sin aceptar.
   - Bloqueos obligatorios en rojo (clicables para saltar directo a la sección y enfocar el campo), avisos no bloqueantes plegables en ámbar (máx. 3 visibles + contador de adicionales), casilla `Acepto publicar la práctica con los avisos señalados` y botón `Publicar ahora` (`.btn-primary`, habilitado solo sin bloqueos y con avisos aceptados).
2. **Modal `Copiloto IA` (`Generar con IA`, `width: 740px`):**
   - Adaptado a Desarrollo (entorno fijado de solo lectura, consigna máx. `1500` caracteres, dificultad) vs. Evaluación (consigna, dificultad, casos públicos `1–5` y ocultos `1–10`). Atajo `Ctrl + Enter` (formal, sin voseo).
   - Devuelve propuesta editable antes de aplicar al borrador (**nunca publica automáticamente**).
   - **Panel desplegable `Mi API Key personal (BYOK)`:** Layout estricto en **2 columnas (`210px` lista de proveedores + `minmax(0, 1fr)` configuración)** con `overflow-x: hidden` para garantizar **cero cortes y cero scroll horizontal**. Incluye switch de activación (`Motor Institucional` vs. `Mi API Key Personal`), API Key enmascarada, filtro de modelos (`Todos` / `Gratuitos` / `De pago`), buscador, `Probar conexión`, `Guardar clave` y `Eliminar clave`.
3. **Modal `Generar archivos con IA` (Proyecto inicial de Desarrollo, `width: 740px`):**
   - Plantilla en solo lectura, 3 campos con límites visibles (`Descripción del proyecto` `2000`, `Propósito` `500`, `Requisitos de archivos` `1200`; total combinado `≤ 4000`) y vista previa de árbol antes de aplicar al borrador local.
4. **Modal `Generar Casos de Prueba` (Evaluación, `width: 740px`):**
   - Dos pestañas: `Plantillas Estructuradas` (5 estructuras: `Un entero N`, `N y luego N enteros`, `Matriz N×M`, `Grafo con aristas`, `Una cadena`; rangos, cantidad `1–100`, semilla opcional) y `Script Python` (editor JSON en sandbox). Incluye vista previa de 3 muestras antes de importar como `Ocultos` con peso `1.0`.
5. **Modal `Editar Caso #N` (`width: 740px`):**
   - Descripción pedagógica opcional, bloques paralelos `STDIN` y `STDOUT` con contador de líneas en vivo, botón `Calcular desde Solución` y `Guardar cambios`.
6. **Modal `Vista Previa del Estudiante` (`width: 960px`):**
   - Adaptado estrictamente a la modalidad activa y al `terminology-gate` (**prohibido** mostrar «Docker», «Paso 3» o siglas `AC`, `WA`, `TLE`, `RE`, `AST`):
     - *Desarrollo:* ficha limpia de solo lectura con título, metadatos, acción ilustrativa deshabilitada de OpenVSCode, plantilla con mini-logos sin píldora, consigna renderizada, archivos del proyecto inicial y criterios de rúbrica.
     - *Evaluación:* split tipo Solver (Arquetipo 2) con límites (`ms`, `MB`, complejidad), enunciado, casos marcados como `Ejemplo`, editor de solo lectura con selector de lenguaje (cuando hay `> 1` permitido) y referencia de estados humanizados (`Cumple`, `Con diferencias`, `Tiempo excedido`, `No evaluado`).

---

## 9. Aula del Docente (`/teacher/aula`) — ✅ CERRADO
*Aplica a la consola de sesiones de clase en vivo del docente: apertura de sesión, seguimiento de participación, emisión de avances, pausa en solo lectura y cierre.*

### 9.1. Estructura General y Reglas de Dominio
- **Sidebar global:** Inicia colapsado en **modo rail (`52px`)**, expandible bajo demanda (`224px`), sin duplicar perfil de usuario en el pie.
- **Topbar global:** Mantiene estrictamente el contrato de la Sección 2 (`48px`, **prohibido** colocar acciones específicas de sesión o breadcrumbs en el topbar global).
- **Redacción:** Español neutro formal (trato de **usted**; **prohibido** el voseo como *«Abrí»*, *«Podés»* o *«si la clase sigue, abrí otra»*).
- **Separación estricta de endpoints:**
  - **Tablero (`/api/v1/teacher/sessions/{id}/board`):** entrega únicamente presencia, estado compuesto (`En aula`, `Se trabó`, `Entregó`, `Salió`, `Sin empezar`), última actividad y datos de entrega (`verdict`, `intentos`). **Prohibido** inventar una columna de «avance aplicado» dentro del tablero principal.
  - **Avances (`/api/v1/teacher/sessions/{id}/advances`):** gestiona las emisiones (título/hora, origen `Web` / `SOLV Companion`, alcance y desglose por destinatario al desplegar una fila).
  - **Veredictos siempre humanizados en español:** `Cumple`, `Con diferencias`, `Tiempo excedido`, `Pendiente de revisión`. **Prohibido** mostrar siglas técnicas (`AC`, `WA`, `TLE`, `RE`, `AST`).

### 9.2. Estado «Sin aula abierta» (Layout 2 Columnas: `1fr` + `340px`)
- **Cabecera compacta:** Título `Aula` + indicador discreto `Sin aula abierta` + línea breve aclarando que los estudiantes pueden continuar trabajando en prácticas publicadas aunque no haya aula abierta.
- **Columna Izquierda (`1fr` — Protagonista: `Abrir nueva clase`):**
  - Cabecera de tarjeta con botón **`Reabrir última configuración`** (`.btn-secondary`, `28px`) para relanzar en 1 clic la última materia, práctica y duración usadas.
  - Selectores de **Materia** y **Práctica publicada** (mostrando modalidad `Desarrollo` / `Evaluación`, propósito `Clase` / `Examen` y capacidades asociadas, o permitiendo `Sin práctica asignada`).
  - **Duración en botones rápidos** (`1 h` · `1 h 30 m` · `2 h` · `3 h` — **prohibido** esconder la duración en un `<select>`).
  - Pie con nota sobre `SOLV Companion` y **`Abrir aula`** (`.btn-primary`, `32px`, **único botón primario sólido**).
- **Columna Derecha (`340px` — Secundario: `Última sesión` + `Historial`):**
  - **`Última sesión`:** Muestra materia, práctica, fecha/hora de cierre y estado de la **Ventana de avances**:
    - Si está **abierta:** muestra minutos restantes, botones `+15 min` / `+30 min`, `Cerrar` y selector de fecha/hora límite (`Fijar`).
    - Si está **cerrada:** se muestra compacta con accesos directos **`Reabrir por 15 min`** y **`Ver opciones`** (desplegando `+30 min`, `+60 min` y fecha límite), evitando ocupar media pantalla con controles deshabilitados.
  - **`Historial de la última sesión`:** **Plegado por defecto** con contador en mono; se despliega bajo demanda.

### 9.3. Estado «Aula en vivo» — Matriz Estricta `Entorno × Propósito`
- **Cabecera operativa en 1 fila (`padding: 14px 24px; border-bottom: 1px solid var(--border-subtle)`):**
  - Izquierda: badge vivo (`• En vivo` verde o `• En pausa · Solo lectura` ámbar) + código mono y nombre de la materia + badge `Examen` (si aplica), y debajo práctica asignada (o `Sin práctica asignada`) + rango horario en mono (`21:15 – 22:45`).
  - Derecha: reloj regresivo en mono (`HH:MM:SS` · `tiempo restante` / `en pausa`) + controles respetando **estrictamente** la matriz y **máximo 1 `.btn-primary` al extremo derecho**:

| Escenario en vivo | `Abrir mi IDE` | `Emitir avance` | Controles de sesión |
|---|---|---|---|
| **Desarrollo + Clase** | Sí (`.btn-secondary`) | **Sí (`.btn-primary`, extremo derecho)** | `Pausar clase` + `⋮ Extender +30 min` / `Cerrar aula` (rojo) |
| **Desarrollo + Examen** | **Sí (`.btn-primary`, extremo derecho)** | Oculto (bloqueado en examen) | `Pausar clase` + `⋮ Extender +30 min` / `Cerrar examen` (rojo) |
| **Evaluación (Clase o Examen)** | Oculto (`JUEZ_EFIMERO`) | Oculto (`JUEZ_EFIMERO`) | `Pausar clase` + `⋮ Extender +30 min` / `Cerrar aula` (rojo) |
| **Sin práctica asignada** | Oculto | Oculto | `⋮ Extender +30 min` / `Cerrar aula` (rojo) |

- Al pausar la clase, mostrar banner explícito indicando que los entornos de los estudiantes quedan en **solo lectura** hasta reanudar.
- El aula del alumno (`/student/aula`) reutiliza este patrón en la tarjeta de la clase abierta: badge de estado como único indicador, código y materia en una fila y una sola línea de metadatos (práctica · fecha y horario · presencia).

### 9.4. Tablero Adaptativo y Panel Lateral
- **Ancho adaptativo:**
  - En **Desarrollo + Clase** (donde aplican avances): layout de 2 columnas (`minmax(0, 1fr) 340px`, gap `20px`).
  - En **Desarrollo + Examen**, **Evaluación** y **Sin práctica asignada** (donde no aplican avances): el tablero ocupa el **ancho completo** (sin columna derecha vacía) y el **`Historial`** pasa a un botón con cajón deslizable (`drawer`) en la cabecera.
- **Barra de filtros y conteos exactos:**
  - Botón desplegable de estado (`Todos los estados`, `En aula`, `Se trabó`, `Entregaron` si hay práctica, `Salió`, `Sin empezar`) con **conteos mutuamente excluyentes cuya suma iguala exactamente al total** (sin badge verde en `Todos` ni alerta ámbar de `sin entregar` al iniciar).
  - Fuera del menú, mostrar una píldora accionable **`N se trabó` / `N se trabaron`** **únicamente cuando `stuckCount > 0`**.
  - Buscador (`220px`, atajo `/`) a la derecha.
- **Tabla densa unificada (3 columnas reales + acciones en *hover*):**
  1. **`Alumno`:** nombre + correo en mono (+ `· Requiere atención` en ámbar si está trabado).
  2. **`Estado`:** punto semántico + estado (`En aula`, `Se trabó`, `Entregó`, `Salió`, `Sin empezar`) + veredicto humanizado e intentos inline cuando exista entrega (eliminando columnas redundantes llenas de guiones `—`).
  3. **`Actividad`:** marca temporal en mono (`hace 1 min`, `21:48`, `—`), seguida de acciones discretas al *hover* (`Revisar` si entregó; `Avance` si aplica envío puntual).
  - Las filas con estado `Se trabó` llevan barra lateral izquierda ámbar de `2px` (`--status-pending-color`) y tinte cálido muy sutil (`--row-stuck-bg`).
- **Pie tranquilo del tablero:** Muestra resumen de presencia/entregas a la izquierda y **`Actualizado HH:mm`** + botón manual de refresco a la derecha (**prohibido** segundero parpadeante `5s → 4s...`).
- **Columna derecha (`Avances emitidos` + `Historial de esta clase`):**
  - Cada emisión en `Avances emitidos` va **colapsada por defecto** (sin duplicar los nombres de toda la clase en pantalla) con hora, badge de origen (`Web` / `SOLV Companion`), pendientes/estado y barra fina de adopción; al expandir muestra el detalle por destinatario.
  - `Historial de esta clase` va **plegado por defecto**.

---

## 10. Portal del Estudiante (v1.5) — ✅ CERRADO

*Aplica a todo el recorrido del rol estudiante en SOLV: `/student` (`Inicio`), `/student/materias` (`Mis materias`), `/student/evaluaciones` (`Evaluaciones y prácticas`), `/student/courses/:id` (`Detalle de la materia`) y `/student/exercises/:id` (`Resolución en Entorno de Desarrollo` y `Resolución en Entorno de Evaluación`).*

> **Actualización en Sección 2 (Rutas con rail por defecto):** Incluir `/student`, `/student/materias`, `/student/evaluaciones`, `/student/courses/:id` y `/student/exercises/:id` en la lista de rutas donde el sidebar global inicia colapsado en modo rail (`52px`).

---

### 10.1. Reglas Generales del Portal Estudiantil

- **Shell SOLV v1.5:** Escala **Zinc** neutra en Light Mode (default) y Dark Workspace (`[data-theme="dark"]`), motor multi-tenant HSL con contraste AA (`--tenant-primary-*`), topbar global de `48px` y sidebar global iniciando en **modo rail (`52px`)**, expandible a `224px` sin perfil duplicado al pie.
  - **Grupos del sidebar:** *Espacio* (`Inicio`, `Aula` con punto verde `--status-running-color` **solo si hay sesión activa**, `Mis materias`) y *Seguimiento* (`Evaluaciones`, `Mis entregas`).
- **Redacción en español neutro formal (trato de «usted»):**
  - **Prohibido** el voseo, el tuteo y los coloquialismos (*«Trabajá en el IDE y entregá»*, *«Podés volver a entregar»*, *«No tienes entregas pendientes»*, *«aparece acá»*, *«Tus materias»*, *«Tu solución en Go»*).
  - Usar siempre trato formal neutro (*«No tiene entregas pendientes»*, *«aparece aquí»*, *«Sus materias»*, *«Su solución en Go»*).
- **Fidelidad de dominio y prohibiciones:**
  - **Prohibido** inventar secciones ajenas a SOLV (`Calendario`), horarios programados de aula (*«No tiene clase programada · Horario por confirmar»*, ya que las sesiones de Aula son abiertas en vivo por el docente) o métricas académicas inexistentes (`0% del semestre`). Mostrar siempre progreso real sobre prácticas (`N / M prácticas`, ej. `0 / 8 prácticas`).
  - **Prohibido** usar jerga docente (*«Para revisar»*) o siglas técnicas de veredicto (`AC`, `WA`, `TLE`, `RE`, `AST`). Usar únicamente veredictos humanizados en español (`Cumple`, `Pendiente de revisión`, `Con diferencias`, `Tiempo excedido`, `Error de ejecución`).
  - **Prohibido** mostrar botón `Resolver` en actividades ya completadas (usar `Ver entrega →`).
  - **Máximo 1 botón `.btn-primary` sólido por vista**, sin paredes de botones azules repetidos por fila ni enlaces azules subrayados clásicos (`Historial`, `Ver prácticas >`, `Resolver >`).

---

### 10.2. Inicio del Estudiante (`/student`)

*Centro de arranque diario del estudiante: visibilidad inmediata de aula en vivo, cierres próximos, acceso rápido a materias y resultados recientes.*

- **Cabecera compacta (`max-width: 1320px; padding: 20px 24px 12px`, sin tarjeta flotante):**
  - Izquierda: saludo formal (`Buenas noches, <Nombre>.`, `20px`, `600`, `letter-spacing: -0.02em`) + resumen de estado en 1 línea (`No tiene entregas pendientes.` o `1 clase en vivo · N prácticas con fecha límite próxima · N entrega pendiente de revisión.`). **Prohibido** colocar botones de refresco o texto `Actualizado ahora` en esta cabecera (patrón exclusivo del tablero en vivo de Aula).
  - Derecha: `Mis entregas` (`.btn-secondary`, `height: 32px`, contador en mono solo si `> 0`).
- **Layout equilibrado de 2 columnas (`minmax(0, 1fr) 340px`, `gap: 20px`, `align-items: start`):**
  - **Columna Izquierda (`1fr`) — `Aula ahora` + `Próximas entregas`:**
    1. **`Aula ahora` (bloque protagonista *border-first*, sin paneles rellenos anidados):**
       - *Sin clase abierta:* badge neutro `Sin clase abierta`, mensaje claro indicando que las prácticas publicadas siguen disponibles sin aula abierta, y **`Ver mis prácticas →`** como **único `.btn-primary`** de la vista (navega a `/student/materias`).
       - *Clase en vivo:* badge `• Clase en vivo` con pulso discreto y tamaño `--text-2xs` como único indicador de estado (sin acento lateral), código de materia en mono y nombre en `--text-base`, y una sola línea de metadatos `--text-xs` con el patrón de la cabecera del aula del docente: práctica con ícono de modalidad (`Laptop` Desarrollo / `Terminal` Evaluación) y elipsis, `·` horario y `·` presencia `N de M en el aula`. La acción es `Entrar al aula en vivo →` en `.btn-primary .btn-sm` (28px), sin botón gigante ni botón secundario redundante.
       - *Clase en pausa (`frozen`):* el badge `Congelada` usa ámbar (`--status-pending-*`), igual que el examen; `closed` y los estados desconocidos usan el tono neutro.
       - *Examen supervisado en vivo:* badge `• Examen en curso` ámbar, **`Entrar al examen →`** (`.btn-primary .btn-sm`) y franja de advertencia clara sobre restricción de internet y congelamiento en solo lectura al cerrar.
    2. **`Próximas entregas`:**
       - Encabezado con contador en mono (`0 pendientes` / `N prácticas`).
       - *Estado vacío:* ícono discreto en superficie `--bg-surface-subtle` + título (`No tiene entregas con fecha límite próxima`) y subtítulo breve (**prohibido** rellenar con listas de consejos genéricos).
       - *Con actividad:* filas *border-first* ordenadas cronológicamente con insignia compacta de fecha en mono (`22 NOV`), ícono de modalidad (`Laptop` / `Terminal`), título, código y nombre de materia, hora de cierre + urgencia (`Cierra 15:18 (Mañana)`), estado con punto semántico y acción `.btn-secondary` (`Continuar →`, `Abrir entorno →`, `Iniciar →`).
  - **Columna Derecha (`340px`) — `Sus materias` (arriba) + `Últimas entregas` (abajo):**
    1. **`Sus materias` (acceso rápido de 1 clic hacia `/student/materias`):**
       - Ubicado primero en la columna derecha por frecuencia de acceso diario.
       - Cabecera con contador en mono y acción `.btn-ghost` **`Ver todas →`** (navega a `/student/materias`).
       - Muestra hasta 4 materias: código en mono con `white-space: nowrap` (**prohibido** partir el código en dos líneas o pegarlo al nombre), nombre truncado, progreso real `N / M prácticas`, etiqueta de próximo cierre o `Al día` y mini-barra de `3.5px` (`--tenant-primary`, o `--status-running-color` si está completada).
    2. **`Últimas entregas`:**
       - Cabecera con acción `.btn-ghost` **`Historial →`** (abre el cajón lateral o navega a `/student/entregas`; **prohibido** enlace azul subrayado).
       - *Estado vacío:* ícono discreto `inbox`, título `Aún no registra entregas` y subtítulo breve.
       - *Con actividad:* hasta 3 entregas recientes con título, puntaje en mono (`100 / 100` cuando exista), veredicto humanizado con punto semántico y `código · fecha` en mono.

---

### 10.3. Mis Materias (`/student/materias`)

*Catálogo personal de materias con filtrado por avance, búsqueda y conmutador de formato.*

- **Cabecera compacta (`max-width: 1320px; padding: 20px 24px 12px`):**
  - Izquierda: `← Inicio` (`.btn-ghost`, `28px`) + divisor `1px × 16px` + `Mis materias` (`20px`, `600`) + subtítulo formal (*«Ingrese a una materia para consultar sus prácticas, su avance y los cierres próximos.»*).
  - Derecha: `Mis entregas` (`.btn-secondary`, `32px`). Sin botón `.btn-primary` forzado en esta vista de exploración.
- **Barra única de controles (`border-bottom: 1px solid var(--border-subtle); padding-bottom: 12px`):**
  - Izquierda: control segmentado de estado (`Todas`, `En curso`, `Completadas`) con **contadores neutros en mono** (**prohibido** fondo/badge verde en `Todas`).
  - Derecha: buscador (`width: 240px`, atajo `/`, *«Buscar por nombre o código...»*) + conmutador **`Tarjetas | Lista`**.
- **Modo `Tarjetas` (`display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px` en `xl`):** Tarjetas limpias *border-first* sobre `--bg-surface` (`border: 1px solid var(--border-subtle); border-radius: 8px; padding: 16px`). **Prohibido** borde izquierdo grueso gris y enlaces azules clásicos `Ver prácticas >`. Estructura interna: código mono + badge de estado (`En curso`, `Completada`, `En vivo` o `Examen`), nombre de la materia (`15px`, `600`), docente, barra de progreso real (`N de M prácticas` + porcentaje mono + barra `4px`), y pie separado por `1px` con próximo cierre + botón `.btn-ghost` `Ver prácticas →`.
- **Modo `Lista`:** Contenedor único *border-first* dividido por líneas de `1px` mostrando código mono, nombre (con punto verde si hay aula en vivo), docente + cierre próximo, barra compacta de avance y botón `.btn-ghost` `Ver prácticas →`.

---

### 10.4. Evaluaciones y Prácticas (`/student/evaluaciones` — Radar Transversal por Prioridad)

*Responde a «¿qué exámenes tengo, qué vence pronto y qué dejé a medias en todas mis materias?». Estructuralmente distinta al Detalle de la materia.*

- **Cabecera con resumen real y 1 único `.btn-primary`:**
  - Izquierda: `Evaluaciones y prácticas` (`20px`, `600`) + resumen con pluralización exacta (`N exámenes próximos · N en curso · N completadas en todas sus materias.`).
  - Derecha: `Mis entregas` (`.btn-secondary`) + **único `.btn-primary`** específico para retomar la actividad más urgente (ej. **`Continuar INF-301 →`** con el título completo en `title` / tooltip).
- **Barra única de búsqueda y filtros (`border-bottom: 1px solid var(--border-subtle)`):**
  - Reemplaza las 5 pestañas + 2 `<select>` anteriores por: buscador (`280px`, atajo `/`) + botón único **`Filtros`** con **popover unificado de 3 columnas (`Estado` | `Materia` | `Tipo`)** + conteo en mono (`N de M actividades`).
  - Chips removibles visibles debajo únicamente cuando existan filtros activos, con acción `Limpiar todo`.
- **Jerarquía en 3 bloques reales (prohibido apilar tarjetas sueltas gigantes, bloques `INTENTOS 0` o botones azules repetidos):**
  1. **`Exámenes y cierres próximos`:** filas *border-first* ordenadas por fecha de cierre con insignia de fecha mono a la izquierda (`22 NOV`), urgencia (`Hoy`, `Mañana`, `En 2 días`), acento lateral izquierdo ámbar de `3px` + tinte cálido sutil (`--row-exam-tint`) **exclusivo para exámenes**, código mono de materia, ícono de modalidad (`Laptop` / `Terminal`), estado humanizado + entregas si `> 0`, y botón `.btn-secondary` acorde (`Iniciar examen →`, `Continuar →`, `Resolver →`).
  2. **`Prácticas abiertas · Sin fecha límite`:** filas compactas y serenas debajo, sin competir visualmente con los exámenes ni cierres próximos.
  3. **`Completadas`:** bloque **plegado por defecto** (se expande automáticamente si el usuario filtra por *Completadas*) con puntaje (`100 / 100`) y **acceso dual**: clic en la fila abre la consigna/código para consulta y el botón derecho **`Ver entrega →`** abre el historial/detalle.

---

### 10.5. Detalle de la Materia (`/student/courses/:id` — Hub de Materia en 2 Columnas)

*Responde a «¿cómo avanzo en esta materia, qué unidades tiene y cuál es mi siguiente paso aquí?».*

- **Cabecera plana (`border-bottom: 1px solid var(--border-subtle); background: var(--bg-surface)`):**
  - Izquierda: `← Mis materias` (`.btn-ghost`) + divisor `1px` + código mono (`SIS-12`) + nombre (`19px`, `600`) + docente y conteo publicado en mono. **Prohibido** incluir botón `Actualizar` innecesario o textos negativos internos (*«El docente no definió módulos»*).
  - Derecha: selector discreto de cambio rápido entre materias inscritas.
- **Layout de 2 columnas (`minmax(0, 1fr) 320px`, `gap: 24px`, `align-items: start`):**
  - **Columna Izquierda (`1fr` — Recorrido por Unidades):**
    - Control segmentado interno (`Todas`, `Pendientes`, `Exámenes`, `Completadas`) con contadores neutros en mono.
    - Actividades agrupadas en contenedores *border-first* por **Unidad / Módulo** con progreso por unidad (`N / M completadas`). Si el docente no definió módulos, se renderiza un único contenedor limpio *border-first* sin subtítulos de aviso.
    - Cada fila muestra nodo visual de recorrido (`✓` verde completada, punto ámbar en curso, aro neutro pendiente), título, badge **`Examen`** en ámbar **reservado exclusivamente a exámenes** (sin colisión amarilla con dificultad), modalidad (`Entorno de Desarrollo` / `Entorno de Evaluación`), cierre en mono, estado humanizado y acción acorde (acceso dual en completadas).
  - **Columna Derecha Sticky (`320px`, `top: 68px`) — Panel de la Materia:**
    1. **Tarjeta `Siguiente paso en <Código>`:** destaca la práctica en curso o el próximo cierre de esa materia con el **único `.btn-primary`** de la vista (`Continuar práctica →` / `Ir al examen →`).
    2. **Bloque *cardless* `Avance de la materia`:** porcentaje y barra real (`N / M`), tabla clave-valor con desglose (`Completadas`, `En curso`, `Exámenes próximos`, `Sin empezar`) y estado de `Aula en vivo` de esa materia.

---

### 10.6. Resolución — Entorno de Desarrollo (`/student/exercises/:id` · `IDE_PERSISTENTE`)

*Espejo estudiantil de la Ficha de Práctica docente: expone consigna limpia, rúbrica oficial antes de abrir VS Code y gestión unificada del contenedor.*

- **Cabecera plana en 1 fila (sin tarjeta flotante):** `← <Materia>` + título + badge `Examen` (si aplica), y debajo `Entorno de Desarrollo`, fecha límite y stack en mono.
- **Layout asimétrico (`minmax(0, 1fr) 340px`, `gap: 24px`):**
  - **Columna Izquierda (`1fr`):**
    - *Solo si es Examen:* aviso ámbar sobrio de 2 líneas indicando que el entorno no tiene acceso a internet y que al finalizar el plazo permanecerá en solo lectura.
    - **`Consigna de la práctica`:** Markdown limpio directo sobre `--bg-surface` (**prohibido** duplicar tarjetas de instrucciones genéricas o mezclar voseo).
    - **`Rúbrica de evaluación`** (consumiendo `GET /api/v1/exercises/{id}/rubric` cuando exista): barra proporcional interactiva de 100 % vinculada por *hover* a cada criterio y su descripción, en modo solo lectura, anticipando en web los criterios que también muestra **SOLV Companion** en VS Code.
  - **Columna Derecha Sticky (`340px`):**
    - **Una sola tarjeta operativa (`Espacio de trabajo y entrega`):** elimina las dos tarjetas separadas anteriores. Muestra estado real (`Sin iniciar`, `En ejecución`, `Solo lectura`), **`Abrir entorno`** como **único `.btn-primary`** (baja a `.btn-secondary` **`Abrir entorno (solo lectura)`** cuando un examen cerró) y **`Entregar proyecto`** como `.btn-secondary` (deshabilitado con nota breve mientras no se haya iniciado el entorno; dispara `confirm-modal` y registra entrega + última fecha).
    - **Bloque *cardless* `Detalles de entrega`:** `Reenvíos`, `Entregas registradas`, `Fecha límite` y `Acceso a red` (en examen).

---

### 10.7. Resolución — Entorno de Evaluación (`/student/exercises/:id` · `JUEZ_EFIMERO`)

*Workbench de altura completa construido sobre la arquitectura desacoplada de **2 Primitivas (`Probar código` vs. `Enviar a evaluación` / `Entregar examen`)**.*

- **Cabecera plana compacta (`48px`, sin tarjeta flotante):**
  - Izquierda: `← <Materia>` + título truncado + badge `Examen` (si aplica) + `Límite N s` integrado en mono.
  - Derecha: selector de `Lenguaje` (si admite varios), `Reiniciar` (`.btn-ghost`), **`Probar código`** (`.btn-secondary`, atajo `Ctrl+Enter`) y **`Enviar a evaluación`** —o **`Entregar examen`** en modo examen— como **único `.btn-primary`** al extremo derecho.
- **Correcciones técnicas obligatorias:**
  - Parsear Markdown y código inicial sin saltos literales `\n` ni encabezados `###` crudos.
  - Mostrar extensiones de archivo reales (`solucion.py`, `solucion.go`, `solucion.cpp` — **prohibido** `solucion.python`).
- **Workbench Split-Pane (`height: calc(100vh - 96px)` en desktop):**
  - **Panel Izquierdo (~42 %, scroll independiente):** aviso claro de modo examen (explicando la política vigente), `Descripción del problema` en Markdown y bloque `Casos de ejemplo` con `STDIN` / `STDOUT` en monospace, botón `Copiar entrada` y contador `+ N ocultos en evaluación`.
  - **Panel Derecho (~58 %):**
    - Sub-barra del archivo (`solucion.py`, indicador `Solo lectura · Examen entregado` si está bloqueado, y posición `Ln, Col`).
    - Editor de código ocupando todo el espacio vertical disponible con soporte `Tab` y `Ctrl+Enter`.
    - **Consola inferior plegable (`248px` abierta / `38px` cerrada) con 2 pestañas diferenciadas:**
      1. **`Prueba rápida` (Primitiva `Run` / Sandbox efímero):** ejecuta únicamente contra casos de ejemplo sin insertar filas en `submissions` ni consumir intentos.
         - Muestra franja explícita **`Ejecución de prueba · No registra entrega`**.
         - **Salvaguarda anti-olvido:** si todos los ejemplos aprueban y aún no existe entrega oficial registrada, la franja cambia a tono ámbar con el mensaje **`Ejemplos cumplen · Falta enviar la entrega oficial`**.
         - Detalla `Esperado`, `Su salida` y errores de compilación/`STDERR`.
      2. **`Resultado oficial` (Primitiva `Submit` formal):** ejecuta la suite completa (ejemplos + públicos + ocultos), registra la entrega, actualiza puntaje (`N / 100`) y estado en listados/historial, y muestra veredicto humanizado, timestamp, tiempo en `ms`, barra de casos aprobados y resumen de ocultos (`N / M casos ocultos aprobados`) sin revelar sus entradas.
- **Gobernanza de Examen:** al pulsar **`Entregar examen`**, abre `confirm-modal` indicando explícitamente si el examen permite **reenvíos hasta la fecha límite (`due_date`)** o si es de **intento único** (bloqueo inmediato del editor en solo lectura tras confirmar).

---

### 10.8. Paneles Deslizables Complementarios (`Drawer`)

- **Historial de entregas (`400px`–`420px`, `border-left: 1px solid var(--border-strong)`):** accesible desde `Mis entregas`, `Historial →` o `Ver entrega →`. Lista todas las entregas con código/fecha en mono, título, puntaje, veredicto humanizado y `Ver detalle →`, o estado vacío limpio.
- **Vista rápida de prácticas por materia:** transición ligera desde las tarjetas de `Inicio` o `Mis materias` hacia el detalle de la materia (`/student/courses/:id`).
