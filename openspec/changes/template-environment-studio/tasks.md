# Tareas: Constructor de Entornos Componibles por Catálogo de Tecnologías y Construcción Local Determinista de Imágenes (`solv-local/stack:<sha256>`)

## Pronóstico de Carga de Revisión

| Campo | Valor |
|-------|-------|
| Líneas estimadas modificadas | 1,400–1,800 (en 4 unidades de trabajo autocontenidas) |
| Riesgo de presupuesto de 400 líneas | Alto |
| PRs encadenados recomendados | Sí |
| División sugerida | 4 unidades de trabajo (~350–400 líneas cada una) |
| Estrategia de entrega | ask-on-risk |

### Unidades de Trabajo

| Unidad | Objetivo | Comando de prueba focalizado | Límite de reversión |
|--------|----------|------------------------------|---------------------|
| 1 | Dominio Backend, Migración y `TemplateComposerService` Determinista (SHA-256 + `pip`/`npm`/`apt` con lista blanca + CRUD del Catálogo de Tecnologías) | `cd backend && go test ./internal/core/domain/... ./internal/core/services/... -run 'Component|Compose|Stack' -count=1` | `template_component.go`, `template_composer_service.go`, migración |
| 2 | Construcción Local de Imágenes en el Nodo (`ImageBuilderPort` en `EnvTestService`), Limpieza de Imágenes Huérfanas, Actualización sin Corte y Endpoints HTTP | `cd backend && go test ./internal/core/services/... ./internal/delivery/http/... -run 'EnvTest|Component|Compose' -count=1` | `env_test_service.go`, `admin_academic_handler.go`, `router.go` |
| 3 | Catálogo Vectorial `TechLogoComponent` (31 Iconos SVG + Monograma de Respaldo) y Pestaña "Catálogo de Tecnologías" + Modal Asistido en `ModelLibraryComponent` | `cd frontend && npx ng test --watch=false --include='**/tech-logo*.spec.ts' --include='**/model-library*.spec.ts'` | `tech-logo/`, `model-library/`, `admin-templates.service.ts` |
| 4 | Pantalla Dedicada `TemplateStudioComponent` (`/admin/plantillas/nueva`, `/admin/plantillas/:id/editar`), Selección de Tecnologías para Docentes y Documentación ADR | `cd frontend && npx ng test --watch=false --include='**/template-studio*.spec.ts' && npm run lint:styles && CI=true npm run build` | `templates/studio/`, `app.routes.ts`, `docs/ARQUITECTURA/ADR/` |

---

## Fase 1: Dominio Backend, Migración y Compositor Determinista

- [x] 1.1 RED: Escribir pruebas unitarias en `backend/internal/core/domain/template_component_test.go` para `ValidatePackageNames` (aceptando paquetes válidos `pip`/`npm`/`apt` y rechazando metacaracteres de shell con error 422), `SynthesizeAssistedBuildLayer` y `ComputeStackSignature` (hash SHA-256 determinista `solv-local/stack:<12-hex>`).
- [x] 1.2 GREEN: Implementar `TemplateComponent`, `StackConfig`, `ComposedTemplatePlan`, `ValidatePackageNames`, `SynthesizeAssistedBuildLayer` y `ComputeStackSignature` en `backend/internal/core/domain/template_component.go` y extender `CreateOfficialTemplateDTO` / `AdminTemplateReviewItem` / `TemplateModelItemDTO` en `admin_governance.go`.
- [x] 1.3 RED: Escribir pruebas unitarias en `backend/internal/core/services/template_composer_service_test.go` cubriendo: (a) resolución directa de imagen para un único runtime (`NeedsLocalBuild = false`), (b) composición multicapa (`Python + Angular + FastAPI + MySQL` -> `NeedsLocalBuild = true`, `solv-local/stack:<hash>`, herramientas combinadas, carpetas y RAM recomendada), y (c) resolución automática de `entrypoint` para `JUEZ_EFIMERO`.
- [x] 1.4 GREEN: Implementar `TemplateComposerService` y los métodos CRUD del catálogo de tecnologías en `backend/internal/core/services/template_composer_service.go` y `backend/internal/infrastructure/storage/postgres/template_components_repository.go` junto con la migración `00022_template_components_and_studio.sql`.

## Fase 2: Construcción Local de Imágenes, Limpieza de Huérfanas y Endpoints HTTP

- [x] 2.1 RED: Escribir pruebas unitarias en `backend/internal/core/services/env_test_building_test.go` para `StartJob` con `DockerfileContent`: (a) acierto de caché en `solv-local/stack:<hash>` omite `BuildLocalStackImage`, (b) fallo de caché invoca `BuildLocalStackImage` y reporta el avance antes de verificar los binarios, y (c) `PruneOrphanStackImages` elimina tags `solv-local/stack:*` no referenciados conservando los activos.
- [x] 2.2 GREEN: Extender `EnvTestService` y el adaptador Docker con `ImageBuilderPort` (`BuildLocalStackImage` y `PruneOrphanStackImages`) y actualización sin corte en `admin_governance_service.go`.
- [x] 2.3 RED: Escribir pruebas de handlers HTTP en `backend/internal/delivery/http/admin_academic_components_test.go` para `GET/POST/PUT/DELETE /api/v1/admin/template-components` (incluyendo 422 ante nombres de paquete inválidos), `POST /api/v1/admin/templates/compose-preview` y `PUT /api/v1/admin/templates/{id}`.
- [x] 2.4 GREEN: Conectar los endpoints en `backend/internal/delivery/http/admin_academic_handler.go` y `backend/internal/delivery/http/router.go`. Ejecutar `cd backend && go vet ./... && go test ./... -count=1`.
- [x] 2.5 GREEN: Blindar el alta y la edición de plantillas con imagen local en `admin_governance_templates.go` y `admin_governance_components.go`: exigir un job `success` del mismo usuario y tenant cuya huella coincida con la configuración efectiva (incluida la entrada de muestra), saltar la whitelist externa solo para `solv-local/stack:*`, responder `422 verification_required` en `CreateTemplate` y `UpdateOfficialTemplate`, y cubrirlo con `admin_governance_verification_test.go` más los casos de pertenencia en `env_test_handler_test.go`. Comando: `cd backend && go test ./internal/core/services/... ./internal/delivery/http/... -run 'Verification|LocalImage|EnvTest' -count=1`.

## Fase 3: Catálogo de Iconos Vectoriales e Interfaz del Catálogo de Tecnologías

- [x] 3.1 RED: Escribir pruebas de componente en `frontend/src/app/features/admin/templates/components/tech-logo/tech-logo.component.spec.ts` verificando la resolución de los 31 iconos vectoriales SVG oficiales en `public/tech-icons/` y el monograma determinista de 2 letras como respaldo.
- [x] 3.2 GREEN: Actualizar `TechLogoComponent` (`tech-logo`) y extender `AdminTemplatesService` (`admin-templates.service.ts`) y `admin-templates.models.ts` con tipos y métodos para el catálogo de tecnologías (`getTechCatalog`, `createTechCatalogItem`, `deleteTechCatalogItem`, `composeTemplatePreview`, `updateOfficialTemplate`).
- [x] 3.3 RED: Escribir pruebas de componente para `ComponentFormModalComponent` y la pestaña **"Catálogo de Tecnologías"** en `ModelLibraryComponent` (`model-library.component.spec.ts`).
- [x] 3.4 GREEN: Implementar `ComponentFormModalComponent` (`selector: 'component-form-modal'` usando `<modal-shell>`) e integrar la pestaña **"Catálogo de Tecnologías"** en `ModelLibraryComponent`. Verificar `npm run lint:styles` y `npm run lint:ts`.

## Fase 4: Pantalla Dedicada del Constructor de Entornos, Solicitudes Docentes y ADR

- [x] 4.1 RED: Escribir pruebas de componente en `frontend/src/app/features/admin/templates/studio/template-studio.component.spec.ts` verificando: (a) estado inicial limpio sin tecnologías ni categorías preseleccionadas, (b) filtrado contextual de la Columna 2 según la Columna 1, (c) sincronización bidireccional entre el **Modo Visual** y el **Modo Avanzado (Dockerfile / Scripts)**, (d) resolución automática del `entrypoint` para `JUEZ_EFIMERO`, y (e) modelo de memoria por procesos residentes sin inflación por librerías en disco, barra segmentada `.memory-bar`, candados de piso vital / techo del servidor y `resource_profile` en `buildCreatePayload()`.
- [x] 4.2 GREEN: Implementar `TemplateStudioComponent` (`selector: 'template-studio'`) y registrar `/admin/plantillas/nueva` y `/admin/plantillas/:id/editar` en `app.routes.ts`.
- [x] 4.3 GREEN: Actualizar el flujo de solicitud docente (`frontend/src/app/features/teacher/templates/`) con selección de tecnologías del catálogo y documentar `ADR-046` en `docs/ARQUITECTURA/ADR/` + `INDEX.md`.
- [x] 4.4 Verificación final: Ejecutar `cd backend && go vet ./... && go test ./... -count=1` y `cd frontend && npm run lint:styles && npm run lint:ts && npx ng test --watch=false && CI=true npm run build`.
- [x] 4.5 GREEN: Convertir las tarjetas de selección de las tres columnas en `<button>` nativos con `aria-pressed`, descripción visible y el control de edición como botón hermano (sin controles anidados), y agregar las media queries de responsive en `template-studio.component.scss` (1100px y 720px). Comando: `cd frontend && npx ng test --watch=false --include='**/template-studio.component.spec.ts' && npm run lint:styles`.
