package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"solv-backend/internal/core/domain"
)

var (
	// ErrVolumeInventoryUnavailable impide ofrecer la limpieza cuando no se
	// pudo leer el daemon o la base. Sin ese cruce no hay forma de saber qué
	// volumen pertenece a un entorno, así que se falla cerrado.
	ErrVolumeInventoryUnavailable = errors.New("volume inventory is unavailable")
	// ErrOrphanVolumeSelectionEmpty rechaza una liberación sin nombres.
	ErrOrphanVolumeSelectionEmpty = errors.New("no orphan volumes selected")
	// ErrOrphanVolumeSelectionTooLarge acota una corrida a MaxOrphanVolumesPerRun.
	ErrOrphanVolumeSelectionTooLarge = errors.New("too many orphan volumes selected")
)

// environmentVolumeInventory es la lectura y el borrado de volúmenes del nodo.
type environmentVolumeInventory interface {
	ListDanglingEnvironmentVolumes(ctx context.Context) ([]domain.DanglingVolume, error)
	RemoveEnvironmentVolume(ctx context.Context, name string) error
}

// registeredVolumeLister devuelve los volúmenes que reclama algún entorno
// registrado, en cualquier estado. Es una capacidad opcional del repositorio:
// sin ella la limpieza no se habilita.
type registeredVolumeLister interface {
	ListRegisteredVolumeNames(ctx context.Context) ([]string, error)
}

// DiskOrphanService libera volúmenes de workspace que ningún entorno reclama.
// La operación es manual y con auditoría por volumen: no hay worker que borre
// solo, porque un volumen recién creado todavía no está reclamado en base.
type DiskOrphanService struct {
	volumes      environmentVolumeInventory
	environments domain.PracticeEnvironmentRepository
	auditRepo    domain.AuditLogRepository
}

func NewDiskOrphanService(volumes environmentVolumeInventory, environments domain.PracticeEnvironmentRepository) *DiskOrphanService {
	return &DiskOrphanService{volumes: volumes, environments: environments}
}

func (s *DiskOrphanService) SetAuditRepository(repo domain.AuditLogRepository) {
	s.auditRepo = repo
}

// Plan arma el inventario completo: qué se puede liberar y qué se conserva.
func (s *DiskOrphanService) Plan(ctx context.Context) (*domain.OrphanVolumePlan, error) {
	if s == nil || s.volumes == nil || s.environments == nil {
		return nil, ErrVolumeInventoryUnavailable
	}
	lister, ok := s.environments.(registeredVolumeLister)
	if !ok {
		return nil, ErrVolumeInventoryUnavailable
	}

	registeredNames, err := lister.ListRegisteredVolumeNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeInventoryUnavailable, err)
	}
	dangling, err := s.volumes.ListDanglingEnvironmentVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVolumeInventoryUnavailable, err)
	}

	registered := make(map[string]struct{}, len(registeredNames))
	for _, name := range registeredNames {
		registered[name] = struct{}{}
	}

	plan := domain.PlanOrphanVolumes(dangling, registered)
	return &plan, nil
}

// Prune libera solo los volúmenes pedidos que sigan siendo huérfanos en el
// momento de borrar. El plan se recalcula dentro de la operación para que una
// pantalla abierta hace diez minutos no borre un entorno que se creó después.
func (s *DiskOrphanService) Prune(
	ctx context.Context,
	tenantID, actorID string,
	req domain.OrphanVolumePruneRequest,
) (*domain.OrphanVolumePruneResult, error) {
	if req.ConfirmationPhrase != domain.OrphanVolumePrunePhrase {
		return nil, ErrInvalidConfirmationPhrase
	}

	requested := make([]string, 0, len(req.VolumeNames))
	seen := make(map[string]struct{}, len(req.VolumeNames))
	for _, raw := range req.VolumeNames {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}
		requested = append(requested, name)
	}
	if len(requested) == 0 {
		return nil, ErrOrphanVolumeSelectionEmpty
	}
	if len(requested) > domain.MaxOrphanVolumesPerRun {
		return nil, ErrOrphanVolumeSelectionTooLarge
	}

	plan, err := s.Plan(ctx)
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]domain.DanglingVolume, len(plan.Orphans))
	for _, volume := range plan.Orphans {
		allowed[volume.Name] = volume
	}
	keptReason := make(map[string]string, len(plan.Kept))
	for _, volume := range plan.Kept {
		keptReason[volume.Name] = volume.Reason
	}

	result := &domain.OrphanVolumePruneResult{
		Deleted: make([]domain.DeletedVolume, 0, len(requested)),
		Skipped: make([]domain.KeptVolume, 0, len(requested)),
	}

	for _, name := range requested {
		volume, ok := allowed[name]
		if !ok {
			reason := keptReason[name]
			if reason == "" {
				reason = domain.VolumeKeepNotDangling
			}
			result.Skipped = append(result.Skipped, domain.KeptVolume{Name: name, Reason: reason})
			continue
		}

		if err := s.volumes.RemoveEnvironmentVolume(ctx, name); err != nil {
			// Un volumen que se montó entre el plan y el borrado se informa
			// como conservado: la corrida sigue con el resto.
			log.Printf("[Disk Orphans] no se pudo liberar %s: %v", name, err)
			result.Skipped = append(result.Skipped, domain.KeptVolume{
				Name:   name,
				SizeMB: volume.SizeMB,
				Reason: domain.VolumeKeepRemovalFailed,
			})
			continue
		}

		result.Deleted = append(result.Deleted, domain.DeletedVolume{Name: name, SizeMB: volume.SizeMB})
		result.FreedMB += volume.SizeMB
		s.recordReleaseAudit(ctx, tenantID, actorID, name, volume.SizeMB, req.Reason)
	}

	result.RemainingOrphanMB = plan.TotalOrphanMB - result.FreedMB
	if result.RemainingOrphanMB < 0 {
		result.RemainingOrphanMB = 0
	}
	return result, nil
}

// recordReleaseAudit deja una entrada por volumen liberado. La auditoría no
// revierte el borrado: el volumen ya no existe, pero el evento queda para
// explicar de dónde salió el espacio.
func (s *DiskOrphanService) recordReleaseAudit(
	ctx context.Context,
	tenantID, actorID, volumeName string,
	sizeMB int64,
	reason string,
) {
	if s.auditRepo == nil {
		return
	}
	metadata, _ := json.Marshal(map[string]any{
		"size_mb": sizeMB,
		"reason":  reason,
	})
	resourceID := volumeName
	err := s.auditRepo.Create(ctx, &domain.AuditLog{
		TenantID:     tenantID,
		ActorID:      actorID,
		Action:       "ORPHAN_VOLUME_RELEASED",
		ResourceType: "docker_volume",
		ResourceID:   &resourceID,
		StatusCode:   http.StatusOK,
		Metadata:     metadata,
	})
	if err != nil {
		log.Printf("[Disk Orphans] no se pudo auditar la liberación de %s: %v", volumeName, err)
	}
}
