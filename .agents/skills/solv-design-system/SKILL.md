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

### Jerarquía Tipográfica de Cabeceras
- **Título de Página (H1, `.page-title`, `.view-title`):** `font-size: var(--text-xl)` (20px / 1.25rem), `font-weight: 700`, `color: var(--text-primary)`, `letter-spacing: -0.01em`, `line-height: var(--leading-tight)`.
- **Subtítulo de Página (`.page-subtitle`, `.view-subtitle`):** `font-size: var(--text-sm)` (13-14px / 0.875rem), `color: var(--text-secondary)`, margen superior 2px, `line-height: var(--leading-normal)`.
- **Título de Sección (H2, `.section-title`):** `font-size: var(--text-lg)` (18px / 1.125rem), `font-weight: 600`, `color: var(--text-primary)`.
- **Título de Tarjeta / Modal (H3, `.card-title`):** `font-size: var(--text-base)` (16px / 1rem), `font-weight: 600`, `color: var(--text-primary)`.
- **Subsección / Pasos (H4, `.step-section-title`):** `font-size: var(--text-sm)` (14px) o `var(--text-base)`, `font-weight: 600`.

### Controles de Selección (`select`, `option`)
- Obligatorio `font-family: var(--font-sans)` en todo `select`, `.form-select`, `.filter-select` y en sus `option` / `optgroup`.
- Native selects deben usar `appearance: none` con el icono SVG de chevron vectorizado (`stroke: #64748B`) y padding derecho para evitar que el motor de renderizado del sistema operativo sustituya la tipografía web por fuentes del sistema.

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
