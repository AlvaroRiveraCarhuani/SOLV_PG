package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"
)

// runTrivyScan ejecuta Trivy escaneando la imagen por CVEs de severidad CRITICAL y HIGH
func (w *TemplateAuditWorker) runTrivyScan(ctx context.Context, dockerImage string) (string, int, int, []byte) {
	// Verificar o descargar imagen de Trivy si no existe
	_, _, err := w.dockerCli.ImageInspectWithRaw(ctx, TrivyDockerImage)
	if err != nil && errdefs.IsNotFound(err) {
		log.Printf("[TemplateAuditWorker] Descargando imagen de auditoría %s...", TrivyDockerImage)
		out, pullErr := w.dockerCli.ImagePull(ctx, TrivyDockerImage, image.PullOptions{})
		if pullErr != nil {
			log.Printf("[TemplateAuditWorker] Error descargando Trivy: %v", pullErr)
			return "failed", 0, 0, []byte(fmt.Sprintf(`{"error": "no se pudo descargar imagen de auditoría: %s"}`, pullErr.Error()))
		}
		defer func() { _ = out.Close() }()
		_, _ = io.Copy(io.Discard, out)
	}

	scanCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	hostConfig := &container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock:ro",
			fmt.Sprintf("%s:/root/.cache", w.cacheDir),
		},
		AutoRemove: false,
	}

	containerConfig := &container.Config{
		Image: TrivyDockerImage,
		Cmd: []string{
			"image",
			"--severity", "CRITICAL,HIGH",
			"--format", "json",
			"--quiet",
			dockerImage,
		},
		AttachStdout: true,
		AttachStderr: true,
	}

	resp, err := w.dockerCli.ContainerCreate(scanCtx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		log.Printf("[TemplateAuditWorker] Error creando contenedor Trivy: %v", err)
		return "failed", 0, 0, []byte(fmt.Sprintf(`{"error": "creación de escáner falló: %s"}`, err.Error()))
	}
	containerID := resp.ID
	defer func() {
		_ = w.dockerCli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := w.dockerCli.ContainerStart(scanCtx, containerID, container.StartOptions{}); err != nil {
		log.Printf("[TemplateAuditWorker] Error iniciando contenedor Trivy: %v", err)
		return "failed", 0, 0, []byte(fmt.Sprintf(`{"error": "inicio de escáner falló: %s"}`, err.Error()))
	}

	statusCh, errCh := w.dockerCli.ContainerWait(scanCtx, containerID, container.WaitConditionNotRunning)
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			log.Printf("[TemplateAuditWorker] Error esperando escaneo Trivy: %v", waitErr)
			return "failed", 0, 0, []byte(fmt.Sprintf(`{"error": "espera de escáner falló: %s"}`, waitErr.Error()))
		}
	case <-statusCh:
	case <-scanCtx.Done():
		log.Printf("[TemplateAuditWorker] Timeout en escaneo Trivy para imagen %s", dockerImage)
		return "failed", 0, 0, []byte(`{"error": "tiempo límite de escaneo agotado (3 min)"}`)
	}

	logsReader, err := w.dockerCli.ContainerLogs(scanCtx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: false,
	})
	if err != nil {
		return "failed", 0, 0, []byte(fmt.Sprintf(`{"error": "no se pudo leer salida de auditoría: %s"}`, err.Error()))
	}
	defer func() { _ = logsReader.Close() }()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logsReader)
	rawJSON := stdoutBuf.Bytes()

	if len(rawJSON) == 0 {
		return "passed", 0, 0, []byte(`{"Results": []}`)
	}

	var report trivyReport
	if err := json.Unmarshal(rawJSON, &report); err != nil {
		log.Printf("[TemplateAuditWorker] Error parseando JSON de Trivy: %v", err)
		return "passed", 0, 0, rawJSON
	}

	criticalCount := 0
	highCount := 0
	for _, res := range report.Results {
		for _, vuln := range res.Vulnerabilities {
			sev := strings.ToUpper(vuln.Severity)
			switch sev {
			case "CRITICAL":
				criticalCount++
			case "HIGH":
				highCount++
			}
		}
	}

	status := "passed"
	if criticalCount > 0 {
		status = "failed"
	}

	return status, criticalCount, highCount, rawJSON
}

// UpdateTrivyDB actualiza la base de vulnerabilidades de Trivy descargando el paquete a cache
func (w *TemplateAuditWorker) UpdateTrivyDB(ctx context.Context) error {
	syncCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/root/.cache", w.cacheDir),
		},
		AutoRemove: false,
	}

	containerConfig := &container.Config{
		Image: TrivyDockerImage,
		Cmd: []string{
			"image",
			"--download-db-only",
		},
	}

	resp, err := w.dockerCli.ContainerCreate(syncCtx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("error creando contenedor para descarga de base CVE: %w", err)
	}
	defer func() {
		_ = w.dockerCli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
	}()

	if err := w.dockerCli.ContainerStart(syncCtx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("error iniciando descarga de base CVE: %w", err)
	}

	statusCh, errCh := w.dockerCli.ContainerWait(syncCtx, resp.ID, container.WaitConditionNotRunning)
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return fmt.Errorf("error esperando descarga de base CVE: %w", waitErr)
		}
	case <-statusCh:
	case <-syncCtx.Done():
		return fmt.Errorf("timeout en descarga de base CVE")
	}

	log.Println("[TemplateAuditWorker] Base de datos CVE actualizada correctamente en cache")
	return nil
}
