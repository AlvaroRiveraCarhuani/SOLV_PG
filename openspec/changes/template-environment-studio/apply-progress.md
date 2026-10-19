# Avance de Aplicación: Constructor de Entornos Componibles (`template-environment-studio`)

## Estado
- Las 4 unidades de trabajo (16/16 tareas) fueron implementadas bajo TDD (RED -> GREEN) y verificadas.

## Resumen de Cambios

### Unidad 1: Dominio Backend, Migración y Compositor Determinista
- `backend/internal/core/domain/template_component.go` y `template_component_test.go`: Modelos `TemplateComponent`, `StackConfig`, `ComposedTemplatePlan`, validación por expresión regular `ValidatePackageNames`, generador asistido `SynthesizeAssistedBuildLayer` (`pip`, `npm`, `apt`) y cálculo de firma SHA-256 `ComputeStackSignature` (`solv-local/stack:<12-hex>`).
- `backend/internal/core/services/template_composer_service.go` y `template_composer_service_test.go`: `TemplateComposerService` con resolución directa de imagen o composición multicapa, comando automático para `JUEZ_EFIMERO` y modelo de memoria por procesos residentes.
- `backend/migrations/00022_template_components_and_studio.sql` y `backend/internal/infrastructure/storage/postgres/template_components_repository.go`: Tabla `template_components` con semilla institucional y columna `stack_config JSONB` en `lab_templates` y `template_models`.

### Unidad 2: Construcción Local de Imágenes en el Nodo, Limpieza de Huérfanas y Endpoints HTTP
- `backend/internal/core/services/env_test_service.go`, `env_test_building_test.go` y `backend/internal/infrastructure/docker/env_test_adapter.go`: `ImageBuilderPort` (`BuildLocalStackImage` y `PruneOrphanStackImages`) para construir una sola vez `solv-local/stack:<sha256>` y reutilizar la caché local en ejecuciones posteriores.
- `backend/internal/delivery/http/admin_academic_handler.go`, `admin_academic_components_test.go` y `router.go`: Endpoints `GET/POST/PUT/DELETE /api/v1/admin/template-components`, `POST /api/v1/admin/templates/compose-preview` y `PUT /api/v1/admin/templates/{id}`.

### Unidad 3: Catálogo de Iconos Vectoriales e Interfaz del Catálogo de Tecnologías
- `frontend/src/app/features/admin/templates/components/tech-logo/`: `TechLogoComponent` con 31 logotipos vectoriales SVG oficiales en `public/tech-icons/` y monograma determinista de 2 letras como respaldo.
- `frontend/src/app/features/admin/templates/components/model-library/`: `ComponentFormModalComponent` (con primitivas `modal-shell` y `form-field`, modo asistido `pip`/`npm`/`apt` y selector `memoryProfile`) y pestaña **"Catálogo de Tecnologías"** en `ModelLibraryComponent`.

### Unidad 4: Pantalla Dedicada del Constructor de Entornos, Solicitud Docente y ADR-046
- `frontend/src/app/features/admin/templates/studio/`: `TemplateStudioComponent` (`/admin/plantillas/nueva` y `/admin/plantillas/:id/editar`) con layout de 2 paneles, selector visual de 3 columnas sincronizado con el Modo Avanzado, barra segmentada de memoria, candados de seguridad y panel fijo de Resumen del Entorno.
- `frontend/src/app/features/teacher/templates/template-request-modal/`: Solicitud docente conectada al catálogo de componentes tecnológicos.
- `docs/ARQUITECTURA/ADR/ADR-046-estudio-plantillas-componentes-construccion-local.md` e `INDEX.md`: Registro documental de `ADR-046`.
