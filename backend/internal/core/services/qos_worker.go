package services

import (
	"context"
	"log"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
)

type containerStatsHistory struct {
	lastCPU     float64
	lastRx      uint64
	lastTx      uint64
	idleCounter int
}

type QoSOrchestratorWorker struct {
	repo               domain.WorkspaceRepository
	docker             domain.WorkspaceOrchestrator
	hostMonitor        domain.HostMonitor
	inactivityTimeout  time.Duration
	checkInterval      time.Duration
	history            map[string]*containerStatsHistory
	mu                 sync.Mutex
	stopChan           chan struct{}
}

func NewQoSOrchestratorWorker(
	repo domain.WorkspaceRepository,
	docker domain.WorkspaceOrchestrator,
	hostMonitor domain.HostMonitor,
	inactivityTimeout time.Duration,
	checkInterval time.Duration,
) *QoSOrchestratorWorker {
	if inactivityTimeout <= 0 {
		inactivityTimeout = 15 * time.Minute
	}
	if checkInterval <= 0 {
		checkInterval = 10 * time.Second
	}
	return &QoSOrchestratorWorker{
		repo:              repo,
		docker:            docker,
		hostMonitor:       hostMonitor,
		inactivityTimeout: inactivityTimeout,
		checkInterval:     checkInterval,
		history:           make(map[string]*containerStatsHistory),
		stopChan:          make(chan struct{}),
	}
}

func (w *QoSOrchestratorWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.checkInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				w.runCycle(ctx)
			case <-w.stopChan:
				ticker.Stop()
				return
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	log.Printf("[QoS Worker] Resource & Hibernation Orchestrator started (Interval: %v, Inactivity Timeout: %v)", w.checkInterval, w.inactivityTimeout)
}

func (w *QoSOrchestratorWorker) Stop() {
	close(w.stopChan)
}

func (w *QoSOrchestratorWorker) runCycle(ctx context.Context) {
	activeWorkspaces, err := w.repo.GetActiveWorkspaces(ctx)
	if err != nil {
		log.Printf("[QoS Worker] Error fetching active workspaces: %v", err)
		return
	}

	for _, ws := range activeWorkspaces {
		if ws.ContainerID == nil || *ws.ContainerID == "" {
			continue
		}
		containerID := *ws.ContainerID

		metrics, err := w.docker.GetContainerMetrics(ctx, containerID)
		if err != nil {
			log.Printf("[QoS Worker] Error getting container metrics for %s: %v", containerID, err)
			continue
		}

		// 1. Detección de OOMKilled / Exit 137
		if !metrics.IsRunning {
			if metrics.OOMKilled || metrics.ExitCode == 137 {
				log.Printf("[QoS Worker] Container %s (Workspace %s) was OOMKilled (Exit 137). Incrementing strike.", containerID, ws.ID)
				_ = w.repo.IncrementOOMStrike(ctx, ws.ID)
			}
			continue
		}

		// 2. Escalamiento Vertical Dinámico (Auto-Bursting)
		if metrics.MemoryLimitBytes > 0 {
			currentLimitMB := metrics.MemoryLimitBytes / (1024 * 1024)
			usagePct := (float64(metrics.MemoryUsageBytes) / float64(metrics.MemoryLimitBytes)) * 100.0

			if usagePct >= 80.0 && currentLimitMB < domain.HardQuotaMemoryMB {
				targetMemoryMB := currentLimitMB + 256
				if targetMemoryMB > domain.HardQuotaMemoryMB {
					targetMemoryMB = domain.HardQuotaMemoryMB
				}

				neededMB := targetMemoryMB - currentLimitMB
				if w.hostMonitor.CanAllocateMemory(neededMB) {
					if err := w.docker.UpdateContainerMemory(ctx, containerID, targetMemoryMB); err == nil {
						_ = w.repo.UpdateMemoryLimit(ctx, ws.ID, targetMemoryMB)
						log.Printf("[QoS Auto-Bursting] Workspace %s scaled UP from %d MB to %d MB (Usage: %.2f%%)", ws.ID, currentLimitMB, targetMemoryMB, usagePct)
					}
				} else {
					log.Printf("[QoS Auto-Bursting GUARD] Skipped scale UP for workspace %s: Host memory depleted (< 15%% free)", ws.ID)
				}
			}
		}

		// 3. Hibernación Segura por Validación Dual (Intención vs Realidad)
		w.mu.Lock()
		hist, exists := w.history[ws.ID]
		if !exists {
			hist = &containerStatsHistory{
				lastCPU: metrics.CPUPercent,
				lastRx:  metrics.RxBytes,
				lastTx:  metrics.TxBytes,
			}
			w.history[ws.ID] = hist
		}

		// Calcular deltas de realidad
		rxDelta := metrics.RxBytes - hist.lastRx
		txDelta := metrics.TxBytes - hist.lastTx
		cpuLow := metrics.CPUPercent < 0.5
		netIdle := (rxDelta + txDelta) < 1024 // Menos de 1KB transferido

		hist.lastCPU = metrics.CPUPercent
		hist.lastRx = metrics.RxBytes
		hist.lastTx = metrics.TxBytes

		if cpuLow && netIdle {
			hist.idleCounter++
		} else {
			hist.idleCounter = 0
		}
		w.mu.Unlock()

		// Validación Dual:
		// Condición A (Intención): tiempo transcurrido desde el último Heartbeat HTTP > inactivityTimeout
		intentionTimeout := time.Since(ws.LastHeartbeatAt) >= w.inactivityTimeout

		// Condición B (Realidad): inactividad física persistente en CPU/Red equivalente a la ventana de hibernación
		maxIdleCycles := int(w.inactivityTimeout / w.checkInterval)
		realityTimeout := hist.idleCounter >= maxIdleCycles

		if intentionTimeout || realityTimeout {
			log.Printf("[QoS Hibernation] Hibernating container %s (Workspace %s). Reason: IntentionTimeout=%v, RealityTimeout=%v", containerID, ws.ID, intentionTimeout, realityTimeout)
			if err := w.docker.StopAndRemoveContainer(ctx, containerID); err == nil {
				_ = w.repo.UpdateStatus(ctx, ws.ID, domain.WorkspaceStatusHibernated)
				w.mu.Lock()
				delete(w.history, ws.ID)
				w.mu.Unlock()
			}
		}
	}
}
