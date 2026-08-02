package services

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"solv-backend/internal/core/domain"
)

type ZombieCollectorWorker struct {
	repo                   domain.WorkspaceRepository
	orchestrator           domain.WorkspaceOrchestrator
	interval               time.Duration
	mu                     sync.Mutex
	reclaimedCountAtomic   uint64
}

func NewZombieCollectorWorker(repo domain.WorkspaceRepository, orchestrator domain.WorkspaceOrchestrator, interval time.Duration) *ZombieCollectorWorker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &ZombieCollectorWorker{
		repo:         repo,
		orchestrator: orchestrator,
		interval:     interval,
	}
}

func (w *ZombieCollectorWorker) GetReclaimedCount() uint64 {
	return atomic.LoadUint64(&w.reclaimedCountAtomic)
}

func (w *ZombieCollectorWorker) Start(ctx context.Context) {
	log.Printf("[Zombie Collector] Worker started (Reconciliation Interval: %v)", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Zombie Collector] Worker stopped.")
			return
		case <-ticker.C:
			w.ReconcileOrphanContainers(ctx)
		}
	}
}

func (w *ZombieCollectorWorker) ReconcileOrphanContainers(ctx context.Context) {
	// Manejo de Concurrencia: Evita reconciliaciones superpuestas si un ciclo tarda más del intervalo
	if !w.mu.TryLock() {
		return
	}
	defer w.mu.Unlock()

	managedContainers, err := w.orchestrator.ListAllManagedContainers(ctx)
	if err != nil {
		log.Printf("[Zombie Collector] Error listing Docker containers: %v", err)
		return
	}

	if len(managedContainers) == 0 {
		return
	}

	activeWorkspaces, err := w.repo.GetActiveWorkspaces(ctx)
	if err != nil {
		log.Printf("[Zombie Collector] Error getting active workspaces from DB: %v", err)
		return
	}

	// Mapear IDs de contenedores legítimos registrados en DB
	legitimateContainers := make(map[string]bool)
	for _, ws := range activeWorkspaces {
		if ws.ContainerID != nil && *ws.ContainerID != "" {
			legitimateContainers[*ws.ContainerID] = true
		}
	}

	// Detectar y destruir contenedores huérfanos
	for _, cid := range managedContainers {
		if !legitimateContainers[cid] {
			log.Printf("[Zombie Collector] WARNING: Found orphan/zombie container %s not present in PostgreSQL. Removing...", cid[:12])
			if err := w.orchestrator.StopAndRemoveContainer(ctx, cid); err == nil {
				atomic.AddUint64(&w.reclaimedCountAtomic, 1)
				log.Printf("[Zombie Collector] Reclaimed orphan container %s successfully.", cid[:12])
			} else {
				log.Printf("[Zombie Collector] Error removing orphan container %s: %v", cid[:12], err)
			}
		}
	}
}
