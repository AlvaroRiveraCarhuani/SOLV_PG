package services

import (
	"context"
	"fmt"
	"strings"

	"solv-backend/internal/core/domain"
)

func (s *TeacherService) ListLiveSessions(ctx context.Context, tenantID, teacherID string) ([]*domain.LiveWorkspaceSession, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}

	sessions, err := s.repo.ListLiveWorkspaceSessions(ctx, tenantID, teacherID)
	if err != nil {
		return nil, fmt.Errorf("error al listar sesiones en vivo: %w", err)
	}
	
	// Completar las métricas de cada sesión usando el snapshot service
	for _, session := range sessions {
		if s.metricsSnapshotService != nil {
			// Obtener el snapshot para esta workspace si existe
			snap, exists := s.metricsSnapshotService.GetSnapshot(session.WorkspaceID)
			if exists {
				session.MemoryUsedMB = snap.MemoryUsedMB
				session.CPUPercent = snap.CPUPercent
				session.MetricsAvailable = snap.MetricsAvailable
				session.MetricsUpdatedAt = snap.MetricsUpdatedAt
			}
		}
	}

	return sessions, nil
}

func (s *TeacherService) GetInitialTerminalBuffer(ctx context.Context, containerID string, tailLines int) (string, error) {
	if s.mirrorService == nil {
		return "[SOLV Shadow Mode] Sesión conectada.\n", nil
	}
	return s.mirrorService.GetInitialTerminalBuffer(ctx, containerID, tailLines)
}

func (s *TeacherService) ExecuteTutorCommand(ctx context.Context, tenantID, teacherID, containerID, cmd string) (*domain.TutorCommandResponse, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}
	if s.mirrorService == nil {
		s.mirrorService = NewTerminalMirrorService(nil)
	}
	return s.mirrorService.ExecuteTutorCommand(ctx, containerID, cmd)
}

func (s *TeacherService) GenerateFuzzCases(ctx context.Context, tenantID, teacherID string, req domain.FuzzGenerationRequest) (*domain.FuzzGenerationReport, error) {
	if tenantID == "" {
		return nil, domain.ErrInvalidTenant
	}

	engine := NewFuzzingEngine()
	report := engine.GenerateTestCases(req)

	if strings.TrimSpace(req.ReferenceCode) != "" && s.evalService != nil && len(report.Cases) > 0 {
		inputs := make([]string, len(report.Cases))
		for i, c := range report.Cases {
			inputs[i] = c.Input
		}
		calcResp, err := s.evalService.CalculateOutputs(ctx, CalculateOutputsRequest{
			Language:   req.TargetLanguage,
			SourceCode: req.ReferenceCode,
			Inputs:     inputs,
		})
		if err == nil && calcResp != nil {
			for _, out := range calcResp.Outputs {
				if out.Index >= 0 && out.Index < len(report.Cases) && out.Status == "ok" {
					report.Cases[out.Index].Expected = out.ExpectedOutput
				}
			}
		}
	}

	return report, nil
}

func (s *TeacherService) ApplyFuzzCases(ctx context.Context, tenantID, teacherID, exerciseID string, cases []domain.GeneratedFuzzCase) (int, error) {
	if tenantID == "" {
		return 0, domain.ErrInvalidTenant
	}
	if exerciseID == "" {
		return 0, domain.ErrNotFound
	}

	testCases := make([]domain.TestCase, 0, len(cases))
	for _, c := range cases {
		testCases = append(testCases, domain.TestCase{
			Input:          c.Input,
			ExpectedOutput: c.Expected,
			IsHidden:       !c.IsPublic,
		})
	}

	if s.evalService != nil {
		err := s.evalService.BulkAddTestCases(ctx, exerciseID, tenantID, testCases)
		if err != nil {
			return 0, fmt.Errorf("error al agregar casos de prueba al ejercicio: %w", err)
		}
	}

	return len(testCases), nil
}
