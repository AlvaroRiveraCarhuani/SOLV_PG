package services

import (
	"context"
	"log"
	"sync"
	"time"

	"solv-backend/internal/core/domain"
)

type ContainerMetricsSnapshot struct {
	WorkspaceID      string
	ContainerID      string
	StudentID        string
	StudentName      string
	StudentEmail     string
	SubjectID        string
	SubjectName      string
	MemoryLimitMB    int64
	OOMStrikes       int
	LastHeartbeatAt  time.Time
	IsAttached       bool
	MemoryUsedMB     int64
	CPUPercent       float64
	MetricsAvailable bool
	MetricsUpdatedAt time.Time
}

type MetricsSnapshotService struct {
	repo           domain.WorkspaceRepository
	docker         domain.WorkspaceOrchestrator
	snapshots      map[string]ContainerMetricsSnapshot
	mu             sync.RWMutex
	stopChan       chan struct{}
	doneChan       chan struct{}
	updateTicker   *time.Ticker
	updateInterval time.Duration
	tenantFilter   func(tenantID string) bool
}

func NewMetricsSnapshotService(
	repo domain.WorkspaceRepository,
	docker domain.WorkspaceOrchestrator,
	updateInterval time.Duration,
) *MetricsSnapshotService {
	if updateInterval <= 0 {
		updateInterval = 5 * time.Second
	}
	s := &MetricsSnapshotService{
		repo:           repo,
		docker:         docker,
		updateInterval: updateInterval,
		snapshots:      make(map[string]ContainerMetricsSnapshot),
		stopChan:       make(chan struct{}),
		doneChan:       make(chan struct{}),
	}
	return s
}

func (s *MetricsSnapshotService) Start(ctx context.Context) {
	if s == nil {
		log.Printf("[Metrics Snapshot] Service not initialized, skipping start")
		return
	}
	s.updateTicker = time.NewTicker(s.updateInterval)
	go func() {
		for {
			select {
			case <-s.updateTicker.C:
				s.updateSnapshots(ctx)
			case <-s.stopChan:
				s.updateTicker.Stop()
				close(s.doneChan)
				return
			case <-ctx.Done():
				s.updateTicker.Stop()
				close(s.doneChan)
				return
			}
		}
	}()
	log.Printf("[Metrics Snapshot] Metrics snapshot service started (Interval: %v)", s.updateInterval)
}

func (s *MetricsSnapshotService) Stop() {
	if s == nil {
		return
	}
	s.stopChan <- struct{}{}
	<-s.doneChan
}

func (s *MetricsSnapshotService) SetTenantFilter(filter func(tenantID string) bool) {
	if s == nil {
		return
	}
	s.tenantFilter = filter
}

func (s *MetricsSnapshotService) GetWorkspaceMetrics(ctx context.Context, tenantID, teacherID string) ([]*ContainerMetricsSnapshot, error) {
	if s == nil {
		return nil, nil
	}

	// updateSnapshots es quien construye y actualiza el mapa con escrituras;
	// aquí solo se leen copias de los snapshots ya calculados.
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*ContainerMetricsSnapshot, 0, len(s.snapshots))
	for _, snap := range s.snapshots {
		snapCopy := snap
		result = append(result, &snapCopy)
	}

	return result, nil
}

func (s *MetricsSnapshotService) updateSnapshots(ctx context.Context) {
	if s == nil {
		return
	}

	activeWorkspaces, err := s.repo.GetActiveWorkspaces(ctx)
	if err != nil {
		log.Printf("[Metrics Snapshot] Error updating snapshots: %v", err)
		return
	}

	for _, ws := range activeWorkspaces {
		if ws.ContainerID == nil || *ws.ContainerID == "" {
			continue
		}

		containerID := *ws.ContainerID

		// Get container metrics
		metrics, err := s.docker.GetContainerMetrics(ctx, containerID)
		if err != nil {
			log.Printf("[Metrics Snapshot] Error getting container metrics: %v", err)
			continue
		}

		// Skip if not running
		if !metrics.IsRunning {
			continue
		}

		// Lock once per workspace (outside slow Docker call)
		s.mu.Lock()
		wsID := ws.ID

		snap, exists := s.snapshots[wsID]

		if !exists {
			// Create new snapshot
			var studentID, studentName, studentEmail string
			var subjectID, subjectName string

			if ws.StudentID != "" {
				studentID = ws.StudentID
			}
			if ws.SubjectID != "" {
				subjectID = ws.SubjectID
			}

			snap = ContainerMetricsSnapshot{
				WorkspaceID:      wsID,
				ContainerID:      containerID,
				StudentID:        studentID,
				StudentName:      studentName,
				StudentEmail:     studentEmail,
				SubjectID:        subjectID,
				SubjectName:      subjectName,
				MemoryLimitMB:    ws.MemoryLimitMB,
				OOMStrikes:       ws.OOMStrikeCount,
				LastHeartbeatAt:  ws.LastHeartbeatAt,
				IsAttached:       ws.AccessURL != "",
				MemoryUsedMB:     metrics.MemoryUsageBytes / (1024 * 1024), // Convert bytes to MB
				CPUPercent:       metrics.CPUPercent,
				MetricsAvailable: true,
				MetricsUpdatedAt: time.Now(),
			}
		}

		// Update existing snapshot
		snap.MemoryUsedMB = metrics.MemoryUsageBytes / (1024 * 1024) // Convert bytes to MB
		snap.CPUPercent = metrics.CPUPercent
		snap.MetricsAvailable = metrics.IsRunning && metrics.MemoryUsageBytes > 0
		snap.MetricsUpdatedAt = time.Now()
		snap.LastHeartbeatAt = ws.LastHeartbeatAt
		snap.IsAttached = ws.AccessURL != ""
		snap.OOMStrikes = ws.OOMStrikeCount
		s.snapshots[wsID] = snap
		s.mu.Unlock()
	}
}

func (s *MetricsSnapshotService) GetSnapshot(workspaceID string) (ContainerMetricsSnapshot, bool) {
	if s == nil {
		var zero ContainerMetricsSnapshot
		return zero, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	snap, exists := s.snapshots[workspaceID]
	return snap, exists
}
