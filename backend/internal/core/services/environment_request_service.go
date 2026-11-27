package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
	repo           domain.EnvironmentRequestRepository
	compRepo       domain.TemplateComponentRepository
	composer       *TemplateComposerService
	templateLookup domain.EnvironmentRequestTemplateLookup
	now            func() time.Time
	// admission aporta la foto del nodo para armar capacity_impact en vivo.
	admission *PracticeEnvironmentService
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

// SetAdmission conecta el presupuesto del nodo para armar capacity_impact.
// Sin esta conexión los previews no traen el bloque humano de capacidad.
func (s *EnvironmentRequestService) SetAdmission(admission *PracticeEnvironmentService) *EnvironmentRequestService {
	s.admission = admission
	return s
}

// SetTemplateLookup conecta el catálogo de plantillas. Sin este puerto no se
// puede aprobar: una solicitud aprobada sin una plantilla válida no le sirve al
// docente.
func (s *EnvironmentRequestService) SetTemplateLookup(lookup domain.EnvironmentRequestTemplateLookup) *EnvironmentRequestService {
	s.templateLookup = lookup
	return s
}

// Preview arma la receta sin guardarla: es lo que el docente revisa antes de
// enviar. Devuelve la plantilla existente o la solicitud gemela si las hay.
func (s *EnvironmentRequestService) Preview(
	ctx context.Context, tenantID, userID string, dto domain.EnvironmentRequestPreviewDTO,
) (*domain.EnvironmentRequestPreview, error) {
	uses := domain.NormalizeEnvironmentUses(dto.Uses)
	env := domain.DeriveTargetEnvironment(uses)
	missingNames := normalizeMissingComponentNames(dto.MissingComponents)

	preview := &domain.EnvironmentRequestPreview{
		TargetEnvironment: env,
		Components:        []domain.EnvironmentRequestPreviewComponent{},
		MissingComponents: []domain.EnvironmentRequestMissingComponent{},
	}

	resolved, err := s.resolveComponents(ctx, tenantID, dto.StackConfig)
	if err != nil {
		return nil, err
	}
	preview.Components = resolved

	plan, err := s.compose(ctx, tenantID, env, dto.StackConfig)
	if err != nil {
		return nil, err
	}

	hasMissing := len(missingNames) > 0 || hasMissingPreviewComponent(resolved)
	if plan != nil {
		preview.SuggestedName = plan.SuggestedName
		preview.RecommendedRAMMB = plan.RecommendedRamMB
	}
	preview.Signature = environmentRequestSignature(plan, env, missingNames)

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

	// Solo cuando no faltan piezas fuera de catálogo y hay capas propias
	// (imagen construida) la referencia de imagen identifica la receta; con una
	// imagen pública el tag lo comparten entornos distintos.
	if !hasMissing && plan != nil && plan.NeedsLocalBuild {
		if ref, err := s.repo.FindApprovedTemplateByImage(ctx, tenantID, plan.ResolvedImage); err == nil && ref != nil {
			preview.ExistingTemplate = ref
		}
	}

	preview.CapacityImpact = s.capacityImpactForPreview(ctx, plan)

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
	cleanedMissing := dedupeMissingComponents(dto.MissingComponents)
	hasTools := hasStackTools(dto.StackConfig)
	if (!hasTools || len(cleanedMissing) > 0) && len([]rune(reason)) < 10 {
		return nil, fmt.Errorf("%w: cuéntanos qué deben lograr los estudiantes (al menos 10 caracteres)", domain.ErrEnvironmentRequestInvalid)
	}

	missingNames := make([]string, 0, len(cleanedMissing))
	for _, item := range cleanedMissing {
		missingNames = append(missingNames, item.Name)
	}

	uses := domain.NormalizeEnvironmentUses(dto.Uses)
	preview, err := s.Preview(ctx, tenantID, userID, domain.EnvironmentRequestPreviewDTO{
		Uses:              uses,
		StackConfig:       dto.StackConfig,
		MissingComponents: missingNames,
	})
	if err != nil {
		return nil, err
	}

	allMissing := append([]domain.EnvironmentRequestMissingComponent{}, cleanedMissing...)
	for _, item := range preview.MissingComponents {
		allMissing = append(allMissing, item)
	}
	dedupedMissing := dedupeMissingComponents(allMissing)
	hasMissing := len(dedupedMissing) > 0 || hasMissingPreviewComponent(preview.Components)
	if hasMissing && len([]rune(reason)) < 10 {
		return nil, fmt.Errorf("%w: cuéntanos qué deben lograr los estudiantes (al menos 10 caracteres)", domain.ErrEnvironmentRequestInvalid)
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
		MissingComponents: dedupedMissing,
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
	if !hasMissing && preview.ExistingTemplate != nil {
		message = fmt.Sprintf("Ya existe el entorno «%s» con estas herramientas. Puedes usarlo ahora mismo mientras revisamos tu caso.", preview.ExistingTemplate.Name)
	}

	return &domain.EnvironmentRequestCreated{Request: created, Joined: false, Message: message}, nil
}

func (s *EnvironmentRequestService) ListMine(
	ctx context.Context, tenantID, userID string,
) ([]*domain.EnvironmentRequest, error) {
	return s.repo.ListMine(ctx, tenantID, userID)
}

// GetForAdmin devuelve la ficha completa de una solicitud para el panel de
// administración: es la misma foto que el listado, con stack_config, piezas,
// faltantes y materias, y el Constructor la usa para precargar el Paso 2.
//
// La consulta va acotada al tenant de la sesión, así que un identificador de
// otra institución se comporta igual que uno inexistente.
func (s *EnvironmentRequestService) GetForAdmin(
	ctx context.Context, tenantID, requestID string,
) (*domain.EnvironmentRequest, error) {
	tenant := strings.TrimSpace(tenantID)
	if tenant == "" {
		return nil, fmt.Errorf("%w: falta la institución de la solicitud", domain.ErrEnvironmentRequestInvalid)
	}
	id := strings.TrimSpace(requestID)
	if id == "" {
		return nil, fmt.Errorf("%w: falta el identificador de la solicitud", domain.ErrEnvironmentRequestInvalid)
	}

	request, err := s.repo.GetByID(ctx, tenant, id)
	if err != nil {
		if errors.Is(err, domain.ErrEnvironmentRequestNotFound) {
			return nil, domain.ErrEnvironmentRequestNotFound
		}
		return nil, err
	}
	if request == nil {
		return nil, domain.ErrEnvironmentRequestNotFound
	}
	request.CapacityImpact = s.capacityImpactForRequest(ctx, request)
	return request, nil
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
// solicitud quede enlazada a una plantilla existente, aprobada, de la misma
// institución y que cubra la modalidad pedida, porque el docente necesita algo
// que pueda usar, no un estado.
func (s *EnvironmentRequestService) Review(
	ctx context.Context, tenantID, requestID, reviewerID string, dto domain.ReviewEnvironmentRequestDTO,
) (*domain.EnvironmentRequest, error) {
	status := strings.ToUpper(strings.TrimSpace(dto.Status))
	switch status {
	case domain.EnvironmentRequestAprobada, domain.EnvironmentRequestNecesitaCambio, domain.EnvironmentRequestNoViable:
	default:
		return nil, fmt.Errorf("%w: estado de revisión inválido %q", domain.ErrEnvironmentRequestInvalid, dto.Status)
	}

	dto.Status = status
	dto.ReviewNote = strings.TrimSpace(dto.ReviewNote)

	if status == domain.EnvironmentRequestAprobada && strings.TrimSpace(dto.ResolvedTemplateID) == "" {
		return nil, fmt.Errorf("%w: para aprobar hay que indicar la plantilla que la resuelve", domain.ErrEnvironmentRequestInvalid)
	}
	if (status == domain.EnvironmentRequestNecesitaCambio || status == domain.EnvironmentRequestNoViable) && dto.ReviewNote == "" {
		return nil, fmt.Errorf("%w: explica qué falta o por qué no es viable", domain.ErrEnvironmentRequestInvalid)
	}

	existing, err := s.repo.GetByID(ctx, tenantID, requestID)
	if err != nil {
		return nil, err
	}

	if status == domain.EnvironmentRequestAprobada {
		template, err := s.resolveApprovedTemplate(ctx, tenantID, strings.TrimSpace(dto.ResolvedTemplateID), existing.TargetEnvironment)
		if err != nil {
			return nil, err
		}
		dto.ResolvedTemplateID = template.ID
		if dto.ReviewNote == "" {
			dto.ReviewNote = fmt.Sprintf("Tu solicitud quedó resuelta con el entorno «%s».", template.Name)
		}
	} else {
		dto.ResolvedTemplateID = ""
	}

	return s.repo.Review(ctx, tenantID, requestID, reviewerID, dto)
}

func (s *EnvironmentRequestService) resolveApprovedTemplate(
	ctx context.Context, tenantID, templateID, requiredEnvironment string,
) (*domain.AdminTemplateReviewItem, error) {
	if s.templateLookup == nil {
		return nil, fmt.Errorf("%w: no se puede validar la plantilla indicada", domain.ErrEnvironmentRequestInvalid)
	}
	template, err := s.templateLookup.GetTemplateByID(ctx, templateID)
	if err != nil || template == nil {
		return nil, fmt.Errorf("%w: la plantilla indicada no existe", domain.ErrEnvironmentRequestInvalid)
	}
	if template.TenantID != nil && strings.TrimSpace(*template.TenantID) != "" && *template.TenantID != tenantID {
		return nil, fmt.Errorf("%w: la plantilla indicada no pertenece a la institución", domain.ErrEnvironmentRequestInvalid)
	}
	if !isApprovedTemplateStatus(template.Status) {
		return nil, fmt.Errorf("%w: la plantilla indicada no está aprobada", domain.ErrEnvironmentRequestInvalid)
	}
	if !domain.TemplateSupportsEnvironment(template.TargetEnvironment, requiredEnvironment) {
		return nil, fmt.Errorf("%w: la plantilla no cubre la modalidad solicitada", domain.ErrEnvironmentRequestInvalid)
	}
	return template, nil
}

func isApprovedTemplateStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved", "aprobada":
		return true
	default:
		return false
	}
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
		// La incompatibilidad real de piezas sí invalida el pedido: el docente
		// eligió un complemento que no aplica al runtime activo y debe corregirlo.
		if errors.Is(err, domain.ErrIncompatibleAddon) {
			return nil, err
		}
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
// de build, servicios satélite, modalidad y piezas faltantes declaradas. No
// coincide necesariamente con el tag de la imagen porque dos entornos con la
// misma imagen pueden diferir en la base de datos o en piezas fuera de catálogo.
func environmentRequestSignature(plan *domain.ComposedTemplatePlan, env string, missingNames []string) string {
	if plan == nil && len(missingNames) == 0 {
		return ""
	}
	if plan == nil {
		layers := []string{
			"env:" + env,
			"missing:" + strings.Join(missingNames, ","),
		}
		return domain.ComputeStackSignature("", layers)
	}
	services := make([]string, 0, len(plan.ServicesConfig.Services))
	for _, svc := range plan.ServicesConfig.Services {
		services = append(services, strings.ToLower(svc.Category+":"+svc.Engine))
	}
	sort.Strings(services)
	layers := []string{
		"sig:" + plan.SignatureHash,
		"env:" + env,
		"svc:" + strings.Join(services, ","),
	}
	if len(missingNames) > 0 {
		layers = append(layers, "missing:"+strings.Join(missingNames, ","))
	}
	return domain.ComputeStackSignature(plan.BaseImage, layers)
}

func hasStackTools(cfg domain.StackConfig) bool {
	if strings.TrimSpace(cfg.RuntimeID) != "" || strings.TrimSpace(cfg.WebPlatformID) != "" {
		return true
	}
	for _, id := range cfg.AddonIDs {
		if strings.TrimSpace(id) != "" {
			return true
		}
	}
	db := strings.TrimSpace(cfg.DatabaseID)
	if db != "" && db != "none" && db != "db-none" {
		return true
	}
	for _, svc := range cfg.Services {
		id := strings.TrimSpace(svc.ComponentID)
		if id != "" && id != "none" && id != "db-none" {
			return true
		}
	}
	return false
}

func hasMissingPreviewComponent(items []domain.EnvironmentRequestPreviewComponent) bool {
	for _, item := range items {
		if item.Missing {
			return true
		}
	}
	return false
}

func normalizeMissingComponentNames(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, raw := range items {
		clean := strings.ToLower(strings.TrimSpace(raw))
		if clean == "" {
			continue
		}
		if _, dup := seen[clean]; dup {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	sort.Strings(result)
	return result
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

// capacityImpactForPreview arma el bloque humano desde la RAM recomendada de
// la receta. Sin conexión al nodo o sin receta armada no hay bloque.
func (s *EnvironmentRequestService) capacityImpactForPreview(ctx context.Context, plan *domain.ComposedTemplatePlan) *domain.CapacityImpact {
	if s == nil || s.admission == nil || plan == nil || plan.RecommendedRamMB <= 0 {
		return nil
	}
	return s.admission.capacityImpact(ctx, nil, int64(plan.RecommendedRamMB))
}

// capacityImpactForRequest arma el bloque humano de la ficha del administrador.
// Con plantilla resuelta usa su RAM base; en revisión compone la receta para
// obtener la recomendación del Constructor. Sin RAM no hay bloque.
func (s *EnvironmentRequestService) capacityImpactForRequest(ctx context.Context, request *domain.EnvironmentRequest) *domain.CapacityImpact {
	if s == nil || s.admission == nil || request == nil {
		return nil
	}
	ramMB := int64(0)
	if strings.TrimSpace(request.ResolvedTemplateID) != "" && s.templateLookup != nil {
		if tpl, err := s.templateLookup.GetTemplateByID(ctx, strings.TrimSpace(request.ResolvedTemplateID)); err == nil && tpl != nil && tpl.BaseRamMB > 0 {
			ramMB = int64(tpl.BaseRamMB)
		}
	}
	if ramMB <= 0 {
		plan, err := s.compose(ctx, request.TenantID, request.TargetEnvironment, request.StackConfig)
		if err == nil && plan != nil && plan.RecommendedRamMB > 0 {
			ramMB = int64(plan.RecommendedRamMB)
		}
	}
	if ramMB <= 0 {
		return nil
	}
	return s.admission.capacityImpact(ctx, nil, ramMB)
}
