package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
	"solv-backend/internal/core/domain"
)

var (
	ErrInvalidReviewStatus      = errors.New("status must be 'approved', 'rejected', 'paused', 'suspended' or 'pending_audit'")
	ErrRejectionReasonRequired  = errors.New("rejection_reason is required when rejecting or suspending a template")
	ErrTemplateQuotaNotEditable = errors.New("la plantilla no admite ajuste de cuota en su estado actual")
	ErrTemplateNotAuditable    = errors.New("la plantilla no admite re-escaneo en su estado actual")
)

// constructorVerifiedSmokeOutput documenta de dónde viene un smoke aprobado
// sin contenedor: la compilación del Constructor ya ejecutó la comprobación.
const constructorVerifiedSmokeOutput = "Verificado durante la compilación en el Constructor"

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

	// La columna guarda el estado canónico en mayúsculas; el DTO habla en
	// minúsculas. Guardar el valor del DTO dejaba la plantilla en un estado que
	// la publicación no reconoce como aprobado.
	dbStatus := dto.Status
	switch status {
	case "approved":
		dbStatus = "APROBADA"
	case "rejected":
		dbStatus = "RECHAZADA"
	case "paused":
		dbStatus = "PAUSADA"
	case "suspended":
		dbStatus = "SUSPENDIDA"
	case "pending_audit":
		dbStatus = "PENDIENTE_AUDITORIA"
	}

	if dto.BaseRamMB != nil && *dto.BaseRamMB > 0 {
		if status == "approved" || status == "pending_audit" || status == "paused" {
			if existing, err := s.govRepo.GetTemplateByID(ctx, templateID); err == nil && existing != nil {
				minFloorMB := calculateMinimumVitalFloorMB(existing.TargetEnvironment)
				if *dto.BaseRamMB < minFloorMB {
					return nil, fmt.Errorf("%w: base_ram_mb=%d es inferior al piso vital requerido (%d MB)", ErrRamBelowVitalFloor, *dto.BaseRamMB, minFloorMB)
				}
			}
		}

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
		totalMB = domain.Uint64ToInt(v.Total / (1024 * 1024))
	}
	return totalMB
}

var (
	dockerImageRegex      = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*:[a-zA-Z0-9_.-]+$`)
	ErrInvalidDockerImage = errors.New("la imagen Docker debe tener formato válido repositorio:tag sin espacios (ej: python:3.12-slim)")
	ErrLatestTagForbidden = errors.New("el tag :latest está prohibido en plantillas oficiales por reproducibilidad")
	ErrRamBelowVitalFloor = errors.New("la memoria asignada es inferior al piso vital requerido por el entorno")
)

// calculateMinimumVitalFloorMB es el piso del contenedor del alumno: editor y procesos.
// Los motores satélite viven en contenedores compartidos y su reserva se descuenta una
// vez en la admisión del nodo, así que no suman a la cuota de cada laboratorio.
func calculateMinimumVitalFloorMB(targetEnvironment string) int {
	env := strings.TrimSpace(targetEnvironment)
	if env == "JUEZ_EFIMERO" || env == "JUEZ_VIRTUAL" {
		return 64
	}
	return 210
}

var ErrTemplateVerificationRequired = errors.New("la configuración técnica requiere una comprobación exitosa vigente")

func validateTemplateVerification(
	ctx context.Context,
	repo domain.EnvTestJobRepository,
	tenantID, jobID string,
	config domain.EnvTestVerificationConfig,
	requiresVerification bool,
) error {
	if !requiresVerification {
		return nil
	}
	if repo == nil || strings.TrimSpace(jobID) == "" {
		return ErrTemplateVerificationRequired
	}
	job, err := repo.GetByID(ctx, strings.TrimSpace(jobID))
	if err != nil || job == nil || job.Status != domain.EnvTestStatusSuccess {
		return ErrTemplateVerificationRequired
	}
	if job.TenantID != tenantID || job.OwnerID == "" || job.OwnerID != domain.GetUserID(ctx) || job.VerificationFingerprint == "" || job.VerificationFingerprint != config.Fingerprint() {
		return ErrTemplateVerificationRequired
	}
	return nil
}

func verificationConfigForCreate(dto domain.CreateOfficialTemplateDTO) domain.EnvTestVerificationConfig {
	var dockerfileContent string
	if dto.StackConfig != nil {
		dockerfileContent = dto.StackConfig.DockerfileContent
	}
	return domain.EnvTestVerificationConfig{
		Image: dto.DockerImage, Tools: dto.ToolsDeclared, TargetEnvironment: dto.TargetEnvironment,
		Entrypoint: dto.Entrypoint, TimeoutMS: dto.TimeoutMS, SampleInput: dto.SampleInput,
		BaseRamMB: dto.BaseRamMB, SetupScript: dto.SetupScript,
		DockerfileContent: dockerfileContent,
	}
}

func verificationConfigForUpdate(dto domain.UpdateOfficialTemplateDTO) domain.EnvTestVerificationConfig {
	var dockerfileContent string
	if dto.StackConfig != nil {
		dockerfileContent = dto.StackConfig.DockerfileContent
	}
	return domain.EnvTestVerificationConfig{
		Image: dto.DockerImage, Tools: dto.ToolsDeclared, TargetEnvironment: dto.TargetEnvironment,
		Entrypoint: dto.Entrypoint, TimeoutMS: dto.TimeoutMS, SampleInput: dto.SampleInput,
		BaseRamMB: dto.BaseRamMB, SetupScript: dto.SetupScript,
		DockerfileContent: dockerfileContent,
	}
}

// validateLanguageRunnerLimits acota la memoria declarada por cada ejecutor al
// techo del host. El tiempo y el rango de memoria ya se validaron en el dominio.
func (s *AdminGovernanceService) validateLanguageRunnerLimits(ctx context.Context, runners []domain.LanguageRunner) error {
	if len(runners) == 0 {
		return nil
	}
	maxAllowedRAM := domain.CalculateHostMaxAllowedRAM(s.getHostTotalRAM(ctx))
	for _, runner := range runners {
		if runner.MemoryLimitMB > 0 {
			if err := domain.ValidateRamAgainstHost(runner.MemoryLimitMB, maxAllowedRAM); err != nil {
				return fmt.Errorf("%s: %w", runner.Language, err)
			}
		}
	}
	return nil
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

	// Un entorno de evaluación declara uno o más ejecutores de lenguaje. La
	// modalidad que no evalúa no los admite: no tendrían quién los ejecute.
	if !domain.IsJudgeEnvironment(dto.TargetEnvironment) {
		dto.LanguageRunners = nil
	} else {
		runners, err := domain.NormalizeLanguageRunners(dto.LanguageRunners)
		if err != nil {
			return nil, err
		}
		dto.LanguageRunners = runners
		if err := s.validateLanguageRunnerLimits(ctx, runners); err != nil {
			return nil, err
		}
		// El entrypoint de la plantilla sigue siendo el del smoke test; con un
		// solo lenguaje se deriva del ejecutor para no pedirlo dos veces.
		if strings.TrimSpace(dto.Entrypoint) == "" && len(runners) > 0 {
			dto.Entrypoint = runners[0].Entrypoint
		}
	}

	if domain.IsBuiltStackImageTag(dto.DockerImage) {
		if err := validateTemplateVerification(ctx, s.envTestJobs, tenantID, dto.VerificationJobID, verificationConfigForCreate(dto), true); err != nil {
			return nil, err
		}
		// El Constructor ya construyó la imagen y ejecutó el smoke con las
		// herramientas declaradas. Se persiste ese resultado en vez de un
		// 'pending' que da a entender que nadie comprobó nada; solo falta el
		// escaneo de Trivy, que el worker toma de inmediato.
		dto.SmokeTestStatus = "passed"
		dto.SmokeTestOutput = constructorVerifiedSmokeOutput
	}

	if domain.IsJudgeEnvironment(dto.TargetEnvironment) {
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

	if dto.BaseRamMB <= 0 {
		dto.BaseRamMB = 512
	} else {
		minFloorMB := calculateMinimumVitalFloorMB(dto.TargetEnvironment)
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
		// Pull asincrono desacoplado del ciclo de vida del request HTTP.
		//nolint:gosec
		go func(tid, tmplID, img string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			log.Printf("[GoldenImage] Pre-warming image %q for official template %s...", img, tmplID)
			_, _ = s.cacheService.WarmTemplate(bgCtx, tid, tmplID)
		}(tenantID, created.ID, created.DockerImage)
	}

	return created, nil
}

// UpdateTemplateQuota ajusta la memoria por alumno sin reconstruir la imagen.
// La huella de verificación incluye la RAM, así que este camino no puede exigir
// el job de build: valida piso y techo y deja que el worker repita el smoke.
func (s *AdminGovernanceService) UpdateTemplateQuota(
	ctx context.Context,
	tenantID, templateID, adminID string,
	dto domain.UpdateTemplateQuotaDTO,
) (*domain.AdminTemplateReviewItem, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, ErrTemplateNotFound
	}
	existing, err := s.govRepo.GetTemplateByID(ctx, templateID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "template not found") {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	if existing == nil {
		return nil, ErrTemplateNotFound
	}
	if existing.TenantID != nil && strings.TrimSpace(*existing.TenantID) != "" && *existing.TenantID != tenantID {
		return nil, ErrTemplateNotFound
	}

	status := strings.ToLower(strings.TrimSpace(existing.Status))
	if status == "rejected" || status == "rechazada" {
		return nil, ErrTemplateQuotaNotEditable
	}
	if dto.BaseRamMB <= 0 {
		return nil, errors.New("base_ram_mb debe ser mayor que cero")
	}

	minFloorMB := calculateMinimumVitalFloorMB(existing.TargetEnvironment)
	if dto.BaseRamMB < minFloorMB {
		return nil, fmt.Errorf("%w: base_ram_mb=%d es inferior al piso vital requerido (%d MB)", ErrRamBelowVitalFloor, dto.BaseRamMB, minFloorMB)
	}
	totalHostMB := s.getHostTotalRAM(ctx)
	maxAllowedRAM := domain.CalculateHostMaxAllowedRAM(totalHostMB)
	if err := domain.ValidateRamAgainstHost(dto.BaseRamMB, maxAllowedRAM); err != nil {
		return nil, err
	}

	// Una plantilla aprobada repite el smoke con la cuota nueva. Una pausada
	// conserva su estado y no se auto-reactiva por una auditoría.
	dto.ResetSmoke = isApprovedTemplateStatus(existing.Status)
	quotaRepo, ok := s.govRepo.(domain.TemplateQuotaRepository)
	if !ok {
		return nil, errors.New("el repositorio no admite ajuste de cuota")
	}
	updated, err := quotaRepo.UpdateTemplateQuota(ctx, tenantID, templateID, adminID, dto)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "template not found") {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	return updated, nil
}

// ReauditTemplate reencola la auditoría técnica sin tocar el estado del
// catálogo: deja el smoke y el escaneo en "pending" y despierta al worker para
// que empiece en segundos. Los estados que decide una persona no se reauditan.
func (s *AdminGovernanceService) ReauditTemplate(
	ctx context.Context,
	tenantID, templateID string,
) (*domain.AdminTemplateReviewItem, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, ErrTemplateNotFound
	}

	existing, err := s.govRepo.GetTemplateByID(ctx, templateID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "template not found") {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	if existing == nil {
		return nil, ErrTemplateNotFound
	}
	if existing.TenantID != nil {
		if owner := strings.TrimSpace(*existing.TenantID); owner != "" && owner != tenantID {
			return nil, ErrTemplateNotFound
		}
	}
	if isHumanReviewStatus(existing.Status) {
		return nil, ErrTemplateNotAuditable
	}

	repo, ok := s.govRepo.(domain.TemplateAuditRepository)
	if !ok {
		return nil, errors.New("el repositorio actual no admite re-escaneo de plantillas")
	}
	updated, err := repo.ResetTemplateAudit(ctx, tenantID, templateID)
	if err != nil {
		return nil, err
	}

	if s.auditWorker != nil {
		s.auditWorker.EnqueueTemplate(templateID)
	}
	return updated, nil
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
