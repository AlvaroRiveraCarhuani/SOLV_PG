---
name: gobernanza
description: Reglas de gobernanza y control de cambios para el proyecto SOLV.
---

# Gobernanza y Control de Cambios

Esta skill define las reglas obligatorias de gobernanza y trazabilidad para cualquier modificación sobre la documentación de arquitectura, ADRs y slices del proyecto SOLV.

## Reglas Permanentes

* **Actualización del Índice de ADRs:** Todo ADR nuevo o cambio de estado de un ADR existente debe actualizar `docs/ARQUITECTURA/ADR/INDEX.md` en el mismo commit de la documentación.
* **Trazabilidad de Slices:** Cualquier modificación a la arquitectura o base de datos debe documentarse en el Slice de origen en su correspondiente archivo `01-ADR.md`.
