package services

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"

	"solv-backend/internal/core/domain"
)

// runSmokeTest ejecuta un contenedor efímero aislado sin red ni permisos para chequear binarios declarados
func (w *TemplateAuditWorker) runSmokeTest(ctx context.Context, item *domain.AdminTemplateReviewItem) (string, string) {
	if len(item.ToolsDeclared) == 0 {
		return "skipped", "No se declararon herramientas requeridas (prueba no aplicable)"
	}

	// Construir script de verificación para /bin/sh o PATH
	var checks []string
	for _, tool := range item.ToolsDeclared {
		cleanTool := strings.TrimSpace(tool)
		if cleanTool == "" {
			continue
		}
		// Sanitizar nombre de herramienta para evitar inyección en el script del contenedor
		cleanTool = strings.ReplaceAll(cleanTool, ";", "")
		cleanTool = strings.ReplaceAll(cleanTool, "&", "")
		cleanTool = strings.ReplaceAll(cleanTool, "|", "")
		cleanTool = strings.ReplaceAll(cleanTool, "`", "")
		cleanTool = strings.ReplaceAll(cleanTool, "$", "")
		cleanTool = strings.TrimSpace(cleanTool)
		if cleanTool != "" {
			checks = append(checks, fmt.Sprintf(`command -v "%s" >/dev/null 2>&1 || which "%s" >/dev/null 2>&1 || missing="$missing %s"`, cleanTool, cleanTool, cleanTool))
		}
	}

	if len(checks) == 0 {
		return "skipped", "No se declararon herramientas requeridas válidas (prueba no aplicable)"
	}

	script := fmt.Sprintf(`missing=""; %s; if [ -n "$missing" ]; then echo "MISSING:$missing"; exit 1; else echo "ALL_FOUND"; exit 0; fi`, strings.Join(checks, "; "))

	testCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	ramMB := item.BaseRamMB
	if ramMB <= 0 {
		ramMB = 256
	}

	hostConfig := &container.HostConfig{
		NetworkMode:    "none",
		ReadonlyRootfs: true,
		Resources: container.Resources{
			Memory:   int64(ramMB) * 1024 * 1024,
			NanoCPUs: 500000000, // 0.5 CPU
		},
		AutoRemove: false,
	}

	containerConfig := &container.Config{
		Image:        item.DockerImage,
		Cmd:          []string{"/bin/sh", "-c", script},
		AttachStdout: true,
		AttachStderr: true,
	}

	resp, err := w.dockerCli.ContainerCreate(testCtx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "failed", fmt.Sprintf("Error creando contenedor efímero: %v", err)
	}
	containerID := resp.ID
	defer func() {
		_ = w.dockerCli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := w.dockerCli.ContainerStart(testCtx, containerID, container.StartOptions{}); err != nil {
		return "failed", fmt.Sprintf("Error arrancando contenedor efímero: %v", err)
	}

	statusCh, errCh := w.dockerCli.ContainerWait(testCtx, containerID, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return "failed", fmt.Sprintf("Error esperando contenedor efímero: %v", waitErr)
		}
	case res := <-statusCh:
		exitCode = res.StatusCode
	case <-testCtx.Done():
		return "failed", "Tiempo de espera agotado al verificar herramientas (timeout 20s)"
	}

	// Leer salida
	logsReader, err := w.dockerCli.ContainerLogs(testCtx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		if exitCode == 0 {
			return "passed", "Todas las herramientas declaradas están instaladas"
		}
		return "failed", "Fallo al verificar herramientas declaradas"
	}
	defer func() { _ = logsReader.Close() }()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logsReader)
	outputStr := strings.TrimSpace(stdoutBuf.String())

	if exitCode == 0 {
		return "passed", "Todas las herramientas declaradas están disponibles y ejecutables en el PATH"
	}

	if strings.Contains(outputStr, "MISSING:") {
		parts := strings.Split(outputStr, "MISSING:")
		missingList := strings.TrimSpace(parts[1])
		return "failed", fmt.Sprintf("Faltan las siguientes herramientas declaradas en la imagen: %s", missingList)
	}

	return "failed", fmt.Sprintf("La imagen no contiene las herramientas requeridas o no soporta /bin/sh (exit %d)", exitCode)
}
