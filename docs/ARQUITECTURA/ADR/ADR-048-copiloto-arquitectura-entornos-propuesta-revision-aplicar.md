# ADR-048: Copiloto IA para Arquitectura de Entornos — Flujo Propuesta → Revisión → Aplicar y Primitivo Compartido de Operaciones de IA

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

## Actualización — Primitivo compartido de operaciones de IA (2026-10-28)

### Contexto y problema

El front consume cuatro acciones de IA. Sus presupuestos de tiempo los resuelve `domain.AIEngineAutoTimeoutSeconds` (`internal/core/domain/ai_settings.go`), y `queued_or_reasoning` y `local_gpu` comparten valores:

| Acción | `fast_cloud` | `queued_or_reasoning` / `local_gpu` |
|---|---|---|
| `suggest-component` | 60 s | 90 s |
| `suggest-template` | 90 s | 120 s |
| `repair-template` | 90 s | 120 s |
| `generate-exercise` | 120 s | 180 s |

Cada una reimplementaba su propio ciclo de vida dentro del componente que la lanzaba: un `signal` de «generando», un `setInterval` de telemetría, una `Subscription` y un handler de cancelación. Cuatro copias del mismo patrón, con variantes.

El defecto común: **el dueño del estado era la vista**. `ngOnDestroy` desuscribía la llamada, lo que aborta el request HTTP. Con presupuestos de 60 a 180 segundos, cualquier destrucción del componente —navegar a otra ruta, cambiar de contexto— era una pérdida garantizada de trabajo. En el Studio de Plantillas eso se manifestaba al salir de la vista, porque el modal solo recibe `input()`/`output()` y el estado real vivía en el componente padre. El modal del docente bloqueaba el cierre con `[disableClose]="isGenerating()"`, así que ahí la pérdida también llegaba por navegación. Y el aviso de finalización solo existía en dos de las cuatro operaciones: `suggest-template` y `repair-template` lo emitían desde el propio componente, mientras `generate-exercise` y `suggest-component` no avisaban nada, ni inline ni por toast. La operación más larga de todas —hasta 180 segundos en perfil de cola— era la más silenciosa.

### Decisión

1. **Un primitivo para el ciclo de vida.** `AiOperation<T>` (`shared/services/ai-operation.ts`) modela la llama como una máquina de estados: `idle → running → ready | error`. Expone `state`, `result`, `errorMessage`, `elapsedSeconds` e `isRunning`, gestiona su propio cronómetro, cancela con `cancel()` y descarta respuestas tardías con un contador de intentos. Es una clase plana, sin DI, para poder testearse aislada.

2. **El alcance lo decide el destinatario del resultado, no la acción.** Una operación vive en `AiOperations` (`providedIn: 'root'`) **si y solo si** su resultado se aplica a un estado que sobrevive a la vista. Si el resultado solo tiene sentido dentro del formulario que lo pidió, la operación muere con ese formulario.

   | Operación | Destinatario | Alcance |
   |---|---|---|
   | `suggest-template` | borrador del constructor (`StudioDraft`) | raíz |
   | `repair-template` | manifiesto del entorno en edición | raíz |
   | `generate-exercise` | ejercicio generado, aplicable al editor | raíz |
   | `suggest-component` | campos de un modal de alta descartable | el modal |

   La distinción es de naturaleza, no de grado: un ejercicio generado es un artefacto de dominio que sirve aunque el modal se haya cerrado; la sugerencia de un componente son campos de un formulario de alta que el administrador ya descartó. Encolar la segunda o conservarla «por las dudas» produce resultados sin destinatario.

3. **Restauración por reclamo.** Cuando una operación de alcance raíz terminó mientras el componente no estaba montado, el feature llama `take()` en `ngOnInit`. `take()` entrega el resultado una sola vez y deja la operación en `idle`: es un *claim*, no un *peek*. Sin él convivían dos caminos de aplicación —el callback `onSuccess` y la restauración en `ngOnInit`— y el mismo resultado podía aplicarse dos veces. Para el alcance local, `dispose()` libera la llamada al morir su dueño sin emitir el toast de cancelación, porque nadie canceló nada.

4. **Separación entre motor y presentación.** La operación no conoce el dominio de cada feature. El feature aporta `resolveError` para traducir el error a un mensaje de usuario, `onSuccess`/`onError` para poblar formularios o diagnóstico de forma sincrónica, y `notify` para decidir qué se anuncia por el toast global (el error inline del formulario no se duplica en pantalla).

5. **El estado del Copiloto y el borrador del Studio.** `StudioCopilotService` (`providedIn: 'root'`) es dueño de `prompt`, `isSuggesting`, `proposal`, `applied`, `elapsedSeconds` y del `StudioDraft` con los campos del formulario (nombre, descripción, categoría, entorno objetivo, selección de piezas, Dockerfile, script de setup, RAM, etc.). Un `effect` del componente actualiza el borrador y `ngOnInit` lo restaura cuando la ruta no es de edición ni duplicación, avisando «Borrador del Studio restaurado». Se descarta al publicar la plantilla. El componente expone esos mismos signals para no cambiar los bindings del template ni la API que consumen los tests.

6. **Aviso global.** `ToastService` + `ToastHostComponent` (montado en `admin-layout`) reutilizan el primitivo `.toast-notification` de `styles.scss`. El aviso sale de la operación, no del componente, para que llegue aunque el administrador esté en otra vista.

7. **No se convierte en job de backend.** Se mantiene el criterio del proyecto: los jobs (`EnvTestJobService`) existen para operaciones con efectos sobre el sistema y no regenerables. Las cuatro acciones de IA son funciones puras sin efectos que devuelven un valor reproducible, así que el ciclo de vida vive en un store de cliente y no en la cola de trabajos.

### Consecuencias

- **Positivas:** el ciclo de vida queda escrito una vez en lugar de cuatro. Las tres operaciones cuyo resultado sobrevive a la vista dejan de abortar su llamada al navegar; el docente puede cerrar el modal y recuperar el ejercicio generado; el aviso de finalización llega esté donde esté el usuario. El botón «Cancelar» aborta de verdad el request. Los tests del primitivo cubren estados, cancelación, reclamo, respuestas tardías y avisos.
- **Negativas / Reducción de impacto:** el estado no sobrevive a recargar la página ni a cerrar la pestaña, porque vive en memoria del cliente. Se acepta de forma explícita: cubrirlo exigiría jobs en backend, coste que no se justifica para un valor regenerable. Las opciones de una operación (`onSuccess`, `notify`) se capturan en su primera obtención, de modo que la restauración tras remontar depende de `ngOnInit` y no del callback original.
- **Riesgo acotado:** `suggest-component` no participa del registro de la raíz, así que un resultado pendiente no puede aplicarse al formulario de otro componente. Cerrar el modal descarta su llamada, que es el comportamiento correcto: el administrador descartó ese componente.
- **Efecto colateral corregido:** cancelar el Copiloto ya no limpia el cronómetro de «Corregir con Copiloto IA» (autorreparación), que compartía el mismo `timerInterval`.
- **Pendiente, fuera de este ADR:** el borrador del Studio se pierde al recargar. Es trabajo manual del administrador, no una llamada regenerable, y corresponde resolverlo con persistencia local o en backend en su propio cambio.
- **Descartado explícitamente:** una cola FIFO de operaciones de IA. El backend no comparte un recurso escaso (`ai_copilot_service.go` dispara un request independiente por llamada; la cola del proveedor es asunto del proveedor), y encolar introduce latencia acumulada y produce resultados sin destinatario.
