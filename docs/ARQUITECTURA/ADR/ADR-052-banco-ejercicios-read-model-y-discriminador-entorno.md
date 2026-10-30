# ADR-052: Banco de Ejercicios como Read Model y Discriminador de Modalidad por Entorno

## Estado
Aprobado

## Fecha
2026-10-30

## Contexto
El banco de ejercicios del docente (`/teacher/exercises`) agrega laboratorios de todas las materias, pero presentaba varias inconsistencias con el modelo de dominio:

1. **Tres vocabularios para la misma modalidad.** El editor escribe `type='algorithm' | 'project'` y `environment_type='JUEZ_EFIMERO' | 'IDE_PERSISTENTE'`; los seeds usaban `type='code'`; y el filtro de la vista comparaba `type` contra `'ALGORITMO' | 'WORKSPACE'` (valores que no existen en el dominio). El endpoint `/labs` ni siquiera devolvía `type`.
2. **Filtro de modalidad roto.** Como `e.type` nunca llegaba del backend, el filtro "Juez Automático" mostraba todos los laboratorios (fallback a `'ALGORITMO'`) y "Laboratorio VS Code" no mostraba ninguno, aunque el docente tuviera laboratorios IDE.
3. **Fuzzing en laboratorios IDE.** El botón Fuzzing aparecía en todas las tarjetas. El flujo real de un laboratorio `IDE_PERSISTENTE` no pasa por el juez I/O: `SubmitWorkspaceProject` hace snapshot del volumen y crea la submission con veredicto `PENDING` para corrección manual en SpeedGrader. `ApplyFuzzCases` escribía casos en `Config.Algorithm.TestCases` igualmente, casos que el juez jamás ejecuta (trabajo muerto).
4. **Semántica de "por revisar" mezclada.** El KPI contaba `verdict IN ('WA','RE','TLE')` sin override, un concepto del juez; las entregas `PENDING` de SpeedGrader (cola de corrección manual de IDE) no se reflejaban.
5. **Datos de demo falsos.** Los seeds insertaban veredictos automáticos (AC/WA/RE/AST_BLOCKED) y casos de prueba en laboratorios IDE, valores que el flujo real no produce. Hacían parecer que los IDE se auto-corrigieran contra casos I/O.

## Decisión

Establecer el **entorno como discriminador único de modalidad** y el **banco como read model dedicado** (CQRS de lectura), sin parchear la vista existente:

1. **Restricción de integridad (`00033_enforce_exercise_type_environment.sql`)**:
   - Backfill de `type='code'` → `'algorithm'` (juez) o `'project'` (IDE) según `environment_type`.
   - CHECK constraint: `type IN ('algorithm','database','project')`. El valor legacy `'code'` deja de ser posible.

2. **Read model del banco (`GET /api/v1/teacher/exercises/bank`)**:
   - DTO `TeacherExerciseBankItem` con `environment_type`, `type`, `language`, límites (`memory_limit_mb`, `time_limit_ms`), materia (id/código/nombre), entregas, estudiantes y `review_pending`.
   - `review_pending` con semántica por entorno: juez → `WA/RE/TLE` sin override; IDE → `PENDING` (cola de SpeedGrader).
   - El endpoint consolida todas las materias del docente en una sola query (sin N+1 por curso).

3. **Invariante de modalidad en la escritura de casos**:
   - `BulkAddTestCases` rechaza (error `ErrTestCasesRequireJudge`) ejercicios con `environment_type='IDE_PERSISTENTE'`: los casos I/O pertenecen al juez virtual. Esto blinda también el fuzzing aplicado.

4. **Frontend del banco**:
   - El filtro de modalidad usa `environment_type` con valores `JUEZ_EFIMERO | IDE_PERSISTENTE` y etiquetas "Juez Virtual" / "IDE Persistente (Workspace)". Fuera el vocabulario `ALGORITMO/WORKSPACE`.
   - El botón Fuzzing se muestra solo en tarjetas de juez (`canFuzz`).
   - Las tarjetas muestran datos reales: lenguaje, límites, `review_pending`.

5. **Sanidad de datos de demo**: los seeds (`seed_teacher_dashboard.sql`, `demo_presentation_seed.sql`) normalizan de forma idempotente los laboratorios IDE: veredictos automáticos → `PENDING`, sin métricas de ejecución, y retiran los casos de prueba de ejercicios IDE.

## Consecuencias

- **Positivas**:
  - El banco refleja fielmente el modelo canónico del editor; el filtro de entorno funciona y muestra los 4 laboratorios IDE de la demo.
  - El fuzzing queda acotado al juez virtual; los IDE se corrigen con rúbrica en SpeedGrader, coherente con el paso `step-ide-environment` del editor.
  - Los estados inválidos (`type='code'`, casos en IDE) se vuelven imposibles a nivel de base de datos y servicio.
- **Negativas**:
  - Los laboratorios IDE existentes con casos de prueba de demo pierden esos casos (se migran a la semántica PENDING/rúbrica).
  - La corrección manual (SpeedGrader) es la vía de evaluación de los IDE; si en el futuro se quiere autoevaluación para IDE, debe modelarse como primitiva distinta (ejecutar la suite de tests del alumno), no como casos I/O del juez.
