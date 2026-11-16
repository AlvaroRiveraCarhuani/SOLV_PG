```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: working-tree-2026-11-16 (base 1a6cf8d)
verdict: superseded
superseded_by: ADR-055 (2026-10-30), ADR-057 (2026-10-30), ADR-062 (2026-11-02)
blockers: 0
critical_findings: 0
requirements: 0/4 vigentes; 4/4 reemplazados
scenarios: 0/3 vigentes; 3/3 reemplazados
test_command: cd frontend && npx ng test --watch=false --include='**/teacher/reviews/teacher-reviews.component.spec.ts'
test_exit_code: 0
test_result: 5 tests passed
test_output_hash: sha256:5ec539910983a7d87085f8d040691bf6cf4bc21a38e5835133bc39737fa552b1
lint_command: cd frontend && npm run lint:styles
lint_exit_code: 0
lint_output_hash: sha256:3134c732ffaa924c4ce09faa65e42fd9b30b3a85572877c65e626fce0e83ef64
```

## Verification Report

**Change**: teacher-workspace-navigation-and-queue
**Mode**: verificación de cambio superado
**Fecha**: 2026-11-16

### Resultado

El cambio implementó una navegación docente con tres secciones —`DOCENCIA`, `CONTENIDOS`, `SUPERVISIÓN`— y una cola global en `/teacher/evaluaciones`. El estado vigente es distinto por decisiones posteriores:

- ADR-055 retiró el centro global de evaluaciones y dejó una sola cola de corrección por `review_status`, con varias puertas.
- ADR-057 fijó `Hoy` como canvas de materias y columna de pendientes; la navegación pasó a `HOY`, `DOCENCIA`, `CONTENIDOS` y `CUENTA`.
- ADR-062 retiró `SUPERVISIÓN` porque la auditoría de similitud se abre desde cada práctica.

La ruta `/teacher/evaluaciones` redirige a `/teacher/por-calificar`. La cola vigente es `TeacherReviewsComponent`, no una tabla global con filtros por veredicto.

Por eso el cambio se archiva como SUPERADO. No se cierra como PASS contra una especificación que el producto ya no sostiene.

### Evidencia de la cola vigente

La suite de `TeacherReviewsComponent` pasa con 5 pruebas:
```text
suma las entregas pendientes de todas las prácticas
selecciona la primera práctica y muestra solo las entregas sin corregir, de la más antigua a la más nueva
usa la materia de la URL como filtro de la cola y lo deja quitar
muestra la espera en lugar del veredicto
muestra el estado vacío cuando no hay nada por calificar
```

El gate de estilos pasa sin hallazgos.

### Correspondencia con la especificación original

| Requisito original | Estado actual | Tratamiento |
|---|---|---|
| REQ-1: sidebar con `DOCENCIA`, `CONTENIDOS`, `SUPERVISIÓN` | El sidebar vigente es `HOY`, `DOCENCIA`, `CONTENIDOS`, `CUENTA`; `SUPERVISIÓN` desapareció | Reemplazado por ADR-057 y ADR-062 |
| REQ-2: cola global en `/teacher/evaluaciones` | `/teacher/evaluaciones` redirige a `/teacher/por-calificar` | Reemplazado por ADR-055 |
| REQ-3: filtros por curso, veredicto y estado | La cola vigente filtra por materia desde la URL y agrupa por práctica; no conserva los filtros de veredicto de la spec | Reemplazado por ADR-055 |
| REQ-4: acceso directo a SpeedGrader | La cola vigente abre la revisión de la entrega | Vigente en la práctica, con otra superficie |

### Issues Found

**CRITICAL**: ninguno.

**WARNING**:

- W1: La spec original describe rutas y secciones que ya no existen. No debe copiarse a `openspec/specs/` porque introduciría un contrato obsoleto.
- W2: No existe una spec del `TeacherLayoutComponent`; la estructura del sidebar se verificó por código y por los ADR, no por una prueba de layout.
- W3: No se ejecutó un E2E de navegación desde el sidebar hasta SpeedGrader en esta sesión.

### Verdict

**SUPERSEDED**

El cambio queda cerrado como historia. La cola vigente es `TeacherReviewsComponent` en `/teacher/por-calificar`, y la navegación se rige por ADR-055, ADR-057 y ADR-062. El spec histórico se conserva en el archivo, pero no se fusiona con las especificaciones actuales.
