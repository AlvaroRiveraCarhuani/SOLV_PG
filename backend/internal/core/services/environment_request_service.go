package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

// EnvironmentRequestService gestiona el pedido de entorno del docente.
//
// El docente describe su práctica y, opcionalmente, elige piezas del catálogo.
// El servicio arma la receta con el mismo constructor que usa el administrador
// (así el docente nunca escribe una imagen), calcula una huella de esa receta y
// la usa para dos cosas: ofrecer una plantilla ya aprobada si existe, y sumar la
// materia a una solicitud igual que ya está en revisión en lugar de duplicarla.
type EnvironmentRequestService struct {
	repo     domain.EnvironmentRequestRepository
	compRepo domain.TemplateComponentRepository
	composer *TemplateComposerService
	now      func() time.Time
}

func NewEnvironmentRequestService(
	repo domain.EnvironmentRequestRepository,
	compRepo domain.TemplateComponentRepository,
) *EnvironmentRequestService {
	svc := &EnvironmentRequestService{
		repo:     repo,
		compRepo: compRepo,
		now:      time.Now,
	}
	if compRepo != nil {
		svc.composer = NewTemplateComposerServiceFromRepo(compRepo)
	}
	return svc
}

// Preview arma la receta sin guardarla: es lo que el docente revisa antes de
// enviar. Devuelve la plantilla existente o la solicitud gemela si las hay.
func (s *EnvironmentRequestService) Preview(
	ctx context.Context, tenantID, userID string, dto domain.EnvironmentRequestPreviewDTO,
) (*domain.EnvironmentRequestPreview, error) {
	uses := domain.NormalizeEnvironmentUses(dto.Uses)
	env := domain.DeriveTargetEnvironment(uses)

	preview := &domain.EnvironmentRequestPreview{
		TargetEnvironment: env,
		Components:        []domain.EnvironmentRequestPreviewComponent{},
		MissingComponents: []domain.EnvironmentRequestMissingComponent{},
	}

	plan, err := s.compose(ctx, tenantID, env, dto.StackConfig)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return preview, nil
	}

	preview.SuggestedName = plan.SuggestedName
	preview.RecommendedRAMMB = plan.RecommendedRamMB
	preview.Signature = environmentRequestSignature(plan, env)

	resolved, err := s.resolveComponents(ctx, tenantID, dto.StackConfig)
	if err != nil {
		return nil, err
	}
	preview.Components = resolved

	if preview.Signature != "" {
		if dup, err := s.repo.FindPendingBySignature(ctx, tenantID, preview.Signature); err == nil && dup != nil {
			subjects := make([]string, 0, len(dup.Subjects))
			for _, subject := range dup.Subjects {
				subjects = append(subjects, subject.SubjectName)
			}
			preview.DuplicateRequest = &domain.EnvironmentRequestDuplicate{
				ID:            dup.ID,
				Title:         dup.Title,
				Status:        dup.Status,
				Subjects:      subjects,
				RequestedByMe: dup.RequestedBy == userID,
			}
		}
	}

	// Solo cuando hay capas propias (imagen construida) la referencia de imagen
	// identifica la receta; con una imagen pública el tag lo comparten entornos
	// distintos y no sirve para detectar duplicados.
	if plan.NeedsLocalBuild {
		if ref, err := s.repo.FindApprovedTemplateByImage(ctx, tenantID, plan.ResolvedImage); err == nil && ref != nil {
			preview.ExistingTemplate = ref
		}
	}

	return preview, nil
}

// Create guarda la solicitud. Si ya hay una idéntica en revisión, agrega la
// materia y devuelve esa ficha con Joined=true.
func (s *EnvironmentRequestService) Create(
	ctx context.Context, tenantID, userID string, dto domain.CreateEnvironmentRequestDTO,
) (*domain.EnvironmentRequestCreated, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: falta el docente que solicita", domain.ErrEnvironmentRequestInvalid)
	}

	reason := strings.TrimSpace(dto.Description)
	if len([]rune(reason)) < 10 {
		return nil, fmt.Errorf("%w: cuéntanos qué deben lograr los estudiantes (al menos 10 caracteres)", domain.ErrEnvironmentRequestInvalid)
	}

	uses := domain.NormalizeEnvironmentUses(dto.Uses)
	preview, err := s.Preview(ctx, tenantID, userID, domain.EnvironmentRequestPreviewDTO{
		Uses:        uses,
		StackConfig: dto.StackConfig,
	})
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(dto.Title)
	if title == "" {
		title = strings.TrimSpace(preview.SuggestedName)
	}
	if title == "" {
		title = "Entorno solicitado"
	}

	subject := domain.EnvironmentRequestSubject{
		SubjectID:   strings.TrimSpace(dto.SubjectID),
		SubjectName: strings.TrimSpace(dto.SubjectName),
		RequestedBy: userID,
		AddedAt:     s.now(),
	}

	// Misma receta en revisión: se suma la materia en lugar de abrir otra ficha.
	if preview.Signature != "" {
		existing, err := s.repo.FindPendingBySignature(ctx, tenantID, preview.Signature)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.RequestedBy != userID {
			if subject.SubjectName != "" {
				if err := s.repo.AddSubject(ctx, existing.ID, subject); err != nil {
					return nil, err
				}
			}
			updated, err := s.repo.GetByID(ctx, tenantID, existing.ID)
			if err != nil {
				return nil, err
			}
			return &domain.EnvironmentRequestCreated{
				Request: updated,
				Joined:  true,
				Message: "Otro docente ya pidió este entorno. Sumamos tu materia a esa solicitud.",
			}, nil
		}
	}

	missing := append([]domain.EnvironmentRequestMissingComponent{}, dto.MissingComponents...)
	for _, item := range preview.MissingComponents {
		missing = append(missing, item)
	}

	request := &domain.EnvironmentRequest{
		TenantID:          tenantID,
		RequestedBy:       userID,
		Title:             title,
		Description:       reason,
		TeacherText:       strings.TrimSpace(dto.TeacherText),
		Uses:              uses,
		TargetEnvironment: preview.TargetEnvironment,
		StackConfig:       dto.StackConfig,
		Components:        previewComponentsToRequest(preview.Components),
		MissingComponents: dedupeMissingComponents(missing),
		Subjects:          []domain.EnvironmentRequestSubject{},
		StackSignature:    preview.Signature,
		Status:            domain.EnvironmentRequestEnRevision,
	}
	if subject.SubjectName != "" || subject.SubjectID != "" {
		request.Subjects = append(request.Subjects, subject)
	}

	created, err := s.repo.Create(ctx, request)
	if err != nil {
		return nil, err
	}

	message := "Recibimos tu solicitud. La administración la revisa y te avisamos en tus solicitudes."
	if preview.ExistingTemplate != nil {
		message = fmt.Sprintf("Ya existe el entorno «%s» con estas herramientas. Puedes usarlo ahora mismo mientras revisamos tu caso.", preview.ExistingTemplate.Name)
	}

	return &domain.EnvironmentRequestCreated{Request: created, Joined: false, Message: message}, nil
}

func (s *EnvironmentRequestService) ListMine(
	ctx context.Context, tenantID, userID string,
) ([]*domain.EnvironmentRequest, error) {
	return s.repo.ListMine(ctx, tenantID, userID)
}

func (s *EnvironmentRequestService) ListForReview(
	ctx context.Context, tenantID, status string,
) ([]*domain.EnvironmentRequest, error) {
	if status != "" {
		status = strings.ToUpper(strings.TrimSpace(status))
		switch status {
		case domain.EnvironmentRequestEnRevision, domain.EnvironmentRequestAprobada,
			domain.EnvironmentRequestNecesitaCambio, domain.EnvironmentRequestNoViable:
		default:
			return nil, fmt.Errorf("%w: estado desconocido %q", domain.ErrEnvironmentRequestInvalid, status)
		}
	}
	return s.repo.ListForReview(ctx, tenantID, status)
}

// Review registra la respuesta del administrador. Aprobar exige que la
// solicitud quede enlazada a una plantilla existente, porque el docente
// necesita algo que pueda usar, no un estado.
func (s *EnvironmentRequestService) Review(
	ctx context.Context, tenantID, requestID, reviewerID string, dto domain.ReviewEnvironmentRequestDTO,
) (*domain.EnvironmentRequest, error) {
	status := strings.ToUpper(strings.TrimSpace(dto.Status))
	switch status {
	case domain.EnvironmentRequestAprobada, domain.EnvironmentRequestNecesitaCambio, domain.EnvironmentRequestNoViable:
	default:
		return nil, fmt.Errorf("%w: estado de revisión inválido %q", domain.ErrEnvironmentRequestInvalid, dto.Status)
	}

	if status == domain.EnvironmentRequestAprobada && strings.TrimSpace(dto.ResolvedTemplateID) == "" {
		return nil, fmt.Errorf("%w: para aprobar hay que indicar la plantilla que la resuelve", domain.ErrEnvironmentRequestInvalid)
	}
	if status != domain.EnvironmentRequestAprobada {
		dto.ResolvedTemplateID = ""
	}
	if status == domain.EnvironmentRequestNecesitaCambio && strings.TrimSpace(dto.ReviewNote) == "" {
		return nil, fmt.Errorf("%w: explica qué falta para poder continuar", domain.ErrEnvironmentRequestInvalid)
	}

	dto.Status = status
	dto.ReviewNote = strings.TrimSpace(dto.ReviewNote)
	return s.repo.Review(ctx, tenantID, requestID, reviewerID, dto)
}

func (s *EnvironmentRequestService) compose(
	ctx context.Context, tenantID, env string, cfg domain.StackConfig,
) (*domain.ComposedTemplatePlan, error) {
	if s.composer == nil {
		return nil, nil
	}
	if strings.TrimSpace(cfg.RuntimeID) == "" && strings.TrimSpace(cfg.WebPlatformID) == "" {
		return nil, nil
	}
	plan, err := s.composer.Compose(ctx, tenantID, env, cfg)
	if err != nil {
		// Un docente puede nombrar una pieza que ya no está en el catálogo. Eso
		// no invalida el pedido: se guarda como faltante y el administrador
		// decide. Solo se propaga el error si no hay nada que guardar.
		return nil, nil
	}
	return plan, nil
}

func (s *EnvironmentRequestService) resolveComponents(
	ctx context.Context, tenantID string, cfg domain.StackConfig,
) ([]domain.EnvironmentRequestPreviewComponent, error) {
	if s.compRepo == nil {
		return []domain.EnvironmentRequestPreviewComponent{}, nil
	}
	ids := make([]string, 0, len(cfg.AddonIDs)+3)
	if cfg.RuntimeID != "" {
		ids = append(ids, cfg.RuntimeID)
	}
	if cfg.WebPlatformID != "" {
		ids = append(ids, cfg.WebPlatformID)
	}
	ids = append(ids, cfg.AddonIDs...)
	if cfg.DatabaseID != "" && cfg.DatabaseID != "none" {
		ids = append(ids, cfg.DatabaseID)
	}

	result := make([]domain.EnvironmentRequestPreviewComponent, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, idOrSlug := range ids {
		key := strings.ToLower(strings.TrimSpace(idOrSlug))
		if key == "" {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		component, err := s.compRepo.GetTemplateComponentByIDOrSlug(ctx, tenantID, idOrSlug)
		if err != nil || component == nil {
			result = append(result, domain.EnvironmentRequestPreviewComponent{
				Slug:    strings.TrimSpace(idOrSlug),
				Name:    strings.TrimSpace(idOrSlug),
				Layer:   "addon",
				Missing: true,
			})
			continue
		}
		result = append(result, domain.EnvironmentRequestPreviewComponent{
			ID:        component.ID,
			Slug:      component.Slug,
			Name:      component.Name,
			Layer:     string(component.Layer),
			IconKey:   component.IconKey,
			BaseImage: component.BaseImage,
		})
	}
	return result, nil
}

// environmentRequestSignature identifica la receta completa: imagen base, capas
// de build, servicios satélite y modalidad. No coincide necesariamente con el
// tag de la imagen porque dos entornos con la misma imagen pueden diferir en la
// base de datos.
func environmentRequestSignature(plan *domain.ComposedTemplatePlan, env string) string {
	if plan == nil {
		return ""
	}
	services := make([]string, 0, len(plan.ServicesConfig.Services))
	for _, svc := range plan.ServicesConfig.Services {
		services = append(services, strings.ToLower(svc.Category+":"+svc.Engine))
	}
	// Se ordenan para que la huella no dependa del orden de selección.
	for i := 0; i < len(services); i++ {
		for j := i + 1; j < len(services); j++ {
			if services[j] < services[i] {
				services[i], services[j] = services[j], services[i]
			}
		}
	}
	layers := []string{
		"sig:" + plan.SignatureHash,
		"env:" + env,
		"svc:" + strings.Join(services, ","),
	}
	return domain.ComputeStackSignature(plan.BaseImage, layers)
}

func previewComponentsToRequest(items []domain.EnvironmentRequestPreviewComponent) []domain.EnvironmentRequestComponent {
	result := make([]domain.EnvironmentRequestComponent, 0, len(items))
	for _, item := range items {
		if item.Missing {
			continue
		}
		result = append(result, domain.EnvironmentRequestComponent{
			ID:        item.ID,
			Slug:      item.Slug,
			Name:      item.Name,
			Layer:     item.Layer,
			IconKey:   item.IconKey,
			BaseImage: item.BaseImage,
		})
	}
	return result
}

func dedupeMissingComponents(items []domain.EnvironmentRequestMissingComponent) []domain.EnvironmentRequestMissingComponent {
	result := make([]domain.EnvironmentRequestMissingComponent, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		layer := strings.ToLower(strings.TrimSpace(item.Layer))
		if layer != "runtime" && layer != "addon" && layer != "database" {
			layer = "addon"
		}
		result = append(result, domain.EnvironmentRequestMissingComponent{
			Name:     name,
			Layer:    layer,
			Category: strings.TrimSpace(item.Category),
			Reason:   strings.TrimSpace(item.Reason),
		})
	}
	return result
}
