---
name: gobernanza
description: Reglas de gobernanza, control de cambios y estándar de commits para el proyecto SOLV.
---

# Gobernanza y Control de Cambios

Esta skill define las reglas obligatorias de gobernanza y trazabilidad para cualquier modificación sobre la documentación de arquitectura, ADRs, código y slices del proyecto SOLV.

## Reglas de Commits (Conventional Commits Estilo Humano)

- Formato: `<tipo>(<ámbito>): <descripción corta en minúsculas>`
- Cuerpo Obligatorio:
  - `- Qué se hizo`: Explicación concisa.
  - `- Por qué se hizo`: Motivación técnica o de negocio.
  - `- Cómo lo soluciona`: Implementación.
  - `- Impacto`: Efecto secundario o beneficio.
- Footers Obligatorios:
  - `Slice: N`
  - `ADRs: ADR-XXX` (si aplica)
- PROHIBIDO: Usar nomenclatura interna "CRIT-XX" o términos de delator de IA (ej. "suite de pruebas", "veredicto PASS", "cobertura automatizada", "evidencias empíricas", "robustecer").
- PROHIBIDO EMOJIS: Queda estrictamente prohibido el uso de emojis en cualquier archivo de documentación Markdown (.md), código fuente Go/TypeScript, o comentarios. Utilizar texto plano formal, diagramas Mermaid o referenciar Lucide Icons (`lucide:name`).

## Reglas de Trazabilidad Documental

- **Actualización de ADRs**: Todo ADR nuevo o cambio de estado debe reflejarse en `docs/ARQUITECTURA/ADR/INDEX.md` en el mismo commit.
- **Trazabilidad de Slices**: Cambios en arquitectura o base de datos deben registrarse en la carpeta del Slice correspondiente (`docs/SLICES/SLICE_N/`).
