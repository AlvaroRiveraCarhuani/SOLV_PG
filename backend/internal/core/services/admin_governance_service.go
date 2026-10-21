package services

import (
	"context"
	"errors"
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
