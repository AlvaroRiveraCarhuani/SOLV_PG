# Catálogo E2E: estado y cierre de la corrida

> **Corte:** 2026-11-15, 22:32 UTC.
> **Alcance:** los 23 flujos de `frontend/e2e/flows/`.
> **Estado:** 23 de 23 en verde. No quedan flujos pendientes.
> **Resumen versionado:** [`03-RESUMEN_CORRIDA_2026-11-15.md`](03-RESUMEN_CORRIDA_2026-11-15.md).

Este documento mantuvo la lista de flujos sin corrida en verde al 10 de noviembre. La corrida del 15 de noviembre ejecutó los 23 flujos de `e2e/flows/` y los cerró. El nombre del archivo se conserva por trazabilidad; el contenido es el catálogo vigente.

---

## 1. Estado por flujo

| Flujo | Categoría | Estado |
| :--- | :--- | :---: |
| `01-core-lifecycle.spec.mjs` | Core | En verde |
| `01-easy/01-catalog-and-templates.spec.mjs` | Fácil | En verde |
| `01-easy/02-course-management.spec.mjs` | Fácil | En verde |
| `01-easy/03-exercise-authoring-simple.spec.mjs` | Fácil | En verde |
| `01-easy/J01-template-to-judge-multilang.spec.mjs` | Fácil | En verde |
| `01-easy/J02-teacher-authoring-pipeline.spec.mjs` | Fácil | En verde |
| `02-medium/04-judge-verdicts.spec.mjs` | Media | En verde |
| `02-medium/05-ast-rules-security.spec.mjs` | Media | En verde |
| `02-medium/06-curricular-modules.spec.mjs` | Media | En verde |
| `02-medium/07-rubrics-and-ide-projects.spec.mjs` | Media | En verde |
| `02-medium/J03-verdict-matrix-and-hidden-isolation.spec.mjs` | Media | En verde |
| `02-medium/J04-ast-security-barrier.spec.mjs` | Media | En verde |
| `02-medium/J08-persistent-ide-lifecycle.spec.mjs` | Media | En verde |
| `03-hard/08-concurrency-and-resource-limits.spec.mjs` | Difícil | En verde |
| `03-hard/09-exam-mode-seed.spec.mjs` | Difícil | En verde |
| `03-hard/10-bulk-import-and-analytics.spec.mjs` | Difícil | En verde |
| `03-hard/J05-curricular-unlock-and-exam-seed.spec.mjs` | Difícil | En verde |
| `04-edge-cases/11-extreme-inputs-and-utf8.spec.mjs` | Borde | En verde |
| `04-edge-cases/12-anti-plagiarism-supervision.spec.mjs` | Borde | En verde |
| `04-edge-cases/J06-extreme-utf8-and-bulk-import.spec.mjs` | Borde | En verde |
| `05-badges-and-grading/13-speedgrader-audit-and-override.spec.mjs` | Badges y calificación | En verde |
| `05-badges-and-grading/14-student-gamification-and-progress.spec.mjs` | Badges y calificación | En verde |
| `05-badges-and-grading/J07-speedgrader-ephemeral-and-plagiarism.spec.mjs` | Badges y calificación | En verde |

---

## 2. Pendientes al 10 de noviembre: cerrados

Los doce flujos que este documento listaba como pendientes quedaron en verde en la corrida del 15 de noviembre.

| Flujo | Estado al 10/11 | Estado al 15/11 |
| :--- | :--- | :---: |
| `02-medium/J03-verdict-matrix-and-hidden-isolation.spec.mjs` | En corrección, sin verificación posterior | En verde |
| `02-medium/J04-ast-security-barrier.spec.mjs` | En corrección, sin verificación posterior | En verde |
| `03-hard/J05-curricular-unlock-and-exam-seed.spec.mjs` | En corrección, sin verificación posterior | En verde |
| `03-hard/08-concurrency-and-resource-limits.spec.mjs` | Sin corrida registrada | En verde |
| `03-hard/09-exam-mode-seed.spec.mjs` | Sin corrida registrada | En verde |
| `03-hard/10-bulk-import-and-analytics.spec.mjs` | Sin corrida registrada | En verde |
| `04-edge-cases/11-extreme-inputs-and-utf8.spec.mjs` | Sin corrida registrada | En verde |
| `04-edge-cases/12-anti-plagiarism-supervision.spec.mjs` | Sin corrida registrada | En verde |
| `04-edge-cases/J06-extreme-utf8-and-bulk-import.spec.mjs` | Sin corrida registrada | En verde |
| `05-badges-and-grading/13-speedgrader-audit-and-override.spec.mjs` | Sin corrida registrada | En verde |
| `05-badges-and-grading/14-student-gamification-and-progress.spec.mjs` | Sin corrida registrada | En verde |
| `05-badges-and-grading/J07-speedgrader-ephemeral-and-plagiarism.spec.mjs` | Sin corrida registrada | En verde |

El detalle de duraciones, distribución por categoría y datos de la corrida está en el [resumen versionado](03-RESUMEN_CORRIDA_2026-11-15.md).

---

## 3. Qué no cubre esta corrida

- `e2e/typography.spec.mjs` usa `playwright.config.mjs` y se ejecuta aparte; no forma parte de este reporte.
- Esta corrida cubre los flujos E2E del plan de impacto. No equivale a la validación L3 completa: integración con PostgreSQL, suites Docker, cadena de migraciones, seguridad y build tienen sus propias suites.
- No mide capacidad ni rendimiento. Para eso están `docs/CAPACITY/load-test-report.md` y `docs/OPERATIONS/capacidad.md`.

---

## 4. Cómo reproducir

Con el stack local levantado (backend, frontend, PostgreSQL y Docker) y los actores E2E sembrados:

```bash
cd frontend
E2E_BASE_URL=http://127.0.0.1:4200 npm run test:e2e:flows
```

El comando escribe el reporte JSON en `frontend/e2e/test-results/playwright-report.json`. Los requisitos de la pila y el sembrado de actores están en [`frontend/e2e/README.md`](../../frontend/e2e/README.md) y en [`docs/OPERATIONS/GUIA_LOCAL_FEDORA.md`](../OPERATIONS/GUIA_LOCAL_FEDORA.md).

---

## 5. Cierre del plan L3

Un cambio queda validado cuando el plan de impacto selecciona sus suites, todas se ejecutan y el informe final verifica la firma y la evidencia. Esta corrida cubre solo la suite E2E; no reemplaza ese informe.

```bash
node tools/test-impact/cli.mjs plan --base origin/main --working-tree \
  --output impact-plan.json --format text
node tools/test-impact/run.mjs --all --plan impact-plan.json \
  --report impact-report.json
node tools/test-impact/cli.mjs report --file impact-report.json \
  --plan impact-plan.json
```

Si el código cambia después de generar el plan, hay que generar uno nuevo. Un plan con `--files` es sintético y no valida el árbol de trabajo.
