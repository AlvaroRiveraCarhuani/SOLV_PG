package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"solv-backend/internal/core/domain"
)

// AICopilotService implementa los casos de uso generativos del Copiloto
// Institucional sobre AISettingsService.CompleteJSON. Aplica cero confianza
// ciega en la salida del LLM: sanea DTOs, prohíbe imágenes sin tag fijo,
// sintetiza capas de build y sella las salidas esperadas con el runner del juez.
type AICopilotService struct {
	aiSettings *AISettingsService
	runner     domain.EvaluationRunner
	compRepo   domain.TemplateComponentRepository
}

// NewAICopilotService acepta un runner opcional (nil desactiva el sello
// determinista de los casos de prueba) y un repositorio de componentes opcional
// para grounding del catálogo activo.
func NewAICopilotService(aiSettings *AISettingsService, runner domain.EvaluationRunner, compRepo ...domain.TemplateComponentRepository) *AICopilotService {
	var repo domain.TemplateComponentRepository
	if len(compRepo) > 0 {
		repo = compRepo[0]
	}
	return &AICopilotService{aiSettings: aiSettings, runner: runner, compRepo: repo}
}

// SuggestTemplate propone el stack completo de una plantilla a partir de la
// descripción del administrador y el entorno objetivo. El backend NO aplica
// nada: devuelve una propuesta estructurada con slugs canónicos de ecosistema
// y el rationale, y el frontend la resuelve contra el catálogo real del tenant
// antes de que el administrador decida aplicarla (flujo propuesta -> revisión
// -> aplicar).
func (s *AICopilotService) SuggestTemplate(ctx context.Context, tenantID, userID string, req domain.AISuggestTemplateRequestDTO) (*domain.AISuggestTemplateResponseDTO, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if n := utf8.RuneCountInString(prompt); n < 3 || n > 600 {
		return nil, &AISettingsValidationError{
			Code:    "prompt_invalid",
			Message: "prompt debe tener entre 3 y 600 caracteres",
		}
	}

	targetEnv := strings.ToUpper(strings.TrimSpace(req.TargetEnvironment))
	if targetEnv == "" {
		targetEnv = string(domain.TargetEnvironmentJuezEfimero)
	}
	switch targetEnv {
	case string(domain.TargetEnvironmentJuezEfimero), string(domain.TargetEnvironmentIDEPersistente), string(domain.TargetEnvironmentAmbos):
	default:
		return nil, &AISettingsValidationError{
			Code:    "target_environment_invalid",
			Message: fmt.Sprintf("target_environment inválido: %q. Valores permitidos: JUEZ_EFIMERO, IDE_PERSISTENTE, AMBOS", req.TargetEnvironment),
		}
	}

	catalogSummary := s.buildCompactCatalogSummary(ctx, tenantID)
	systemPrompt := suggestTemplateSystemPrompt(catalogSummary)
	userPrompt := fmt.Sprintf("Entorno objetivo: %s\nRequisitos del administrador:\n%s", targetEnv, prompt)

	raw, latencyMS, err := s.aiSettings.CompleteJSON(ctx, tenantID, userID, "suggest-template", systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Name              string   `json:"name"`
		Description       string   `json:"description"`
		RuntimeSlug       string   `json:"runtime_slug"`
		WebPlatformSlug   string   `json:"web_platform_slug"`
		AddonSlugs        []string `json:"addon_slugs"`
		DatabaseSlug      string   `json:"database_slug"`
		SetupScript       string   `json:"setup_script"`
		BaseRamMB         int      `json:"base_ram_mb"`
		Rationale         string   `json:"rationale"`
		MissingComponents []struct {
			Name     string `json:"name"`
			Layer    string `json:"layer"`
			Category string `json:"category"`
			Reason   string `json:"reason"`
		} `json:"missing_components"`
	}
	if err := json.Unmarshal([]byte(cleanJSONFence(raw)), &payload); err != nil {
		return nil, &AISettingsValidationError{
			Code:    "ai_response_invalid",
			Message: "el motor de IA devolvió JSON inválido para la arquitectura del entorno",
		}
	}

	rationale := strings.TrimSpace(payload.Rationale)
	if rationale == "" {
		return nil, &AISettingsValidationError{
			Code:    "ai_response_invalid",
			Message: "el motor de IA no devolvió el rationale con la justificación del stack",
		}
	}

	runtimeSlug := strings.ToLower(strings.TrimSpace(payload.RuntimeSlug))
	if runtimeSlug == "" {
		return nil, &AISettingsValidationError{
			Code:    "runtime_slug_missing",
			Message: "el motor de IA no devolvió el runtime principal (runtime_slug)",
		}
	}

	// Saneo de slugs: minúsculas, sin vacíos ni duplicados, de rango 3..64.
	webSlug := strings.ToLower(strings.TrimSpace(payload.WebPlatformSlug))
	dbSlug := strings.ToLower(strings.TrimSpace(payload.DatabaseSlug))
	addons := make([]string, 0, len(payload.AddonSlugs))
	seen := make(map[string]struct{}, len(payload.AddonSlugs))
	for _, a := range payload.AddonSlugs {
		slug := strings.ToLower(strings.TrimSpace(a))
		if slug == "" || slug == "none" || slug == "null" {
			continue
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		addons = append(addons, slug)
	}
	if dbSlug == "none" || dbSlug == "null" {
		dbSlug = ""
	}

	// Memoria dentro de la horquilla homologada de los perfiles del Studio.
	baseRamMB := payload.BaseRamMB
	if baseRamMB <= 0 {
		baseRamMB = 512
	}
	baseRamMB = clampInt(baseRamMB, 256, 4096)

	// Saneo de componentes faltantes reportados por la IA.
	missingComponents := make([]domain.AIMissingComponentDTO, 0, len(payload.MissingComponents))
	seenMissing := make(map[string]struct{}, len(payload.MissingComponents))
	for _, m := range payload.MissingComponents {
		name := strings.TrimSpace(m.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, dup := seenMissing[key]; dup {
			continue
		}
		seenMissing[key] = struct{}{}

		layer := strings.ToLower(strings.TrimSpace(m.Layer))
		if layer != "runtime" && layer != "addon" && layer != "database" {
			layer = "addon"
		}

		reason := strings.TrimSpace(m.Reason)
		if reason == "" {
			reason = fmt.Sprintf("Tecnología %s requerida pero no disponible en el catálogo", name)
		}

		missingComponents = append(missingComponents, domain.AIMissingComponentDTO{
			Name:     name,
			Layer:    layer,
			Category: strings.ToLower(strings.TrimSpace(m.Category)),
			Reason:   reason,
		})
	}

	return &domain.AISuggestTemplateResponseDTO{
		Name:              strings.TrimSpace(payload.Name),
		Description:       strings.TrimSpace(payload.Description),
		RuntimeSlug:       runtimeSlug,
		WebPlatformSlug:   webSlug,
		AddonSlugs:        addons,
		DatabaseSlug:      dbSlug,
		SetupScript:       strings.TrimSpace(payload.SetupScript),
		BaseRamMB:         baseRamMB,
		Rationale:         rationale,
		MissingComponents: missingComponents,
		LatencyMS:         latencyMS,
	}, nil
}

func (s *AICopilotService) buildCompactCatalogSummary(ctx context.Context, tenantID string) string {
	if s.compRepo == nil {
		return ""
	}
	comps, err := s.compRepo.ListTemplateComponents(ctx, tenantID, "", false)
	if err != nil || len(comps) == 0 {
		return ""
	}

	var runtimes, webs []string
	addonsByFam := make(map[string][]string)

	for _, c := range comps {
		if !c.IsActive {
			continue
		}
		if c.Layer == "runtime" {
			if c.SubCategory == "web" {
				webs = append(webs, fmt.Sprintf("%s (%s)", c.Slug, c.Name))
			} else {
				runtimes = append(runtimes, fmt.Sprintf("%s (%s)", c.Slug, c.Name))
			}
		} else if c.Layer == "addon" {
			fam := "general"
			if len(c.CompatibleRuntimes) > 0 && c.CompatibleRuntimes[0] != "" {
				fam = c.CompatibleRuntimes[0]
			}
			addonsByFam[fam] = append(addonsByFam[fam], fmt.Sprintf("%s (%s)", c.Slug, c.Name))
		}
	}

	var sb strings.Builder
	sb.WriteString("\nCATÁLOGO REAL INSTALADO EN EL TENANT (Prioridad absoluta para reutilizar):\n")
	if len(runtimes) > 0 {
		sb.WriteString(fmt.Sprintf("- Lenguajes Principales (runtime_slug): %s\n", strings.Join(runtimes, ", ")))
	}
	if len(webs) > 0 {
		sb.WriteString(fmt.Sprintf("- Plataformas Web (web_platform_slug): %s\n", strings.Join(webs, ", ")))
	}
	for fam, items := range addonsByFam {
		if len(items) > 0 {
			sb.WriteString(fmt.Sprintf("- Complementos %s (addon_slugs): %s\n", strings.Title(fam), strings.Join(items, ", ")))
		}
	}
	sb.WriteString("- Motores Base de Datos (database_slug): sqlite, postgres, mysql, mongodb, redis (o vacío si no requiere)\n")

	return sb.String()
}

func suggestTemplateSystemPrompt(catalogSummary string) string {
	base := `Sos el copiloto de arquitectura de entornos de SOLV, la plataforma de laboratorios virtuales.
Respondé EXCLUSIVAMENTE con un objeto JSON válido (sin Markdown, sin fences, sin comentarios) con esta forma exacta:
{
  "name": "Nombre corto y descriptivo del entorno",
  "description": "Una frase breve de qué cubre el entorno",
  "runtime_slug": "slug del lenguaje de programación principal",
  "web_platform_slug": "slug de la plataforma web o vacío",
  "addon_slugs": ["slugs de frameworks o librerías"],
  "database_slug": "slug del motor de base de datos o vacío",
  "setup_script": "script shell de preparación o cadena vacía",
  "base_ram_mb": 512,
  "rationale": "justificación técnica breve en español del stack elegido",
  "missing_components": [
    {
      "name": "Nombre de tecnología faltante",
      "layer": "runtime | addon | database",
      "category": "backend | web | framework | orm | toolkit | heavy",
      "reason": "breve justificación técnica de por qué no está en el catálogo pero es necesaria"
    }
  ]
}
Reglas innegociables:
- Si se suministra un CATÁLOGO REAL INSTALADO, debes elegir ÚNICAMENTE entre los slugs listados en dicho catálogo para runtime_slug, web_platform_slug y addon_slugs.
- Si una herramienta solicitada no está en el catálogo pero existe una alternativa directa instalada en la misma categoría (ej. un ORM o cliente alternativo), usa la del catálogo y explícalo en el rationale.
- Solo si el usuario solicita explícitamente una tecnología o librería que NO existe ni tiene alternativa en el catálogo provisto, repórtala en "missing_components" y NO inventes slugs dentro de addon_slugs.
- database_slug SOLO puede ser sqlite, postgres, mysql, mongodb o redis; si no hace falta base de datos o si el entorno objetivo es de juez o dual (JUEZ_EFIMERO, AMBOS), dejalo como cadena vacía.
- base_ram_mb entre 256 y 4096: 512 para entornos ligeros de juez, 1024 o más para laboratorios IDE con compilación pesada o servicios acoplados.
- web_platform_slug SOLO tiene sentido cuando el entorno es un laboratorio IDE de tipo web (Angular, React, Vue) y el entorno objetivo es IDE_PERSISTENTE; en jueces automáticos o entornos duales (JUEZ_EFIMERO, AMBOS) dejalo vacío.
- setup_script es OPCIONAL y solo de preparación del espacio de trabajo (crear carpetas, alias, variables de entorno); no incluyas comandos de instalación de paquetes (eso va en capas de la imagen).
- name y description en español, descriptivos para un catálogo académico.`

	if catalogSummary != "" {
		return base + "\n" + catalogSummary
	}
	return base
}

// SuggestComponent propone un componente de catálogo a partir de la
// descripción del administrador y devuelve el DTO normalizado con la capa de
// build sintetizada de forma determinista.
func (s *AICopilotService) SuggestComponent(ctx context.Context, tenantID, userID string, req domain.AISuggestComponentRequestDTO) (*domain.AISuggestComponentResponseDTO, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if n := utf8.RuneCountInString(prompt); n < 3 || n > 600 {
		return nil, &AISettingsValidationError{
			Code:    "prompt_invalid",
			Message: "prompt debe tener entre 3 y 600 caracteres",
		}
	}

	layer := domain.ComponentLayer(strings.ToLower(strings.TrimSpace(string(req.Layer))))
	if layer == "" {
		layer = domain.ComponentLayerAddon
	}
	switch layer {
	case domain.ComponentLayerRuntime, domain.ComponentLayerAddon, domain.ComponentLayerDatabase:
	default:
		return nil, &AISettingsValidationError{
			Code:    "layer_invalid",
			Message: fmt.Sprintf("layer inválido: %q. Valores permitidos: runtime, addon, database", req.Layer),
		}
	}

	targetRuntime := strings.TrimSpace(req.TargetRuntime)
	if targetRuntime == "" {
		targetRuntime = "el que mejor encaje con la descripción"
	}

	systemPrompt := suggestComponentSystemPrompt()
	if layer == domain.ComponentLayerDatabase {
		systemPrompt += suggestDatabaseComponentRules()
	}
	userPrompt := fmt.Sprintf("Capa solicitada: %s\nRuntime objetivo: %s\nDescripción del administrador:\n%s", layer, targetRuntime, prompt)

	raw, latencyMS, err := s.aiSettings.CompleteJSON(ctx, tenantID, userID, "suggest-component", systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Component json.RawMessage `json:"component"`
		Rationale string          `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(cleanJSONFence(raw)), &payload); err != nil {
		return nil, &AISettingsValidationError{
			Code:    "ai_response_invalid",
			Message: "el motor de IA devolvió JSON inválido para la sugerencia de componente",
		}
	}
	if len(payload.Component) == 0 {
		return nil, &AISettingsValidationError{
			Code:    "ai_response_invalid",
			Message: "el motor de IA no devolvió el objeto component",
		}
	}

	var dto domain.CreateTemplateComponentDTO
	if err := json.Unmarshal(payload.Component, &dto); err != nil {
		return nil, &AISettingsValidationError{
			Code:    "ai_response_invalid",
			Message: "el motor de IA devolvió un componente con formato inválido",
		}
	}
	if strings.TrimSpace(string(dto.Layer)) == "" {
		dto.Layer = layer
	}
	// La capa solicitada manda: un componente pedido como base de datos nunca se
	// guarda como runtime o complemento, y viceversa.
	if layer == domain.ComponentLayerDatabase {
		dto.Layer = domain.ComponentLayerDatabase
	}

	// Prohibido :latest: toda imagen debe llevar tag de versión fija.
	baseImage := strings.TrimSpace(dto.BaseImage)
	if baseImage != "" {
		lower := strings.ToLower(baseImage)
		if strings.Contains(lower, ":latest") {
			return nil, &AISettingsValidationError{
				Code:    "base_image_latest",
				Message: fmt.Sprintf("el motor de IA propuso la imagen %q: el tag :latest está prohibido por reproducibilidad académica", baseImage),
			}
		}
		if !strings.Contains(baseImage, ":") {
			return nil, &AISettingsValidationError{
				Code:    "base_image_unpinned",
				Message: fmt.Sprintf("el motor de IA propuso la imagen %q sin tag de versión fija", baseImage),
			}
		}
		dto.BaseImage = baseImage
	}

	// Saneo de metadatos del catálogo y síntesis determinista de la capa de build.
	// En la capa de base de datos el saneo es cerrado: el motor, la clasificación, la
	// imagen versionada, la huella de memoria y el modo de instalación quedan
	// gobernados por el catálogo, no por la salida del motor de IA.
	dto = domain.NormalizeCreateComponentDTO(dto)
	synthesized, err := domain.SynthesizeAssistedBuildLayer(dto.InstallMode, dto.PackageNames, dto.BuildLayerSnippet)
	if err != nil {
		return nil, &AISettingsValidationError{
			Code:    "build_layer_invalid",
			Message: fmt.Sprintf("no se pudo sintetizar la capa de build: %v", err),
		}
	}
	dto.BuildLayerSnippet = synthesized

	return &domain.AISuggestComponentResponseDTO{
		Component:     dto,
		ServiceEngine: strings.TrimSpace(dto.ServiceEngine),
		Rationale:     strings.TrimSpace(payload.Rationale),
		LatencyMS:     latencyMS,
	}, nil
}

// RepairTemplate repara el manifiesto técnico de una plantilla a partir de la
// radiografía fresca del fallo del contenedor (sin historial de chat): corrige
// el Dockerfile, el setup_script, tools_declared, el entrypoint y la memoria
// base, y devuelve la causa raíz explicada en español. Aplica guardrails
// deterministas sobre la salida del motor.
func (s *AICopilotService) RepairTemplate(ctx context.Context, tenantID, userID string, req domain.AIRepairTemplateRequestDTO) (*domain.AIRepairTemplateResponseDTO, error) {
	systemPrompt := repairTemplateSystemPrompt()
	userPrompt := repairTemplateUserPrompt(req)

	raw, latencyMS, err := s.aiSettings.CompleteJSON(ctx, tenantID, userID, "repair-template", systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var payload struct {
		DockerImage       string   `json:"docker_image"`
		DockerfileContent string   `json:"dockerfile_content"`
		SetupScript       string   `json:"setup_script"`
		ToolsDeclared     []string `json:"tools_declared"`
		Entrypoint        string   `json:"entrypoint"`
		BaseRamMB         int      `json:"base_ram_mb"`
		Rationale         string   `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(cleanJSONFence(raw)), &payload); err != nil {
		return nil, &AISettingsValidationError{
			Code:    "ai_response_invalid",
			Message: "el motor de IA devolvió JSON inválido para la reparación de la plantilla",
		}
	}

	rationale := strings.TrimSpace(payload.Rationale)
	if rationale == "" {
		return nil, &AISettingsValidationError{
			Code:    "ai_response_invalid",
			Message: "el motor de IA no devolvió el rationale con la causa raíz del fallo",
		}
	}

	// Cada campo cae en su valor actual si el motor lo omitió: la reparación
	// nunca borra estado que ya funcionaba.
	dockerImage := strings.TrimSpace(payload.DockerImage)
	if dockerImage == "" {
		dockerImage = strings.TrimSpace(req.DockerImage)
	}
	dockerfile := strings.TrimSpace(payload.DockerfileContent)
	if dockerfile == "" {
		dockerfile = strings.TrimSpace(req.DockerfileContent)
	}
	setupScript := payload.SetupScript
	if strings.TrimSpace(setupScript) == "" {
		setupScript = req.SetupScript
	}
	toolsDeclared := sanitizeToolsList(payload.ToolsDeclared, req.ToolsDeclared)
	entrypoint := strings.TrimSpace(payload.Entrypoint)
	if entrypoint == "" {
		entrypoint = strings.TrimSpace(req.Entrypoint)
	}
	baseRamMB := payload.BaseRamMB
	if baseRamMB == 0 {
		baseRamMB = req.BaseRamMB
	}
	if baseRamMB == 0 {
		baseRamMB = 1024
	}
	baseRamMB = clampInt(baseRamMB, 256, 4096)

	// Guardrail 1: prohibido :latest en la imagen o en cualquier FROM.
	if strings.Contains(strings.ToLower(dockerImage), ":latest") {
		return nil, &AISettingsValidationError{
			Code:    "base_image_latest",
			Message: fmt.Sprintf("el motor de IA propuso la imagen %q: el tag :latest está prohibido por reproducibilidad académica", dockerImage),
		}
	}
	if line, ok := dockerfileLatestTagLine(dockerfile); ok {
		return nil, &AISettingsValidationError{
			Code:    "base_image_latest",
			Message: fmt.Sprintf("el motor de IA devolvió un Dockerfile con :latest (%s): el tag :latest está prohibido por reproducibilidad académica", line),
		}
	}

	// Guardrail 2: un Dockerfile con capas RUN requiere build local, por lo que
	// la imagen debe arrastrar el prefijo solv-local/stack:...
	if dockerfileNeedsLocalBuild(dockerfile) && !strings.HasPrefix(dockerImage, localStackImagePrefix) {
		dockerImage = localStackImagePrefix + repairLocalImageSlug(req.Name)
	}

	return &domain.AIRepairTemplateResponseDTO{
		DockerImage:       dockerImage,
		DockerfileContent: dockerfile,
		SetupScript:       setupScript,
		ToolsDeclared:     toolsDeclared,
		Entrypoint:        entrypoint,
		BaseRamMB:         baseRamMB,
		Rationale:         rationale,
		LatencyMS:         latencyMS,
	}, nil
}

// localStackImagePrefix fuerza el build local de la imagen cuando el
// Dockerfile define capas propias.
const localStackImagePrefix = "solv-local/stack:"

// sanitizeToolsList limpia la lista de herramientas verificadas: recorta,
// descarta vacíos y duplicados preservando el orden. Si el motor no devolvió
// ninguna, conserva las declaradas originalmente.
func sanitizeToolsList(tools, fallback []string) []string {
	cleaned := make([]string, 0, len(tools))
	seen := make(map[string]bool, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		cleaned = append(cleaned, name)
	}
	if len(cleaned) > 0 {
		return cleaned
	}
	for _, tool := range fallback {
		name := strings.TrimSpace(tool)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		cleaned = append(cleaned, name)
	}
	return cleaned
}

// dockerfileNeedsLocalBuild reporta si el Dockerfile define capas RUN propias
// (y por lo tanto no puede resolverse con una imagen preconstruida).
func dockerfileNeedsLocalBuild(dockerfile string) bool {
	for _, raw := range strings.Split(dockerfile, "\n") {
		line := strings.TrimSpace(raw)
		if line == "RUN" || strings.HasPrefix(line, "RUN ") || strings.HasPrefix(line, "RUN\t") {
			return true
		}
	}
	return false
}

// dockerfileLatestTagLine devuelve la instrucción FROM con :latest si existe.
func dockerfileLatestTagLine(dockerfile string) (string, bool) {
	for _, raw := range strings.Split(dockerfile, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "FROM ") && line != "FROM" {
			continue
		}
		if strings.Contains(strings.ToLower(line), ":latest") {
			return line, true
		}
	}
	return "", false
}

// repairSlugReplacer elimina tildes y eñes para que el tag local quede en
// ASCII puro, como exige el formato de tag de Docker.
var repairSlugReplacer = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n", "ç", "c",
)

// repairLocalImageSlug deriva el tag local solv-local/stack:<slug> desde el
// nombre de la plantilla.
func repairLocalImageSlug(name string) string {
	var b strings.Builder
	pendingDash := false
	normalized := repairSlugReplacer.Replace(strings.ToLower(strings.TrimSpace(name)))
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingDash = false
			b.WriteRune(r)
			continue
		}
		pendingDash = true
	}
	slug := b.String()
	if len(slug) > 60 {
		slug = strings.TrimRight(slug[:60], "-")
	}
	if slug == "" || slug == "latest" {
		return "plantilla-reparada"
	}
	return slug
}

// GenerateExerciseDraft genera el borrador completo de un ejercicio y sella
// las salidas esperadas ejecutando la solución de referencia en el runner.
func (s *AICopilotService) GenerateExerciseDraft(ctx context.Context, tenantID, userID string, req domain.AIGenerateExerciseRequestDTO) (*domain.AIGenerateExerciseResponseDTO, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if n := utf8.RuneCountInString(prompt); n < 5 || n > 1500 {
		return nil, &AISettingsValidationError{
			Code:    "prompt_invalid",
			Message: "prompt debe tener entre 5 y 1500 caracteres",
		}
	}
	language := strings.TrimSpace(req.Language)
	if language == "" {
		return nil, &AISettingsValidationError{
			Code:    "language_invalid",
			Message: "language es obligatorio (ej: python, cpp, java, javascript, go, rust, c, csharp)",
		}
	}

	difficulty := strings.ToLower(strings.TrimSpace(req.Difficulty))
	switch domain.ExerciseDifficulty(difficulty) {
	case domain.ExerciseDifficultyEasy, domain.ExerciseDifficultyMedium, domain.ExerciseDifficultyHard:
	default:
		difficulty = string(domain.ExerciseDifficultyMedium)
	}

	publicCount := req.PublicCasesCount
	if publicCount == 0 {
		publicCount = 2
	}
	publicCount = clampInt(publicCount, 1, 5)

	hiddenCount := req.HiddenCasesCount
	if hiddenCount == 0 {
		hiddenCount = 3
	}
	hiddenCount = clampInt(hiddenCount, 1, 10)

	systemPrompt := generateExerciseSystemPrompt(publicCount, hiddenCount, difficulty)
	userPrompt := fmt.Sprintf("Lenguaje: %s\nDificultad: %s\nConsigna del docente:\n%s", language, difficulty, prompt)

	raw, latencyMS, err := s.aiSettings.CompleteJSON(ctx, tenantID, userID, "generate-exercise", systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var draft domain.AIGenerateExerciseResponseDTO
	if err := json.Unmarshal([]byte(cleanJSONFence(raw)), &draft); err != nil {
		return nil, &AISettingsValidationError{
			Code:    "ai_response_invalid",
			Message: "el motor de IA devolvió JSON inválido para el borrador de ejercicio",
		}
	}
	if len(draft.TestCases) == 0 {
		return nil, &AISettingsValidationError{
			Code:    "ai_response_invalid",
			Message: "el motor de IA no devolvió casos de prueba",
		}
	}

	// Normalización de casos: orden secuencial, visibilidad válida, weight >= 1.0
	// y primer caso como example si no existe ninguno.
	hasExample := false
	for i := range draft.TestCases {
		tc := &draft.TestCases[i]
		tc.OrderIndex = i + 1
		switch tc.Visibility {
		case domain.TestCaseVisibilityExample, domain.TestCaseVisibilityPublic, domain.TestCaseVisibilityHidden:
		default:
			tc.Visibility = domain.TestCaseVisibilityPublic
		}
		if tc.Visibility == domain.TestCaseVisibilityExample {
			hasExample = true
		}
		if tc.Weight < 1.0 {
			tc.Weight = 1.0
		}
	}
	if !hasExample {
		draft.TestCases[0].Visibility = domain.TestCaseVisibilityExample
	}

	draft.Difficulty = difficulty
	draft.Language = CanonicalLanguage(language)
	draft.LatencyMS = latencyMS

	// Sello determinista del Juez: la salida esperada es la del runner, nunca
	// el texto alucinado por el LLM.
	s.sealTestCasesWithRunner(ctx, &draft, req, language)

	return &draft, nil
}

// sealTestCasesWithRunner ejecuta la solución de referencia por cada caso y
// sobrescribe ExpectedOutput con la salida real. Si algún caso falla en el
// runner conserva el borrador y documenta el motivo en VerificationWarning.
func (s *AICopilotService) sealTestCasesWithRunner(
	ctx context.Context,
	draft *domain.AIGenerateExerciseResponseDTO,
	req domain.AIGenerateExerciseRequestDTO,
	language string,
) {
	verifyEnabled := req.VerifyWithRunner == nil || *req.VerifyWithRunner
	if s.runner == nil || !verifyEnabled {
		return
	}
	if strings.TrimSpace(draft.ReferenceSolution) == "" {
		draft.RunnerVerifiedAll = false
		draft.VerificationWarning = "el motor de IA no devolvió reference_solution: los casos quedaron sin verificar en el runner"
		return
	}

	var warnings []string
	allVerified := true
	for i := range draft.TestCases {
		tc := &draft.TestCases[i]
		runRes, err := s.runner.RunTestCase(ctx, domain.EvaluationRunConfig{
			Language:      CanonicalLanguage(language),
			SourceCode:    draft.ReferenceSolution,
			MemoryLimitMB: 256,
			TimeLimitMS:   5000,
			TestCase: domain.TestCase{
				OrderIndex:     tc.OrderIndex,
				Input:          tc.Input,
				ExpectedOutput: tc.ExpectedOutput,
				Visibility:     tc.Visibility,
				Weight:         tc.Weight,
			},
		})
		if err != nil {
			allVerified = false
			warnings = append(warnings, fmt.Sprintf("caso %d: error de ejecución de la solución de referencia: %v", tc.OrderIndex, err))
			continue
		}
		switch runRes.Verdict {
		case domain.VerdictCE, domain.VerdictRE, domain.VerdictTLE, domain.VerdictMLE:
			allVerified = false
			warnings = append(warnings, fmt.Sprintf("caso %d: la solución de referencia terminó con veredicto %s", tc.OrderIndex, runRes.Verdict))
			continue
		}
		if strings.TrimSpace(runRes.ActualOutput) == "" {
			allVerified = false
			warnings = append(warnings, fmt.Sprintf("caso %d: el runner no produjo salida (veredicto %s)", tc.OrderIndex, runRes.Verdict))
			continue
		}
		tc.ExpectedOutput = strings.TrimRight(runRes.ActualOutput, "\r\n")
		tc.VerifiedByRunner = true
	}

	draft.RunnerVerifiedAll = allVerified
	if !allVerified {
		draft.VerificationWarning = strings.Join(warnings, "; ")
	}
}

// cleanJSONFence recorta los bloques Markdown (```json ... ```) que muchos
// modelos devuelven aun con response_format: json_object.
func cleanJSONFence(raw string) string {
	s := strings.TrimSpace(raw)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if nl := strings.IndexByte(s, '\n'); nl >= 0 {
		s = s[nl+1:]
	} else {
		s = ""
	}
	if end := strings.LastIndex(s, "```"); end >= 0 {
		s = s[:end]
	}
	return strings.TrimSpace(s)
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func suggestComponentSystemPrompt() string {
	return `Sos el copiloto de componentes tecnológicos de SOLV, la plataforma de laboratorios virtuales.
Respondé EXCLUSIVAMENTE con un objeto JSON válido (sin Markdown, sin fences, sin comentarios) con esta forma exacta:
{
  "component": {
    "name": "",
    "slug": "",
    "layer": "runtime|addon|database",
    "sub_category": "",
    "description": "",
    "icon_key": "",
    "base_image": "",
    "compatible_runtimes": [],
    "install_mode": "none|pip|npm|apt|cargo|go|composer|dotnet|custom",
    "package_names": [],
    "build_layer_snippet": "",
    "setup_snippet": "",
    "judge_entrypoint": "",
    "tools_provided": [],
    "ram_delta_mb": 0
  },
  "rationale": "explicación técnica breve en español"
}
Reglas innegociables:
- PROHIBIDO el tag :latest. Toda base_image debe llevar un tag de versión fija sobre Debian Bookworm/Bullseye slim (ej: python:3.12-slim-bookworm, node:20-bookworm-slim, gcc:13.2-bookworm). En complementos (addon) la base_image puede ir vacía.
- install_mode debe corresponder al ecosistema y package_names debe listar paquetes reales sin metacaracteres de shell.
- build_layer_snippet puede ir vacío: el backend lo sintetiza de forma determinista desde install_mode + package_names.
- Los paquetes ocupan disco, no memoria residente: ram_delta_mb solo refleja procesos residentes (0 salvo servidores o librerías pesadas).`
}

// suggestDatabaseComponentRules amplía el prompt del copiloto con las reglas del
// catálogo de motores de base de datos: el motor se aprovisiona como servicio, no se
// instala dentro de la imagen del laboratorio.
func suggestDatabaseComponentRules() string {
	return `
Reglas adicionales para un componente de capa database:
- service_engine SOLO puede ser sqlite, postgres, mysql, mariadb, mongodb o redis.
- sub_category es "relational" (sqlite, postgres, mysql, mariadb) o "nosql" (mongodb, redis).
- base_image lleva tag de versión fija para los motores de contenedor (mysql:8.4, mariadb:11.4, mongo:7.0, redis:7.2-alpine) y va vacía en sqlite y en el PostgreSQL de aula.
- install_mode siempre "none" y package_names vacío: el motor se aprovisiona como servicio acoplado, no se instala en la imagen.
- compatible_runtimes debe ser ["universal"] y ram_delta_mb la huella residente del motor (0 sqlite, 50 postgres, 64 redis, 128 mysql/mariadb/mongodb).`
}

func generateExerciseSystemPrompt(publicCount, hiddenCount int, difficulty string) string {
	return fmt.Sprintf(`Sos el copiloto pedagógico de ejercicios de SOLV, la plataforma de laboratorios virtuales.
Respondé EXCLUSIVAMENTE con un objeto JSON válido (sin Markdown, sin fences, sin comentarios) con esta forma exacta:
{
  "title": "",
  "description": "",
  "difficulty": "%s",
  "boilerplate": "",
  "reference_solution": "",
  "test_cases": [
    { "input": "", "expected_output": "", "visibility": "example|public|hidden", "weight": 1.0 }
  ]
}
Reglas innegociables:
- description: enunciado en Markdown con formato de entrada, formato de salida y restricciones.
- reference_solution: programa completo y ejecutable que lee de stdin y escribe en stdout, sin interacción.
- boilerplate: esqueleto de código para el estudiante que NO resuelve el problema.
- Generá exactamente %d caso(s) público(s) (visibility "example" o "public") y %d caso(s) oculto(s) (visibility "hidden").
-		weight >= 1.0 en todos los casos.
- expected_output debe ser lo que produce reference_solution ante cada input; el backend lo re-verifica en el runner del juez antes de aceptarlo.`,
		difficulty, publicCount, hiddenCount)
}

// repairTemplateSystemPrompt describe el contrato JSON de la autorreparación
// diagnóstica de plantillas (Modo Visual y Modo Avanzado).
func repairTemplateSystemPrompt() string {
	return `Sos el ingeniero de plataforma del Copiloto de SOLV, la plataforma de laboratorios virtuales.
Recibís la radiografía de un contenedor que falló y debés devolver el manifiesto técnico YA REPARADO.
Respondé EXCLUSIVAMENTE con un objeto JSON válido (sin Markdown, sin fences, sin comentarios) con esta forma exacta:
{
  "docker_image": "",
  "dockerfile_content": "",
  "setup_script": "",
  "tools_declared": [],
  "entrypoint": "",
  "base_ram_mb": 1024,
  "rationale": ""
}
Reglas innegociables:
- PROHIBIDO el tag :latest. Toda imagen y todo FROM deben llevar un tag de versión fija (ej: python:3.12-slim-bookworm, node:20-bookworm-slim).
- Si el Dockerfile necesita capas RUN (instalación de paquetes, compilación), docker_image debe ser una imagen local con prefijo solv-local/stack: seguida de un slug en minúsculas del nombre de la plantilla, para forzar el build local.
- tools_declared debe listar SOLO binarios o paquetes reales verificables en la imagen (Debian/Alpine: nombres de binarios del PATH o paquetes apt/apk), sin metacaracteres de shell y sin herramientas que no se instalaron.
- entrypoint es el comando único que el contenedor ejecuta al arrancar, tal como lo espera el Juez; si la verificación falló por el comando de arranque, corregilo.
- setup_script debe ser idempotente y no repetir instalaciones ya cubiertas por el Dockerfile.
- base_ram_mb es un entero entre 256 y 4096; si el fallo fue por memoria (OOM), subilo lo mínimo necesario.
- rationale es obligatorio: causa raíz del error_message en español y lista concreta de lo que corregiste.`
}

// repairTemplateUserPrompt arma el snapshot diagnóstico sin historial de chat.
func repairTemplateUserPrompt(req domain.AIRepairTemplateRequestDTO) string {
	tools := make([]string, 0, len(req.ToolsDeclared))
	for _, tool := range req.ToolsDeclared {
		if t := strings.TrimSpace(tool); t != "" {
			tools = append(tools, t)
		}
	}
	failed := make([]string, 0, len(req.FailedTools))
	for _, tool := range req.FailedTools {
		if t := strings.TrimSpace(tool); t != "" {
			failed = append(failed, t)
		}
	}

	return fmt.Sprintf(`Plantilla: %s
Descripción: %s
Entorno objetivo: %s

-- Estado actual del contenedor --
docker_image: %s
base_ram_mb: %d
entrypoint: %s
herramientas declaradas: %s
herramientas con present=false: %s

-- Dockerfile actual --
%s

-- setup_script actual --
%s

-- Error reportado por el contenedor --
%s`,
		strings.TrimSpace(req.Name),
		strings.TrimSpace(req.Description),
		strings.TrimSpace(req.TargetEnvironment),
		strings.TrimSpace(req.DockerImage),
		req.BaseRamMB,
		strings.TrimSpace(req.Entrypoint),
		strings.Join(tools, ", "),
		strings.Join(failed, ", "),
		strings.TrimSpace(req.DockerfileContent),
		strings.TrimSpace(req.SetupScript),
		strings.TrimSpace(req.ErrorMessage),
	)
}
