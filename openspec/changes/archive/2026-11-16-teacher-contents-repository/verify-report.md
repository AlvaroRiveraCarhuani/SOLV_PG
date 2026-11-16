```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: working-tree-2026-11-16 (base 1a6cf8d)
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 3/3 implemented
scenarios: 0/0; la spec usa criterios numerados, no Given/When/Then
test_command: cd frontend && npx ng test --watch=false --include='**/teacher/exercises/teacher-exercises.component.spec.ts' --include='**/teacher/templates/teacher-templates.component.spec.ts'
test_exit_code: 0
test_result: 25 tests passed
test_output_hash: sha256:5ec539910983a7d87085f8d040691bf6cf4bc21a38e5835133bc39737fa552b1
backend_command: cd backend && go test ./internal/core/services -run 'TestBulkAddTestCases_EnvironmentGuard' -count=1
backend_exit_code: 0
backend_output_hash: sha256:c141d62632e448977e77cf69ebfdb940c305267f6238806faf52e3f74f53748a
lint_command: cd frontend && npm run lint:styles
lint_exit_code: 0
lint_output_hash: sha256:3134c732ffaa924c4ce09faa65e42fd9b30b3a85572877c65e626fce0e83ef64
```

## Verification Report

**Change**: teacher-contents-repository
**Mode**: Strict TDD, verificación acotada por impacto
**Fecha**: 2026-11-16

### Completeness

| Métrica | Valor |
|---|---|
| Tareas | 10/10 marcadas |
| Fases | 3/3 |
| Requisitos | 3/3 implementados |
| Criterios R1 | 7/7 implementados; integración de banco no ejecutada |
| Criterios R2 | 5/5 implementados |
| Criterios R3 | 4/4 implementados |

Las tareas están completas. El cambio es fundamentalmente frontend, pero su especificación depende del read model `GET /api/v1/teacher/exercises/bank` y de la invariante de casos I/O, que se verificaron por ruta y prueba unitaria de backend.

### Build and Tests Execution

**Frontend**: pasan 25 pruebas.
```text
teacher-exercises.component.spec.ts: 10 tests
teacher-templates.component.spec.ts: 15 tests
```

**Backend**: pasa la guarda de modalidad.
```text
TestBulkAddTestCases_EnvironmentGuard
ok solv-backend/internal/core/services
```

**Gate de estilos**: pasa.

### Spec Compliance Matrix

| Criterio | Evidencia | Resultado |
|---|---|---|
| R1.1 banco global con datos reales | Ruta `GET /api/v1/teacher/exercises/bank`, `TeacherCourseService.getExerciseBank`, componente consume el read model | Implementado; integración PostgreSQL no ejecutada |
| R1.2 KPIs agregados | Prueba “load the bank read model and calculate KPI metrics correctly” | Cumplido |
| R1.3 filtros por materia, entorno, estado y texto | Pruebas de filtros por curso, entorno, estado y búsqueda | Cumplido |
| R1.4 modalidad derivada de `environment_type` | Filtros usan `JUEZ_EFIMERO` e `IDE_PERSISTENTE`; prueba de fuzzing solo para juez | Cumplido |
| R1.5 `review_pending` por entorno | Campo presente en el modelo y en los datos de prueba; la semántica de backend se verificó por código | Parcial: sin integración de banco |
| R1.6 acciones por tarjeta | Pruebas de fuzzing, edición y navegación | Cumplido |
| R1.7 casos I/O solo juez | `ErrTestCasesRequireJudge` y `TestBulkAddTestCases_EnvironmentGuard` | Cumplido |
| R2.1 catálogo de plantillas | Componente consume `/api/v1/templates` | Cumplido |
| R2.2 filtros por categoría y búsqueda | Pruebas de categoría, entorno y búsqueda | Cumplido |
| R2.3 especificaciones técnicas | Tarjetas y modal de detalle con RAM, CPU, puertos y stack | Cumplido por componente y pruebas |
| R2.4 solicitud de plantilla | Modal `TemplateRequestModalComponent`; prueba de apertura/cierre | Cumplido |
| R2.5 crear laboratorio desde plantilla | Navegación con `template_id` y `environment_type` | Cumplido |
| R3.1 tokens sin colores literales | `lint:styles` pasa | Cumplido |
| R3.2 `machineData` | Gate `machine-data-gate` pasa | Cumplido |
| R3.3 fechas con `dateText` | Gate `date-format-gate` pasa | Cumplido |
| R3.4 pruebas de agregación, filtros y modales | 25 pruebas pasan | Cumplido |

### Issues Found

**CRITICAL**: ninguno.

**WARNING**:

- W1: `TestSlice13_TeacherExerciseBank` no se ejecutó contra PostgreSQL; la agregación multi-materia y la semántica de `review_pending` se sostienen en el contrato, el código y mocks, no en una corrida de integración.
- W2: La suite frontend no prueba el caso de un docente sin materias contra el backend real; sí prueba el estado vacío con datos simulados.
- W3: El catálogo de plantillas se apoya en AdminTemplatesService y en el contrato `/api/v1/templates`; no se ejecutó un E2E de solicitud y aprobación.

### Verdict

**PASS WITH WARNINGS**

Los tres requisitos están implementados y las 25 pruebas frontend más la guarda de modalidad pasan. La advertencia principal es que el read model del banco no se verificó contra PostgreSQL en esta sesión.
