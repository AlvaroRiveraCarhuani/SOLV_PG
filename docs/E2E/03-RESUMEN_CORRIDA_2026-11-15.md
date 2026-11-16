# Resumen de la corrida E2E — 2026-11-15

> **Alcance:** los 23 flujos de `frontend/e2e/flows/`.
> **Resultado:** 23 de 23 en verde, sin fallos, sin omitidos y sin inestables.
> **Evidencia cruda:** `frontend/e2e/test-results/playwright-report.json` (artefacto local, ignorado por git).
> **Catálogo:** [`02-SPECS_PENDIENTES.md`](02-SPECS_PENDIENTES.md).

---

## 1. Datos de la corrida

| Campo | Valor |
| :--- | :--- |
| Fecha y hora | 2026-11-15, 22:32:36 UTC (18:32, UTC-4) |
| Duración total | 554 s (9 min 14 s) |
| Comando | `npm run test:e2e:flows` desde `frontend/` |
| Configuración | `playwright.e2e.config.mjs` |
| Navegador | Chromium |
| Workers | 1 (local) |
| Reintentos | 0 (local) |
| URL base | `http://localhost:4200` por defecto |
| Flujos esperados | 23 |
| En verde | 23 |
| Fallidos | 0 |
| Omitidos | 0 |
| Inestables | 0 |
| Errores de configuración | 0 |

La suma de las duraciones de los 23 flujos es 537 s. Los 17 s restantes corresponden al arranque y cierre del runner.

---

## 2. Resultado por flujo

| Flujo | Resultado | Duración | Qué valida |
| :--- | :---: | ---: | :--- |
| `01-core-lifecycle.spec.mjs` | En verde | 32 s | Ciclo multi-rol completo: admin, docente, estudiante y SpeedGrader |
| `01-easy/01-catalog-and-templates.spec.mjs` | En verde | 12 s | Catálogo de tecnologías y creación de plantillas oficiales |
| `01-easy/02-course-management.spec.mjs` | En verde | 10 s | Materia, docente titular y matriculación |
| `01-easy/03-exercise-authoring-simple.spec.mjs` | En verde | 13 s | Editor en 3 pasos, publicación y resolución con AC |
| `01-easy/J01-template-to-judge-multilang.spec.mjs` | En verde | 22 s | Plantilla multi-lenguaje y resolución en C++ y Python |
| `01-easy/J02-teacher-authoring-pipeline.spec.mjs` | En verde | 45 s | Autoría por interfaz, validación de stack y publicación |
| `02-medium/04-judge-verdicts.spec.mjs` | En verde | 50 s | Matriz de veredictos del juez |
| `02-medium/05-ast-rules-security.spec.mjs` | En verde | 4 s | Reglas AST y bloqueo previo a la ejecución |
| `02-medium/06-curricular-modules.spec.mjs` | En verde | 6 s | Módulos curriculares y desbloqueo |
| `02-medium/07-rubrics-and-ide-projects.spec.mjs` | En verde | 3 s | IDE persistente con rúbrica estricta |
| `02-medium/J03-verdict-matrix-and-hidden-isolation.spec.mjs` | En verde | 81 s | WA, RE, TLE y AC sin exponer casos ocultos |
| `02-medium/J04-ast-security-barrier.spec.mjs` | En verde | 97 s | AST_BLOCKED sin ejecutar el contenedor |
| `02-medium/J08-persistent-ide-lifecycle.spec.mjs` | En verde | 42 s | Ciclo de IDE persistente, base satélite y SpeedGrader |
| `03-hard/08-concurrency-and-resource-limits.spec.mjs` | En verde | 24 s | TLE en algoritmos cuadráticos y AC en solución lineal |
| `03-hard/09-exam-mode-seed.spec.mjs` | En verde | 4 s | Examen con semilla por estudiante |
| `03-hard/10-bulk-import-and-analytics.spec.mjs` | En verde | 6 s | Importación masiva dry-run y analíticas |
| `03-hard/J05-curricular-unlock-and-exam-seed.spec.mjs` | En verde | 22 s | Desbloqueo curricular y semillas distintas |
| `04-edge-cases/11-extreme-inputs-and-utf8.spec.mjs` | En verde | 4 s | UTF-8 extremo, emojis y 10^18 |
| `04-edge-cases/12-anti-plagiarism-supervision.spec.mjs` | En verde | 3 s | Anti-plagio y reporte de similitud |
| `04-edge-cases/J06-extreme-utf8-and-bulk-import.spec.mjs` | En verde | 14 s | Importación masiva, analíticas y UTF-8 extremo |
| `05-badges-and-grading/13-speedgrader-audit-and-override.spec.mjs` | En verde | 11 s | SpeedGrader, comentarios por línea y override |
| `05-badges-and-grading/14-student-gamification-and-progress.spec.mjs` | En verde | 9 s | Progreso, historial y estadísticas |
| `05-badges-and-grading/J07-speedgrader-ephemeral-and-plagiarism.spec.mjs` | En verde | 23 s | SpeedGrader, efímera, override auditado y anti-plagio |

---

## 3. Distribución por categoría

| Categoría | Flujos | Duración |
| :--- | ---: | ---: |
| Flujo core | 1 | 32 s |
| Fáciles | 5 | 102 s |
| Medias | 7 | 283 s |
| Difíciles | 4 | 56 s |
| Casos borde | 3 | 21 s |
| Badges y calificación | 3 | 43 s |
| **Total** | **23** | **537 s** |

---

## 4. Lectura de la corrida

- Los 23 flujos terminaron en verde y no hubo reintentos ni resultados inestables.
- Los flujos más largos son J04 (97 s), J03 (81 s), 04 (50 s), J02 (45 s) y J08 (42 s). El resto se mantiene por debajo de 32 s.
- Los flujos que levantan contenedores reales (J01, J03, J04, J05, J06, J07, J08, 08 y 09) terminaron dentro del tiempo configurado de 360 s por flujo.
- La corrida cubre los tres roles y los dos entornos de práctica: juez efímero, IDE persistente con base satélite y revisión en SpeedGrader.

---

## 5. Límites de esta evidencia

- `e2e/typography.spec.mjs` usa `playwright.config.mjs` y no forma parte de esta corrida.
- Esta corrida cubre la suite E2E del plan de impacto. No equivale a la validación L3 completa: integración con PostgreSQL, suites Docker, cadena de migraciones, seguridad y build tienen sus propias suites.
- El JSON crudo vive en `frontend/e2e/test-results/`, que el `.gitignore` excluye. Este documento es el resumen versionado; si hace falta el artefacto crudo, se regenera con el comando de la sección 6.

---

## 6. Reproducción

Con el stack local levantado (backend, frontend, PostgreSQL y Docker) y los actores E2E sembrados:

```bash
cd frontend
E2E_BASE_URL=http://127.0.0.1:4200 npm run test:e2e:flows
```

El reporte JSON se escribe en `frontend/e2e/test-results/playwright-report.json`. Los requisitos de la pila y el sembrado de actores están en [`frontend/e2e/README.md`](../../frontend/e2e/README.md) y en [`docs/OPERATIONS/GUIA_LOCAL_FEDORA.md`](../OPERATIONS/GUIA_LOCAL_FEDORA.md).
