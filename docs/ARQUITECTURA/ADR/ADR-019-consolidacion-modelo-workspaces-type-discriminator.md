# ADR-019: Consolidación del Modelo de Dominio de Workspaces y Discriminador `type` (D3)

* **Estado:** Aceptado (Supersedes y elimina la tabla `lab_instances` de ADR-000)
* **Fecha:** 2026-08-04

## Contexto y Problema
En las etapas iniciales del proyecto existían dos tablas con atributos y propósito casi idénticos: `workspaces` (para entornos interactivos persistentes VS Code) y `lab_instances` (para ejecuciones efímeras del juez virtual). Esta duplicidad generaba redundancia en la capa de persistencia, mantenimiento de repositorios duplicados (`lab_instance_repository.go` y `workspace_repository.go`) e inconsistencias semánticas.

## Decisión Tomada
1. Adoptar la decisión de arquitectura **D3**: Consolidar ambos conceptos en la tabla unificada **`workspaces`** mediante la adición de la columna discriminadora `type VARCHAR(50) NOT NULL DEFAULT 'IDE_PERSISTENTE'`.
2. Definir los valores de tipo en el dominio:
   * `IDE_PERSISTENTE`: Entornos interactivos OpenVSCode Server de larga duración.
   * `JUEZ_EFIMERO`: Entornos de evaluación de código o bases de datos de corta duración.
3. Migrar automáticamente los registros existentes de `lab_instances` a `workspaces` con `type = 'JUEZ_EFIMERO'` y eliminar físicamente la tabla `lab_instances` (`DROP TABLE IF EXISTS lab_instances CASCADE`).
4. Implementar el método de repositorio `GetByType(ctx, workspaceType)` y remover los repositorios legacy.

## Consecuencias
* **Positivas:**
  * Fuente de verdad única e inambigua para toda la orquestación de contenedores en el sistema.
  * Eliminación de deuda técnica en la capa de infraestructura y controladores HTTP.
  * Simplificación de las consultas relacionales y contratos de la API.
