# Feature: Vocabulario Residual del Entorno Docente

Objetivo: terminar el barrido de términos viejos («Juez Virtual», «Boilerplate», «Laboratorio IDE») en el copy visible, sin renombrar campos de API, tablas ni selectores.

Branch: `chore/teacher-vocabulary-sweep` (desde `main`) | Delivery strategy: inline | Test runner: `npm run test:ci`

Estado: **planificada**. Bloqueada por D1 (alcance admin).

Fuente: `proyecto-grado/Observaciones/Corregir.md` §1, §2 y §5; `proyecto-grado/Observaciones/Reestructuracion Entorno Docente y Vocabulario.md` §2. Decisión del usuario: «Limpieza de vocabulario residual: Barrer los textos viejos registrados en Corregir.md».

## Decisiones cerradas

- El vocabulario canónico es: **Entorno de Desarrollo** (no «IDE Persistente», «Laboratorio IDE» ni «Entorno de Proyecto»), **Evaluador Automático** (no «Juez Virtual») y **Código Base para el Estudiante** (no «Boilerplate» ni «Firma inicial»).
- Las siglas de veredicto no se muestran; `OUTCOME_COPY` ya resuelve las etiquetas humanas.
- «Fuzzing» se conserva en el evaluador: el documento de referencia lo acepta como generador de estrés de casos.

## Qué ya está hecho (verificado en el código)

- La ficha de práctica usa «Entorno de Desarrollo» / «Evaluador Automático» y «Código Base para el Estudiante» (`frontend/src/app/features/teacher/courses/practice-detail/practice-detail.component.html:30-40,298`), con `code-editor` en vez de `<pre>`.
- `Mi entorno` ya no existe: no hay carpeta `frontend/src/app/features/teacher/environment/` ni entrada en el sidebar.
- El sidebar no tiene entradas con nombres viejos.
- `docs/UI` y los prototipos ya usan el vocabulario nuevo en las fichas de práctica.

## Inventario pendiente (copy visible)

| Archivo | Línea | Texto actual | Reemplazo |
|---|---|---|---|
| `frontend/.../exercise-editor/ai-exercise-generator-modal/ai-exercise-generator-modal.component.html` | 3 | «casos de prueba sellados por el Juez Virtual» | «…por el Evaluador Automático» |
| mismo archivo | 85 | «ejecutor del Juez Virtual» | «ejecutor del Evaluador Automático» |
| mismo archivo | 90 | «no pudieron sellarse con el Juez Virtual» | «…con el Evaluador Automático» |
| `frontend/.../admin/templates/studio/template-studio.component.html` | 928 | «Evaluación automática de código (Juez)» | «Evaluación automática de código (Evaluador Automático)» |
| `frontend/.../admin/templates/admin-templates.component.html` | 133 | «Laboratorio IDE» | «Entorno de Desarrollo» |
| mismo archivo | 287, 524, 892 | «Comando Juez:» | «Comando del evaluador:» |
| `frontend/.../admin/templates/components/model-library/model-library.component.html` | 394 | «Juez listo» | «Evaluador listo» |
| `frontend/.../admin/templates/components/model-library/components/component-form-modal/component-form-modal.component.html` | 404 | «Comando de evaluación para Juez Virtual (Opcional)» | «Comando de evaluación del Evaluador Automático (Opcional)» |
| `frontend/.../admin/templates/components/template-edit-modal/template-edit-modal.component.html` | 21, 25 | «Entorno Dual (Laboratorio IDE + Evaluador Juez)», «Laboratorio IDE» | «Entorno Dual (Entorno de Desarrollo + Evaluador Automático)», «Entorno de Desarrollo» |
| mismo archivo | 64 | «Laboratorio IDE persistente… Juez Efímero» | «Entorno de Desarrollo… Evaluador Automático» |
| `frontend/.../admin/templates/components/template-review-modal/template-review-modal.component.html` | 21, 25, 61 | mismos textos | mismos reemplazos |
| `frontend/.../admin/configuracion/tabs/admin-config-ia.component.html` | 524 | «Ejercicios Juez» | «Ejercicios del evaluador» |

Fuera del barrido: comentarios internos (`<!-- Entorno Juez Virtual -->`), identificadores (`boilerplate`, `boilerplate-tabs`, `loadBoilerplate`, `resetBoilerplate`), campos de API (`boilerplate`), rutas (`/teacher/anti-plagio`) y valores de enum (`JUEZ_EFIMERO`, `IDE_PERSISTENTE`).

## Specs

- **S1 — Copy del docente.**
  - Los tres textos del generador con IA pasan a «Evaluador Automático».
  - Criterio: `rg -n "Juez Virtual" frontend/src/app --glob '*.html'` no devuelve nodos de texto visibles.
- **S2 — Copy del admin.**
  - Studio, catálogo de plantillas, biblioteca de modelos y configuración de IA usan los términos canónicos.
  - Criterio: ninguna pantalla del admin muestra «Juez Virtual», «Laboratorio IDE» ni «Juez Efímero» como copy.
- **S3 — Verificación.**
  - `rg` de control sobre `.html` y `.ts` excluyendo comentarios, specs y nombres de identificadores.
  - `npm run test:ci` de los componentes tocados y `CI=true npm run build`.
  - No se agrega gate de CI nuevo; el proyecto ya tiene los gates de estilo y de datos técnicos.

## Tasks

- [ ] **T1 — S1**: `ai-exercise-generator-modal.component.html` (3 reemplazos).
- [ ] **T2 — S2**: `template-studio.component.html`, `admin-templates.component.html`, `model-library.component.html`, `component-form-modal.component.html`, `template-edit-modal.component.html`, `template-review-modal.component.html`, `admin-config-ia.component.html`.
- [ ] **T3 — S3**: barrido de control, specs afectados y build.

## Dudas abiertas

- **D1 — Alcance admin.** Recomendación: incluir las pantallas del admin porque el usuario las ve y el vocabulario debe ser uno solo. Alternativa: dejar el admin para otra pasada y limitar esta ficha a docente/alumno.
- **D2 — «Juez Efímero» y «Laboratorio IDE».** Son términos técnicos del catálogo, no el «Juez Virtual» que pide `Corregir.md`. Recomendación: reemplazarlos también en el copy; no tocar los enums.
- **D3 — Renombres internos.** Recomendación: no renombrar `boilerplate` en API/DB ni selectores. Es un cambio incompatible que no mejora la pantalla y obliga a migrar contratos.

## Verificación

```bash
cd frontend && rg -n "Juez Virtual|Laboratorio IDE|Juez Efímero|Evaluador Juez" src/app --glob '*.html'
cd frontend && npx eslint src/app/features/teacher/exercise-editor/ai-exercise-generator-modal src/app/features/admin/templates src/app/features/admin/configuracion/tabs
cd frontend && npm run test:ci -- --include='**/ai-exercise-generator-modal/**' --include='**/template-studio/**'
cd frontend && npm run lint:styles && CI=true npm run build
```

## Log

- **L1 (verbatim):** «Limpieza de vocabulario residual: Barrer los textos viejos registrados en Corregir.md.»
- **Hallazgo verificado:** los badges y el visor de la ficha de práctica ya están corregidos; el inventario de `Corregir.md` está desactualizado en ese punto.
- **Hallazgo verificado:** el copy viejo que queda está concentrado en el generador con IA y en el Studio/catálogo del admin.
