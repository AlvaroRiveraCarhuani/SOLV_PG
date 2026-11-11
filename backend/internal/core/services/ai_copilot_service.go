package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

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

	var runtimes, webs, databases []string
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
		} else if c.Layer == "database" && c.ServiceEngine != "none" {
			// Los slugs salen del catálogo real: antes se anunciaban nombres
			// genéricos ("sqlite") que luego no resolvían contra ninguna pieza.
			databases = append(databases, fmt.Sprintf("%s (%s)", c.Slug, c.Name))
		}
	}

	// Orden estable: el mismo catálogo debe producir el mismo prompt.
	sort.Strings(runtimes)
	sort.Strings(webs)
	sort.Strings(databases)
	fams := make([]string, 0, len(addonsByFam))
	for fam := range addonsByFam {
		fams = append(fams, fam)
	}
	sort.Strings(fams)
	for _, fam := range fams {
		sort.Strings(addonsByFam[fam])
	}

	var sb strings.Builder
	sb.WriteString("\nCATÁLOGO REAL INSTALADO EN EL TENANT (Prioridad absoluta para reutilizar):\n")
	if len(runtimes) > 0 {
		sb.WriteString(fmt.Sprintf("- Lenguajes Principales (runtime_slug): %s\n", strings.Join(runtimes, ", ")))
	}
	if len(webs) > 0 {
		sb.WriteString(fmt.Sprintf("- Plataformas Web (web_platform_slug): %s\n", strings.Join(webs, ", ")))
	}
	for _, fam := range fams {
		items := addonsByFam[fam]
		if len(items) > 0 {
			sb.WriteString(fmt.Sprintf("- Complementos %s (addon_slugs): %s\n", strings.Title(fam), strings.Join(items, ", ")))
		}
	}
	if len(databases) > 0 {
		sb.WriteString(fmt.Sprintf("- Motores de Base de Datos (database_slug): %s\n", strings.Join(databases, ", ")))
	}

	return sb.String()
}

type templateLister interface {
	ListTemplates(ctx context.Context, tenantID, status, search string) ([]*domain.AdminTemplateReviewItem, error)
}

func (s *AICopilotService) buildApprovedTemplatesSummary(ctx context.Context, tenantID string) (string, []*domain.AdminTemplateReviewItem) {
	if s.compRepo == nil {
		return "", nil
	}
	lister, ok := s.compRepo.(templateLister)
	if !ok {
		return "", nil
	}
	items, err := lister.ListTemplates(ctx, tenantID, "approved", "")
	if err != nil || len(items) == 0 {
		return "", nil
	}
	var b strings.Builder
	var valid []*domain.AdminTemplateReviewItem
	for _, it := range items {
		if !strings.EqualFold(it.Status, "approved") && !strings.EqualFold(it.Status, "APROBADA") {
			continue
		}
		targetEnv := strings.ToUpper(strings.TrimSpace(it.TargetEnvironment))
		if targetEnv != "" && targetEnv != string(domain.TargetEnvironmentIDEPersistente) && targetEnv != string(domain.TargetEnvironmentAmbos) && targetEnv != "WORKSPACE" {
			continue
		}
		valid = append(valid, it)
		stackConfig, _ := json.Marshal(it.StackConfig)
		servicesConfig, _ := json.Marshal(it.ServicesConfig)
		b.WriteString(fmt.Sprintf("- ID: %s | Nombre: %s | Descripción: %s | Imagen: %s | Stack tecnológico: %s | Servicios: %s\n", it.ID, it.Name, it.Description, it.DockerImage, stackConfig, servicesConfig))
	}
	return b.String(), valid
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
- database_slug SOLO puede ser uno de los slugs exactos de "Motores de Base de Datos" del CATÁLOGO REAL, nunca un nombre genérico. Si se nombra un motor que está en el catálogo, elegilo aunque el entorno sea de juez. Dejalo vacío solo si nadie pidió base de datos ni la práctica la necesita.
- base_ram_mb entre 256 y 4096: 512 para entornos ligeros de juez, 1024 o más para laboratorios IDE con compilación pesada o servicios acoplados.
- web_platform_slug SOLO puede ser uno de los slugs exactos de "Plataformas Web" del CATÁLOGO REAL. Elegilo cuando se pide explícitamente una interfaz (React, Angular, Vue) aunque el entorno sea de juez; si no se pide, dejalo vacío.
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

	// Si la IA devolvió el prompt del usuario en la descripción o una frase coloquial en primera persona, sanearla.
	desc := strings.TrimSpace(dto.Description)
	promptLower := strings.ToLower(prompt)
	descLower := strings.ToLower(desc)
	if desc == "" || descLower == promptLower || strings.HasPrefix(descLower, "quiero ") || strings.HasPrefix(descLower, "necesito ") || strings.HasPrefix(descLower, "crear ") {
		dto.Description = fmt.Sprintf("Entorno y herramientas de %s para desarrollo y prácticas en laboratorios virtuales", dto.Name)
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

	envType := strings.ToUpper(strings.TrimSpace(req.EnvironmentType))
	if envType == string(domain.TargetEnvironmentIDEPersistente) || envType == "WORKSPACE" {
		return s.generateIdeExerciseDraft(ctx, tenantID, userID, req, prompt)
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
	draft.EnvironmentType = string(domain.TargetEnvironmentJuezEfimero)
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

func (s *AICopilotService) generateIdeExerciseDraft(
	ctx context.Context,
	tenantID, userID string,
	req domain.AIGenerateExerciseRequestDTO,
	prompt string,
) (*domain.AIGenerateExerciseResponseDTO, error) {
	difficulty := strings.ToLower(strings.TrimSpace(req.Difficulty))
	switch domain.ExerciseDifficulty(difficulty) {
	case domain.ExerciseDifficultyEasy, domain.ExerciseDifficultyMedium, domain.ExerciseDifficultyHard:
	default:
		difficulty = string(domain.ExerciseDifficultyMedium)
	}

	templatesSummary, approvedTemplates := s.buildApprovedTemplatesSummary(ctx, tenantID)

	var lockedTemplate *domain.AdminTemplateReviewItem
	reqTemplateID := strings.TrimSpace(req.TemplateID)
	if reqTemplateID != "" {
		for _, tmpl := range approvedTemplates {
			if tmpl.ID == reqTemplateID {
				lockedTemplate = tmpl
				break
			}
		}
	}

	systemPrompt := generateIdeExerciseSystemPrompt(templatesSummary)
	userPrompt := fmt.Sprintf("Dificultad: %s\nConsigna del docente:\n%s", difficulty, prompt)
	if lockedTemplate != nil {
		userPrompt = fmt.Sprintf("Dificultad: %s\nEntorno homologado obligatorio fijado por el docente: %s (ID: %s)\nConsigna del docente:\n%s\nImportante: Debes mantener exactamente este 'template_id': %q y diseñar la consigna, requerimientos y rúbrica exclusivamente para las tecnologías de este entorno.", difficulty, lockedTemplate.Name, lockedTemplate.ID, prompt, lockedTemplate.ID)
	}

	raw, latencyMS, err := s.aiSettings.CompleteJSON(ctx, tenantID, userID, "generate-ide-exercise", systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Title             string `json:"title"`
		Description       string `json:"description"`
		Difficulty        string `json:"difficulty"`
		Language          string `json:"language"`
		TemplateID        string `json:"template_id"`
		MissingTechnology string `json:"missing_technology"`
		RubricCriteria    []struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Weight      float64 `json:"weight"`
		} `json:"rubric_criteria"`
	}
	if err := json.Unmarshal([]byte(cleanJSONFence(raw)), &payload); err != nil {
		return nil, &AISettingsValidationError{
			Code:    "ai_response_invalid",
			Message: "el motor de IA devolvió JSON inválido para la práctica de desarrollo",
		}
	}

	title := strings.TrimSpace(payload.Title)
	if title == "" {
		title = "Práctica de Desarrollo"
	}
	description := strings.TrimSpace(payload.Description)
	if description == "" {
		description = prompt
	}

	language := strings.TrimSpace(payload.Language)
	if language == "" {
		language = strings.TrimSpace(req.Language)
	}
	if language == "" {
		language = "desarrollo"
	}

	draft := &domain.AIGenerateExerciseResponseDTO{
		Title:             title,
		Description:       description,
		Difficulty:        difficulty,
		Language:          language,
		EnvironmentType:   string(domain.TargetEnvironmentIDEPersistente),
		LatencyMS:         latencyMS,
		RunnerVerifiedAll: true,
	}

	// Filtro 1: Resolución determinista de Plantilla homologada y detección de tecnologías faltantes
	var selectedTemplateID, selectedTemplateName, missingTech string
	if lockedTemplate != nil {
		selectedTemplateID = lockedTemplate.ID
		selectedTemplateName = lockedTemplate.Name
		missingTech = ""
	} else {
		selectedTemplateID, selectedTemplateName, missingTech = s.resolveIdeTemplate(payload.TemplateID, payload.MissingTechnology, prompt, approvedTemplates)
	}
	draft.TemplateID = selectedTemplateID
	draft.TemplateName = selectedTemplateName
	draft.MissingTechnology = missingTech

	// Filtro 2: Normalización matemática de rúbrica a 100% exacto con niveles institucionales
	var rawCriteria []rawRubricCriterion
	for _, c := range payload.RubricCriteria {
		rawCriteria = append(rawCriteria, rawRubricCriterion{
			Name:        c.Name,
			Description: c.Description,
			Weight:      c.Weight,
		})
	}
	draft.Rubric = normalizeRubricCriteria(rawCriteria)

	return draft, nil
}

func (s *AICopilotService) resolveIdeTemplate(
	suggestedID, suggestedMissing, prompt string,
	approvedTemplates []*domain.AdminTemplateReviewItem,
) (templateID, templateName, missingTech string) {
	suggestedMissing = strings.TrimSpace(suggestedMissing)
	if len(approvedTemplates) == 0 {
		missing := strings.TrimSpace(suggestedMissing)
		if missing == "" {
			missing = "No hay entornos IDE aprobados disponibles en el catálogo"
		}
		return "", "", missing
	}

	suggestedID = strings.TrimSpace(suggestedID)
	requiredTechnologies := requestedIdeTechnologies(prompt)
	if suggestedMissing != "" {
		return "", "", suggestedMissing
	}
	compatible := func(template *domain.AdminTemplateReviewItem) bool {
		return len(missingTemplateTechnologies(template, requiredTechnologies)) == 0
	}

	// 1. Coincidencia exacta por ID, solo si cubre las tecnologías solicitadas.
	if suggestedID != "" {
		for _, tmpl := range approvedTemplates {
			if tmpl.ID == suggestedID && compatible(tmpl) {
				return tmpl.ID, tmpl.Name, ""
			}
		}
	}

	// 2. Coincidencia por texto/nombre de plantilla, también validando compatibilidad.
	promptLower := strings.ToLower(prompt)
	for _, tmpl := range approvedTemplates {
		nameLower := strings.ToLower(tmpl.Name)
		if compatible(tmpl) && (strings.Contains(promptLower, nameLower) || (suggestedID != "" && strings.Contains(nameLower, strings.ToLower(suggestedID)))) {
			return tmpl.ID, tmpl.Name, ""
		}
	}

	// 3. Usar alternativa únicamente cuando la consigna nombra tecnologías
	// conocidas y la plantilla cubre todas. Sin señales suficientes, no adivinar.
	if len(requiredTechnologies) > 0 {
		for _, tmpl := range approvedTemplates {
			if compatible(tmpl) {
				return tmpl.ID, tmpl.Name, ""
			}
		}
	}

	missing := strings.TrimSpace(suggestedMissing)
	if missing == "" {
		missingList := make([]string, 0, len(requiredTechnologies))
		for _, technology := range requiredTechnologies {
			missingList = append(missingList, technology.label)
		}
		missing = strings.Join(missingList, ", ")
	}
	if missing == "" {
		missing = "No hay una plantilla homologada compatible con esta práctica"
	}
	return "", "", missing
}

type ideTechnology struct {
	label string
	terms []string
}

var ideTechnologyCatalog = []ideTechnology{
	{label: "Angular", terms: []string{"angular"}},
	{label: "React", terms: []string{"react"}},
	{label: "Vue", terms: []string{"vue"}},
	{label: "NestJS", terms: []string{"nestjs", "nest.js", "nest js"}},
	{label: "FastAPI", terms: []string{"fastapi", "fast api"}},
	{label: "Django", terms: []string{"django"}},
	{label: "Flask", terms: []string{"flask"}},
	{label: "Spring", terms: []string{"spring boot", "spring"}},
	{label: "Node.js", terms: []string{"node.js", "nodejs", "node js"}},
	{label: "TypeScript", terms: []string{"typescript", "type script"}},
	{label: "JavaScript", terms: []string{"javascript", "java script"}},
	{label: "Python", terms: []string{"python"}},
	{label: "PostgreSQL", terms: []string{"postgresql", "postgres"}},
	{label: "SQLite", terms: []string{"sqlite"}},
	{label: "MySQL", terms: []string{"mysql"}},
	{label: "MariaDB", terms: []string{"mariadb"}},
	{label: "MongoDB", terms: []string{"mongodb", "mongo db"}},
	{label: "Redis", terms: []string{"redis"}},
	{label: "Java", terms: []string{"java"}},
	{label: "C++", terms: []string{"c++", "cpp"}},
	{label: "C#", terms: []string{"c#", "csharp", ".net"}},
	{label: "Go", terms: []string{"golang"}},
	{label: "Rust", terms: []string{"rust"}},
}

func requestedIdeTechnologies(prompt string) []ideTechnology {
	lower := strings.ToLower(prompt)
	var required []ideTechnology
	for _, technology := range ideTechnologyCatalog {
		for _, term := range technology.terms {
			if technology.label == "Java" && strings.Contains(lower, "javascript") {
				continue
			}
			matched := strings.Contains(lower, term)
			if technology.label == "Go" {
				matched = matched || regexp.MustCompile(`\bgo\b`).MatchString(lower)
			}
			if matched {
				required = append(required, technology)
				break
			}
		}
	}
	return required
}

func missingTemplateTechnologies(template *domain.AdminTemplateReviewItem, required []ideTechnology) []ideTechnology {
	if len(required) == 0 {
		return nil
	}
	stackConfig, _ := json.Marshal(template.StackConfig)
	servicesConfig, _ := json.Marshal(template.ServicesConfig)
	searchable := strings.ToLower(strings.Join([]string{
		template.Name,
		template.Description,
		template.DockerImage,
		string(stackConfig),
		string(servicesConfig),
	}, " "))
	var missing []ideTechnology
	for _, technology := range required {
		found := false
		for _, term := range technology.terms {
			if strings.Contains(searchable, term) {
				found = true
				break
			}
		}
		if !found && technology.label == "Node.js" {
			found = strings.Contains(searchable, "node")
		}
		if !found && technology.label == "Go" {
			found = regexp.MustCompile(`\bgo\b`).MatchString(searchable)
		}
		if !found {
			missing = append(missing, technology)
		}
	}
	return missing
}

type rawRubricCriterion struct {
	Name        string
	Description string
	Weight      float64
}

func normalizeRubricCriteria(raw []rawRubricCriterion) *domain.Rubric {
	var cleaned []rawRubricCriterion
	for _, r := range raw {
		name := strings.TrimSpace(r.Name)
		if name != "" {
			cleaned = append(cleaned, rawRubricCriterion{
				Name:        name,
				Description: strings.TrimSpace(r.Description),
				Weight:      r.Weight,
			})
		}
	}
	if len(cleaned) == 0 {
		cleaned = []rawRubricCriterion{
			{Name: "Funcionalidades Principales y Lógica de Negocio", Description: "Implementación completa de los requerimientos pedidos.", Weight: 40},
			{Name: "Estructura del Proyecto y Buenas Prácticas", Description: "Organización modular, legibilidad y estándares de código.", Weight: 30},
			{Name: "Manejo de Errores y Validaciones", Description: "Control de excepciones y validación de entradas de usuario.", Weight: 30},
		}
	}

	if len(cleaned) == 1 {
		cleaned[0].Weight = 100
	} else {
		var sum float64
		for _, c := range cleaned {
			if c.Weight > 0 {
				sum += c.Weight
			}
		}

		n := len(cleaned)
		if sum <= 0 {
			base := 100 / n
			remainder := 100 % n
			for i := range cleaned {
				w := base
				if i < remainder {
					w++
				}
				cleaned[i].Weight = float64(w)
			}
		} else {
			totalAllocated := 0
			maxIdx := 0
			maxVal := -1.0
			for i, c := range cleaned {
				scaled := int(math.Round((c.Weight / sum) * 100.0))
				if scaled < 1 {
					scaled = 1
				}
				cleaned[i].Weight = float64(scaled)
				totalAllocated += scaled
				if c.Weight > maxVal {
					maxVal = c.Weight
					maxIdx = i
				}
			}
			diff := 100 - totalAllocated
			cleaned[maxIdx].Weight += float64(diff)
		}
	}

	rubricID := uuid.New()
	criteria := make([]domain.RubricCriterion, len(cleaned))
	for i, c := range cleaned {
		critID := uuid.NewString()
		criteria[i] = domain.RubricCriterion{
			ID:          critID,
			Name:        c.Name,
			Description: c.Description,
			Weight:      c.Weight,
			Levels: []domain.RubricLevel{
				{
					Name:        "Logrado",
					Score:       100.0,
					Description: "Cumple completamente con los requisitos del criterio.",
				},
				{
					Name:        "Parcial",
					Score:       50.0,
					Description: "Cumple parcialmente con la funcionalidad o presenta observaciones menores.",
				},
				{
					Name:        "No logrado",
					Score:       0.0,
					Description: "No cumple con los requisitos o no presenta la funcionalidad solicitada.",
				},
			},
		}
	}

	rubric := &domain.Rubric{
		ID:        rubricID,
		Criteria:  criteria,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_ = rubric.Validate()
	return rubric
}

func generateIdeExerciseSystemPrompt(approvedTemplatesSummary string) string {
	catalogClause := ""
	if approvedTemplatesSummary != "" {
		catalogClause = fmt.Sprintf("\nPlantillas de desarrollo homologadas disponibles en el servidor:\n%s\nDebes elegir una plantilla de la lista solo si cubre las tecnologías requeridas e incluir su ID exacto en 'template_id'. Si ninguna plantilla es compatible o la tecnología pedida no está en la lista, deja 'template_id' vacío y reporta la tecnología faltante en 'missing_technology'; no sugieras una plantilla alternativa incompatible.", approvedTemplatesSummary)
	}
	return fmt.Sprintf(`Sos el copiloto pedagógico de prácticas de desarrollo (IDE persistente) de SOLV, la plataforma académica de laboratorios virtuales.
Tu objetivo es generar una práctica de desarrollo completa, clara y motivante para estudiantes universitarios de informática.
Respondé EXCLUSIVAMENTE con un objeto JSON válido (sin Markdown, sin fences, sin comentarios) con esta forma exacta:
{
  "title": "",
  "description": "",
  "difficulty": "easy|medium|hard",
  "language": "",
  "template_id": "",
  "missing_technology": "",
  "rubric_criteria": [
    {
      "name": "Nombre conciso del criterio de evaluación",
      "description": "Qué se evalúa en este punto",
      "weight": 25
    }
  ]
}
Reglas innegociables:
- title: Título académico claro y representativo de la práctica (ej: "Sistema de Gestión de Alumnos con NestJS y PostgreSQL").
- description: Enunciado detallado en Markdown estructurado con:
  - ## Contexto y Objetivos
  - ## Requerimientos Funcionales (con especificación de endpoints, vistas o módulos paso a paso)
  - ## Criterios de Entrega y Estructura esperada
- difficulty: "easy", "medium" o "hard".
- language: Lenguaje o framework principal (ej: typescript, python, java, etc.).
- template_id: solo una plantilla compatible del catálogo; si no hay una compatible, cadena vacía.
- missing_technology: tecnología requerida sin plantilla compatible; cadena vacía si el catálogo cubre la práctica.
- rubric_criteria: Entre 3 y 6 criterios pedagógicos claros que cubran la práctica. La suma de sus pesos (weight) debe ser aproximadamente 100.%s`, catalogClause)
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
- description: redacción técnica concisa y formal del componente para docentes y administradores (ej: "Entorno oficial de ejecución de Ruby para scripts y desarrollo de aplicaciones"). PROHIBIDO copiar la consigna informal o coloquial del administrador (como "quiero tener...", "necesito...").
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
- base_image lleva tag de versión fija para los motores de contenedor (postgres:18-alpine, mysql:8.4, mariadb:11.4, mongo:7.0, redis:7.2-alpine) y va vacía solo en sqlite, que vive en un archivo del laboratorio.
- install_mode siempre "none" y package_names vacío: el motor se aprovisiona como servicio acoplado, no se instala en la imagen.
- compatible_runtimes debe ser ["universal"] y ram_delta_mb el techo de cgroup del satélite del motor (0 sqlite, 384 postgres, 64 redis, 128 mysql/mariadb/mongodb).`
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
