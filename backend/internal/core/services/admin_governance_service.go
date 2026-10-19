package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"solv-backend/internal/core/domain"
)

var (
	ErrReasonTooShort               = errors.New("justification reason must have at least 10 characters")
	ErrTeacherNotFoundOrRole        = errors.New("assigned user does not exist or does not have teacher role")
	ErrEmergencyExecutorUnavailable = errors.New("emergency executor not configured in this deployment")
)

type AdminGovernanceService struct {
	subjectRepo domain.SubjectRepository
	govRepo     domain.AdminGovernanceRepository
	auditRepo   domain.AuditLogRepository
	// dockerPruner y poolResetter son puntos de extensión inyectados por
	// main.go: mantienen el servicio desacoplado del daemon Docker y del
	// pool de conexiones (ADR-032, acciones docker-prune y reset-pools).
	dockerPruner func(ctx context.Context) (containersPruned int64, spaceReclaimedMB int64, err error)
	poolResetter func(ctx context.Context) (connectionsRecycled int64, err error)
	cacheService *TemplateCacheService
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

func (s *AdminGovernanceService) SetCacheService(cache *TemplateCacheService) *AdminGovernanceService {
	s.cacheService = cache
	return s
}

func (s *AdminGovernanceService) SetAuditRepo(repo domain.AuditLogRepository) {
	s.auditRepo = repo
}

// SetDockerPruner inyecta el executor de poda de emergencia (ADR-032,
// docker-prune): poda selectiva de contenedores detenidos gestionados por
// SOLV e imágenes huérfanas, sin tocar volúmenes nombrados de estudiantes.
func (s *AdminGovernanceService) SetDockerPruner(
	pruner func(ctx context.Context) (containersPruned int64, spaceReclaimedMB int64, err error),
) {
	s.dockerPruner = pruner
}

// SetPoolResetter inyecta el executor de reinicio de pools (ADR-032,
// reset-pools): recicla conexiones inactivas del pool de PostgreSQL y
// reinicializa contadores de circuitos en memoria si existen.
func (s *AdminGovernanceService) SetPoolResetter(
	resetter func(ctx context.Context) (connectionsRecycled int64, err error),
) {
	s.poolResetter = resetter
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
	ErrInvalidReviewStatus     = errors.New("status must be 'approved', 'rejected', 'paused', 'suspended' or 'pending_audit'")
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
	switch status {
	case "suspended":
		dbStatus = "SUSPENDIDA"
	case "pending_audit":
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
	dockerImageRegex      = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*:[a-zA-Z0-9_.-]+$`)
	ErrInvalidDockerImage = errors.New("la imagen Docker debe tener formato válido repositorio:tag sin espacios (ej: python:3.12-slim)")
	ErrLatestTagForbidden = errors.New("el tag :latest está prohibido en plantillas oficiales por reproducibilidad")
	ErrRamBelowVitalFloor = errors.New("la memoria asignada es inferior al piso vital requerido por el entorno")
)

func calculateMinimumVitalFloorMB(targetEnvironment string, servicesCfg *domain.ServicesConfig) int {
	env := strings.TrimSpace(targetEnvironment)
	if env == "JUEZ_EFIMERO" || env == "JUEZ_VIRTUAL" {
		return 64
	}
	floorMB := 210
	if servicesCfg != nil {
		for _, svc := range servicesCfg.Services {
			switch strings.ToLower(strings.TrimSpace(svc.Engine)) {
			case "postgres":
				floorMB += 50
			case "mysql", "mongodb":
				floorMB += 128
			case "redis":
				floorMB += 64
			}
		}
	}
	return floorMB
}

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

	if dto.ServicesConfig == nil {
		dto.ServicesConfig = &domain.ServicesConfig{
			Services: []domain.ServiceRequirement{},
		}
	}

	if dto.BaseRamMB <= 0 {
		dto.BaseRamMB = 512
	} else {
		minFloorMB := calculateMinimumVitalFloorMB(dto.TargetEnvironment, dto.ServicesConfig)
		if dto.BaseRamMB < minFloorMB {
			return nil, fmt.Errorf("%w: base_ram_mb=%d es inferior al piso vital requerido (%d MB)", ErrRamBelowVitalFloor, dto.BaseRamMB, minFloorMB)
		}
	}

	totalHostMB := s.getHostTotalRAM(ctx)
	maxAllowedRAM := domain.CalculateHostMaxAllowedRAM(totalHostMB)
	if err := domain.ValidateRamAgainstHost(dto.BaseRamMB, maxAllowedRAM); err != nil {
		return nil, err
	}

	// Derivación obligatoria del perfil cgroups v2: min = base/2, high = base*1.5, max = base*2
	derived := domain.DeriveResourceProfile(dto.BaseRamMB)
	dto.ResourceProfile = &derived
	created, err := s.govRepo.CreateOfficialTemplate(ctx, tenantID, adminID, dto)
	if err != nil {
		return nil, err
	}

	// Golden Image Invariant (Solución 4): Precalentamiento asíncrono en background
	if s.cacheService != nil && created != nil {
		go func(tid, tmplID, img string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			log.Printf("[GoldenImage] Pre-warming image %q for official template %s...", img, tmplID)
			_, _ = s.cacheService.WarmTemplate(bgCtx, tid, tmplID)
		}(tenantID, created.ID, created.DockerImage)
	}

	return created, nil
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
	ActionDockerPrune  = "docker_prune"
	ActionResetPools   = "reset_pools"

	PhraseTerminateAll = "TERMINAR TODOS LOS WORKSPACES"
	PhraseHibernateAll = "HIBERNAR TODOS LOS WORKSPACES"
	PhraseKillZombies  = "LIMPIAR ZOMBIES DOCKER"
	PhraseDockerPrune  = "PURGAR CAPAS HUERFANAS"
	PhraseResetPools   = "REINICIAR POOLS"
)

// emergencyConfirmationPhrase centraliza el contrato de doble confirmación
// (ADR-032): cada acción exige la frase exacta tipeada por el administrador.
func emergencyConfirmationPhrase(action string) (string, bool) {
	switch action {
	case ActionTerminateAll:
		return PhraseTerminateAll, true
	case ActionHibernateAll:
		return PhraseHibernateAll, true
	case ActionKillZombies:
		return PhraseKillZombies, true
	case ActionDockerPrune:
		return PhraseDockerPrune, true
	case ActionResetPools:
		return PhraseResetPools, true
	}
	return "", false
}

// emergencyAuditActions mapea cada acción a su evento de auditoría EMERGENCY_*
// (registros filtrables por el índice parcial de ADR-032).
var emergencyAuditActions = map[string]string{
	ActionTerminateAll: "EMERGENCY_TERMINATE_ALL",
	ActionHibernateAll: "EMERGENCY_HIBERNATE_ALL",
	ActionKillZombies:  "EMERGENCY_KILL_ZOMBIES",
	ActionDockerPrune:  "EMERGENCY_PRUNE_DOCKER",
	ActionResetPools:   "EMERGENCY_RESET_POOLS",
}

var (
	ErrUnknownEmergencyAction    = errors.New("unknown emergency action")
	ErrInvalidConfirmationPhrase = errors.New("invalid confirmation phrase")
)

func (s *AdminGovernanceService) ExecuteEmergencyAction(
	ctx context.Context,
	tenantID, adminID, action string,
	req domain.EmergencyActionRequest,
) (*domain.EmergencyActionResult, error) {
	phrase, known := emergencyConfirmationPhrase(action)
	if !known {
		return nil, ErrUnknownEmergencyAction
	}
	if req.ConfirmationPhrase != phrase {
		return nil, ErrInvalidConfirmationPhrase
	}

	var result *domain.EmergencyActionResult
	var err error

	switch action {
	case ActionTerminateAll:
		var count int64
		count, err = s.govRepo.TerminateAllWorkspaces(ctx, tenantID)
		if err == nil {
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: count,
				ExecutedBy:    adminID,
				Message:       fmt.Sprintf("Se terminaron forzosamente %d workspaces activos", count),
			}
		}

	case ActionHibernateAll:
		var count int64
		count, err = s.govRepo.HibernateAllWorkspaces(ctx, tenantID)
		if err == nil {
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: count,
				ExecutedBy:    adminID,
				Message:       fmt.Sprintf("Se hibernaron exitosamente %d workspaces activos", count),
			}
		}

	case ActionKillZombies:
		if s.dockerPruner != nil {
			var pruned, _ int64
			pruned, _, err = s.dockerPruner(ctx)
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: pruned,
				ExecutedBy:    adminID,
				Message:       fmt.Sprintf("Barrido de contenedores zombies completado: %d contenedores eliminados", pruned),
			}
		} else {
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: 0,
				ExecutedBy:    adminID,
				Message:       "Barrido de contenedores zombies ejecutado exitosamente",
			}
		}

	case ActionDockerPrune:
		if s.dockerPruner == nil {
			return nil, ErrEmergencyExecutorUnavailable
		}
		var pruned, spaceMB int64
		pruned, spaceMB, err = s.dockerPruner(ctx)
		if err == nil {
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: pruned,
				ExecutedBy:    adminID,
				Message:       fmt.Sprintf("Poda completada: %d contenedores e imágenes huérfanos eliminados (%d MB liberados)", pruned, spaceMB),
			}
		}

	case ActionResetPools:
		if s.poolResetter == nil {
			return nil, ErrEmergencyExecutorUnavailable
		}
		var recycled int64
		recycled, err = s.poolResetter(ctx)
		if err == nil {
			result = &domain.EmergencyActionResult{
				Action:        action,
				AffectedCount: recycled,
				ExecutedBy:    adminID,
				Message:       fmt.Sprintf("Pool de conexiones reiniciado: %d conexiones recicladas y circuit breakers inicializados", recycled),
			}
		}
	}

	if err != nil {
		return nil, err
	}

	// Trazabilidad obligatoria (ADR-032): toda ejecución de emergencia queda
	// registrada en audit_logs con evento EMERGENCY_* y el motivo ingresado.
	s.recordEmergencyAudit(ctx, tenantID, adminID, action, req, result)

	return result, nil
}

// recordEmergencyAudit persiste el evento EMERGENCY_* correspondiente. Un
// fallo de auditoría no revierte la acción ejecutada (el efecto operativo ya
// ocurrió), pero se reporta en el log del sistema.
func (s *AdminGovernanceService) recordEmergencyAudit(
	ctx context.Context,
	tenantID, adminID, action string,
	req domain.EmergencyActionRequest,
	result *domain.EmergencyActionResult,
) {
	if s.auditRepo == nil || result == nil {
		return
	}
	meta, _ := json.Marshal(map[string]any{
		"reason":         req.Reason,
		"affected_count": result.AffectedCount,
		"message":        result.Message,
	})
	event := emergencyAuditActions[action]
	err := s.auditRepo.Create(ctx, &domain.AuditLog{
		TenantID:     tenantID,
		ActorID:      adminID,
		Action:       event,
		ResourceType: "emergency_action",
		StatusCode:   200,
		Metadata:     meta,
	})
	if err != nil {
		log.Printf("[EMERGENCY] No se pudo registrar el evento de auditoría %s: %v", event, err)
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

	// 3. Catálogo real de servicios satélite soportados en la plataforma (ADR-045)
	pgVersion := strings.TrimSpace(os.Getenv("POSTGRES_VERSION"))
	if s.govRepo != nil {
		if probe, ok := s.govRepo.(domain.PostgresVersionProbe); ok {
			if liveVer, err := probe.GetServerMajorVersion(ctx); err == nil && strings.TrimSpace(liveVer) != "" {
				pgVersion = strings.TrimSpace(liveVer)
			}
		}
	}
	if pgVersion == "" {
		pgVersion = "18"
	}

	satellites := []domain.SatelliteServiceCapability{
		{
			Category:    "database",
			Engine:      "sqlite",
			Label:       "SQLite3 (Nativo)",
			Version:     "3.x",
			Description: "Base de datos embebida en archivo local (/home/workspace). Consumo nulo (+0 MB RAM), ideal para materias iniciales y SQL básico.",
			EnvVar:      "SQLITE_PATH",
			IsAvailable: true,
			BaseRAMMB:   0,
		},
		{
			Category:    "database",
			Engine:      "postgres",
			Label:       "PostgreSQL (Aula)",
			Version:     pgVersion,
			Description: "Perfil sintonizado para laboratorio (~50 MB RAM) con corte de consultas largas (5s), transaccionalidad completa y aislamiento por estudiante.",
			EnvVar:      "DATABASE_URL",
			IsAvailable: true,
			BaseRAMMB:   50,
		},
		{
			Category:    "database",
			Engine:      "mysql",
			Label:       "MySQL",
			Version:     "8.4",
			Description: "Base de datos relacional MySQL para ejercicios de SQL",
			EnvVar:      "DATABASE_URL",
			IsAvailable: strings.TrimSpace(os.Getenv("MYSQL_SATELLITE_HOST")) != "",
			BaseRAMMB:   128,
		},
		{
			Category:    "database",
			Engine:      "mongodb",
			Label:       "MongoDB",
			Version:     "7.0",
			Description: "Base de datos de documentos NoSQL para proyectos web",
			EnvVar:      "MONGODB_URI",
			IsAvailable: strings.TrimSpace(os.Getenv("MONGODB_SATELLITE_HOST")) != "",
			BaseRAMMB:   128,
		},
		{
			Category:    "cache",
			Engine:      "redis",
			Label:       "Redis",
			Version:     "7.2",
			Description: "Almacén en memoria y caché clave-valor",
			EnvVar:      "REDIS_URL",
			IsAvailable: strings.TrimSpace(os.Getenv("REDIS_SATELLITE_HOST")) != "",
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
	var synthesized string
	if strings.TrimSpace(dto.InstallMode) != "" {
		mode := domain.InstallMode(strings.TrimSpace(dto.InstallMode))
		if mode != domain.InstallModeNone {
			sLayer, errSynth := domain.SynthesizeAssistedBuildLayer(mode, dto.PackageNames, dto.BuildLayerSnippet)
			if errSynth != nil {
				return nil, errSynth
			}
			synthesized = sLayer
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
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(dto.DockerImage)), ":latest") {
		return nil, errors.New("el tag :latest está prohibido por reproducibilidad académica")
	}
	if dto.BaseRamMB > 0 {
		minFloorMB := calculateMinimumVitalFloorMB(dto.TargetEnvironment, dto.ServicesConfig)
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

