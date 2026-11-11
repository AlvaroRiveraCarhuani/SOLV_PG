//go:build integration

package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"

	"solv-backend/internal/core/domain"
)

// TestDependencyPreparationRunnerIntegration ejercita las dos pasadas contra
// Docker de verdad: un entorno virtual de Python se crea dentro del árbol como
// usuario no privilegiado, la instalación usa la red y la comprobación corre con
// la red apagada. Es la prueba que los tests unitarios no pueden dar: que el
// usuario del contenedor escriba en el árbol y que el resultado resuelva sin
// conexión.
func TestDependencyPreparationRunnerIntegration(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de Docker: %v", err)
	}
	runner := NewDependencyPreparationDockerRunner(cli)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("six==1.16.0\n"), 0o644); err != nil {
		t.Fatalf("no se pudo escribir el manifiesto: %v", err)
	}

	// El contenedor corre con el mismo usuario que la prueba: así se comprueba
	// que el árbol es escribible sin privilegios, que es la condición real.
	user := fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
	request := domain.DependencyPreparationRequest{
		TenantID:       "tenant-integration",
		VersionID:      "version-integration",
		ComponentSlug:  "python",
		BaseImage:      "python:3.12-slim-bookworm",
		ProfileVersion: 1,
		ProjectPath:    "/project",
		User:           user,
		Install:        "python -m venv .venv && .venv/bin/pip install --no-cache-dir -r requirements.txt",
		Verify:         ".venv/bin/python -c 'import six'",
		Network:        "build",
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("la solicitud del test no es válida: %v", err)
	}

	logPath := filepath.Join(t.TempDir(), "preparacion.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatalf("no se pudo crear el log: %v", err)
	}
	defer func() { _ = logFile.Close() }()

	if err := runner.RunInstall(ctx, request, dir, logFile); err != nil {
		content, _ := os.ReadFile(logPath)
		t.Fatalf("la instalación debía completarse: %v\n%s", err, tail(string(content), 2000))
	}
	// bin/python es un symlink a una ruta del contenedor que no existe en el
	// host; pyvenv.cfg es un archivo real y prueba lo mismo.
	if _, err := os.Stat(filepath.Join(dir, ".venv", "pyvenv.cfg")); err != nil {
		t.Fatalf("el entorno virtual debía quedar dentro del árbol: %v", err)
	}

	if err := runner.RunVerify(ctx, request, dir, logFile); err != nil {
		content, _ := os.ReadFile(logPath)
		t.Fatalf("la comprobación sin red debía pasar: %v\n%s", err, tail(string(content), 2000))
	}

	content, _ := os.ReadFile(logPath)
	if strings.TrimSpace(string(content)) == "" {
		t.Fatal("el log de la preparación no debía quedar vacío")
	}
}
