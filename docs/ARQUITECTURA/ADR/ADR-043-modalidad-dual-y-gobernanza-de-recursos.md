# ADR-043: Modalidad Dual y Gobernanza de Recursos vía Catálogo de Plantillas

## Estado
Aprobado

## Fecha
2026-10-09

## Contexto
SOLV es una plataforma de orquestación de laboratorios que soporta dos modalidades de ejecución en runtime: evaluación mediante juez virtual efímero (`JUEZ_EFIMERO`) y entornos interactivos persistentes OpenVSCode (`IDE_PERSISTENTE`).

Previamente a esta decisión:
1. La tabla `exercises` no discriminaba formalmente la modalidad de runtime del laboratorio (`environment_type`).
2. No existía la obligatoriedad de vincular cada ejercicio a una plantilla homologada (`template_id`).
3. El campo `memory_limit_mb` estaba expuesto en la API para los docentes, permitiéndoles especificar asignaciones de RAM arbitrarias. Esto violaba la regla de gobernanza donde la asignación de recursos es responsabilidad exclusiva del administrador técnico mediante el catálogo `lab_templates`.

## Decisión

Establecer la gobernanza estricta de recursos y la clasificación de modalidad dual basada en plantillas:

1. **Esquema de Base de Datos y Migración de Backfill (`00020_dual_modality_and_template_governance.sql`)**:
   - Incorporación de `environment_type VARCHAR(50) NOT NULL DEFAULT 'JUEZ_EFIMERO'` con restricción `CHECK (environment_type IN ('JUEZ_EFIMERO', 'IDE_PERSISTENTE'))`.
   - Incorporación de `template_id UUID NOT NULL REFERENCES lab_templates(id) ON DELETE RESTRICT`.
   - Sembrado idempotente de plantillas runner base (`target_environment='JUEZ_EFIMERO'`, `status='approved'`, `description='SYSTEM_SEED_RUNNER'`) para cada lenguaje soportado (python, java, cpp, go, javascript, csharp), calculando `base_ram_mb` mediante la moda (`MODE()`) del historial de ejercicios (con fallback a 128 MB).
   - Backfill de todos los ejercicios existentes a `JUEZ_EFIMERO` vinculados a la plantilla runner correspondiente de su lenguaje.

2. **Capa de Dominio y Aplicación (`EvaluationService`)**:
   - En creación, edición y publicación de ejercicios: `EvaluationService` resuelve la plantilla homologada (vía `template_id` explícito o implícito por lenguaje para juez).
   - Valida que `template.status == 'approved'` y `template.target_environment == exercise.environment_type`.
   - Sobreescribe `exercise.MemoryLimitMB` con `template.BaseRamMB` antes de guardar.
   - Adjunta la ficha de resumen (`TemplateSummary`) a la respuesta de la API.

3. **Contrato de API HTTP (`EvaluationHandler`)**:
   - Si una petición para crear o modificar un ejercicio contiene el campo `memory_limit_mb`, el servidor la rechaza inmediatamente con estado **HTTP 422 (`memory_governed_by_template`)**.

## Consecuencias

- **Positivas**:
  - Control total y gobernanza del administrador sobre la RAM asignada a los entornos.
  - Cero cambios requeridos en el motor de ejecución Docker ni en los comparadores.
  - Separación clara de responsabilidades: los administradores gobiernan la infraestructura y los docentes componen el material académico.
- **Negativas**:
  - Todo ejercicio requiere de una plantilla aprobada disponible en `lab_templates` para su lenguaje.
