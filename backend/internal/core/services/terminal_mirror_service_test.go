package services

import (
	"context"
	"strings"
	"testing"
)

func TestTerminalMirrorService_CleanTerminalOutput(t *testing.T) {
	svc := NewTerminalMirrorService(nil)

	// Simular stream multiplexado de Docker: header de 8 bytes [1, 0, 0, 0, 0, 0, 0, 5] + "hello"
	rawMux := []byte{1, 0, 0, 0, 0, 0, 0, 5, 'h', 'e', 'l', 'l', 'o'}
	cleaned := svc.CleanTerminalOutput(rawMux)
	if cleaned != "hello" {
		t.Fatalf("expected 'hello', got %q", cleaned)
	}

	// Texto plano sin header
	plain := []byte("plain text line\n")
	cleanedPlain := svc.CleanTerminalOutput(plain)
	if cleanedPlain != "plain text line\n" {
		t.Fatalf("expected 'plain text line\\n', got %q", cleanedPlain)
	}
}

func TestTerminalMirrorService_GetInitialTerminalBuffer(t *testing.T) {
	svc := NewTerminalMirrorService(nil)
	ctx := context.Background()

	// Contenedor vacio
	buf, err := svc.GetInitialTerminalBuffer(ctx, "", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf, "Contenedor no asignado") {
		t.Fatalf("expected unassigned message, got %q", buf)
	}

	// Mock con container ID
	buf, err = svc.GetInitialTerminalBuffer(ctx, "container-123", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf, "container-123") {
		t.Fatalf("expected container id in buffer, got %q", buf)
	}
}

func TestTerminalMirrorService_ExecuteTutorCommand(t *testing.T) {
	svc := NewTerminalMirrorService(nil)
	ctx := context.Background()

	// Error si containerID esta vacio
	_, err := svc.ExecuteTutorCommand(ctx, "", "ls -la")
	if err == nil {
		t.Fatalf("expected error for empty container id")
	}

	// Error si comando esta vacio
	_, err = svc.ExecuteTutorCommand(ctx, "c123", "   ")
	if err == nil {
		t.Fatalf("expected error for empty command")
	}

	// Ejecucion exitosa
	resp, err := svc.ExecuteTutorCommand(ctx, "c123", "pytest tests/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ContainerID != "c123" || resp.Command != "pytest tests/" || resp.ExitCode != 0 {
		t.Fatalf("invalid tutor command response: %+v", resp)
	}
}
