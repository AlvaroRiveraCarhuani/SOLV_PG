# Specs E2E pendientes y estado de la corrida

> **Corte:** 2026-11-10, 19:00 UTC.
> **Alcance:** flujos de `frontend/e2e/flows/` ejecutados contra el stack local.
> **Estado del plan L3:** no validado. Hay specs sin corrida en verde y el plan
> de impacto debe regenerarse después de las últimas ediciones.

Un spec se considera pendiente si no tiene una corrida en verde posterior a su
última edición. Los cambios de esta sesión están aplicados en el árbol de
trabajo, pero varios todavía no se volvieron a ejecutar.

---

## 1. En corrección (cambio aplicado, sin verificación posterior)

| Spec | Último fallo conocido | Cambio aplicado | Siguiente paso |
| :--- | :--- | :--- | :--- |
| `02-medium/J03-verdict-matrix-and-hidden-isolation.spec.mjs` | La matrícula devolvía 403 porque `POST /api/v1/subjects/:id/enroll` es exclusivo de admin y el fixture usaba la sesión del estudiante. Al corregirla, el `POST /api/v1/evaluations` del paso **WA** devolvió una respuesta no exitosa. | La matrícula usa `adminPage`. Se agregó al spec el status y el cuerpo de la respuesta de evaluación para diagnosticar el fallo. | Reejecutar J03; con el nuevo mensaje identificar el status real del `POST /evaluations`. |
| `02-medium/J04-ast-security-barrier.spec.mjs` | `ast-rules-builder` quedaba oculto durante 45 s. | El fixture expande el `<details>` **Restricciones didácticas de código** antes de buscar el builder. La matrícula también usa `adminPage`. | Reejecutar J04 y verificar que el builder sea visible y que el veredicto sea `AST_BLOCKED`. |
| `03-hard/J05-curricular-unlock-and-exam-seed.spec.mjs` | El badge del módulo no existía con la clase `.module-state-badge`; el componente usa `.module-state`. | Los dos selectores usan `.module-state`. Las dos matrículas usan `adminPage`. | Reejecutar J05 y comprobar los estados **Bloqueado** y **Disponible**. |

---

## 2. Sin corrida en esta sesión

Estos flujos no tienen una corrida registrada después de sus últimas ediciones.
No se deben declarar en verde sin ejecutarlos.

| Spec | Categoría | Nota |
| :--- | :--- | :--- |
| `03-hard/08-concurrency-and-resource-limits.spec.mjs` | Difíciles | Límites de tiempo ajustados a 15000 ms. |
| `03-hard/09-exam-mode-seed.spec.mjs` | Difíciles | Límites de tiempo ajustados a 15000 ms. |
| `03-hard/10-bulk-import-and-analytics.spec.mjs` | Difíciles | Sin corrida registrada. |
| `04-edge-cases/11-extreme-inputs-and-utf8.spec.mjs` | Borde | Sin corrida registrada. |
| `04-edge-cases/12-anti-plagiarism-supervision.spec.mjs` | Borde | Sin corrida registrada. |
| `04-edge-cases/J06-extreme-utf8-and-bulk-import.spec.mjs` | Borde | Sin corrida registrada. |
| `05-badges-and-grading/13-speedgrader-audit-and-override.spec.mjs` | Badges y SpeedGrader | Sin corrida registrada. |
| `05-badges-and-grading/14-student-gamification-and-progress.spec.mjs` | Badges y SpeedGrader | Sin corrida registrada. |
| `05-badges-and-grading/J07-speedgrader-ephemeral-and-plagiarism.spec.mjs` | Badges y SpeedGrader | Sin corrida registrada. |

---

## 3. En verde (referencia)

Estos flujos tienen una corrida en verde en esta sesión y no forman parte de la
lista pendiente:

- `01-core-lifecycle.spec.mjs`
- `01-easy/01-catalog-and-templates.spec.mjs`
- `01-easy/02-course-management.spec.mjs`
- `01-easy/03-exercise-authoring-simple.spec.mjs`
- `01-easy/J01-template-to-judge-multilang.spec.mjs`
- `01-easy/J02-teacher-authoring-pipeline.spec.mjs`
- `02-medium/04-judge-verdicts.spec.mjs`
- `02-medium/05-ast-rules-security.spec.mjs`
- `02-medium/06-curricular-modules.spec.mjs`
- `02-medium/07-rubrics-and-ide-projects.spec.mjs`
- `02-medium/J08-persistent-ide-lifecycle.spec.mjs`

---

## 4. Cómo ejecutar un spec pendiente

Desde `frontend/`, con el backend y el frontend levantados:

```bash
cd frontend

# Un flujo concreto
E2E_BASE_URL=http://127.0.0.1:4200 \
  npx playwright test e2e/flows/02-medium/J03-verdict-matrix-and-hidden-isolation.spec.mjs \
  --config=playwright.e2e.config.mjs --reporter=line

# Una categoría completa, sin cortar al primer fallo
E2E_BASE_URL=http://127.0.0.1:4200 \
  npx playwright test e2e/flows/03-hard \
  --config=playwright.e2e.config.mjs --reporter=line --max-failures=3
```

El fixture lee `JWT_SECRET` de `.env`. Si se ejecuta contra otra base o contra
el stack aislado de `tools/test-impact`, hay que exportar el secreto y la URL
correspondientes, y sembrar los actores E2E en esa base:

```bash
docker exec -i solv_db psql -U solv_user -d solv_db \
  < backend/seeds/e2e_actors.sql
```

> El seed actualiza el tenant `00000000-0000-0000-0000-000000000001` con el
> nombre `SOLV E2E`. Usarlo solo en una base local de pruebas, no en una base
> con datos reales.

---

## 5. Cierre del plan L3

Cuando los 12 specs estén en verde:

```bash
# 1. Regenerar el plan: hubo cambios de código y de specs.
node tools/test-impact/cli.mjs plan \
  --base origin/main --working-tree \
  --output impact-plan.json --format text

# 2. Ejecutar todas las suites seleccionadas por el plan.
node tools/test-impact/run.mjs --all \
  --plan impact-plan.json --report impact-report.json

# 3. Verificar el recibo y la evidencia.
node tools/test-impact/cli.mjs report \
  --file impact-report.json --plan impact-plan.json
```

El cambio queda validado solo si `validated` es verdadero, no hay suites
pendientes o fallidas y `cli report --plan` verifica la firma y devuelve 0.
No editar el JSON del informe para declarar éxito.
