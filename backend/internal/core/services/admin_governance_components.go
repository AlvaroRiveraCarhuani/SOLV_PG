package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"solv-backend/internal/core/domain"
	"strings"
)

func (s *AdminGovernanceService) ListTemplateCategories(ctx context.Context, tenantID string) ([]*domain.TemplateCategory, error) {
	return s.govRepo.ListTemplateCategories(ctx, tenantID)
}

func (s *AdminGovernanceService) CreateTemplateCategory(ctx context.Context, tenantID string, dto domain.CreateCategoryDTO) (*domain.TemplateCategory, error) {
	dto.Name = strings.TrimSpace(dto.Name)
	if dto.Name == "" {
		return nil, errors.New("el nombre de la categoría es obligatorio")
	}
	return s.govRepo.CreateTemplateCategory(ctx, tenantID, dto)
}

func (s *AdminGovernanceService) UpdateTemplateCategory(ctx context.Context, tenantID, categoryID string, dto domain.UpdateCategoryDTO) (*domain.TemplateCategory, error) {
	dto.Name = strings.TrimSpace(dto.Name)
	if dto.Name == "" {
		return nil, errors.New("el nombre de la categoría es obligatorio")
	}
	return s.govRepo.UpdateTemplateCategory(ctx, tenantID, categoryID, dto)
}

func (s *AdminGovernanceService) DeleteTemplateCategory(ctx context.Context, tenantID, actorID, categoryID string) error {
	if err := s.govRepo.DeleteTemplateCategory(ctx, tenantID, categoryID); err != nil {
		return err
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      actorID,
			Action:       "TEMPLATE_CATEGORY_DELETED",
			ResourceType: "template_category",
			ResourceID:   &categoryID,
			StatusCode:   200,
		})
	}
	return nil
}

func (s *AdminGovernanceService) ReorderTemplateCategories(ctx context.Context, tenantID string, items []domain.ReorderCategoryItemDTO) error {
	return s.govRepo.ReorderTemplateCategories(ctx, tenantID, items)
}

func (s *AdminGovernanceService) ListTemplateModels(ctx context.Context, tenantID, targetEnv, categoryID string, includeInactive bool) ([]*domain.TemplateModelItemDTO, error) {
	return s.govRepo.ListTemplateModels(ctx, tenantID, targetEnv, categoryID, includeInactive)
}

func (s *AdminGovernanceService) UpdateTemplateModel(ctx context.Context, tenantID, modelID string, dto domain.UpdateTemplateModelDTO) (*domain.TemplateModelItemDTO, error) {
	dto.Title = strings.TrimSpace(dto.Title)
	if dto.Title == "" {
		return nil, errors.New("el título del modelo es obligatorio")
	}
	if dto.CategoryID == "" {
		return nil, errors.New("la categoría es obligatoria para el modelo")
	}
	return s.govRepo.UpdateTemplateModel(ctx, tenantID, modelID, dto)
}

func (s *AdminGovernanceService) DeactivateTemplateModel(ctx context.Context, tenantID, actorID, modelID string) error {
	if actorID == "" {
		actorID = domain.DefaultTenantID
	}
	if err := s.govRepo.SetTemplateModelActive(ctx, tenantID, modelID, false); err != nil {
		return err
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      actorID,
			Action:       "TEMPLATE_MODEL_DEACTIVATED",
			ResourceType: "template_model",
			ResourceID:   &modelID,
			StatusCode:   200,
		})
	}
	return nil
}

func (s *AdminGovernanceService) ReactivateTemplateModel(ctx context.Context, tenantID, actorID, modelID string) error {
	if actorID == "" {
		actorID = domain.DefaultTenantID
	}
	if err := s.govRepo.SetTemplateModelActive(ctx, tenantID, modelID, true); err != nil {
		return err
	}
	if s.auditRepo != nil {
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      actorID,
			Action:       "TEMPLATE_MODEL_REACTIVATED",
			ResourceType: "template_model",
			ResourceID:   &modelID,
			StatusCode:   200,
		})
	}
	return nil
}

func (s *AdminGovernanceService) PromoteTemplateToModel(
	ctx context.Context,
	tenantID, templateID, adminID string,
	dto domain.PromoteTemplateToModelDTO,
) (*domain.TemplateModelItemDTO, error) {
	dto.Title = strings.TrimSpace(dto.Title)
	if dto.Title == "" {
		return nil, errors.New("el título del modelo es obligatorio")
	}
	if dto.CategoryID == "" {
		return nil, errors.New("la categoría es obligatoria para el modelo")
	}

	model, err := s.govRepo.PromoteTemplateToModel(ctx, tenantID, templateID, adminID, dto)
	if err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      adminID,
			Action:       "TEMPLATE_PROMOTED_TO_MODEL",
			ResourceType: "template_model",
			ResourceID:   &model.ID,
			StatusCode:   201,
		})
	}

	return model, nil
}

func (s *AdminGovernanceService) SaveTemplateDraft(ctx context.Context, tenantID, userID string, formData json.RawMessage, templateID *string) (*domain.TemplateDraft, error) {
	if tenantID == "" || userID == "" {
		return nil, errors.New("tenant_id y user_id son requeridos")
	}
	if len(formData) == 0 {
		formData = json.RawMessage("{}")
	}
	return s.govRepo.SaveDraft(ctx, tenantID, userID, formData, templateID)
}

func (s *AdminGovernanceService) GetTemplateDraft(ctx context.Context, tenantID, userID string) (*domain.TemplateDraft, error) {
	if tenantID == "" || userID == "" {
		return nil, errors.New("tenant_id y user_id son requeridos")
	}
	return s.govRepo.GetDraftByUser(ctx, tenantID, userID)
}

func (s *AdminGovernanceService) DeleteTemplateDraft(ctx context.Context, tenantID, userID string) error {
	if tenantID == "" || userID == "" {
		return errors.New("tenant_id y user_id son requeridos")
	}
	return s.govRepo.DeleteDraft(ctx, tenantID, userID)
}

func (s *AdminGovernanceService) GetImageUsageCounts(ctx context.Context, tenantID string) (map[string]int, error) {
	return s.govRepo.GetImageUsageCounts(ctx, tenantID)
}

func (s *AdminGovernanceService) componentRepo() (domain.TemplateComponentRepository, error) {
	if repo, ok := s.govRepo.(domain.TemplateComponentRepository); ok {
		return repo, nil
	}
	return nil, errors.New("el repositorio actual no soporta gestión de componentes tecnológicos")
}

func (s *AdminGovernanceService) ListTemplateComponents(ctx context.Context, tenantID, layer string, includeInactive bool) ([]*domain.TemplateComponent, error) {
	repo, err := s.componentRepo()
	if err != nil {
		return nil, err
	}
	return repo.ListTemplateComponents(ctx, tenantID, layer, includeInactive)
}

func (s *AdminGovernanceService) CreateTemplateComponent(ctx context.Context, tenantID, adminID string, dto domain.CreateTemplateComponentDTO) (*domain.TemplateComponent, error) {
	repo, err := s.componentRepo()
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(dto.Name)
	if len(name) < 2 {
		return nil, errors.New("el nombre del componente tecnológico debe tener al menos 2 caracteres")
	}
	if dto.Layer != domain.ComponentLayerRuntime && dto.Layer != domain.ComponentLayerAddon && dto.Layer != domain.ComponentLayerDatabase {
		return nil, errors.New("capa de componente inválida: debe ser runtime, addon o database")
	}
	if dto.Layer == domain.ComponentLayerRuntime && strings.TrimSpace(dto.BaseImage) == "" {
		return nil, errors.New("la imagen base es obligatoria para componentes de tipo runtime")
	}
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(dto.BaseImage)), ":latest") {
		return nil, errors.New("el tag :latest está prohibido por reproducibilidad académica")
	}

	dto = domain.NormalizeCreateComponentDTO(dto)

	// Gobernanza del catálogo de motores: solo se aceptan los motores canónicos, con
	// su modo de aprovisionamiento conocido (archivo, host de aula o contenedor
	// satélite auto-orquestado).
	if dto.Layer == domain.ComponentLayerDatabase && !domain.IsSupportedServiceEngine(dto.ServiceEngine) {
		return nil, fmt.Errorf(
			"motor de base de datos no soportado: %q (motores válidos: %s)",
			dto.ServiceEngine, strings.Join(domain.SupportedServiceEngines(), ", "),
		)
	}

	synthesizedLayer, errSynth := domain.SynthesizeAssistedBuildLayer(dto.InstallMode, dto.PackageNames, dto.BuildLayerSnippet)
	if errSynth != nil {
		return nil, errSynth
	}

	comp := &domain.TemplateComponent{
		TenantID:           tenantID,
		Slug:               strings.TrimSpace(dto.Slug),
		Layer:              dto.Layer,
		SubCategory:        strings.TrimSpace(dto.SubCategory),
		Name:               name,
		Description:        strings.TrimSpace(dto.Description),
		IconKey:            strings.TrimSpace(dto.IconKey),
		BaseImage:          strings.TrimSpace(dto.BaseImage),
		CompatibleRuntimes: dto.CompatibleRuntimes,
		InstallMode:        dto.InstallMode,
		PackageNames:       dto.PackageNames,
		BuildLayerSnippet:  synthesizedLayer,
		SetupSnippet:       strings.TrimSpace(dto.SetupSnippet),
		JudgeEntrypoint:    strings.TrimSpace(dto.JudgeEntrypoint),
		ToolsProvided:      dto.ToolsProvided,
		ServiceEngine:      strings.TrimSpace(dto.ServiceEngine),
		RamDeltaMB:         dto.RamDeltaMB,
		SortOrder:          dto.SortOrder,
	}

	created, err := repo.CreateTemplateComponent(ctx, tenantID, comp)
	if err != nil {
		return nil, err
	}
	if s.auditRepo != nil && adminID != "" {
		_ = s.auditRepo.Create(ctx, &domain.AuditLog{
			TenantID:     tenantID,
			ActorID:      adminID,
			Action:       "TEMPLATE_COMPONENT_CREATED",
			ResourceType: "template_component",
			ResourceID:   &created.ID,
			StatusCode:   201,
		})
	}
	return created, nil
}

func (s *AdminGovernanceService) UpdateTemplateComponent(ctx context.Context, tenantID, componentID, adminID string, dto domain.UpdateTemplateComponentDTO) (*domain.TemplateComponent, error) {
	repo, err := s.componentRepo()
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(dto.Name)
	if len(name) < 2 {
		return nil, errors.New("el nombre del componente tecnológico debe tener al menos 2 caracteres")
	}
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(dto.BaseImage)), ":latest") {
		return nil, errors.New("el tag :latest está prohibido por reproducibilidad académica")
	}
	// El motor de servicio solo existe para la capa de base de datos: se canonicaliza
	// (mongo -> mongodb) y se rechaza cualquier motor fuera del dominio soportado.
	if engine := strings.TrimSpace(dto.ServiceEngine); engine != "" {
		canonical := domain.NormalizeServiceEngine(engine, "", dto.Name, "", dto.BaseImage)
		if canonical == "" {
			return nil, fmt.Errorf(
				"motor de base de datos no soportado: %q (motores válidos: %s)",
				engine, strings.Join(domain.SupportedServiceEngines(), ", "),
			)
		}
		dto.ServiceEngine = canonical
	}
	var synthesized string
	if strings.TrimSpace(dto.InstallMode) != "" {
		mode := domain.InstallMode(strings.TrimSpace(dto.InstallMode))
		if mode != domain.InstallModeNone {
			sLayer, errSynth := domain.SynthesizeAssistedBuildLayer(mode, dto.PackageNames, dto.BuildLayerSnippet)
			if errSynth != nil {
				return nil, errSynth
			}
			synthesized = sLayer
		} else if strings.TrimSpace(dto.BuildLayerSnippet) == "" {
			dto.BuildLayerSnippet = ""
		}
	}
	return repo.UpdateTemplateComponent(ctx, tenantID, componentID, dto, synthesized)
}

func (s *AdminGovernanceService) DeleteTemplateComponent(ctx context.Context, tenantID, componentID string) error {
	repo, err := s.componentRepo()
	if err != nil {
		return err
	}
	return repo.DeleteTemplateComponent(ctx, tenantID, componentID)
}

func (s *AdminGovernanceService) ComposeTemplatePreview(
	ctx context.Context,
	tenantID, targetEnvironment string,
	cfg domain.StackConfig,
) (*domain.ComposedTemplatePlan, error) {
	repo, err := s.componentRepo()
	if err != nil {
		return nil, err
	}
	composer := NewTemplateComposerServiceFromRepo(repo)
	return composer.Compose(ctx, tenantID, targetEnvironment, cfg)
}

func (s *AdminGovernanceService) UpdateOfficialTemplate(
	ctx context.Context,
	tenantID, templateID, adminID string,
	dto domain.UpdateOfficialTemplateDTO,
) (*domain.AdminTemplateReviewItem, error) {
	repo, err := s.componentRepo()
	if err != nil {
		return nil, err
	}

	dto.Name = strings.TrimSpace(dto.Name)
	if dto.Name == "" {
		return nil, errors.New("el nombre de la plantilla es requerido")
	}

	dto.DockerImage = strings.TrimSpace(dto.DockerImage)
	if dto.DockerImage == "" {
		return nil, errors.New("la imagen Docker es requerida")
	}

	if strings.HasSuffix(strings.ToLower(dto.DockerImage), ":latest") {
		return nil, ErrLatestTagForbidden
	}

	if !dockerImageRegex.MatchString(dto.DockerImage) {
		return nil, ErrInvalidDockerImage
	}

	if !domain.IsBuiltStackImageTag(dto.DockerImage) {
		if err := ValidateAllowedRegistry(dto.DockerImage); err != nil {
			return nil, err
		}
	}

	if dto.TargetEnvironment == "" {
		dto.TargetEnvironment = "IDE_PERSISTENTE"
	}
	if dto.TimeoutMS <= 0 {
		dto.TimeoutMS = 5000
	}
	if domain.IsBuiltStackImageTag(dto.DockerImage) {
		if err := validateTemplateVerification(ctx, s.envTestJobs, tenantID, dto.VerificationJobID, verificationConfigForUpdate(dto), true); err != nil {
			return nil, err
		}
	}

	if dto.TargetEnvironment == "JUEZ_EFIMERO" || dto.TargetEnvironment == "JUEZ_VIRTUAL" || dto.TargetEnvironment == "AMBOS" {
		dto.ServicesConfig = &domain.ServicesConfig{
			Services: []domain.ServiceRequirement{},
		}
		if strings.TrimSpace(dto.Entrypoint) == "" {
			return nil, errors.New("el comando de ejecución (entrypoint) es obligatorio para entornos con evaluador automático")
		}
	} else if dto.ServicesConfig == nil {
		dto.ServicesConfig = &domain.ServicesConfig{
			Services: []domain.ServiceRequirement{},
		}
	}

	if err := domain.NormalizeServiceBindings(dto.ServicesConfig); err != nil {
		return nil, err
	}

	if dto.BaseRamMB > 0 {
		minFloorMB := calculateMinimumVitalFloorMB(dto.TargetEnvironment)
		if dto.BaseRamMB < minFloorMB {
			return nil, fmt.Errorf("%w: base_ram_mb=%d es inferior al piso vital requerido (%d MB)", ErrRamBelowVitalFloor, dto.BaseRamMB, minFloorMB)
		}
		totalHostMB := s.getHostTotalRAM(ctx)
		maxAllowedRAM := domain.CalculateHostMaxAllowedRAM(totalHostMB)
		if err := domain.ValidateRamAgainstHost(dto.BaseRamMB, maxAllowedRAM); err != nil {
			return nil, err
		}
	}
	return repo.UpdateOfficialTemplate(ctx, tenantID, templateID, adminID, dto)
}
