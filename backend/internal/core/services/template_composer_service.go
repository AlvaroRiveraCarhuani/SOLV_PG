package services

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"solv-backend/internal/core/domain"
)

// ComponentLookupPort permite desacoplar el compositor del repositorio completo en pruebas unitarias.
type ComponentLookupPort interface {
	GetComponent(ctx context.Context, tenantID, idOrSlug string) (*domain.TemplateComponent, error)
}

type repoComponentLookupAdapter struct {
	repo domain.TemplateComponentRepository
}

func (a *repoComponentLookupAdapter) GetComponent(ctx context.Context, tenantID, idOrSlug string) (*domain.TemplateComponent, error) {
	return a.repo.GetTemplateComponentByIDOrSlug(ctx, tenantID, idOrSlug)
}

var validFolderRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// TemplateComposerService compila una selección de componentes tecnológicos en un plan técnico determinista.
type TemplateComposerService struct {
	lookup ComponentLookupPort
}

func NewTemplateComposerService(lookup ComponentLookupPort) *TemplateComposerService {
	return &TemplateComposerService{lookup: lookup}
}

func NewTemplateComposerServiceFromRepo(repo domain.TemplateComponentRepository) *TemplateComposerService {
	return &TemplateComposerService{lookup: &repoComponentLookupAdapter{repo: repo}}
}

func (s *TemplateComposerService) Compose(
	ctx context.Context,
	tenantID, targetEnvironment string,
	cfg domain.StackConfig,
) (*domain.ComposedTemplatePlan, error) {
	if strings.TrimSpace(cfg.RuntimeID) == "" {
		return nil, fmt.Errorf("runtime_id is required to compose a template")
	}

	rt, err := s.lookup.GetComponent(ctx, tenantID, strings.TrimSpace(cfg.RuntimeID))
	if err != nil {
		return nil, fmt.Errorf("primary runtime component not found: %w", err)
	}

	isJudge := strings.TrimSpace(targetEnvironment) == "JUEZ_EFIMERO" || strings.TrimSpace(targetEnvironment) == "JUEZ_VIRTUAL"

	baseImage := strings.TrimSpace(rt.BaseImage)
	if baseImage == "" {
		baseImage = "python:3.12-slim-bookworm"
	}

	var nameParts []string
	nameParts = append(nameParts, rt.Name)

	var buildLayers []string
	if strings.TrimSpace(rt.BuildLayerSnippet) != "" {
		buildLayers = append(buildLayers, strings.TrimSpace(rt.BuildLayerSnippet))
	}

	var setupLines []string
	var cleanFolders []string
	if !isJudge {
		for _, f := range cfg.WorkspaceFolders {
			clean := strings.Trim(strings.TrimSpace(f), "/")
			if clean != "" && validFolderRegex.MatchString(clean) {
				cleanFolders = append(cleanFolders, "/home/workspace/"+clean)
			}
		}
		if len(cleanFolders) > 0 {
			setupLines = append(setupLines, "mkdir -p "+strings.Join(cleanFolders, " "))
		}
	}

	if strings.TrimSpace(rt.SetupSnippet) != "" && !isJudge {
		setupLines = append(setupLines, strings.TrimSpace(rt.SetupSnippet))
	}

	toolsSet := make(map[string]bool)
	var orderedTools []string
	addTool := func(t string) {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" && !toolsSet[trimmed] {
			toolsSet[trimmed] = true
			orderedTools = append(orderedTools, trimmed)
		}
	}
	for _, t := range rt.ToolsProvided {
		addTool(t)
	}
	if !isJudge {
		addTool("git")
	}

	var webComp *domain.TemplateComponent
	// 2. Plataforma Web opcional (Solo para entornos IDE interactivos)
	if !isJudge && strings.TrimSpace(cfg.WebPlatformID) != "" && cfg.WebPlatformID != cfg.RuntimeID {
		resolvedWeb, errWeb := s.lookup.GetComponent(ctx, tenantID, strings.TrimSpace(cfg.WebPlatformID))
		if errWeb == nil && resolvedWeb != nil {
			webComp = resolvedWeb
			nameParts = append(nameParts, webComp.Name)
			// Si la imagen base no es Node, asegurar que Node/npm estén instalados antes de la capa web
			if !strings.Contains(strings.ToLower(baseImage), "node") {
				nodeBootstrap := "RUN apt-get update && apt-get install -y --no-install-recommends nodejs npm && rm -rf /var/lib/apt/lists/*"
				buildLayers = append(buildLayers, nodeBootstrap)
			}
			if strings.TrimSpace(webComp.BuildLayerSnippet) != "" {
				buildLayers = append(buildLayers, strings.TrimSpace(webComp.BuildLayerSnippet))
			}
			if strings.TrimSpace(webComp.SetupSnippet) != "" {
				setupLines = append(setupLines, strings.TrimSpace(webComp.SetupSnippet))
			}
			for _, t := range webComp.ToolsProvided {
				addTool(t)
			}
		}
	}

	// 3. Complementos (Addons) ordenados deterministamente por ID/Slug
	sortedAddonIDs := make([]string, 0, len(cfg.AddonIDs))
	seenAddon := make(map[string]bool)
	for _, id := range cfg.AddonIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" && !seenAddon[trimmed] {
			seenAddon[trimmed] = true
			sortedAddonIDs = append(sortedAddonIDs, trimmed)
		}
	}
	sort.Strings(sortedAddonIDs)

	selectedAddons := make([]*domain.TemplateComponent, 0, len(sortedAddonIDs))
	for _, addonID := range sortedAddonIDs {
		addon, errAddon := s.lookup.GetComponent(ctx, tenantID, addonID)
		if errAddon != nil || addon == nil {
			continue
		}
		selectedAddons = append(selectedAddons, addon)
		nameParts = append(nameParts, addon.Name)
		if strings.TrimSpace(addon.BuildLayerSnippet) != "" {
			buildLayers = append(buildLayers, strings.TrimSpace(addon.BuildLayerSnippet))
		}
		if strings.TrimSpace(addon.SetupSnippet) != "" && !isJudge {
			setupLines = append(setupLines, strings.TrimSpace(addon.SetupSnippet))
		}
		for _, t := range addon.ToolsProvided {
			addTool(t)
		}
	}

	// 4. Base de datos acoplada (Solo para entornos IDE interactivos)
	var dbComp *domain.TemplateComponent
	servicesCfg := domain.ServicesConfig{Services: []domain.ServiceRequirement{}}
	if !isJudge && strings.TrimSpace(cfg.DatabaseID) != "" && cfg.DatabaseID != "none" {
		resolvedDB, errDB := s.lookup.GetComponent(ctx, tenantID, strings.TrimSpace(cfg.DatabaseID))
		if errDB == nil && resolvedDB != nil && resolvedDB.ServiceEngine != "" && resolvedDB.ServiceEngine != "none" {
			dbComp = resolvedDB
			nameParts = append(nameParts, dbComp.Name)
			category := "database"
			if dbComp.ServiceEngine == "redis" {
				category = "cache"
			}
			servicesCfg.Services = append(servicesCfg.Services, domain.ServiceRequirement{
				Category: category,
				Engine:   dbComp.ServiceEngine,
			})
			for _, t := range dbComp.ToolsProvided {
				addTool(t)
			}
		}
	}

	// 5. Cálculo de memoria basado en procesos residentes activos (sin inflación lineal por librerías en disco)
	recommendedRam := calculateResidentRecommendedRAM(rt, webComp, selectedAddons, dbComp, isJudge)

	// Construcción determinista de imagen y Dockerfile
	needsBuild := len(buildLayers) > 0
	sigHash := domain.ComputeStackSignature(baseImage, buildLayers)
	resolvedImage := baseImage
	dockerfileContent := "FROM " + baseImage
	if needsBuild {
		resolvedImage = domain.FormatBuiltStackImageTag(sigHash)
		var dfLines []string
		dfLines = append(dfLines, "# Dockerfile generado deterministamente por el Constructor de Entornos SOLV")
		dfLines = append(dfLines, "FROM "+baseImage)
		dfLines = append(dfLines, buildLayers...)
		dockerfileContent = strings.Join(dfLines, "\n")
	}

	prefix := "Laboratorio: "
	if isJudge {
		prefix = "Evaluador: "
	}
	suggestedName := prefix + strings.Join(nameParts, " + ")

	defaultEntrypoint := ""
	if isJudge {
		defaultEntrypoint = strings.TrimSpace(rt.JudgeEntrypoint)
		if defaultEntrypoint == "" {
			defaultEntrypoint = inferJudgeEntrypoint(baseImage)
		}
	}

	return &domain.ComposedTemplatePlan{
		SuggestedName:     suggestedName,
		ResolvedImage:     resolvedImage,
		BaseImage:         baseImage,
		NeedsLocalBuild:   needsBuild,
		SignatureHash:     sigHash,
		DockerfileContent: dockerfileContent,
		SetupScript:       strings.Join(setupLines, "\n"),
		ToolsDeclared:     orderedTools,
		DefaultEntrypoint: defaultEntrypoint,
		RecommendedRamMB:  recommendedRam,
		ServicesConfig:    servicesCfg,
	}, nil
}

func isCppRuntime(rt *domain.TemplateComponent) bool {
	if rt == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(rt.IconKey), "cpp") {
		return true
	}
	for _, fam := range rt.CompatibleRuntimes {
		if strings.EqualFold(strings.TrimSpace(fam), "cpp") {
			return true
		}
	}
	return rt.RamDeltaMB > 0 && rt.RamDeltaMB <= 384
}

func isJavaRuntime(rt *domain.TemplateComponent) bool {
	if rt == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(rt.IconKey), "java") {
		return true
	}
	for _, fam := range rt.CompatibleRuntimes {
		if strings.EqualFold(strings.TrimSpace(fam), "java") {
			return true
		}
	}
	return rt.RamDeltaMB >= 1024
}

func isResidentAddon(addon *domain.TemplateComponent) bool {
	if addon == nil {
		return false
	}
	sub := strings.ToLower(strings.TrimSpace(addon.SubCategory))
	slug := strings.ToLower(strings.TrimSpace(addon.Slug))
	if sub == "devserver" || sub == "heavy" || sub == "data" || strings.Contains(slug, "datascience") || strings.Contains(slug, "data-science") {
		return true
	}
	return addon.RamDeltaMB >= 256
}

func calculateResidentRecommendedRAM(
	rt *domain.TemplateComponent,
	webComp *domain.TemplateComponent,
	addons []*domain.TemplateComponent,
	dbComp *domain.TemplateComponent,
	isJudge bool,
) int {
	if isJudge {
		const runtimeBaseMB = 32
		processFootprintMB := 224
		if isJavaRuntime(rt) {
			processFootprintMB = 480
		} else if isCppRuntime(rt) {
			processFootprintMB = 96
		}
		for _, a := range addons {
			if !isResidentAddon(a) {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(a.SubCategory), "heavy") || a.RamDeltaMB >= 384 {
				if processFootprintMB < 480 {
					processFootprintMB = 480
				}
			} else if processFootprintMB < 224 {
				processFootprintMB = 224
			}
		}
		return roundToRamPreset(runtimeBaseMB+processFootprintMB, true)
	}

	const editorBaseMB = 210
	satelliteBaseMB := 0
	if dbComp != nil && dbComp.ServiceEngine != "" && dbComp.ServiceEngine != "none" && dbComp.RamDeltaMB > 0 {
		satelliteBaseMB = dbComp.RamDeltaMB
	}

	activeProcessFootprintMB := 256
	if isCppRuntime(rt) {
		activeProcessFootprintMB = 128
	}
	if isJavaRuntime(rt) {
		if webComp != nil || len(addons) > 0 || satelliteBaseMB > 0 {
			activeProcessFootprintMB = 1280
		} else {
			activeProcessFootprintMB = 640
		}
	}
	if webComp != nil && activeProcessFootprintMB < 640 {
		activeProcessFootprintMB = 640
	}

	for _, a := range addons {
		if !isResidentAddon(a) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(a.SubCategory), "heavy") || a.RamDeltaMB >= 384 {
			if webComp != nil || isJavaRuntime(rt) {
				if activeProcessFootprintMB < 1280 {
					activeProcessFootprintMB = 1280
				}
			} else if activeProcessFootprintMB < 512 {
				activeProcessFootprintMB = 512
			}
		} else if activeProcessFootprintMB < 384 {
			activeProcessFootprintMB = 384
		}
	}

	return roundToRamPreset(editorBaseMB+satelliteBaseMB+activeProcessFootprintMB, false)
}

func roundToRamPreset(rawMB int, isJudge bool) int {
	if isJudge {
		if rawMB <= 128 {
			return 128
		}
		if rawMB <= 256 {
			return 256
		}
		return 512
	}
	switch {
	case rawMB <= 384:
		return 384
	case rawMB <= 512:
		return 512
	case rawMB <= 768:
		return 768
	case rawMB <= 1024:
		return 1024
	case rawMB <= 2048:
		return 2048
	default:
		return 4096
	}
}

func inferJudgeEntrypoint(image string) string {
	lower := strings.ToLower(image)
	switch {
	case strings.Contains(lower, "python"):
		return "python3 solution.py"
	case strings.Contains(lower, "gcc") || strings.Contains(lower, "clang"):
		return "gcc -O2 solution.c -o solution && ./solution"
	case strings.Contains(lower, "java") || strings.Contains(lower, "temurin") || strings.Contains(lower, "openjdk"):
		return "javac Solution.java && java -Xmx256m Solution"
	case strings.Contains(lower, "golang") || strings.HasPrefix(lower, "go:"):
		return "go run main.go"
	case strings.Contains(lower, "node"):
		return "node solution.js"
	case strings.Contains(lower, "rust"):
		return "rustc -O solution.rs -o solution && ./solution"
	default:
		return "sh solution.sh"
	}
}
