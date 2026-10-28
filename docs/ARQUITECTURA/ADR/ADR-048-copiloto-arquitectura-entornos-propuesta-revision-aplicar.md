# ADR-048: Copiloto IA para Arquitectura de Entornos — Flujo Propuesta → Revisión → Aplicar

- **Estado:** Aprobado
- **Fecha:** 2026-10-25
- **Contexto:** Core / Gestión de Entornos, Studio de Plantillas y Gobernanza de IA

## Contexto y Problema

La sección «Copiloto IA para Arquitectura de Entornos» del Studio de Plantillas (`/admin/plantillas/nueva`, rol admin) presentaba dos problemas estructurales:

1. **Falsa promesa de aplicación inmediata.** El botón «Sugerir y Configurar Entorno» invocaba `POST /api/v1/admin/ai/suggest-template` y el texto de ayuda afirmaba que el Copiloto «actualizará» el constructor visual y el manifiesto técnico. Ese endpoint **nunca fue registrado en el router HTTP** (solo existían `suggest-component`, `repair-template` y `generate-exercise`): cualquier clic terminaba en `404`. La acción `suggest-template` ya estaba presupuestada en `ai_settings.go` (timeouts 90–120 s), pero la implementación quedó a medio camino.

2. **Aplicación opaca y sin reversa.** En el `next:` de la suscripción, la respuesta del LLM se aplicaba en caliente sobre el constructor (runtime, web, addons, base de datos, script de setup, RAM, nombre y descripción) sin que el administrador pudiera ver *qué* se proponía antes de que se aplicara, sin indicar qué piezas no existían en el catálogo del tenant y sin forma de descartar. Combinado con el catálogo componible del ADR-046 (slugs propios por componente), un slug inventado o no presente (p. ej. `gcc` frente a perfiles `c-gcc13`/`cpp-gcc13`) pasaba desapercibido y se perdía en silencio.

## Decisión

1. **Endpoint real y sanado** `POST /api/v1/admin/ai/suggest-template` (rol admin), implementado en `AICopilotService.SuggestTemplate`:
   - `prompt` obligatorio de 3..600 caracteres; `target_environment` opcional (`JUEZ_EFIMERO` | `IDE_PERSISTENTE` | `AMBOS`, default `JUEZ_EFIMERO`) validado en servidor con `target_environment_invalid`.
   - El LLM responde JSON con `name`, `description`, `runtime_slug`, `web_platform_slug`, `addon_slugs`, `database_slug`, `setup_script`, `base_ram_mb` y `rationale`. El servicio sanea sin confianza ciega: limpia fences Markdown, normaliza slugs a minúsculas sin duplicados ni `"none"`/`"null"`, acota `base_ram_mb` a [256, 4096] con default 512, exige `runtime_slug` no vacío (`runtime_slug_missing`) y `rationale` obligatorio (`ai_response_invalid`).
   - Errores mapeados como en el resto de acciones de IA: `prompt_invalid`, `target_environment_invalid`, `ai_response_invalid`, `runtime_slug_missing` (400); `ai_disabled` (409); `ai_quota_exceeded` (429); `ai_timeout` (504).

2. **El backend propone, el frontend resuelve, el administrador decide.** El endpoint **no aplica ni persiste nada**: devuelve la propuesta estructurada. El Studio (`template-studio`) la resuelve contra el catálogo real del tenant mediante helpers puros en `template-studio.utils.ts` (`findCatalogComponentMatch` por id → slug → `icon_key`, y `findDatabaseOptionMatch` por `service_engine` → id → slug):

   - **Nueva vista de resolución de la propuesta (`aiProposalView`):** cada pieza se muestra como chip con su logo técnico y sello de disponibilidad:
     - `✓` pieza existente en el catálogo (chip con `check-circle`), lista para aplicarse.
     - `✗` slug ausente (chip con `alert-triangle` mostrando el slug en mono con `machineData`), con la advertencia global «No está en el catálogo actual y no se aplicará» listando los slugs faltantes.
   - **Estado `aiProposalApplied`:** los botones «Aplicar al constructor» / «Descartar propuesta» solo aparecen tras la revisión. Al deshacer tras aplicar, el constructor conserva los cambios ya sincronizados. Solo «Aplicar al constructor» (`applyAiProposal`) ejecuta la sincronización visual + manifiesto avanzado (recalcula `recalculateFromVisualSelection`, hereda nombre/descripción solo si no fueron editados manualmente, script de setup y RAM acotada a 256+).
   - El texto de ayuda de la tarjeta se reescribió para describir con precisión el contrato: *«El Copiloto propondrá una arquitectura completa: usted la revisa y decide si aplicarla»*.

3. **Un único flujo conserva el patrón container-presentational del ADR-046:** no se crea un componente nuevo; la lógica de resolución vive en utilidades puras y el panel de propuesta se renderiza en `template-studio.component.html` con los tokens existentes (`tech-chip`, `status-running-*`, `status-failed-*`, `status-pending-*`).

## Consecuencias

- **Positivas:** el administrador ve *qué* se va a aplicar antes de que se toque el constructor, incluidas las piezas ausentes del catálogo (educa el prompt y evita stacks rotos en silencio); el endpoint queda alineado con los budgets de `suggest-template` ya reservados; el saneo de slugs y memoria ocurre en servidor, con tests (`TestAICopilot_SuggestTemplate_ReturnsProposalWithSanitizedSlugs`).
- **Negativas / Reducción de impacto:** un paso más entre «sugerir» y «aplicar» (revisión explícita), compensado porque la propuesta descarta piezas no disponibles de forma visible; los slugs canónicos del LLM pueden no empatar con slugs de catálogo personalizados, que se marcan como «no disponible» y exigen selección manual (comportamiento intencionado, no silencioso).
- **No cubierto:** no se persiste la propuesta ni existe historial de sugerencias; el Copiloto sigue siendo una asistencia de autoría, no un automatismo de publicación (la publicación mantiene el blindaje de «Comprobar funcionamiento» previo del ADR-046).

## Actualización — Operación desacoplada de la vista (2026-10-28)

### Contexto y problema

El estado del Copiloto vivía en `TemplateStudioComponent`. Como el endpoint `suggest-template` tiene un presupuesto de 90–120 s (`AIEngineAutoTimeoutSeconds`), el administrador podía salir de la vista mientras esperaba, y eso destruía el trabajo: `ngOnDestroy` desuscribía `suggestSub`, lo que aborta el request HTTP, y la propuesta y el borrador del constructor se perdían sin aviso. Cerrar el modal, en cambio, no cancelaba nada porque el modal solo recibe `input()`/`output()` y el estado real estaba en el componente padre.

### Decisión

1. **El estado de la operación sale del componente.** `StudioCopilotService` (`providedIn: 'root'`) es dueño de `prompt`, `isSuggesting`, `proposal`, `applied`, `elapsedSeconds` y de la suscripción al endpoint, junto con su propio cronómetro. El componente expone esos mismos signals para no cambiar los bindings del template ni la API que consumen los tests. Navegar a otra vista ya no aborta la operación, y cerrar el modal tampoco.
2. **El borrador del Studio también sobrevive.** `StudioCopilotService` guarda un `StudioDraft` con los campos del formulario (nombre, descripción, categoría, entorno objetivo, selección de piezas, Dockerfile, script de setup, RAM, etc.). Un `effect` del componente lo actualiza, y `ngOnInit` lo restaura cuando la ruta no es de edición ni duplicación, avisando «Borrador del Studio restaurado». Se descarta al publicar la plantilla.
3. **Aviso global.** Se agrega `ToastService` + `ToastHostComponent` (montado en `admin-layout`), reutilizando el primitivo global `.toast-notification` de `styles.scss`. El aviso de propuesta lista, el error y la cancelación salen del servicio, no del componente, para que lleguen aunque el administrador esté en otra vista.
4. **No se convierte en job de backend.** Se mantiene el criterio del proyecto: los jobs (`EnvTestJobService`) existen para operaciones con efectos sobre el sistema y no regenerables. `suggest-template` es una función pura sin efectos que devuelve un valor reproducible con un clic, así que el ciclo de vida vive en un store de cliente y no en la cola de trabajos.

### Consecuencias

- **Positivas:** el administrador puede seguir trabajando mientras el Copiloto diseña; al volver encuentra el estado real (en curso con su cronómetro, o la propuesta lista) y el constructor conserva lo que había armado. El botón «Cancelar» aborta de verdad el request.
- **Negativas / Reducción de impacto:** el cambio no sobrevive a recargar la página ni a cerrar la pestaña, porque el estado vive en memoria del cliente. Se acepta de forma explícita: cubrirlo exigiría jobs en backend, coste que no se justifica para un valor regenerable.
- **Efecto colateral corregido:** cancelar el Copiloto ya no limpia el cronómetro de «Corregir con Copiloto IA» (autorreparación), que compartía el mismo `timerInterval`.
