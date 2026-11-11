package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"solv-backend/internal/core/domain"
)

// DependencyPreparationDockerRunner ejecuta las dos pasadas de una preparación:
// la instalación con la red que pida el perfil y la comprobación sin red.
// El contenedor no recibe privilegios, trabaja como el usuario declarado y su
// único punto de escritura es el directorio montado.
type DependencyPreparationDockerRunner struct {
	cli *client.Client
}

// NewDependencyPreparationDockerRunner construye el runner sobre el daemon local.
func NewDependencyPreparationDockerRunner(cli *client.Client) *DependencyPreparationDockerRunner {
	return &DependencyPreparationDockerRunner{cli: cli}
}

// RunInstall ejecuta la instalación del perfil.
func (r *DependencyPreparationDockerRunner) RunInstall(
	ctx context.Context,
	req domain.DependencyPreparationRequest,
	dir string,
	logWriter io.Writer,
) error {
	// La instalación sigue lo que declara `network`: las banderas de resolución
	// sin red no van acá o dejarían al gestor sin índice cuando sí puede usarlo.
	return r.run(ctx, req, dir, req.Network, req.InstallCommand(), logWriter)
}

// RunVerify comprueba el árbol con la red desactivada. Una preparación que solo
// funciona con red en el arranque no está lista.
func (r *DependencyPreparationDockerRunner) RunVerify(
	ctx context.Context,
	req domain.DependencyPreparationRequest,
	dir string,
	logWriter io.Writer,
) error {
	// La comprobación es la pasada que debe probar la resolución sin conexión.
	command := applyOfflineFlags(strings.TrimSpace(req.Verify), req.OfflineFlags)
	return r.run(ctx, req, dir, "none", command, logWriter)
}

// runtimeDirScript prepara el directorio de ejecución del usuario del contenedor
// antes de correr el gestor. La CLI de .NET crea un mutex con nombre antes de
// restaurar y lo respalda en el directorio XDG del usuario: si ese directorio no
// existe, la CLI muere con "cannot open the device or file specified" sin llegar
// a tocar el proyecto. `docker run -u uid:gid` no trae entrada en passwd ni
// directorio propio, así que se crean acá, como el mismo usuario que ejecuta el
// comando, y no desde el host con otro dueño.
func runtimeDirScript(home string) string {
	runtimeDir := strings.TrimRight(home, "/") + "/run"
	return fmt.Sprintf("mkdir -p %s %s; chmod 700 %s",
		shellQuote(home), shellQuote(runtimeDir), shellQuote(runtimeDir))
}

func (r *DependencyPreparationDockerRunner) run(
	ctx context.Context,
	req domain.DependencyPreparationRequest,
	dir string,
	network string,
	command string,
	logWriter io.Writer,
) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("comando vacío para la preparación")
	}
	networkMode := "bridge"
	if strings.TrimSpace(network) == "none" {
		networkMode = "none"
	}
	user := strings.TrimSpace(req.User)
	if user == "" {
		user = "1000:1000"
	}
	home := strings.TrimSpace(req.Home)
	if home == "" {
		home = strings.TrimRight(req.ProjectPath, "/") + "/.solv-home"
	}

	script := fmt.Sprintf("set -eu; cd %s; %s; %s", shellQuote(req.ProjectPath), runtimeDirScript(home), command)

	containerConfig := &container.Config{
		Image:        req.BaseImage,
		Cmd:          []string{"/bin/sh", "-c", script},
		WorkingDir:   req.ProjectPath,
		User:         user,
		Env:          preparationEnv(req, home),
		AttachStdout: true,
		AttachStderr: true,
	}
	hostConfig := &container.HostConfig{
		Binds:       []string{dir + ":" + req.ProjectPath},
		NetworkMode: container.NetworkMode(networkMode),
		CapDrop:     []string{"ALL"},
		SecurityOpt: []string{"no-new-privileges"},
		Resources: container.Resources{
			Memory:    int64(2) << 30,
			NanoCPUs:  1000000000,
			PidsLimit: int64Ptr(512),
		},
		AutoRemove: false,
	}

	resp, err := r.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return wrapTransientPreparationError(fmt.Errorf("no se pudo crear el contenedor de preparación: %w", err))
	}
	containerID := resp.ID
	defer func() {
		_ = r.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := r.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return wrapTransientPreparationError(fmt.Errorf("no se pudo arrancar el contenedor de preparación: %w", err))
	}

	statusCh, errCh := r.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case waitErr := <-errCh:
		if waitErr != nil {
			return wrapTransientPreparationError(fmt.Errorf("error esperando el contenedor de preparación: %w", waitErr))
		}
	case res := <-statusCh:
		exitCode = res.StatusCode
	case <-ctx.Done():
		return fmt.Errorf("tiempo de preparación agotado: %w", ctx.Err())
	}

	output := r.collectLogs(ctx, containerID, logWriter)
	if exitCode != 0 {
		detail := tail(output, 2000)
		if isTransientPreparationFailure(detail) {
			return fmt.Errorf("%w: el comando terminó con código %d: %s",
				domain.ErrDependencyPreparationTransient, exitCode, detail)
		}
		return fmt.Errorf("el comando terminó con código %d: %s", exitCode, detail)
	}
	return nil
}

// transientPreparationMarkers son las señales de un fallo que puede resolverse
// repitiendo el intento. Un error de compilación o un comando mal escrito no
// aparecen acá: esos son deterministas y reintentarlos solo hace perder tiempo.
var transientPreparationMarkers = []string{
	"connection refused",
	"connection reset",
	"i/o timeout",
	"no such host",
	"temporary failure in name resolution",
	"could not resolve host",
	"tls handshake timeout",
	"network is unreachable",
	"unexpected eof",
	"temporarily unavailable",
	"socket hang up",
	"econnreset",
	"etimedout",
	"eai_again",
	"econnrefused",
	"502 bad gateway",
	"503 service unavailable",
	"504 gateway timeout",
	"too many requests",
	"toomanyrequests",
}

func isTransientPreparationFailure(message string) bool {
	lower := strings.ToLower(message)
	for _, marker := range transientPreparationMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func wrapTransientPreparationError(err error) error {
	if err == nil {
		return nil
	}
	if isTransientPreparationFailure(err.Error()) {
		return fmt.Errorf("%w: %v", domain.ErrDependencyPreparationTransient, err)
	}
	return err
}

func (r *DependencyPreparationDockerRunner) collectLogs(ctx context.Context, containerID string, logWriter io.Writer) string {
	reader, err := r.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return ""
	}
	defer func() { _ = reader.Close() }()

	var stdout, stderr bytes.Buffer
	stdoutWriter := io.Writer(&stdout)
	stderrWriter := io.Writer(&stderr)
	if logWriter != nil {
		stdoutWriter = io.MultiWriter(&stdout, logWriter)
		stderrWriter = io.MultiWriter(&stderr, logWriter)
	}
	_, _ = stdcopy.StdCopy(stdoutWriter, stderrWriter, reader)
	return strings.TrimSpace(stdout.String() + "\n" + stderr.String())
}

// preparationEnv arma el entorno del contenedor. Las rutas de caché declaradas
// bajo /opt/solv/cache se remapean dentro del árbol mientras la caché compartida
// del nodo no exista: si quedaran fuera del montaje, la caché se perdería al
// terminar el contenedor y la comprobación sin red no tendría de dónde resolver.
func preparationEnv(req domain.DependencyPreparationRequest, home string) []string {
	env := []string{"HOME=" + home, "SOLV_DEPENDENCY_PREPARE=1"}
	// El directorio de ejecución se declara si el perfil no lo define: es lo que
	// permite a la CLI de .NET crear su mutex con nombre como usuario sin cuenta.
	if _, declared := req.CacheEnv["XDG_RUNTIME_DIR"]; !declared {
		env = append(env, "XDG_RUNTIME_DIR="+strings.TrimRight(home, "/")+"/run")
	}
	for name, value := range req.CacheEnv {
		env = append(env, name+"="+remapCachePath(req.ProjectPath, value))
	}
	return env
}

func remapCachePath(projectPath, value string) string {
	const sharedCacheRoot = "/opt/solv/cache"
	if strings.HasPrefix(value, sharedCacheRoot) {
		suffix := strings.TrimPrefix(value, sharedCacheRoot)
		return strings.TrimRight(projectPath, "/") + "/.solv-cache" + suffix
	}
	return value
}

// applyOfflineFlags coloca los modificadores del perfil en el comando: como
// asignaciones de entorno si tienen forma NAME=valor, como banderas al final si
// empiezan con guion.
func applyOfflineFlags(command, flags string) string {
	command = strings.TrimSpace(command)
	flags = strings.TrimSpace(flags)
	if flags == "" {
		return command
	}
	allAssignments := true
	for _, token := range strings.Fields(flags) {
		if !strings.Contains(token, "=") {
			allAssignments = false
			break
		}
	}
	if allAssignments {
		return flags + " " + command
	}
	if strings.HasPrefix(flags, "-") {
		return command + " " + flags
	}
	return flags + " " + command
}

func tail(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= limit {
		return trimmed
	}
	return "…" + trimmed[len(trimmed)-limit:]
}

func int64Ptr(value int64) *int64 { return &value }
