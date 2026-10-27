# Índice de Decisiones de Arquitectura (ADRs) — Slice 17

Este Slice implementa la veracidad de la telemetría del panel de salud y recursos, respaldado por:

* [ADR-049: Telemetría de Host Persistida y Estados de Reconciliación (unknown / orphaned)](../../ARQUITECTURA/ADR/ADR-049-telemetria-de-host-persistida-y-estados-de-reconciliacion.md)
* [ADR-008: Arquitectura Hexagonal y Puertos de Dominio](../../ARQUITECTURA/ADR/ADR-008-hexagonal-ports-and-adapters.md) *(El puerto `domain.HostTelemetry` sigue este patrón)*
* [ADR-043: Modalidad Dual y Gobernanza de Recursos](../SLICE_12/01-ADR.md) *(El techo de seguridad y la política de capacidad vienen del catálogo)*