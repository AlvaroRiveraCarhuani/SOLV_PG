# ADR-047: Gobierna del Límite de Tiempo por Caso — Validación en Servidor y Contrato de Plantilla como Arquitectura Objetivo

- **Estado:** Aprobado
- **Fecha:** 2026-10-25
- **Contexto:** Core / Evaluación y Gobernanza de Plantillas

## Contexto y Problema

Una auditoría del Constructor de Entornos (`TemplateStudioComponent` y su resumen lateral) reveló que el campo `timeout_ms` de `lab_templates` era **metadata muerta**:

1. En el studio, al seleccionar un entorno de juez (`JUEZ_EFIMERO`) o dual (`AMBOS`), el resumen mostraba `Tiempo máximo: 5000 ms` con apariencia de techo homologado, cuando en realidad no ejerce ninguna restricción: el límite efectivo por caso lo define el docente por ejercicio (`time_limit_ms`, rango 100–30.000 ms validado solo en el formulario) y el runner lo consume desde `config.algorithm.time_limit_ms`, nunca desde la plantilla.
2. El backend **no validaba** el rango de `time_limit_ms`: aceptaba valores negativos o absurdos (`-1`, `999999`) y los reemplazaba en silencio por 1000 ms en `exercise_repository` (defensa inexistente fuera de la UI).
3. El único uso real de `timeout_ms` de plantilla es el *smoke test* de verificación del entorno (`EnvTestService`), no la evaluación de entregas.
4. El default `1000 ms` vivía duplicado en tres lugares (repositorio, importador y frontend), sin una fuente de verdad única.

## Decisión

### Parte 1 — Inmediata (aplicada con este ADR)

- **Validación de rango en servidor:** `Exercise.Validate()` rechaza `time_limit_ms` fuera de `[100, 30.000] ms` con `ErrTimeLimitOutOfRange` (solo se admite `0`, que el repositorio normaliza a 1000 ms en flujos históricos). Cubre creación, actualización e importación, que pasan por el mismo `Validate()`.
- **Fuente de verdad única:** `normalizeTimeLimit` en `EvaluationService` (ejecutado en `CreateExercise` y `UpdateExercise`) normaliza la columna `time_limit_ms` y la sincroniza en `config.algorithm.time_limit_ms`, que es lo que consume el runner y el dry-run. Así un payload con `config.algorithm.time_limit_ms` divergente queda bajo la misma validación.
- **El error es de dominio y se responde 422** vía `isValidationError`, coherente con los demás errores de gobernanza (`ErrMemoryGovernedByTemplate`, `ErrTemplateNotApproved`).
- **Semántica del campo de plantilla:** `timeout_ms` de la plantilla pasa a ser explícitamente un **valor orientativo** (sugerencia informativa). El studio lo etiqueta como "Tiempo máximo sugerido (ms)" con texto de ayuda que aclara que el límite real lo fija el docente al crear el ejercicio, y el resumen lateral lo muestra como "Límite sugerido". No se elimina la columna; se documenta su significado.

### Parte 2 — Arquitectura objetivo (decidida, no construida aún)

El rumbo registrado para cuando exista un disparador real:

- **Contrato de plantilla inmutable y direccionado por contenido:** la plantilla aprobada emite un `template_contract` (`contract_hash` + `contract_version`) con tres bloques: `environment` (imagen, RAM, servicios), `execution` (entrypoint, `verify_timeout_ms`) y `judge_policy` (`default_time_limit_ms`, `envelope: {min_ms, max_ms}`). `timeout_ms` se separa en `verify_timeout_ms` (operacional) y `judge_policy` (dominio): un concepto, un nombre.
- **Política en capas con resolución en servidor:** envolvente efectiva = intersección de techo de tenant (`server_policies`, hoy solo RAM/containers) y envolvente de plantilla, resuelta por un `PolicyResolver` idempotente con asiento de auditoría. El frontend negocia con el contrato resuelto; nunca impone límites.
- **Snapshot por ligadura:** al vincular plantilla a ejercicio se congela `policy_snapshot` + `contract_version` en la fila del ejercicio. Los ejercicios publicados se validan contra la versión de contrato a la que están ligados, no contra la más reciente; el re-bind explícito aplica la nueva envolvente. Sin esto, editar una plantilla bloquearía guardados legítimos de ejercicios antiguos.
- **Desviación gestionada:** superar la envolvente exige `policy_deviation_reason` obligatorio (mismo patrón `Reason` de `language_profile_service`).
- **Defaults calibrados por telemetría (opcional):** con `run_metrics` (que ya guarda `language`, `duration_ms`, `verdict`) el default por idioma puede proponerse desde p95 histórico vía JOIN con `exercises.template_id`; la telemetría propone, el admin decide.

**Criterios de activación** (no construir antes de que ocurra al menos uno):

1. Aparece un integrador externo que consuma `/api/v1/templates` y necesite límites autodescriptivos (requisito BaaS).
2. Un tenant pide techos de tiempo propios (hoy `server_policies` solo gobierna RAM, inactividad y contenedores).
3. Gobernanza institucional exige auditoría de decisiones de límites con periodicidad real.

## Consecuencias

- **Positivas:** la API deja de aceptar límites inválidos sin importar quién llame; el runner, el dry-run y la columna comparten una única fuente de verdad; el studio deja de mostrar un número con apariencia de techo; la decisión de arquitectura queda registrada para no re-derivarla.
- **Neutrales / sin cambios:** el flujo del docente (formulario de ejercicio, rango 100–30.000 ms) no cambia; el runner y los comparadores no se tocan; `timeout_ms` de plantilla se conserva como columna legacy (espejo de `verify_timeout_ms` cuando se construya el contrato).
- **Negativas / impacto esperado:** solicitudes directas a la API con `time_limit_ms` fuera de rango ahora reciben 422 en lugar de ser corregidas en silencio — es la consecuencia buscada.

## Relación con Otros ADRs

- **ADR-005** (Juez Virtual): el runner consume `config.algorithm.time_limit_ms`; este ADR garantiza que ese valor siempre pasa por validación de dominio.
- **ADR-008 / ADR-043** (Recursos y Modalidad Dual): mismo lenguaje de gobernanza de recursos (RAM homologada por plantilla); el tiempo queda documentado como no-gobernado por plantilla hasta la Parte 2.
- **ADR-030 / ADR-046** (Catálogo y Studio de Plantillas): la Parte 2 extiende el versionado y el direccionamiento por contenido (`solv-local/stack:<sha256>`) al contrato de evaluación.
- **ADR-042** (Auditoría Time-Travel): el snapshot por ligadura es coherente con la trazabilidad del estado previo.