---
name: solv-design-system
description: Sistema de diseño, tokens, tipografía, paleta semántica y componentes de UI para SOLV.
---

# SOLV — Sistema de Diseño de UI

## Filosofía
- Herramienta de ingeniería sólida: Claridad > Decoración. Content-first. Consistencia > Novedad.
- PROHIBIDO: gradientes excesivos, cards redondeadas gigantes, sombras pesadas, animaciones decorativas, KPI cards vistosos, ilustraciones o estética "AI SaaS".

## Tipografía y Espaciado
- Fuente UI Principal: Inter.
- Fuente Datos/Código: JetBrains Mono (para UUIDs, URLs, métricas, veredictos, RAM/CPU).
- Escala de espaciado: múltiplos de 4px (4px, 8px, 12px, 16px, 24px, 32px).
- Radios de borde: 6px, 8px, 12px máximos.

### Contrato Tipográfico (personas vs. máquina)
- Inter para personas (todo texto de UI); JetBrains Mono SOLO para datos de máquina: imágenes Docker, RAM/CPU, UUIDs, logs, código.
- Escala: `xs` 12px / `sm` 14px / `base` 16px / `lg` 18px / `xl` 20px (`var(--text-xs)` … `var(--text-xl)`).
- Tablas: cabeceras en `xs`, mayúsculas, `var(--text-secondary)`.
- Cards: texto `base`, `600`.

### Jerarquía Tipográfica de Cabeceras
- **Título de Página (H1, `.page-title`, `.view-title`):** `font-size: var(--text-xl)` (20px / 1.25rem), `font-weight: 700`, `color: var(--text-primary)`, `letter-spacing: -0.01em`, `line-height: var(--leading-tight)`.
- **Subtítulo de Página (`.page-subtitle`, `.view-subtitle`):** `font-size: var(--text-sm)` (13-14px / 0.875rem), `color: var(--text-secondary)`, margen superior 2px, `line-height: var(--leading-normal)`.
- **Título de Sección (H2, `.section-title`):** `font-size: var(--text-lg)` (18px / 1.125rem), `font-weight: 600`, `color: var(--text-primary)`.
- **Título de Tarjeta / Modal (H3, `.card-title`):** `font-size: var(--text-base)` (16px / 1rem), `font-weight: 600`, `color: var(--text-primary)`.
- **Subsección / Pasos (H4, `.step-section-title`):** `font-size: var(--text-sm)` (14px) o `var(--text-base)`, `font-weight: 600`.

### Canon de Modal (obligatorio, sin excepciones)
Todo modal del admin replica esta estructura. PROHIBIDO inventar otra:
- Backdrop: `rgba(15, 23, 42, 0.5)`, `blur(2px)`, `padding: var(--space-4)`.
- Caja: `var(--bg-surface)`, `var(--radius-lg)`, `var(--shadow-dropdown)`, `max-width: 540px` (wizard: lo que pida, pero mismo shell).
- Header: `var(--space-4) var(--space-5)`, sin fondo, borde inferior `var(--border-subtle)`.
- Icono: círculo 36px, `var(--tenant-primary-subtle)` + `var(--tenant-primary)` para info; fondo/color semántico (`failed/pending/running`) según intención (peligro, aviso, éxito). PROHIBIDO cuadrado 8px.
- Título: `var(--text-base)`, `600`, `var(--text-primary)`. Subtítulo: `var(--text-sm)`, `var(--text-secondary)`.
- Body: `var(--space-5)`, `gap: var(--space-4)`.
- Label: `var(--text-xs)`, `600`, `var(--text-primary)`. Input: `var(--text-sm)`, `var(--border-strong)`, `var(--radius-sm)`, foco con `var(--border-focus)` + `0 0 0 2px var(--tenant-primary-border)`.
- Footer: `var(--space-4) var(--space-5)`, `gap: var(--space-3)`, `var(--bg-surface-subtle)`, borde superior.
- Botones: cancelar `var(--text-sm)` `500` `var(--text-secondary)` sobre `var(--bg-surface)` con `var(--border-strong)`; primario `600` `var(--text-inverse)` sobre `var(--tenant-primary)`, hover `var(--tenant-primary-hover)`. PROHIBIDO azul fijo (`#2563EB`, `--color-primary-*`): rompe el white-label.

### Primitivos Compartidos (usar siempre, PROHIBIDO duplicar su SCSS)
- `modal-shell` (`shared/components/modal-shell`): `title*`, `subtitle?`, `intent?: info|success|warning|error` (color del icono), `maxWidth?=540px`, `showClose?=true`, `(close)`. Proyecta icono con `<svg modal-icon ...>`, cuerpo directo y pie con `<ng-container modal-footer>`. El shell es dueño de Escape y click en backdrop: el modal hijo NO registra su propio `HostListener Escape`.
- Intención semántica de `modal-shell`: `info` = creación o flujo informativo (azul tenant). `warning` / `success` / `error` solo ante intención semántica real (aviso, confirmación de éxito, error). El hijo solo proyecta el `svg` con `modal-icon`; NUNCA define color o forma de icono propia.
- `form-field` (`shared/components/form-field`): `label*`, `for?`, `required?`, `hint?`, `error?` (error pisa a hint). Envuelve UN control (input o `combobox`).
- Controles sueltos usan las clases globales de `styles.scss` (`.form-input`, `.form-textarea`, `.form-select`, `.input-with-icon`, `.field-icon`, `.btn-primary`, `.btn-outline`, `.form-row`): PROHIBIDO redefinirlos en SCSS de vista/modal.

### Botones Globales (contrato único)
- Existe UN solo `.btn-primary` y UN solo `.btn-outline` globales. Primario: `600`, `var(--text-inverse)` sobre `var(--tenant-primary)`, hover `var(--tenant-primary-hover)`. Cancelar: `500`, `var(--text-secondary)` sobre `var(--bg-surface)` con `var(--border-strong)`.
- PROHIBIDO redefinir botones por vista o modal (clases locales tipo `.btn-cancel` / `.btn-submit`, u overrides con `brightness()` / `opacity()` sobre los globales).

### Superficies de Código / Terminal
Visores de logs, auditoría dry-run y previsualizaciones usan la escala `--code-*` de `_primitives.scss` (`--code-bg`, `--code-bg-deep`, `--code-border`, `--code-text`, `--code-muted`, `--code-accent`, `--code-warning`, `--code-error`). Niveles de log: info=`--code-accent`, warn=`--code-warning`, error/stderr=`--code-error`. NUNCA grises del tema claro sobre fondo oscuro.
- Hover de primario sólido: `--color-success-dark` (valor sancionado por este sistema).

### Controles de Selección (`select`, `option`)- Obligatorio `font-family: var(--font-sans)` en todo `select`, `.form-select`, `.filter-select` y en sus `option` / `optgroup`.
- Native selects deben usar `appearance: none` con el icono SVG de chevron vectorizado (`stroke: #64748B`) y padding derecho para evitar que el motor de renderizado del sistema operativo sustituya la tipografía web por fuentes del sistema.

### Combobox (contrato cerrado/abierto)
- Estado cerrado: tipografía Inter `var(--text-sm)`, idéntica a `.form-input`.
- Lista abierta: opciones en Inter `var(--text-xs)`; metadatos secundarios (descripciones, contadores) con color muted.
- TODO resuelto por tokens (`var(--text-*)`, `--text-secondary`/`--text-muted`). PROHIBIDO `var(--x, #hex)` en fallbacks: todo fallback con hexadecimal viola la regla `color-no-hex`; usar token puro.

### Padding de Contenedor de Vistas
- El cascarón principal (`.shell-content`) ya aplica `padding: var(--space-6, 24px)`.
- PROHIBIDO agregar `padding: 24px` en el contenedor raíz de una vista hija (ej. `.model-library-container`), ya que genera doble espaciado (48px) y desalinea la cabecera respecto a las demás vistas.

## Color y Tema
- Tema Claro por defecto: Fondo `#F6F7F9`, Superficie `#FFFFFF`, Bordes `gray-200`.
- Separación visual mediante bordes limpios, no mediante sombras.
- Primario White-Label: `var(--tenant-primary)` (Default `#2563EB`), provisto dinámicamente por `/api/v1/config/public`.
- Semánticos Fijos:
  - Success: `#16A34A` / `#15803D`
  - Warning: `#D97706`
  - Error: `#DC2626`
  - Neutral: `gray-500`

- **REGLA ESTRICTA DE FEEDBACK Y TOASTS (PROHIBIDO AZUL O SLATE):**
  - Los toasts/tooltips de éxito o confirmación NUNCA deben tener fondo azul, slate ni azul oscuro (`#0F172A`, `#1E293B`, `#2563EB`).
  - Deben ser SIEMPRE color VERDE semántico (`#15803D` con texto blanco) para feedback positivo.
  - Toasts de error: SIEMPRE color ROJO semántico (`#DC2626` con texto blanco).

## Semántica de Estados (Workspaces y Juez)
- `running`: Success (Verde)
- `pending`: Warning (Ámbar)
- `hibernated`: Neutral (Gris)
- `failed` / `oom_killed`: Error (Rojo)
- `terminated`: Neutral (Gris)

## Veredictos del Juez
- `AC`: Verde (`#16A34A`)
- `WA`: Rojo (`#DC2626`)
- `TLE`: Ámbar (`#D97706`)
- `RE`: Naranja (`#EA580C`)
- `AST_BLOCKED`: Rojo Intenso (`#991B1B`)

## Inventario de Componentes UI Obligatorios
- `StatusBadge`: Estado de workspaces con animación pulse sutil para `running`.
- `VerdictBadge`: Veredictos de calificación del juez.
- `ResourceMeter`: Barras de consumo de CPU y RAM.
- `IframeWrapper`: Contenedor seguro para OpenVSCode Server.
- `MonacoWrapper`: Componente nativo encapsulado cargado dinámicamente con `@defer`.
- `EmptyState`: Mensajes de estado vacío con máx 1 animación Lottie (<100KB, lazy).

## Leyes de Interfaz (UX)
1. **Dualidad de Estados:** Estado académico (entrega) y técnico (contenedor) visibles en la misma tarjeta.
2. **Carga Cognitiva Cero:** Errores de Docker/OOM traducidos a mensajes humanos y accionables.
3. **Disclosure Progresivo:** Logs y métricas avanzadas ocultas hasta que el usuario las solicita.
4. **Inmutabilidad en Revisión:** Bloqueo de edición docente respaldado en backend (`:ro`), no solo en CSS.
5. **Comunicación de Frescura:** Indicadores transitorios explícitos ("Reconectando...", "Guardado hace 2s").

## Regla de Componentes
Antes de crear cualquier componente UI nuevo, verificar si el Inventario existente ya resuelve la necesidad.
Queda estrictamente PROHIBIDO usar el prefijo `solv-` en selectors o clases de componentes (`selector: 'solv-...'` o `Solv...Component`). Utilizar nombres semánticos limpios en kebab-case para el selector (ej. `kpi-card`, `search-bar`, `status-tabs`, `pagination-bar`, `view-switcher`).

## Reglas Blindadas por Auditoría
Contrato normativo. Ningún cambio posterior puede derivar de estas reglas sin actualizar esta sección en el mismo cambio.
- Vocabulario de flujos multipaso: PROHIBIDA la palabra "asistente" para wizards (desde 2026 se lee como IA). Usar siempre "wizard" en código, UI visible y documentación.
- Opción fantasma PROHIBIDA: ningún `select` o `combobox` puede incluir una opción del tipo `-- Sin categoría asignada --` (ni variantes) como opción o valor. Campo vacío con `placeholder` + `required` + validación de frontend alineada al backend: si el backend exige el campo, el frontend bloquea antes del 422 con el mismo mensaje.
- Regla de salida del ignore: limpiar el archivo y sacarlo de `.stylelintignore` / `ignoreFiles` en el MISMO cambio. Nunca todo de golpe: un archivo por cambio, verificado con el linter.
