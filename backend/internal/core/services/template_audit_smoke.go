package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"

	"solv-backend/internal/core/domain"
)

// isDockerInfrastructureError distingue un daemon caído o una conexión
// interrumpida de un fallo real de la imagen.
func isDockerInfrastructureError(err error) bool {
	if err == nil {
		return false
	}
	return client.IsErrConnectionFailed(err) ||
		errdefs.IsUnavailable(err) ||
		errdefs.IsSystem(err) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled)
}

// isImageReferenceError reconoce los errores que sí describen una imagen
// inexistente o inaccesible. El resto de fallos de preparación se reintenta.
func isImageReferenceError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{
		"manifest unknown",
		"manifest for",
		"not found",
		"pull access denied",
		"unauthorized",
		"no such image",
		"invalid reference format",
		"repository does not exist",
		"name unknown",
		"denied: requested access",
		"tag invalid",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

// ensureImageAvailable comprueba que la imagen esté en el daemon y la descarga
// si falta. Es el paso previo del smoke para imágenes públicas.
func (w *TemplateAuditWorker) ensureImageAvailable(ctx context.Context, imageRef string) error {
	if _, _, err := w.dockerCli.ImageInspectWithRaw(ctx, imageRef); err == nil {
		return nil
	} else if !errdefs.IsNotFound(err) {
		return err
	}

	pullCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	reader, err := w.dockerCli.ImagePull(pullCtx, imageRef, image.PullOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	_, err = io.Copy(io.Discard, reader)
	return err
}

// runSmokeTest ejecuta un contenedor efímero aislado sin red ni permisos para chequear binarios declarados
func (w *TemplateAuditWorker) runSmokeTest(ctx context.Context, item *domain.AdminTemplateReviewItem) (string, string) {
	// La imagen debe estar en el daemon antes del smoke y del escaneo. Una
	// imagen pública puede faltar; se descarga una sola vez.
	if err := w.ensureImageAvailable(ctx, item.DockerImage); err != nil {
		status := auditStatusError
		if isImageReferenceError(err) && !isDockerInfrastructureError(err) {
			status = auditStatusFailed
		}
		return status, fmt.Sprintf("No se pudo preparar la imagen para la comprobación: %v", err)
	}

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
		if isDockerInfrastructureError(err) {
			return auditStatusError, fmt.Sprintf("Error de infraestructura creando contenedor efímero: %v", err)
		}
		return auditStatusFailed, fmt.Sprintf("Error creando contenedor efímero: %v", err)
	}
	containerID := resp.ID
	defer func() {
		_ = w.dockerCli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := w.dockerCli.ContainerStart(testCtx, containerID, container.StartOptions{}); err != nil {
		if isDockerInfrastructureError(err) {
			return auditStatusError, fmt.Sprintf("Error de infraestructura arrancando contenedor efímero: %v", err)
		}
		return auditStatusFailed, fmt.Sprintf("Error arrancando contenedor efímero: %v", err)
	}

	statusCh, errCh := w.dockerCli.ContainerWait(testCtx, containerID, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return auditStatusError, fmt.Sprintf("Error esperando contenedor efímero: %v", waitErr)
		}
	case res := <-statusCh:
		exitCode = res.StatusCode
	case <-testCtx.Done():
		return auditStatusError, "Tiempo de espera agotado al verificar herramientas (timeout 20s); se reintentará"
	}

	// Leer salida
	logsReader, err := w.dockerCli.ContainerLogs(testCtx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		if isDockerInfrastructureError(err) {
			return auditStatusError, fmt.Sprintf("Error de infraestructura leyendo la salida del smoke: %v", err)
		}
		if exitCode == 0 {
			return auditStatusPassed, "Todas las herramientas declaradas están instaladas"
		}
		return auditStatusFailed, "Fallo al verificar herramientas declaradas"
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
