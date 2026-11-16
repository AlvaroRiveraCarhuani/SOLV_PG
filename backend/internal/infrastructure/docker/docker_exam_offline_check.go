package docker

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/pkg/stdcopy"

	"solv-backend/internal/core/domain"
)

const (
	// examOfflineCheckScript se escribe en la raíz del volumen temporal. El
	// setup y el arranque viajan dentro del script para no pelear con comillas.
	examOfflineCheckScript = "solv-offline-check.sh"
	// examOfflineCheckLabel identifica la corrida y permite encontrarla.
	examOfflineCheckLabel = "solv.exam.offline-check"
	// examOfflineCheckLimit acota la corrida completa: una instalación eterna
	// no puede dejar colgada la publicación.
	examOfflineCheckLimit = 5 * time.Minute
	// examOfflineRunTimeout deja arrancar un servidor y lo corta: 124 cuenta
	// como arranque correcto porque el proceso seguía vivo.
	examOfflineRunTimeout = 20
	// examOfflineOutputCap acota el log que viaja al servicio.
	examOfflineOutputCap = 8000
)

// VerifyExamProjectOffline arma un volumen temporal con el proyecto base, lo
// corre en la red interna del examen y devuelve qué paso falló. El contenedor
// no publica puertos, no tiene salida y se descarta con su volumen.
func (c *Client) VerifyExamProjectOffline(ctx context.Context, check domain.ExamProjectOfflineCheck) (*domain.ExamProjectOfflineCheckResult, error) {
	started := time.Now()
	if strings.TrimSpace(check.Image) == "" {
		return nil, fmt.Errorf("exam offline check requires an image")
	}
	if err := c.ensureExamNetwork(ctx); err != nil {
		return nil, err
	}

	name := fmt.Sprintf("solv-exam-check-%d", started.UnixNano())
	if _, err := c.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   name,
		Driver: "local",
		Labels: mergeTestRunLabels(map[string]string{"solv.managed": "true", "solv.purpose": "exam-offline-check"}),
	}); err != nil {
		return nil, fmt.Errorf("failed to create exam offline check volume: %w", err)
	}
	defer func() { _ = c.RemoveVolume(context.Background(), name) }()

	files := make(map[string]string, len(check.Files)+1)
	for path, content := range check.Files {
		files[path] = content
	}
	files[examOfflineCheckScript] = buildExamOfflineCheckScript(check)
	if err := c.WritePracticeEnvironmentVolumeFiles(ctx, name, files); err != nil {
		return &domain.ExamProjectOfflineCheckResult{
			Status:     domain.ExamOfflineFailed,
			Step:       "proyecto",
			Output:     err.Error(),
			DurationMS: time.Since(started).Milliseconds(),
		}, nil
	}

	memoryMB := check.MemoryLimitMB
	if memoryMB <= 0 {
		memoryMB = 1024
	}

	resp, err := c.cli.ContainerCreate(ctx, &container.Config{
		Image:      check.Image,
		Cmd:        []string{"sh", "/home/workspace/" + examOfflineCheckScript},
		WorkingDir: "/home/workspace",
		Labels: mergeTestRunLabels(map[string]string{
			examNoEgressLabel:     "true",
			examOfflineCheckLabel: name,
		}),
	}, &container.HostConfig{
		NetworkMode: container.NetworkMode(examNetworkName),
		SecurityOpt: []string{"no-new-privileges:true"},
		Mounts: []mount.Mount{{
			Type:   mount.TypeVolume,
			Source: name,
			Target: "/home/workspace",
		}},
		Resources: container.Resources{
			Memory:   int64(memoryMB) << 20,
			NanoCPUs: 1_000_000_000,
		},
	}, nil, nil, name+"-run")
	if err != nil {
		return nil, fmt.Errorf("failed to create exam offline check container: %w", err)
	}
	defer func() {
		_ = c.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
	}()

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start exam offline check: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, examOfflineCheckLimit)
	defer cancel()
	statusCh, errCh := c.cli.ContainerWait(waitCtx, resp.ID, container.WaitConditionNotRunning)
	var exitCode int64
	timedOut := false
	select {
	case status := <-statusCh:
		exitCode = status.StatusCode
	case <-errCh:
		timedOut = true
	}

	output := c.examOfflineCheckLogs(ctx, resp.ID)
	result := &domain.ExamProjectOfflineCheckResult{
		Status:     domain.ExamOfflinePassed,
		Step:       "setup",
		Output:     output,
		DurationMS: time.Since(started).Milliseconds(),
	}
	if strings.Contains(output, "::solv-step:run") {
		result.Step = "run"
	}

	switch {
	case timedOut:
		result.Status = domain.ExamOfflineFailed
		result.Output = fmt.Sprintf("La verificación superó %s en el paso %q.", examOfflineCheckLimit, result.Step)
		result.Step = "tiempo"
	case exitCode == 90:
		result.Status = domain.ExamOfflineFailed
		result.Step = "proyecto"
	case strings.Contains(output, "::solv-step-failed:setup"):
		result.Status = domain.ExamOfflineFailed
		result.Step = "setup"
	case strings.Contains(output, "::solv-step-failed:run"):
		result.Status = domain.ExamOfflineFailed
		result.Step = "run"
	case exitCode != 0:
		result.Status = domain.ExamOfflineFailed
	}

	result.Output = trimExamOfflineOutput(output)
	return result, nil
}

// examOfflineCheckLogs lee la salida del contenedor de verificación. El log se
// pide después de esperar; si el contenedor todavía corre, devuelve lo que haya.
func (c *Client) examOfflineCheckLogs(ctx context.Context, containerID string) string {
	reader, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return ""
	}
	defer reader.Close()

	var out bytes.Buffer
	_, _ = stdcopy.StdCopy(&out, &out, reader)
	return out.String()
}

// buildExamOfflineCheckScript arma el script que corre dentro del contenedor.
// Los marcadores permiten saber si falló la preparación o el arranque, y el
// heredoc evita reinterpretar comillas del setup o del comando.
func buildExamOfflineCheckScript(check domain.ExamProjectOfflineCheck) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	b.WriteString("cd /home/workspace || exit 90\n")
	b.WriteString("printf '::solv-step:setup\\n'\n")
	b.WriteString("cat > /tmp/solv-setup.sh <<'SOLV_SETUP_EOF'\n")
	b.WriteString(check.SetupScript)
	b.WriteString("\nSOLV_SETUP_EOF\n")
	b.WriteString("sh /tmp/solv-setup.sh || { printf '::solv-step-failed:setup\\n'; exit 10; }\n")

	if run := strings.TrimSpace(check.RunCommand); run != "" {
		b.WriteString("printf '::solv-step:run\\n'\n")
		b.WriteString("cat > /tmp/solv-run.sh <<'SOLV_RUN_EOF'\n")
		b.WriteString(run)
		b.WriteString("\nSOLV_RUN_EOF\n")
		fmt.Fprintf(&b, "timeout %d sh /tmp/solv-run.sh\n", examOfflineRunTimeout)
		b.WriteString("code=$?\n")
		b.WriteString("printf '::solv-step-exit:%s\\n' \"$code\"\n")
		b.WriteString("if [ \"$code\" -ne 0 ] && [ \"$code\" -ne 124 ]; then printf '::solv-step-failed:run\\n'; exit 11; fi\n")
	}

	b.WriteString("exit 0\n")
	return b.String()
}

// trimExamOfflineOutput quita los marcadores del log y recorta el final: es lo
// que el docente ve como motivo del rechazo.
func trimExamOfflineOutput(output string) string {
	lines := strings.Split(output, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "::solv-") {
			continue
		}
		filtered = append(filtered, line)
	}
	cleaned := strings.TrimSpace(strings.Join(filtered, "\n"))
	if len(cleaned) > examOfflineOutputCap {
		cleaned = "…" + cleaned[len(cleaned)-examOfflineOutputCap:]
	}
	return cleaned
}
