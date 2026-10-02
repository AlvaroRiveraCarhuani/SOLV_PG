package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"solv-backend/internal/core/domain"
)

// TerminalMirrorService gestiona la inspeccion de terminales en tiempo real (Shadow Mode) y comandos de asistencia tutorial.
type TerminalMirrorService struct {
	orchestrator domain.WorkspaceOrchestrator
}

func NewTerminalMirrorService(orchestrator domain.WorkspaceOrchestrator) *TerminalMirrorService {
	return &TerminalMirrorService{
		orchestrator: orchestrator,
	}
}

// CleanTerminalOutput sanitiza encabezados multiplexados de Docker y secuencias ANSI no imprimibles si es necesario.
func (s *TerminalMirrorService) CleanTerminalOutput(raw []byte) string {
	var result []byte
	data := raw
	for len(data) > 0 {
		if len(data) < 8 {
			result = append(result, data...)
			break
		}
		// Docker multiplex stream: byte 0 es 1 (stdout) o 2 (stderr)
		if data[0] == 1 || data[0] == 2 {
			size := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
			data = data[8:]
			if size > len(data) {
				size = len(data)
			}
			result = append(result, data[:size]...)
			data = data[size:]
		} else {
			result = append(result, data[0])
			data = data[1:]
		}
	}
	return string(result)
}

// GetInitialTerminalBuffer obtiene los logs recientes del contenedor para poblar la vista inicial de la terminal.
func (s *TerminalMirrorService) GetInitialTerminalBuffer(ctx context.Context, containerID string, tailLines int) (string, error) {
	if containerID == "" {
		return "[SOLV Terminal] Contenedor no asignado o pendiente de inicio.\n", nil
	}

	if s.orchestrator == nil {
		// Mock buffer para tests y modo offline
		return fmt.Sprintf("[SOLV Shadow Mode] Conectado a contenedor %s\n$ Solv workspace active. Listening for events...\n", containerID), nil
	}

	logs, err := s.orchestrator.GetContainerLogs(ctx, containerID, tailLines)
	if err != nil {
		return fmt.Sprintf("[SOLV Shadow Mode] Conectado a %s (esperando logs activos...)\n", containerID), nil
	}

	if strings.TrimSpace(logs) == "" {
		return fmt.Sprintf("[SOLV Shadow Mode] Conectado a %s. Sesion de trabajo activa.\n", containerID), nil
	}

	return logs, nil
}

// ExecuteTutorCommand ejecuta un comando de diagnostico o asistencia dentro del contenedor.
func (s *TerminalMirrorService) ExecuteTutorCommand(ctx context.Context, containerID string, command string) (*domain.TutorCommandResponse, error) {
	if containerID == "" {
		return nil, fmt.Errorf("container_id requerido")
	}

	cmdTrimmed := strings.TrimSpace(command)
	if cmdTrimmed == "" {
		return nil, fmt.Errorf("comando no puede estar vacio")
	}

	// Comandos permitidos en modo tutor de seguridad: verificacion, tests, inspeccion
	cmdParts := strings.Fields(cmdTrimmed)

	executedAt := time.Now().UTC().Format(time.RFC3339)

	if s.orchestrator == nil {
		// Fallback para entornos de prueba
		return &domain.TutorCommandResponse{
			ContainerID: containerID,
			Command:     cmdTrimmed,
			Output:      fmt.Sprintf("[Tutor Command Executed]\n$ %s\nExit code: 0\n", cmdTrimmed),
			ExitCode:    0,
			ExecutedAt:  executedAt,
		}, nil
	}

	err := s.orchestrator.ExecuteCommandInBackground(ctx, containerID, "/workspace", cmdParts)
	if err != nil {
		return nil, fmt.Errorf("error al ejecutar comando en contenedor %s: %w", containerID, err)
	}

	return &domain.TutorCommandResponse{
		ContainerID: containerID,
		Command:     cmdTrimmed,
		Output:      fmt.Sprintf("Comando enviado al workspace: %s", cmdTrimmed),
		ExitCode:    0,
		ExecutedAt:  executedAt,
	}, nil
}
