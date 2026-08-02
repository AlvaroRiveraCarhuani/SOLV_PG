package services

import (
	"context"
	"fmt"
	"log"

	"solv-backend/internal/core/domain"
)

type SemgrepWorker struct {
	repo         domain.WorkspaceRepository
	orchestrator domain.WorkspaceOrchestrator
}

func NewSemgrepWorker(repo domain.WorkspaceRepository, orchestrator domain.WorkspaceOrchestrator) *SemgrepWorker {
	return &SemgrepWorker{
		repo:         repo,
		orchestrator: orchestrator,
	}
}

// AuditWorkspace ejecuta la auditoría AST semántica sobre el volumen de un estudiante y persiste el JSONB en PostgreSQL
func (w *SemgrepWorker) AuditWorkspace(ctx context.Context, workspaceID string, volumeName string) ([]byte, error) {
	log.Printf("[Semgrep Worker] Starting AST semantic code audit on workspace %s (Volume: %s)...", workspaceID, volumeName)

	// 1. Ejecutar escaneo efímero en contenedor semgrep/semgrep en modo solo lectura (:ro)
	auditJSON, err := w.orchestrator.RunSemgrepScanOnVolume(ctx, volumeName)
	if err != nil {
		return nil, fmt.Errorf("semgrep scan failed for volume %s: %w", volumeName, err)
	}

	if len(auditJSON) == 0 {
		auditJSON = []byte("{}")
	}

	// 2. Persistir resultado de la auditoría AST en la columna JSONB de PostgreSQL
	if err := w.repo.SaveSemgrepAudit(ctx, workspaceID, auditJSON); err != nil {
		return nil, fmt.Errorf("failed to persist semgrep audit for workspace %s: %w", workspaceID, err)
	}

	log.Printf("[Semgrep Worker] AST semantic audit completed successfully for workspace %s. Persisted %d bytes to PostgreSQL JSONB.", workspaceID, len(auditJSON))
	return auditJSON, nil
}
