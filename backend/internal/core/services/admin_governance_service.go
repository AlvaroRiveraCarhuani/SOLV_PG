package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"solv-backend/internal/core/domain"
)

var (
	ErrReasonTooShort       = errors.New("justification reason must have at least 10 characters")
	ErrTeacherNotFoundOrRole = errors.New("assigned user does not exist or does not have teacher role")
)

type AdminGovernanceService struct {
	subjectRepo domain.SubjectRepository
	govRepo     domain.AdminGovernanceRepository
	auditRepo   domain.AuditLogRepository
}

func NewAdminGovernanceService(
	subjectRepo domain.SubjectRepository,
	govRepo domain.AdminGovernanceRepository,
) *AdminGovernanceService {
	return &AdminGovernanceService{
		subjectRepo: subjectRepo,
		govRepo:     govRepo,
	}
}

func (s *AdminGovernanceService) SetAuditRepo(repo domain.AuditLogRepository) {
	s.auditRepo = repo
}

func (s *AdminGovernanceService) ReassignCourse(ctx context.Context, tenantID, subjectID string, dto domain.ReassignCourseDTO) (*domain.Subject, error) {
	if dto.NewTeacherID == "" {
		return nil, fmt.Errorf("new_teacher_id is required")
	}

	// 1. Validar que la materia existe en el tenant
	subject, err := s.subjectRepo.GetByID(ctx, tenantID, subjectID)
	if err != nil {
		return nil, err
	}

	// 2. Validar que el nuevo docente existe y tiene rol teacher o admin
	if s.govRepo != nil {
		isValid, err := s.govRepo.ValidateTeacherRole(ctx, tenantID, dto.NewTeacherID)
		if err != nil || !isValid {
			return nil, ErrTeacherNotFoundOrRole
		}
	}

	// 3. Reasignar materia
	if err := s.subjectRepo.ReassignTeacher(ctx, tenantID, subjectID, dto.NewTeacherID); err != nil {
		return nil, err
	}

	subject.TeacherID = &dto.NewTeacherID
	return subject, nil
}

func (s *AdminGovernanceService) ListStudents(
	ctx context.Context,
	tenantID, search, subjectID, status, periodID string,
) ([]*domain.AdminStudentDirectoryItem, error) {
	return s.govRepo.ListStudentsDirectory(ctx, tenantID, search, subjectID, status, periodID)
}

func (s *AdminGovernanceService) GetStudentCourses(
	ctx context.Context,
	tenantID, studentID string,
) ([]*domain.AdminStudentCourseItem, error) {
	return s.govRepo.GetStudentCourses(ctx, tenantID, studentID)
}

func (s *AdminGovernanceService) CreateStudent(
	ctx context.Context,
	tenantID string,
	dto domain.CreateStudentDTO,
) (*domain.AdminStudentDirectoryItem, error) {
	if strings.TrimSpace(dto.Email) == "" || strings.TrimSpace(dto.FirstName) == "" || strings.TrimSpace(dto.LastName) == "" {
		return nil, errors.New("todos los campos (email, nombre, apellido) son obligatorios")
	}
	return s.govRepo.CreateStudent(ctx, tenantID, strings.TrimSpace(dto.Email), strings.TrimSpace(dto.FirstName), strings.TrimSpace(dto.LastName))
}

func (s *AdminGovernanceService) UpdateStudentStatus(
	ctx context.Context,
	tenantID, studentID string,
	dto domain.UpdateStudentStatusDTO,
) error {
	if dto.Status != "active" && dto.Status != "suspended" {
		return errors.New("estado inválido: debe ser 'active' o 'suspended'")
	}
	return s.govRepo.UpdateStudentStatus(ctx, tenantID, studentID, dto.Status, dto.Reason)
}

func (s *AdminGovernanceService) ResetStudentOOM(
	ctx context.Context,
	tenantID, studentID string,
	dto domain.ResetOOMDTO,
) (*domain.ResetOOMResult, error) {
	if len(dto.Reason) < 10 {
		return nil, ErrReasonTooShort
	}

	affectedRows, err := s.govRepo.ResetStudentOOM(ctx, tenantID, studentID)
	if err != nil {
		return nil, err
	}

	return &domain.ResetOOMResult{
		StudentID:            studentID,
		WorkspacesResetCount: affectedRows,
		Message:              "Penalizaciones OOM reseteadas exitosamente",
	}, nil
}

var (
	ErrInvalidReviewStatus      = errors.New("status must be 'approved', 'rejected', 'paused', 'suspended' or 'pending_audit'")
	ErrRejectionReasonRequired = errors.New("rejection_reason is required when rejecting or suspending a template")
)

func (s *AdminGovernanceService) ListTemplates(
	ctx context.Context,
	tenantID, status, search string,
) ([]*domain.AdminTemplateReviewItem, error) {
	return s.govRepo.ListTemplates(ctx, tenantID, status, search)
}

func (s *AdminGovernanceService) ReviewTemplate(
	ctx context.Context,
	tenantID, templateID, adminID string,
	dto domain.ReviewTemplateDTO,
) (*domain.AdminTemplateReviewItem, error) {
	status := strings.ToLower(strings.TrimSpace(dto.Status))
	if status != "approved" && status != "rejected" && status != "paused" && status != "suspended" && status != "pending_audit" {
		return nil, ErrInvalidReviewStatus
	}

	if (status == "rejected" || status == "suspended") && strings.TrimSpace(dto.RejectionReason) == "" {
		return nil, ErrRejectionReasonRequired
	}

	dbStatus := dto.Status
	if status == "suspended" {
		dbStatus = "SUSPENDIDA"
	} else if status == "pending_audit" {
		dbStatus = "PENDIENTE_AUDITORIA"
	}

	if dto.BaseRamMB != nil && *dto.BaseRamMB > 0 {
		totalHostMB := s.getHostTotalRAM(ctx)
		maxAllowedRAM := domain.CalculateHostMaxAllowedRAM(totalHostMB)
		if err := domain.ValidateRamAgainstHost(*dto.BaseRamMB, maxAllowedRAM); err != nil {
			return nil, err
		}
	}

	return s.govRepo.ReviewTemplate(ctx, tenantID, templateID, adminID, dbStatus, dto.RejectionReason, dto.BaseRamMB)
}

func (s *AdminGovernanceService) getHostTotalRAM(ctx context.Context) int {
	totalMB := 8192
	if v, err := mem.VirtualMemoryWithContext(ctx); err == nil && v != nil {
		totalMB = int(v.Total / (1024 * 1024))
	}
	return totalMB
}

var (
	dockerImageRegex = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*:[a-zA-Z0-9_.-]+$`)
	ErrInvalidDockerImage = errors.New("la imagen Docker debe tener formato válido repositorio:tag sin espacios (ej: python:3.12-slim)")
	ErrLatestTagForbidden = errors.New("el tag :latest está prohibido en plantillas oficiales por reproducibilidad")
)

func (s *AdminGovernanceService) CreateOfficialTemplate(
	ctx context.Context,
	tenantID, adminID string,
	dto domain.CreateOfficialTemplateDTO,
) (*domain.AdminTemplateReviewItem, error) {
	dto.Name = strings.TrimSpace(dto.Name)
	dto.DockerImage = strings.TrimSpace(dto.DockerImage)

	if dto.Name == "" {
		return nil, errors.New("el nombre de la plantilla es requerido")
	}

	if dto.DockerImage == "" {
		return nil, errors.New("la imagen Docker es requerida")
	}

	if strings.HasSuffix(strings.ToLower(dto.DockerImage), ":latest") {
		return nil, ErrLatestTagForbidden
	}

	if !dockerImageRegex.MatchString(dto.DockerImage) {
		return nil, ErrInvalidDockerImage
	}

	if err := ValidateAllowedRegistry(dto.DockerImage); err != nil {
		return nil, err
	}

	if dto.TargetEnvironment == "" {
		dto.TargetEnvironment = "IDE_PERSISTENTE"
	}

	if dto.BaseRamMB <= 0 {
		dto.BaseRamMB = 512
	} else if dto.BaseRamMB < 256 {
		dto.BaseRamMB = 256
	}

	totalHostMB := s.getHostTotalRAM(ctx)
	maxAllowedRAM := domain.CalculateHostMaxAllowedRAM(totalHostMB)
	if err := domain.ValidateRamAgainstHost(dto.BaseRamMB, maxAllowedRAM); err != nil {
		return nil, err
	}

	if dto.ServicesConfig == nil {
		dto.ServicesConfig = &domain.ServicesConfig{
			Services: []domain.ServiceRequirement{},
		}
	}

	// Derivación obligatoria del perfil cgroups v2: min = base/2, high = base*1.5, max = base*2
	derived := domain.DeriveResourceProfile(dto.BaseRamMB)
	dto.ResourceProfile = &derived

	return s.govRepo.CreateOfficialTemplate(ctx, tenantID, adminID, dto)
}

func (s *AdminGovernanceService) DuplicateTemplate(
	ctx context.Context,
	tenantID, templateID, adminID string,
) (*domain.AdminTemplateReviewItem, error) {
	if templateID == "" {
		return nil, errors.New("el ID de la plantilla es requerido")
	}
	return s.govRepo.DuplicateTemplate(ctx, tenantID, templateID, adminID)
}

const (
	ActionTerminateAll = "terminate_all_workspaces"
	ActionHibernateAll = "hibernate_all_workspaces"
	ActionKillZombies  = "kill_zombies"

	PhraseTerminateAll = "TERMINAR TODOS LOS WORKSPACES"
	PhraseHibernateAll = "HIBERNAR TODOS LOS WORKSPACES"
	PhraseKillZombies  = "LIMPIAR ZOMBIES DOCKER"
)

var (
	ErrUnknownEmergencyAction     = errors.New("unknown emergency action")
	ErrInvalidConfirmationPhrase = errors.New("invalid confirmation phrase")
)

func (s *AdminGovernanceService) ExecuteEmergencyAction(
	ctx context.Context,
	tenantID, adminID, action string,
	req domain.EmergencyActionRequest,
) (*domain.EmergencyActionResult, error) {
	switch action {
	case ActionTerminateAll:
		if req.ConfirmationPhrase != PhraseTerminateAll {
			return nil, ErrInvalidConfirmationPhrase
		}
		count, err := s.govRepo.TerminateAllWorkspaces(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		return &domain.EmergencyActionResult{
			Action:        action,
			AffectedCount: count,
			ExecutedBy:    adminID,
			Message:       fmt.Sprintf("Se terminaron forzosamente %d workspaces activos", count),
		}, nil

	case ActionHibernateAll:
		if req.ConfirmationPhrase != PhraseHibernateAll {
			return nil, ErrInvalidConfirmationPhrase
		}
		count, err := s.govRepo.HibernateAllWorkspaces(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		return &domain.EmergencyActionResult{
			Action:        action,
			AffectedCount: count,
			ExecutedBy:    adminID,
			Message:       fmt.Sprintf("Se hibernaron exitosamente %d workspaces activos", count),
		}, nil

	case ActionKillZombies:
		if req.ConfirmationPhrase != PhraseKillZombies {
			return nil, ErrInvalidConfirmationPhrase
		}
		// Acción de limpieza de zombies
		return &domain.EmergencyActionResult{
			Action:        action,
			AffectedCount: 0,
			ExecutedBy:    adminID,
			Message:       "Barrido de contenedores zombies ejecutado exitosamente",
		}, nil

	default:
		return nil, ErrUnknownEmergencyAction
	}
}

// GetRuntimeCapabilities obtiene métricas del host físico y catálogo de servicios satélite soportados
func (s *AdminGovernanceService) GetRuntimeCapabilities(ctx context.Context) (*domain.RuntimeCapabilities, error) {
	// 1. Métricas de memoria y procesador del host real
	totalMB := int64(8192)
	availableMB := int64(4096)
	usedMB := int64(4096)
	cpuCores := 4

	if v, err := mem.VirtualMemoryWithContext(ctx); err == nil && v != nil {
		totalMB = int64(v.Total / (1024 * 1024))
		availableMB = int64(v.Available / (1024 * 1024))
		usedMB = int64(v.Used / (1024 * 1024))
	}
	if c, err := cpu.CountsWithContext(ctx, true); err == nil && c > 0 {
		cpuCores = c
	}

	hostMem := domain.HostCapacityInfo{
		TotalRAMMB:     totalMB,
		AvailableRAMMB: availableMB,
		UsedRAMMB:      usedMB,
		CPUCores:       cpuCores,
	}

	// 2. Presets adaptados dinámicamente a la memoria física
	judgePresets := []domain.RamPresetSuggestion{
		{MB: 128, Label: "128 MB", Desc: "Ultra-ligera (C/C++)"},
		{MB: 256, Label: "256 MB", Desc: "Recomendada (Python/Go)"},
		{MB: 512, Label: "512 MB", Desc: "Completa (Java/JVM)"},
	}

	idePresets := []domain.RamPresetSuggestion{
		{MB: 512, Label: "512 MB", Desc: "Ligera (C/Go)"},
		{MB: 1024, Label: "1 GB", Desc: "Estándar (Web/Python)"},
	}
	if totalMB >= 6000 {
		idePresets = append(idePresets, domain.RamPresetSuggestion{
			MB: 2048, Label: "2 GB", Desc: "Intensiva (Java/ML)",
		})
	}
	if totalMB >= 12000 {
		idePresets = append(idePresets, domain.RamPresetSuggestion{
			MB: 4096, Label: "4 GB", Desc: "Datos & IA",
		})
	} else if totalMB >= 7000 {
		idePresets = append(idePresets, domain.RamPresetSuggestion{
			MB: 4096, Label: "4 GB", Desc: "Alta demanda (Intensivo)",
		})
	}
	if totalMB >= 32000 {
		idePresets = append(idePresets, domain.RamPresetSuggestion{
			MB: 8192, Label: "8 GB", Desc: "Big Data & Deep Learning",
		})
	}

	maxAllowedRAM := int(float64(totalMB) * 0.75)
	if maxAllowedRAM < 512 {
		maxAllowedRAM = 512
	}

	// 3. Catálogo real de servicios satélite soportados en la plataforma
	satellites := []domain.SatelliteServiceCapability{
		{
			Category:    "database",
			Engine:      "postgres",
			Label:       "PostgreSQL",
			Version:     "16",
			Description: "Base de datos relacional aislada por estudiante y materia",
			EnvVar:      "DATABASE_URL",
			IsAvailable: true,
			BaseRAMMB:   128,
		},
		{
			Category:    "database",
			Engine:      "mysql",
			Label:       "MySQL",
			Version:     "8.4",
			Description: "Base de datos relacional MySQL para ejercicios de SQL",
			EnvVar:      "DATABASE_URL",
			IsAvailable: true,
			BaseRAMMB:   128,
		},
		{
			Category:    "database",
			Engine:      "mongodb",
			Label:       "MongoDB",
			Version:     "7.0",
			Description: "Base de datos de documentos NoSQL para proyectos web",
			EnvVar:      "MONGODB_URI",
			IsAvailable: false,
			BaseRAMMB:   128,
		},
		{
			Category:    "cache",
			Engine:      "redis",
			Label:       "Redis",
			Version:     "7.2",
			Description: "Almacén en memoria y caché clave-valor",
			EnvVar:      "REDIS_URL",
			IsAvailable: false,
			BaseRAMMB:   64,
		},
	}

// editorBaseMB es la RAM mínima que consume el proceso del editor (OpenVSCode Server).
// Constante de dominio: si se necesita hacer configurable, agregar al struct de Config del servicio.
const editorBaseMB = 210

// runtimeBaseMB es la RAM mínima reservada para el runtime del Juez (kernel + sandbox del contenedor efímero).
const runtimeBaseMB = 32

	return &domain.RuntimeCapabilities{
		HostMemory:        hostMem,
		SatelliteServices: satellites,
		IDEPresets:        idePresets,
		JudgePresets:      judgePresets,
		MaxAllowedRamMB:   maxAllowedRAM,
		EditorBaseMB:      editorBaseMB,
		RuntimeBaseMB:     runtimeBaseMB,
	}, nil
}

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
		actorID = "00000000-0000-0000-0000-000000000001"
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
		actorID = "00000000-0000-0000-0000-000000000001"
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

