package lang

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"solv-backend/internal/core/domain"
)

func runContainerExecution(ctx context.Context, cli *client.Client, imageName string, fileName string, cmd []string, config domain.EvaluationRunConfig) (domain.TestCaseRunResult, error) {
	// 1. Crear directorio temporal en el host para montaje solo lectura (:ro)
	tmpDir, err := os.MkdirTemp("", "solv-eval-*")
	if err != nil {
		return domain.TestCaseRunResult{}, fmt.Errorf("failed to create temp dir for evaluation: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(filePath, []byte(config.SourceCode), 0644); err != nil {
		return domain.TestCaseRunResult{}, fmt.Errorf("failed to write solution file: %w", err)
	}

	// 2. Inspeccionar/Descargar imagen
	_, _, err = cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil && errdefs.IsNotFound(err) {
		out, pullErr := cli.ImagePull(ctx, imageName, image.PullOptions{})
		if pullErr != nil {
			return domain.TestCaseRunResult{}, fmt.Errorf("failed to pull image %s: %w", imageName, pullErr)
		}
		_, _ = io.Copy(io.Discard, out)
		out.Close()
	}

	// 3. Aislamiento Estricto (NetworkMode: "none", RAM Limit, :ro)
	memLimit := int64(config.MemoryLimitMB)
	if memLimit <= 0 {
		memLimit = 128
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/runner:ro", tmpDir),
		},
		Resources: container.Resources{
			Memory:     memLimit * 1024 * 1024,
			MemorySwap: memLimit * 1024 * 1024,
		},
		NetworkMode: "none",
	}

	containerConfig := &container.Config{
		Image:        imageName,
		Cmd:          cmd,
		OpenStdin:    true,
		StdinOnce:    true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	// 4. Crear Contenedor Efímero
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return domain.TestCaseRunResult{}, fmt.Errorf("failed to create evaluation container: %w", err)
	}
	containerID := resp.ID

	defer func() {
		_ = cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	// 5. Streaming IO
	attachResp, err := cli.ContainerAttach(ctx, containerID, container.AttachOptions{
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Stream: true,
	})
	if err != nil {
		return domain.TestCaseRunResult{}, fmt.Errorf("failed to attach to container streams: %w", err)
	}
	defer attachResp.Close()

	// Inyectar stdin
	go func() {
		_, _ = attachResp.Conn.Write([]byte(config.TestCase.Input))
		_ = attachResp.CloseWrite()
	}()

	// 6. Iniciar Contenedor
	startTime := time.Now()
	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return domain.TestCaseRunResult{}, fmt.Errorf("failed to start evaluation container: %w", err)
	}

	// 7. Esperar salida con Límite de Tiempo (TLE)
	timeLimitMS := config.TimeLimitMS
	if timeLimitMS <= 0 {
		timeLimitMS = 2000
	}
	timeoutDuration := time.Duration(timeLimitMS) * time.Millisecond

	evalCtx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	statusCh, errCh := cli.ContainerWait(evalCtx, containerID, container.WaitConditionNotRunning)

	var stdoutBuf, stderrBuf bytes.Buffer
	doneCopy := make(chan error, 1)

	go func() {
		_, err := stdCopy(&stdoutBuf, &stderrBuf, attachResp.Reader)
		doneCopy <- err
	}()

	select {
	case <-evalCtx.Done():
		timeoutVal := 5
		_ = cli.ContainerStop(context.Background(), containerID, container.StopOptions{Timeout: &timeoutVal})
		return domain.TestCaseRunResult{
			Verdict:       domain.VerdictTLE,
			ExecutionTime: timeoutDuration,
			ErrorDetails:  fmt.Sprintf("Tiempo límite de ejecución superado (%d ms)", timeLimitMS),
		}, nil

	case err := <-errCh:
		if err != nil {
			return domain.TestCaseRunResult{}, fmt.Errorf("container wait error: %w", err)
		}

	case status := <-statusCh:
		execTime := time.Since(startTime)
		<-doneCopy

		if status.StatusCode != 0 {
			errStr := strings.TrimSpace(stderrBuf.String())
			if errStr == "" {
				errStr = strings.TrimSpace(stdoutBuf.String())
			}
			return domain.TestCaseRunResult{
				Verdict:       domain.VerdictRE,
				ExecutionTime: execTime,
				StdErr:        errStr,
				ErrorDetails:  fmt.Sprintf("Error de ejecución (Exit Code %d): %s", status.StatusCode, errStr),
			}, nil
		}

		actualOutput := strings.TrimSpace(stdoutBuf.String())
		expectedOutput := strings.TrimSpace(config.TestCase.ExpectedOutput)

		if actualOutput == expectedOutput {
			return domain.TestCaseRunResult{
				Verdict:       domain.VerdictAC,
				ExecutionTime: execTime,
				ActualOutput:  actualOutput,
			}, nil
		}

		return domain.TestCaseRunResult{
			Verdict:       domain.VerdictWA,
			ExecutionTime: execTime,
			ActualOutput:  actualOutput,
			ErrorDetails:  fmt.Sprintf("Respuesta incorrecta. Esperado: %q, Obtenido: %q", expectedOutput, actualOutput),
		}, nil
	}

	return domain.TestCaseRunResult{
		Verdict:      domain.VerdictRE,
		ErrorDetails: "Execution completed abnormally",
	}, nil
}
