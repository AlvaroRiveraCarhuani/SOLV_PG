# Feature: Grounding del Catálogo para Copiloto IA, Resolución Semántica y Creación Contextual de Tecnologías Faltantes
Branch: main | Delivery strategy: ask-on-risk | Test runner: go test ./... & npm run test:ci

## Specs

- **S1**: **Inyección Compacta del Catálogo Real (Grounding) en `SuggestTemplate` (Backend Go)**:
  - En `backend/internal/core/services/ai_copilot_service.go`:
    - Permitir inyectar `domain.TemplateComponentRepository` en `AICopilotService` (de forma retrocompatible en `NewAICopilotService(aiSettings, runner, compRepo...)` o via setter/constructor).
    - En `SuggestTemplate(ctx, tenantID, userID, req)`:
      - Si el repositorio de componentes está disponible, listar los componentes activos del tenant (`ListTemplateComponents(ctx, tenantID, "", false)`).
      - Construir un resumen textual compacto y estructurado por familias (menos de 300 tokens):
        - `Runtimes`: `[slug (Nombre), ...]`.
        - `Web Platforms`: `[slug (Nombre), ...]`.
        - `Addons agrupados por ecosistema`: `[slug (Nombre), ...]`.
        - `Databases`: `sqlite`, `postgres`, `mysql`, `redis`, `mongodb`.
      - Incorporar este resumen en el `suggestTemplateSystemPrompt(catalogSummary)`.
      - Instruir al modelo:
        1. Debe priorizar y reutilizar **exclusivamente** los slugs exactos del catálogo provisto cuando correspondan a la necesidad.
        2. Si una herramienta solicitada no está en el catálogo pero existe una alternativa directa instalada en la misma categoría, usar la del catálogo y justificarlo en `rationale`.
        3. Solo si no existe alternativa viable en el catálogo, declarar la tecnología faltante en el campo estructurado `missing_components`.

- **S2**: **Contrato de Respuesta con Faltantes Estructurados (`missing_components`) en Backend**:
  - En `backend/internal/core/domain/ai_copilot.go`:
    - Añadir `AIMissingComponentDTO`:
      - `Name string` (`json:"name"`): Nombre canónico de la herramienta (ej. "PyTorch", "Celery").
      - `Layer string` (`json:"layer"`): `"runtime"`, `"addon"`, `"database"`.
      - `Category string` (`json:"category"`): `"backend"`, `"web"`, `"framework"`, `"orm"`, `"toolkit"`, `"heavy"`.
      - `Reason string` (`json:"reason"`): Justificación técnica breve de por qué se requiere y no está en el catálogo.
    - Ampliar `AISuggestTemplateResponseDTO` con `MissingComponents []AIMissingComponentDTO` (`json:"missing_components,omitempty"`).
  - En `backend/internal/core/services/ai_copilot_service.go`:
    - Parsear y sanitizar `missing_components` asegurando que no vengan vacíos ni duplicados.
  - Actualizar tests unitarios en `ai_copilot_service_test.go` para verificar el grounding y la detección de faltantes estructurados.

- **S3**: **Reactividad Desacoplada y Resolución Semántica en el Frontend (`template-studio`)**:
  - En `frontend/src/app/features/admin/templates/studio/template-studio.component.ts`:
    - Corregir el cálculo de `addonChoices` dentro de `aiProposalView`: en lugar de leer el estado estático del constructor (`this.compatibleAddons()`), calcular dinámicamente con `computeCompatibleAddonsForSelection(this.techComponents(), runtime, web, this.targetEnvironment())`.
  - En `frontend/src/app/features/admin/templates/studio/template-studio.utils.ts`:
    - Mejorar `findCatalogComponentMatch` para que resuelva tanto coincidencia exacta (`id`, `slug`, `icon_key`) como variantes de versión (`python` -> `python-312`) y nombres de paquetes incluidos (`package_names` / `tools_provided`).

- **S4**: **Acción Contextual "Crear Tecnología Faltante" con Retorno de Estado (UX Frontend)**:
  - En `frontend/src/app/features/admin/templates/studio/components/studio-ai-copilot-modal/studio-ai-copilot-modal.component.html` y `.ts`:
    - Reemplazar el banner pasivo de advertencia por tarjetas de acción de faltantes:
      - Mostrar el nombre y motivo del componente faltante.
      - Botón de acción: **`Diseñar [Nombre] con Copiloto IA`**.
  - En `frontend/src/app/features/admin/templates/studio/template-studio.component.ts` y `.html`:
    - Conectar el botón con la apertura de `component-form-modal`, manteniendo en pausa la propuesta del Copiloto (sin perder el prompt ni el estado actual).
    - Pre-llenar el prompt del asistente de componentes con el nombre y contexto del faltante.
    - Al emitir `(saved)`, agregar el componente guardado a `techComponents` y auto-seleccionarlo inmediatamente en la propuesta activa, cerrando el ciclo sin clics redundantes.

- **S5**: **Blindaje de Calidad, Tests y Gates de Diseño**:
  - Actualizar y ejecutar los tests unitarios de Go (`go test ./internal/...`).
  - Actualizar y ejecutar los tests unitarios de Angular (`npm run test:ci`).
  - Verificar gates de estilo (`npm run lint:styles`) y build de producción (`CI=true npm run build`).

## Tasks

- [x] **T1** (S1, S2): Extender `AICopilotService` en Go para recibir `TemplateComponentRepository`, generar el resumen estructurado del catálogo e instruir al modelo con grounding y `missing_components` estructurados. Añadir tests unitarios en `ai_copilot_service_test.go`.
- [x] **T2** (S3): Corregir la reactividad de `addonChoices` en `aiProposalView` y enriquecer `findCatalogComponentMatch` con resolución por prefijos y paquetes en `template-studio.utils.ts`.
- [ ] **T3** (S4): Implementar en la UI del modal de propuesta las tarjetas de faltantes con botón `Diseñar con Copiloto IA`, enlazadas con `component-form-modal` y preservación de estado en `template-studio.component.ts`.
- [ ] **T4** (S5): Ejecución y pase de verificación integral: `go test ./...`, `npm run test:ci`, `npm run lint:styles` y build de producción.

## Log
- **L1 (2026-10-27)**: Diagnóstico de la inconsistencia en el Copiloto de Entornos: el modelo generaba slugs genéricos a ciegas (`python`, `sqlalchemy`), el backend no le suministraba el catálogo del tenant, y el frontend leía los complementos compatibles del constructor en lugar de la propuesta provisional. Se acordó la arquitectura con Grounding contextual compacto, detección estructurada de faltantes y flujo de creación con 1 clic sin pérdida de contexto.
