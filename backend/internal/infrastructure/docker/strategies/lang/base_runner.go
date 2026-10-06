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
	"solv-backend/internal/core/services/comparators"
)

// maxBuildStderr limita el stderr de compilación devuelto en un CE.
const maxBuildStderr = 4000

// asciiWhitespace es el cutset para sanear bordes sin tocar la comparación
// (el recorte de comparación vive solo en el comparador exact del registro).
const asciiWhitespace = " \t\n\v\f\r"

// BuildResult es el resultado de la fase de compilación (D-EJ-03).
type BuildResult struct {
	Verdict       domain.Verdict
	ExecutionTime time.Duration
	ErrorDetails  string
}

// resolveCrashVerdict mapea la salida del contenedor a veredicto: el OOM
// kill de cgroups es MLE, cualquier otra salida distinta de cero es RE.
func resolveCrashVerdict(exitCode int64, oomKilled bool) domain.Verdict {
	if oomKilled {
		return domain.VerdictMLE
	}
	return domain.VerdictRE
}

// sanitizeBuildStderr quita rutas internas del host y limita el tamaño del
// stderr de compilación antes de devolverlo en un CE.
func sanitizeBuildStderr(raw, hostTmpDir string) string {
	s := strings.Trim(raw, asciiWhitespace)
	if hostTmpDir != "" {
		s = strings.ReplaceAll(s, hostTmpDir, "")
	}
	if len(s) > maxBuildStderr {
		s = s[:maxBuildStderr]
	}
	return strings.Trim(s, asciiWhitespace)
}

// imageDigestOf extrae el digest de una referencia repo@digest. Sin digest
// devuelve vacío (las imágenes del juez van pineadas, spec 3.1).
func imageDigestOf(imageName string) string {
	if i := strings.LastIndex(imageName, "@"); i >= 0 {
		return imageName[i+1:]
	}
	return ""
}

// checkerOutcome es la salida del checker docente del comparador custom ya
// ejecutado en su sidecar (sin red, solo lectura, con timeout).
type checkerOutcome struct {
	verdict domain.Verdict
	message string
}

// checkerResultToRunResult convierte la salida del checker a resultado de
// caso: el fallo o timeout del checker es VE (D-EJ-02).
func checkerResultToRunResult(out checkerOutcome, execTime time.Duration) domain.TestCaseRunResult {
	res := domain.TestCaseRunResult{
		Verdict:       out.verdict,
		ExecutionTime: execTime,
	}
	if out.message != "" {
		res.ErrorDetails = out.message
	}
	return res
}

// RunBuildPhase ejecuta la fase de compilación en un contenedor efímero con
// el mismo aislamiento que run. Sin comando no hay fase (interpretados).
// El fallo de compilación es CE con stderr sanitizado (D-EJ-03).
func RunBuildPhase(ctx context.Context, cli *client.Client, imageName, fileName, sourceCode, buildCmd string, timeoutMS, memoryMB int) (BuildResult, error) {
	if strings.Trim(buildCmd, asciiWhitespace) == "" {
		return BuildResult{Verdict: domain.VerdictAC}, nil
	}

	tmpDir, err := os.MkdirTemp("", "solv-build-*")
	if err != nil {
		return BuildResult{}, fmt.Errorf("failed to create temp dir for build: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, fileName), []byte(sourceCode), 0644); err != nil {
		return BuildResult{}, fmt.Errorf("failed to write solution file: %w", err)
	}

	ensureImage(ctx, cli, imageName)

	memLimit := int64(memoryMB)
	if memLimit <= 0 {
		memLimit = 512
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
		Cmd:          []string{"sh", "-c", buildCmd},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return BuildResult{}, fmt.Errorf("failed to create build container: %w", err)
	}
	containerID := resp.ID
	defer func() {
		_ = cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	attachResp, err := cli.ContainerAttach(ctx, containerID, container.AttachOptions{
		Stdout: true,
		Stderr: true,
		Stream: true,
	})
	if err != nil {
		return BuildResult{}, fmt.Errorf("failed to attach to build streams: %w", err)
	}
	defer attachResp.Close()

	startTime := time.Now()
	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return BuildResult{}, fmt.Errorf("failed to start build container: %w", err)
	}

	buildTimeout := timeoutMS
	if buildTimeout <= 0 {
		buildTimeout = 10000
	}
	timeoutDuration := time.Duration(buildTimeout) * time.Millisecond
	buildCtx, cancel := context.WithTimeout(ctx, timeoutDuration)
	defer cancel()

	statusCh, errCh := cli.ContainerWait(buildCtx, containerID, container.WaitConditionNotRunning)

	var stdoutBuf, stderrBuf bytes.Buffer
	doneCopy := make(chan error, 1)
	go func() {
		_, err := stdCopy(&stdoutBuf, &stderrBuf, attachResp.Reader)
		doneCopy <- err
	}()

	select {
	case <-buildCtx.Done():
		timeoutVal := 5
		_ = cli.ContainerStop(context.Background(), containerID, container.StopOptions{Timeout: &timeoutVal})
		return BuildResult{
			Verdict:       domain.VerdictCE,
			ExecutionTime: timeoutDuration,
			ErrorDetails:  fmt.Sprintf("Build timeout superado (%d ms)", buildTimeout),
		}, nil
	case err := <-errCh:
		if err != nil {
			return BuildResult{}, fmt.Errorf("build container wait error: %w", err)
		}
	case status := <-statusCh:
		execTime := time.Since(startTime)
		<-doneCopy
		if status.StatusCode != 0 {
			errStr := sanitizeBuildStderr(stderrBuf.String()+"\n"+stdoutBuf.String(), tmpDir)
			if errStr == "" {
				errStr = fmt.Sprintf("exit code %d", status.StatusCode)
			}
			return BuildResult{
				Verdict:       domain.VerdictCE,
				ExecutionTime: execTime,
				ErrorDetails:  fmt.Sprintf("Error de compilación (Exit Code %d): %s", status.StatusCode, errStr),
			}, nil
		}
		return BuildResult{Verdict: domain.VerdictAC, ExecutionTime: execTime}, nil
	}

	return BuildResult{Verdict: domain.VerdictCE, ErrorDetails: "Build completed abnormally"}, nil
}

// oomKilled informa si el contenedor fue terminado por el OOM killer de
// cgroups (veredicto MLE). Sin daemon disponible devuelve falso.
func oomKilled(cli *client.Client, containerID string) bool {
	inspectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, err := cli.ContainerInspect(inspectCtx, containerID)
	if err != nil {
		return false
	}
	return info.State != nil && info.State.OOMKilled
}
func ensureImage(ctx context.Context, cli *client.Client, imageName string) {
	_, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil && errdefs.IsNotFound(err) {
		out, pullErr := cli.ImagePull(ctx, imageName, image.PullOptions{})
		if pullErr != nil {
			return
		}
		_, _ = io.Copy(io.Discard, out)
		out.Close()
	}
}

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
			errStr := strings.Trim(stderrBuf.String(), asciiWhitespace)
			if errStr == "" {
				errStr = strings.Trim(stdoutBuf.String(), asciiWhitespace)
			}
			return domain.TestCaseRunResult{
				Verdict:       resolveCrashVerdict(status.StatusCode, oomKilled(cli, containerID)),
				ExecutionTime: execTime,
				StdErr:        errStr,
				ErrorDetails:  fmt.Sprintf("Error de ejecución (Exit Code %d): %s", status.StatusCode, errStr),
				ImageDigest:   imageDigestOf(imageName),
			}, nil
		}

		// Comparación vía registro PR2 (D-EJ-01/D-EJ-02): el recorte de bordes
		// vive solo en el comparador exact (corrige DESVÍO-02).
		spec := comparators.DefaultSpec()
		if config.Comparator != nil && config.Comparator.ID != "" {
			spec = comparators.Spec{
				ID:     config.Comparator.ID,
				Params: config.Comparator.Params,
			}
		}
		cmp, cmpErr := comparators.Compare(
			spec,
			config.TestCase.ExpectedOutput,
			stdoutBuf.String(),
		)
		if cmpErr != nil {
			return domain.TestCaseRunResult{}, fmt.Errorf("comparator error: %w", cmpErr)
		}

		actualOutput := stdoutBuf.String()
		if cmp.Verdict == comparators.VerdictAC {
			return domain.TestCaseRunResult{
				Verdict:       domain.VerdictAC,
				ExecutionTime: execTime,
				ActualOutput:  actualOutput,
				ImageDigest:   imageDigestOf(imageName),
			}, nil
		}

		return domain.TestCaseRunResult{
			Verdict:       domain.VerdictWA,
			ExecutionTime: execTime,
			ActualOutput:  actualOutput,
			ErrorDetails:  fmt.Sprintf("Respuesta incorrecta. Esperado: %q, Obtenido: %q", config.TestCase.ExpectedOutput, actualOutput),
			ImageDigest:   imageDigestOf(imageName),
		}, nil
	}

	return domain.TestCaseRunResult{
		Verdict:      domain.VerdictRE,
		ErrorDetails: "Execution completed abnormally",
	}, nil
}
