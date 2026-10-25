# Feature: Copiloto IA Autoadaptativo, Autorreparación de Entornos (Modo Visual + Experto), BYOK Docente y Rediseño UX
Branch: main | Delivery strategy: ask-on-risk | Test runner: go test ./... & npm run test:ci

## Specs
- **S1**: **Gobernanza Autoadaptativa (Zero-Config), Medición Real de Latencia y Error HTTP 504 (`ai_timeout`) en Backend**:
  - En `backend/internal/core/domain/ai_settings.go`:
    - Elevar `DefaultAITimeoutSeconds = 60` y `MaxAITimeoutSeconds = 240` (manteniendo `MinAITimeoutSeconds = 5`). Declarar `ErrAITimeout = errors.New("el motor de IA superó el tiempo máximo de espera configurado")`.
    - Agregar en `AISettingsResponseDTO` y `AIStatusDTO` descriptores de la política autoadaptativa resuelta por el servidor: `EngineProfile string` (`"fast_cloud"` | `"queued_or_reasoning"` | `"local_gpu"`) y `AutoTimeoutRangeLabel string` (ej. `"60s – 180s según complejidad de la tarea"`).
  - En `backend/internal/core/services/ai_settings_service.go` (`openAICompatibleProvider.Complete`):
    - Calcular `latencyMS := p.nowFn().Sub(start).Milliseconds()` **después** de `io.ReadAll(io.LimitReader(httpResp.Body, aiMaxResponseBytes))` (y también en las ramas de error de `client.Do` e `io.ReadAll`) para medir la descarga completa del stream JSON y no solo el TTFB de los headers HTTP.
    - Si `err != nil` en `client.Do` o `io.ReadAll` y `errors.Is(err, context.DeadlineExceeded)` o `errors.Is(ctx.Err(), context.DeadlineExceeded)`, devolver un error que envuelva `%w` con `domain.ErrAITimeout`.
  - En `backend/internal/core/services/ai_settings_service.go` (`CompleteJSON`):
    - Clasificar automáticamente el motor activo (`classifyAIEngine(provider, model)`):
      - `"local_gpu"`: cuando `provider == "ollama_lan"` o `domain.IsCustomAIProvider(provider)`.
      - `"queued_or_reasoning"`: cuando el `model` (en minúsculas) contiene `:free`, `r1`, `reasoner`, `thinking`, `nemotron`, `qwq`, `o1`, `o3`.
      - `"fast_cloud"`: cualquier otro modelo comercial en la nube (`gemini-2.5-flash`, `gpt-4o-mini`, `deepseek-chat`, etc.).
    - Calcular el timeout automático por `(engineProfile, action)` sin depender de que el usuario adivine segundos:
      - `"suggest-component"`: `60 s` en `fast_cloud` | `90 s` en `queued_or_reasoning` / `local_gpu`.
      - `"suggest-template"` y `"repair-template"`: `90 s` en `fast_cloud` | `120 s` en `queued_or_reasoning` / `local_gpu`.
      - `"generate-exercise"`: `120 s` en `fast_cloud` | `180 s` en `queued_or_reasoning` / `local_gpu`.
    - El timeout efectivo final es `min(MaxAITimeoutSeconds, max(autoTimeout, settings.TimeoutSeconds))`, garantizando que un `30` antiguo guardado en la base jamás recorte una operación pesada o un modelo gratuito, pero respetando si un administrador en Modo Experto fijó un tope mayor.
    - Para motores `"local_gpu"` (`ollama_lan`), elevar el piso efectivo de cuota anti-abuso a `max(settings.HourlyQuotaPerUser, 60)` solicitudes/hora dado que no existe costo por token externo.
  - En `backend/internal/delivery/http/ai_copilot_handler.go` (`sendAICopilotError`):
    - Si `errors.Is(err, domain.ErrAITimeout)`, responder `http.StatusGatewayTimeout` (`504`) con código `"ai_timeout"` y mensaje `"El motor de IA superó el tiempo límite de espera para esta operación. Probá nuevamente, usá un modelo más ágil o verificá la carga del proveedor."`.
    - En la rama `500` por defecto, incluir el motivo técnico en el mensaje (`fmt.Sprintf("%s: %s", fallbackMessage, err.Error())`) para nunca ocultar la causa real en el frontend.
- **S2**: **Credenciales en Cascada: Base Institucional + Override Personal por Usuario/Docente (BYOK) en Backend**:
  - En `backend/internal/core/domain/ai_settings.go`:
    - Añadir `AIUserOverrideStored` dentro de `AISettingsStored` como `UserOverrides map[string]AIUserOverrideStored` (`json:"user_overrides,omitempty"`), indexado por `userID`:
      - Campos: `Enabled bool` (`json:"enabled"`), `Provider string` (`json:"provider"`), `Model string` (`json:"model"`), `BaseURL string` (`json:"base_url,omitempty"`), `EncryptedAPIKey string` (`json:"encrypted_api_key,omitempty"`), `APIKeyMasked string` (`json:"api_key_masked,omitempty"`), `UpdatedAt string` (`json:"updated_at,omitempty"`).
    - Añadir DTOs públicos:
      - `AIUserOverrideResponseDTO`: `enabled` (`bool`), `provider` (`string`), `provider_label` (`string`), `model` (`string`), `base_url` (`string`), `has_api_key` (`bool`), `api_key_masked` (`string`), `updated_at` (`string`), `institutional_enabled` (`bool`), `institutional_provider_label` (`string`), `institutional_model` (`string`).
      - `UpdateAIUserOverrideDTO`: `enabled` (`bool`), `provider` (`string`), `model` (`string`), `base_url` (`string`), `api_key` (`string`), `clear_api_key` (`bool`).
    - Ampliar `AIStatusDTO` (`GET /api/v1/ai/status`) agregando `UsingPersonalKey bool` (`json:"using_personal_key"`) e `InstitutionalEnabled bool` (`json:"institutional_enabled"`).
  - En `backend/internal/core/services/ai_settings_service.go`:
    - Preservar siempre `UserOverrides` al leer (`Get`) y al actualizar la configuración institucional del administrador (`Update`), inicializando el mapa si es `nil`.
    - Implementar `GetUserOverride`, `UpdateUserOverride` y `GetEffectiveStatus` aplicando las mismas validaciones de proveedor, URL oficial en cloud, escudo anti-SSRF en `ollama_lan`/`custom_*`, y cifrado AES-256-GCM (`encryptAPIKey` / `maskAPIKey`).
    - En `CompleteJSON(ctx, tenantID, userID, action, systemPrompt, userPrompt)`: si el `userID` tiene un override personal activo y válido en `settings.UserOverrides[userID]`, ejecutar la inferencia usando el proveedor, modelo, `BaseURL` y `EncryptedAPIKey` personales del usuario **sin consumir la cuota horaria institucional compartida** y permitiendo operar aunque `settings.Enabled` institucional sea `false`.
  - En `backend/internal/delivery/http/ai_settings_handler.go` y `backend/internal/delivery/http/router.go`:
    - Actualizar `GET /api/v1/ai/status` para resolver `userID := getUserIDFromCtx(r)` y responder con `GetEffectiveStatus`.
    - Registrar para roles `teacher` y `admin` (rechazando `student` con `403 Forbidden`):
      - `GET /api/v1/ai/my-settings`
      - `PUT /api/v1/ai/my-settings`
      - `POST /api/v1/ai/my-settings/test-connection`
      - `POST /api/v1/ai/my-settings/models`
- **S3**: **Autorreparación Diagnóstica de Entornos (`POST /api/v1/admin/ai/repair-template`) para Modo Visual y Modo Avanzado en Backend**:
  - En `backend/internal/core/domain/ai_copilot.go`:
    - Definir `AIRepairTemplateRequestDTO`:
      - `Name string` (`json:"name"`)
      - `Description string` (`json:"description"`)
      - `TargetEnvironment string` (`json:"target_environment"`)
      - `DockerImage string` (`json:"docker_image"`)
      - `DockerfileContent string` (`json:"dockerfile_content"`)
      - `SetupScript string` (`json:"setup_script"`)
      - `ToolsDeclared []string` (`json:"tools_declared"`)
      - `Entrypoint string` (`json:"entrypoint"`)
      - `BaseRamMB int` (`json:"base_ram_mb"`)
      - `ErrorMessage string` (`json:"error_message"`)
      - `FailedTools []string` (`json:"failed_tools"`)
    - Definir `AIRepairTemplateResponseDTO`:
      - `DockerImage string` (`json:"docker_image"`)
      - `DockerfileContent string` (`json:"dockerfile_content"`)
      - `SetupScript string` (`json:"setup_script"`)
      - `ToolsDeclared []string` (`json:"tools_declared"`)
      - `Entrypoint string` (`json:"entrypoint"`)
      - `BaseRamMB int` (`json:"base_ram_mb"`)
      - `Rationale string` (`json:"rationale"`)
      - `LatencyMS int64` (`json:"latency_ms"`)
  - En `backend/internal/core/services/ai_copilot_service.go`:
    - Implementar `RepairTemplate(ctx context.Context, tenantID, userID string, req domain.AIRepairTemplateRequestDTO) (*domain.AIRepairTemplateResponseDTO, error)` que invoca `s.aiSettings.CompleteJSON(ctx, tenantID, userID, "repair-template", systemPrompt, userPrompt)`.
    - El prompt recibe el snapshot diagnóstico (sin historial de chat): la imagen/Dockerfile actual, `setup_script`, `tools_declared`, `entrypoint`, `base_ram_mb`, el `error_message` devuelto por el contenedor Docker y la lista `failed_tools` (herramientas que dieron `present: false`).
    - El motor devuelve el manifiesto técnico reparado (`dockerfile_content` corregido con `FROM` con tag explícito sin `:latest`, lista `tools_declared` depurada con los nombres reales de binarios/paquetes Debian/Alpine, `entrypoint` corregido si falló la ejecución del Juez, `base_ram_mb` ajustada entre `256` y `4096` si hubo fallo de memoria, y `rationale` en español explicando la causa raíz y qué corrigió).
    - Aplicar guardrails deterministas: prohibir `:latest` en `FROM` o `docker_image`, asegurar que si `dockerfile_content` tiene capas `RUN` adicionales entonces `docker_image` mantenga o reciba un tag local `solv-local/stack:...` para forzar build local, y validar `rationale` no vacío.
  - En `backend/internal/delivery/http/ai_copilot_handler.go` y `backend/internal/delivery/http/router.go`:
    - Exponer `POST /api/v1/admin/ai/repair-template` (rol `admin`) conectado a `AICopilotHandler.RepairTemplate`.
- **S4**: **Tests Unitarios en Go y Documentación en `docs/API_CONTRACTS.md`**:
  - En `backend/internal/core/services/ai_settings_service_test.go` y `ai_copilot_service_test.go`:
    1. Probar que `latency_ms` mide el tiempo real tras `io.ReadAll`.
    2. Probar que `context.DeadlineExceeded` produce `domain.ErrAITimeout` y HTTP `504` (`ai_timeout`).
    3. Probar la gobernanza autoadaptativa por perfil de motor (`fast_cloud` vs `queued_or_reasoning` `:free` vs `local_gpu`) y acción (`suggest-component`, `suggest-template`, `repair-template`, `generate-exercise`), ignorando techos viejos de `30 s` pero respetando overrides mayores de Modo Experto.
    4. Probar la cascada BYOK por docente (`UpdateUserOverride`, `CompleteJSON`, `GetEffectiveStatus`).
    5. Probar `RepairTemplate` (`POST /api/v1/admin/ai/repair-template`) reparando un `Dockerfile` y `tools_declared` tras un fallo de verificación de contenedor.
  - Actualizar `docs/API_CONTRACTS.md` con `POST /api/v1/admin/ai/repair-template`, `/api/v1/ai/my-settings*`, `GET /api/v1/ai/status` y el error `504` (`ai_timeout`).
- **S5**: **Gobernanza Autoadaptativa con Modo Experto Colapsado en `admin-config-ia` (Frontend)**:
  - En `frontend/src/app/features/admin/configuracion/admin-config-ia.service.ts`:
    - Añadir métodos `repairTemplate(dto)`, `getMySettings()`, `updateMySettings(dto)`, `testMyConnection(dto)` y `listMyModels(dto)`.
    - Mejorar `extractErrorMessage` para traducir `504` / `ai_timeout` y mostrar el detalle técnico real cuando `message` es genérico.
  - En `frontend/src/app/features/admin/configuracion/admin-config-ia.component.ts`, `.html` y `.scss`:
    - Eliminar de la vista principal los dos inputs numéricos manuales y las 6 tarjetas gigantes de `Cuota Horaria` y `Timeout de Inferencia` que obligaban al administrador a adivinar segundos y concurrencia.
    - Reemplazarlos por un panel limpio de **"Gobernanza Autoadaptativa del Motor"** que muestre dinámicamente según el proveedor/modelo seleccionado arriba:
      - **Perfil detectado**: *Motor Ágil en Nube*, *Modelo Gratuito / Razonamiento (Cola extendida)* o *Servidor GPU Local Universitario*.
      - **Presupuesto de tiempo automático por tarea**: `60s–90s` en Componentes · `90s–120s` en Stacks Completos y Autorreparación · `120s–180s` en Ejercicios con Juez.
      - **Protección de cuota**: automática por docente (más soporte de **API Key Personal BYOK** para que cada docente pueda usar su propia cuenta sin consumir el cupo institucional).
    - Incluir un botón discreto **`Ajuste manual de límites (Modo Experto)`** (colapsado por defecto) que, solo si el personal de infraestructura lo abre, permite sobreescribir el piso de `timeout_seconds` (`5–240 s`) y `hourly_quota_per_user` (`1–200`).
- **S6**: **Rediseño UX del Copiloto en Constructor (`template-studio`) + Autorreparación en Modo Visual y Modo Avanzado**:
  - En `frontend/src/app/features/admin/templates/studio/template-studio.component.html`, `.ts`, `.scss` y `components/studio-summary-sidebar/studio-summary-sidebar.component.*`:
    - Elevar el Copiloto IA a su **propia sección/tarjeta colapsable arriba del Constructor**, disponible **tanto cuando el usuario está en `Modo Visual` como cuando está en `Modo Avanzado (Dockerfile / Scripts)`** (quitar la restricción `studioMode() === 'visual'`).
    - Usar un `<textarea class="form-textarea" rows="2">` limpio (sin la fila de píldoras de "Ejemplos rápidos", que fueron descartadas por innecesarias), cronómetro en vivo (`Transcurrido: X.X s` con `machineData`) y botón **`Cancelar`** (`Subscription.unsubscribe()`).
    - **Tarjeta de Resumen del Stack Configurado y Sincronización con Modo Avanzado**: cuando el Copiloto arma el stack, actualiza tanto las 3 columnas visuales como el `Dockerfile`, `setup.sh`, `toolsDeclared` y `entrypoint` del Modo Avanzado, y dentro de `.ai-rationale-box` muestra las insignias visuales de cada tecnología seleccionada (`<tech-logo [iconType]="comp.icon_key" [size]="16" />` + `comp.name`) junto con el chip de memoria (`{{ baseRamMB() }} MB RAM` con `machineData`).
    - **Flujo de Autorreparación (`Corregir con Copiloto IA`) cuando falla `Comprobar funcionamiento`**:
      - En `studio-summary-sidebar`, cuando `job.status === 'failed'` y `aiEnabled()` es `true`, mostrar dentro de la alerta de fallo el botón **`Corregir con Copiloto IA`** (con estado de carga, cronómetro en vivo y opción de cancelar).
      - Al pulsarlo, `template-studio` invoca `aiService.repairTemplate(...)` enviando el estado actual real (`name`, `description`, `target_environment`, `docker_image`, `dockerfile_content`, `setup_script`, `tools_declared`, `entrypoint`, `base_ram_mb`, `job.error_message` y los nombres de `job.result?.tools` con `present === false`).
      - Al recibir el parche de la IA:
        1. Aplica las correcciones directamente sobre los campos técnicos y del **Modo Avanzado** (`dockerfileContent.set(res.dockerfile_content)`, `hasCustomDockerfile.set(true)`, `dockerImage.set(res.docker_image)`, `needsLocalBuild.set(res.docker_image.startsWith('solv-local/stack:'))`, `setupScript.set(res.setup_script)`, `toolsDeclared.set(res.tools_declared.join(', '))`, `entrypoint.set(res.entrypoint)`, `baseRamMB.set(res.base_ram_mb)`).
        2. Conmuta automáticamente `studioMode.set('code')` (**Modo Avanzado: Dockerfile / Scripts**) y muestra un banner con el diagnóstico y reparación realizada por la IA (`res.rationale` + `res.latency_ms ms`), para que el administrador vea exactamente qué corrigió en el `Dockerfile`, en los programas verificados o en el comando de ejecución antes de volver a pulsar **`Comprobar funcionamiento`**.
- **S7**: **Rediseño UX en Modal de Componente (`component-form-modal`) y `modal-shell`: Sin Ejemplos Rápidos, Tarjeta de Previsualización Inteligente y Cero Scroll Prematuro**:
  - En `frontend/src/app/shared/components/modal-shell/modal-shell.component.scss`:
    - Garantizar que `.modal-dialog` tenga `max-height: calc(100vh - 2 * var(--space-4))` y `.modal-body` tenga `overflow-y: auto; overscroll-behavior: contain; flex: 1 1 auto; min-height: 0;` para que la cabecera y el pie queden fijos y cualquier scroll ocurra limpiamente en el cuerpo.
  - En `frontend/src/app/features/admin/templates/components/model-library/components/component-form-modal/component-form-modal.component.html`, `.ts` y `.scss`:
    - **Eliminar por completo los "Ejemplos rápidos" (`.ai-examples-row` / `.pill-example`)** de la pestaña `Diseñar con Copiloto IA`.
    - **Eliminar los inputs redundantes y el texto largo de ayuda que forzaban scroll desde la 3ra línea**:
      - Si `!isContextLocked() && !editingComponent`, reemplazar el `combobox` pesado de `Clasificación del Componente del Catálogo *` y su hint de 3 líneas por una barra segmentada compacta de 1 línea (`#comp-layer.catalog-kind-segmented`) con 3 botones: `Lenguaje Principal`, `Interfaz Web`, `Complemento / Librería`.
      - En modo creación (`!editingComponent`), **ocultar por defecto los campos manuales inferiores** (`Nombre Visible *`, `Descripción breve`, `smart-icon-picker`, instalador, paquetes y `Modo experto (contenedor)`) mientras el usuario está eligiendo en el `Catálogo Oficial (1 clic)` o escribiendo su prompt en `Diseñar con Copiloto IA`. El formulario manual completo (`.manual-form-panel`) solo se despliega automáticamente si el usuario pulsa **`+ Personalizado`** (`isCustomPresetSelected() && !hasPreviewReady()`), si está editando un componente existente (`editingComponent`), o si pulsa el botón **`Personalizar campos`** (`showManualFineTune()`).
    - **Tarjeta de Previsualización de la Tecnología (`.component-preview-card`)**:
      - Cuando el Copiloto IA devuelve la sugerencia o el usuario elige una tarjeta del Catálogo Oficial (`hasPreviewReady(): !editingComponent && name().trim().length >= 2`), mostrar una **Tarjeta de Previsualización** limpia y estructurada que presenta todos los campos resueltos de un vistazo sin inputs repetidos:
        - Cabecera con `<tech-logo [iconType]="effectiveIconKey()" [size]="38" />`, nombre (`name()`), badge de rol (`Lenguaje Principal` | `Plataforma Web / Frontend` | `Complemento / Librería`), badge `En catálogo (Actualizará)` si coincide con uno existente, descripción (`description()`), y botón **`Personalizar campos`** (`toggleManualFineTune()`) para abrir/cerrar la edición manual fina si desea retocar algún valor.
        - Grilla técnica compacta (`.preview-specs-grid`) mostrando con `font-mono` / `machineData`: **Imagen Base Oficial / Instalación** (`baseImage()` o `${installPreset()}: ${packageNamesInput()}`), **Herramientas Verificadas** (chips con `previewToolsList()`), **Evaluador Juez Virtual** (`judgeEntrypoint()` cuando aplica) y **Presupuesto de Memoria** (`previewMemoryBadge()`).
      - Además, en `ai-exercise-generator-modal.component.html` y `.scss`, transformar la previsualización del Paso B (`generatedResult()`) en una tarjeta estructurada `.exercise-preview-card` con título, badges de lenguaje, dificultad, cantidad de casos, latencia (`ms`), extracto del enunciado y el banner del sello del Juez Virtual.
- **S8**: **Panel BYOK Docente ("Mi API Key Personal") y Telemetría en Vivo en el Modal de Ejercicios (`ai-exercise-generator-modal`)**:
  - En `frontend/src/app/features/teacher/services/teacher-ai-copilot.service.ts`, `exercise-editor.component.*` y `ai-exercise-generator-modal.component.*`:
    - Permitir abrir el modal del Copiloto IA siempre (incluso si la IA institucional está deshabilitada, para que el docente pueda activar su propia API Key).
    - Mostrar arriba una barra de estado indicando si opera con el **Motor Institucional** o con **Mi API Key Personal**, con botón **`Configurar mi API Key`** que despliega el panel BYOK colapsable.
    - En el panel BYOK permitir: activar/desactivar `"Usar mi propia API Key"`, elegir proveedor (`gemini`, `openai`, `openrouter`, `deepseek`), pegar API Key, probar conexión (`POST /api/v1/ai/my-settings/test-connection`), buscar y filtrar modelos con **buscador de texto + filtro `Todos | Gratis` + modo manual** (`POST /api/v1/ai/my-settings/models`) y guardar (`PUT /api/v1/ai/my-settings`).
    - Durante `isGenerating()`, mostrar cronómetro en vivo (`Generando y sellando casos de prueba... X.X s` con `<code class="font-mono">`) y botón **`Cancelar`**.
- **S9**: **Cumplimiento estricto de gates de diseño, tipografía y tests de Frontend**:
  - Cero colores hexadecimales literales (`#...`) ni `px`/`rem` en `font-size` o `font-family` dentro de archivos `.scss` (`npm run lint:styles` en verde).
  - Todo dato técnico visible (latencia, segundos del cronómetro, IDs de modelos/proveedores, comandos/herramientas) marcado con `machineData` o `<code>`/`font-mono`.
  - Actualizar/ampliar los tests `.spec.ts` de `admin-config-ia`, `template-studio`, `component-form-modal` y `ai-exercise-generator-modal` (`npm run test:ci` y `CI=true npm run build` en verde).
- **S10**: **Blindaje de Transporte para Modelos Gratuitos/Razonamiento en OpenRouter y Ollama (`TestConnection` tolerante a `<think>` + Fallback automático `response_format` en HTTP 400)**:
  - En `backend/internal/core/services/ai_settings_service.go`:
    1. **Fallback automático de `response_format` en `openAICompatibleProvider.Complete`**: muchos modelos `:free` en OpenRouter rechazan `"response_format": {"type": "json_object"}` con HTTP `400 Bad Request`. Si `httpResp.StatusCode == http.StatusBadRequest` y el payload tenía `ResponseFormat.Type != ""`, reintentar automáticamente una única vez enviando el mismo payload con `ResponseFormat` vacío (omisivo en JSON vía `omitempty` o puntero `*aiResponseFormat`), sumando la latencia total de ambos intentos.
    2. **`TestConnection` y `TestUserConnection` tolerantes a modelos de razonamiento**:
       - Extraer el bloque JSON con `extractJSONObject(content)` antes de validar `json.Valid([]byte(...))`, de modo que si un modelo de razonamiento (`deepseek-r1`, `nemotron`, `qwq`) responde `<think>...</think>\n{"ok":true}` o envuelve el JSON en bloques Markdown ````json ... ````, la prueba de conexión sea exitosa (`OK: true`).
       - Usar `effectiveAITimeout(stored, classifyAIEngine(provider, model), "test-connection")` (con `30 s` en `fast_cloud` y `60 s` en `queued_or_reasoning` / `local_gpu` en `AIEngineAutoTimeoutSeconds`) en lugar de `aiSettingsTimeout` plano.
    3. **Tests unitarios en `ai_settings_service_test.go`**:
       - Probar que si el servidor upstream devuelve `400 Bad Request` al recibir `response_format: json_object` y devuelve `200 OK` en el reintento sin `response_format`, `CompleteJSON` tiene éxito y acumula la latencia real.
       - Probar que `TestConnection` da `OK: true` cuando el modelo responde `<think>verificando...</think> ```json {"ok":true} ``` `.

## Tasks
- [x] **T1** (S1, S2, S3) [delegated: Freebuff]: Implementar en Backend Go la gobernanza autoadaptativa por perfil de motor y operación, medición real de latencia post-`io.ReadAll`, tipado `ErrAITimeout` -> HTTP `504` (`ai_timeout`), cascada BYOK (`UserOverrides` + `/api/v1/ai/my-settings*`) y el endpoint de autorreparación `POST /api/v1/admin/ai/repair-template` — verificado con `go vet ./...` y `go test ./internal/...`.
- [x] **T2** (S4) [delegated: Freebuff]: Añadir tests unitarios en `ai_settings_service_test.go`, `ai_copilot_service_test.go` y `ai_copilot_handler_test.go` cubriendo latencia real, timeout `504`, gobernanza autoadaptativa, cascada BYOK docente y `RepairTemplate` + actualizar `docs/API_CONTRACTS.md` (`cd backend && go vet ./... && go test ./internal/...` OK).
- [x] **T3** (S5, S6, S7) [delegated: Antigravity]: Implementar el panel de Gobernanza Autoadaptativa con Modo Experto colapsado en `admin-config-ia`, la sección dedicada del Copiloto + flujo de autorreparación (`Corregir con Copiloto IA` sincronizado con `Modo Avanzado: Dockerfile / Scripts`) en `template-studio` y `studio-summary-sidebar`, y las pestañas (`Catálogo Oficial` | `Diseñar con Copiloto IA`) integradas con el `Modo experto (contenedor)` en `component-form-modal`.
- [x] **T4** (S8, S9) [delegated: Antigravity]: Incorporar el panel colapsable de **API Key Personal (BYOK)** con buscador/filtro `Gratis` y cronómetro/cancelación en `ai-exercise-generator-modal` para el docente, asegurando tests unitarios y gates (`npm run lint:styles && npm run test:ci && CI=true npm run build`).
- [x] **T5** (S10) [delegated: Freebuff]: Implementar fallback automático sin `response_format` ante HTTP 400 en `openAICompatibleProvider.Complete`, tolerancia a `<think>`/fences y timeout adaptativo en `TestConnection`/`TestUserConnection`, más sus tests unitarios en `ai_settings_service_test.go`.

## Log
- **L1 (2026-10-25)**: Solicitud del usuario tras diagnosticar en `scripts/.run/logs/backend.log` dos timeouts clavados en 30003/30005 ms y uno en 27348 ms (con latencia reportada falsa de 705 ms por medir antes de `io.ReadAll`, y error 500 genérico ocultando `context deadline exceeded`).
- **L2 (2026-10-25)**: Tras revisión arquitectónica con el usuario, se decidió: (1) reemplazar los selectores manuales de segundos/cuota por **Gobernanza Autoadaptativa (Zero-Config)** calculada por el sistema según perfil del modelo (`fast_cloud`, `queued_or_reasoning` `:free`, `local_gpu`) y complejidad de la operación, dejando el ajuste manual oculto en un *Modo Experto* colapsado; (2) incorporar **Autorreparación Diagnóstica (`POST /api/v1/admin/ai/repair-template` + botón `Corregir con Copiloto IA`)** cuando falla `Comprobar funcionamiento`, que repara directamente el `Dockerfile`, `setup.sh`, `tools_declared`, `entrypoint` y `base_ram_mb` del **Modo Avanzado / Experto** usando una radiografía fresca del fallo sin arrastrar historial de chat; y (3) conectar las sugerencias de `component-form-modal` con su `Modo experto (contenedor)`.
- **L3 (2026-10-25)**: Corrección de UX solicitada por el usuario tras revisar capturas de `component-form-modal` y `template-studio`: "si esta dividio en otra seccion la parte de ia, pero, esos ejemplos rapidos son super inncesarios siento yo, para lo elimimnamos y ya, y en ves de eso y para que sirve los otros inputs de Clasificación del Componente del Catálogo * ... Nombre Visible * Descripción breve para docentes y administradores si tambien lo rellenara verdad, no seria mejor, como tener una previsualizcion de los campso que lleno la ia, algo bonito y tal, como una carta, o algoa si los mismo pasa con el otro, en encima a la 3ra linea ya tengo que ahccer scroll, una locura, mejoremos el diseño, quiero un diseño super hermoso". Se actualizaron S6 y S7 para eliminar los ejemplos rápidos, ocultar los campos manuales redundantes por defecto, agregar la Tarjeta de Previsualización (`.component-preview-card` / `.exercise-preview-card` / chips en `.ai-rationale-box`) con botón `Personalizar campos`, y fijar `max-height` + scroll limpio en `.modal-body` de `modal-shell`.
