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

## Color y Tema
- Tema Claro por defecto: Fondo `#F6F7F9`, Superficie `#FFFFFF`, Bordes `gray-200`.
- Separación visual mediante bordes limpios, no mediante sombras.
- Primario White-Label: `var(--tenant-primary)` (Default `#2563EB`), provisto dinámicamente por `/api/v1/config/public`.
- Semánticos Fijos:
  - Success: `#16A34A`
  - Warning: `#D97706`
  - Error: `#DC2626`
  - Neutral: `gray-500`

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
