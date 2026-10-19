---
name: gobernanza
description: Reglas de gobernanza, control de cambios y estándar de commits para el proyecto SOLV.
---

# Gobernanza y Control de Cambios

Esta skill define las reglas obligatorias de gobernanza y trazabilidad para cualquier modificación sobre la documentación de arquitectura, ADRs, código y slices del proyecto SOLV.

## Reglas de Commits (Conventional Commits Estilo Humano)

Esta sección es la **fuente única de verdad** del estándar de commits de SOLV. AGENTS.md y CONTRIBUTING.md referencian este documento; no duplican estas reglas.

- Formato: `<tipo>(<ámbito>): <descripción corta en minúsculas>`
- Cuerpo Obligatorio (1-2 líneas):
  - `- Qué se hizo`: Explicación concisa.
  - `- Por qué se hizo`: Motivación técnica o de negocio.
  - `- Cómo lo soluciona`: Implementación.
  - `- Impacto`: Efecto secundario o beneficio.
- Tono: primera persona del plural o impersonal técnico: "Se agregó X", "Agregamos X", "Agregado X".
- Frases cortas. Evitar subordinadas largas.
- Números concretos mejor que adjetivos (ej. "5 tests pasan").
- Sin superlativos ni marketing ("excelente", "óptimo", "poderoso", "elegante").
- Usar lenguaje de dominio puro (ej. `feat(exercises): implementar importacion masiva`).
- Footers Obligatorios:
  - `Slice: N` — solo el número de carpeta `docs/SLICES/SLICE_N/` (trazabilidad documental).
  - `ADRs: ADR-XXX` (si aplica)
- PROHIBIDO: Usar nomenclatura interna "CRIT-XX" o términos de delator de IA (ej. "suite de pruebas", "veredicto PASS", "cobertura automatizada", "evidencias empíricas", "robustecer").
- PROHIBIDO: Cualquier etiqueta de coordinación interna en la descripción o cuerpo: `Slice F1`, `Cajón A-D`, `OA-XX`, `Adendas 1-6`, `INV-*`, `PU-*`, `MO-*`, `ST-*`, `PB-*`, `TE-*`, `D1-D5`. El footer `Slice: N` es trazabilidad estructural, no vocabulario de coordinación: ambas reglas conviven sin conflicto.
- PROHIBIDO: Atribución de IA (`Co-Authored-By: AI...` o similar) y emojis en mensajes de commit.
- PROHIBIDO EMOJIS: Queda estrictamente prohibido el uso de emojis en cualquier archivo de documentación Markdown (.md), código fuente Go/TypeScript, o comentarios. Utilizar texto plano formal, diagramas Mermaid o referenciar Lucide Icons (`lucide:name`).
- PROHIBIDO PREFIJO SOLV EN SELECTORES: Los selectores y clases de componentes de Angular deben usar nombres de dominio limpios (`kpi-card`, `search-bar`, `status-tabs`, `view-switcher`, etc.), nunca prefijados con `solv-` ni clases `Solv*Component`.

## Reglas de Trazabilidad Documental

- **Actualización de ADRs**: Todo ADR nuevo o cambio de estado debe reflejarse en `docs/ARQUITECTURA/ADR/INDEX.md` en el mismo commit.
- **Trazabilidad de Slices**: Cambios en arquitectura o base de datos deben registrarse en la carpeta del Slice correspondiente (`docs/SLICES/SLICE_N/`).
- **Documentación en Cambios Relevantes**: Cada modificación estructural, adición de capacidades (ej. soporte SQLite / optimizaciones de BD) o ajuste de contratos debe documentarse en la carpeta correspondiente (`docs/ARQUITECTURA/`, `docs/SLICES/` o especificaciones activas) antes de dar por cerrada la tarea.
