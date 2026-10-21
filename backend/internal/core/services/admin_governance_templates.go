package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/shirou/gopsutil/v3/mem"
	"log"
	"regexp"
	"solv-backend/internal/core/domain"
	"strings"
	"time"
)

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
		if status == "approved" || status == "pending_audit" || status == "paused" {
			if existing, err := s.govRepo.GetTemplateByID(ctx, templateID); err == nil && existing != nil {
				minFloorMB := calculateMinimumVitalFloorMB(existing.TargetEnvironment, &existing.ServicesConfig)
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
