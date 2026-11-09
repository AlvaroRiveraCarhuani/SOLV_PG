package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"

	"solv-backend/internal/core/domain"
)

const (
	// environmentServerPort es el único puerto que un environment expone al host.
	environmentServerPort = "3000/tcp"

	// pureRuntimePracticeEnvironmentCmd se usa en imágenes que no traen OpenVSCode: prepara el
	// environment, ejecuta el setup_script de la plantilla una sola vez y arranca
	// OpenVSCode abriendo directamente el directorio del proyecto.
	pureRuntimePracticeEnvironmentCmd = `mkdir -p /home/workspace && chmod 0777 /home/workspace && if [ -n "$SOLV_SETUP_SCRIPT" ] && [ ! -f /home/workspace/.solv_initialized ]; then sh -c "$SOLV_SETUP_SCRIPT" > /home/workspace/.solv_setup.log 2>&1; touch /home/workspace/.solv_initialized; fi && if [ -x /home/.openvscode-server/bin/openvscode-server ]; then exec /home/.openvscode-server/bin/openvscode-server --without-connection-token --host 0.0.0.0 --port 3000 --default-folder /home/workspace --extensions-dir /home/.openvscode-server/extensions; else exec sleep infinity; fi`
)

// imageShipsOpenVSCode detecta si la imagen trae el servidor OpenVSCode como
// entrypoint (gitpod/openvscode-server y derivadas).
func imageShipsOpenVSCode(imageRef string) bool {
	return strings.Contains(strings.ToLower(imageRef), "openvscode-server")
}

func (c *Client) StartContainer(ctx context.Context, config domain.LabContainerConfig) (string, error) {
	mountMode := "rw"
	if config.ReadOnly {
		mountMode = "ro"
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/environment:%s", config.VolumeName, mountMode),
		},
		Resources: container.Resources{
			Memory: config.MemoryLimitMB * 1024 * 1024, // Conversión a bytes
		},
		NetworkMode: container.NetworkMode(config.NetworkMode),
		SecurityOpt: []string{"no-new-privileges:true"},
	}

	containerConfig := &container.Config{
		Image:  config.Image,
		Labels: mergeTestRunLabels(config.Labels),
	}

	networkingConfig := &network.NetworkingConfig{}
	cached, _, inspectErr := c.InspectLocalImage(ctx, config.Image)
	if inspectErr != nil {
		return "", fmt.Errorf("failed to inspect image %q: %w", config.Image, inspectErr)
	}
	if !cached {
		log.Printf("[SingleFlight] Image %q not found locally. Initiating single-flight pull...", config.Image)
		if err := c.PullImage(ctx, config.Image); err != nil {
			return "", err
		}
		log.Printf("Image %q pulled successfully.", config.Image)
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, config.ContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container %q: %w", config.ContainerName, err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Un contenedor creado que no arranca no debe quedar como residuo. Se usa
		// un contexto propio porque el de la petición puede estar cancelado.
		_ = c.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true, RemoveVolumes: true})
		return "", fmt.Errorf("failed to start container %q (ID: %s): %w", config.ContainerName, resp.ID, err)
	}

	return resp.ID, nil
}

// environmentVolumeBind arma el montaje del volumen del alumno. En un examen
// cerrado va en solo lectura: el sistema de archivos es el que bloquea la
// escritura, no la pantalla (ADR-005 §5).
func environmentVolumeBind(volumeName string, readOnly bool) string {
	mode := "rw"
	if readOnly {
		mode = "ro"
	}
	return fmt.Sprintf("%s:/home/workspace:%s", volumeName, mode)
}

func (c *Client) StartPracticeEnvironmentContainer(ctx context.Context, config domain.PracticeEnvironmentContainerConfig) (string, error) {
	memLimit := config.MemoryLimitMB
	if memLimit <= 0 {
		memLimit = domain.DefaultBaseMemoryMB // 256MB base limit
	}

	memReservation := int64(float64(memLimit) * 0.75) // 75% memory.low para contención suave
	hostPort := strings.TrimSpace(config.HostPort)
	if hostPort == "" {
		hostPort = "0" // puerto dinámico elegido por el daemon
	}
	hostConfig := &container.HostConfig{
		Binds: []string{
			environmentVolumeBind(config.VolumeName, config.ReadOnly),
		},
		Resources: container.Resources{
			Memory:            memLimit * 1024 * 1024,
			MemoryReservation: memReservation * 1024 * 1024,
		},
		NetworkMode: container.NetworkMode(config.NetworkName),
		SecurityOpt: []string{"no-new-privileges:true"},
		// Publicación local: el environment se alcanza por 127.0.0.1 y nunca por
		// un dominio .local inventado.
		PortBindings: nat.PortMap{
			nat.Port(environmentServerPort): []nat.PortBinding{
				{HostIP: "127.0.0.1", HostPort: hostPort},
			},
		},
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
	}

	if companionMount := resolveCompanionExtensionMount(); companionMount != "" {
		hostConfig.Binds = append(hostConfig.Binds, companionMount)
	}

	shipsOpenVSCode := imageShipsOpenVSCode(config.Image)

	// Inyección de variables de entorno del environment.
	envList := config.Env
	if strings.TrimSpace(config.SetupScript) != "" {
		envList = append(envList, fmt.Sprintf("SOLV_SETUP_SCRIPT=%s", config.SetupScript))
	}
	// Evita duplicar NODE_OPTIONS cuando el cliente ya lo provee.
	hasNodeOptions := false
	for _, env := range envList {
		if strings.HasPrefix(env, "NODE_OPTIONS=") {
			hasNodeOptions = true
			break
		}
	}
	if shipsOpenVSCode && !hasNodeOptions {
		maxOldSpace := int(float64(memLimit) * 0.65)
		if maxOldSpace < 128 {
			maxOldSpace = 128
		}
		envList = append(envList, fmt.Sprintf("NODE_OPTIONS=--max-old-space-size=%d", maxOldSpace))
	}

	containerConfig := &container.Config{
		Image:        config.Image,
		Labels:       mergeTestRunLabels(config.Labels),
		Env:          envList,
		ExposedPorts: nat.PortSet{nat.Port(environmentServerPort): struct{}{}},
	}

	if shipsOpenVSCode {
		// El entrypoint de la imagen ya es el binario: solo se le pasan flags.
		containerConfig.User = "1000:1000"
		containerConfig.Cmd = []string{
			"--without-connection-token",
			"--host", "0.0.0.0",
			"--port", "3000",
			"--default-folder", "/home/workspace",
			"--extensions-dir", "/home/.openvscode-server/extensions",
		}
	} else {
		// Runtime puro (python, node, solv-local/stack): NO se le pasa
		// --without-connection-token como Cmd porque el binario no existe y el
		// contenedor moriría con "executable file not found". El shell prepara el
		// environment y arranca OpenVSCode solo si está disponible.
		containerConfig.Cmd = []string{"/bin/sh", "-c", pureRuntimePracticeEnvironmentCmd}
	}

	cached, _, inspectErr := c.InspectLocalImage(ctx, config.Image)
	if inspectErr != nil {
		return "", fmt.Errorf("failed to inspect image %q: %w", config.Image, inspectErr)
	}
	if !cached {
		log.Printf("[SingleFlight] Image %q not found locally. Initiating single-flight pull...", config.Image)
		if err := c.PullImage(ctx, config.Image); err != nil {
			return "", err
		}
		log.Printf("Image %q pulled successfully.", config.Image)
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, config.ContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to create environment container %q: %w", config.ContainerName, err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// La creación pudo completarse aunque el arranque falle (por ejemplo, una
		// red inexistente). El contenedor se elimina antes de devolver el error.
		_ = c.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true, RemoveVolumes: true})
		return "", fmt.Errorf("failed to start environment container %q: %w", config.ContainerName, err)
	}

	// El laboratorio que declara un motor gestionado se conecta a la red interna de
	// satélites para resolver 'solv-sat-<engine>'. Si la conexión falla, el contenedor
	// se descarta: un laboratorio sin su base de datos fallaría de forma confusa y
	// silenciosa más adelante.
	if config.AttachSatellitesNet {
		if netErr := c.attachContainerToSatellitesNetwork(ctx, resp.ID); netErr != nil {
			_ = c.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
			return "", fmt.Errorf("failed to attach environment container %q to %s: %w", config.ContainerName, satellitesNetworkName, netErr)
		}
	}

	return resp.ID, nil
}

// StartReviewEnvironmentContainer ejecuta la app de una entrega en un
// contenedor efímero montado sobre una copia del snapshot. No abre el editor ni
// toca el volumen vivo del alumno: el único puerto publicado es el de la app.
func (c *Client) StartReviewEnvironmentContainer(ctx context.Context, config domain.ReviewEnvironmentContainerConfig) (string, error) {
	if config.AppPort <= 0 || config.AppPort > 65535 {
		return "", fmt.Errorf("invalid review environment app port %d", config.AppPort)
	}
	if strings.TrimSpace(config.Command) == "" {
		return "", fmt.Errorf("review environment app command is required")
	}

	memLimit := config.MemoryLimitMB
	if memLimit <= 0 {
		memLimit = domain.ReviewEnvironmentMinMemoryMB
	}
	memReservation := int64(float64(memLimit) * 0.75)
	hostPort := strings.TrimSpace(config.HostPort)
	if hostPort == "" {
		hostPort = "0"
	}
	appPort := fmt.Sprintf("%d/tcp", config.AppPort)

	portBindings := nat.PortMap{
		nat.Port(appPort): []nat.PortBinding{
			{HostIP: "127.0.0.1", HostPort: hostPort},
		},
	}

	pidsLimit := int64(512)
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/home/workspace:rw", config.VolumeName),
		},
		Resources: container.Resources{
			Memory:            memLimit * 1024 * 1024,
			MemoryReservation: memReservation * 1024 * 1024,
			NanoCPUs:          1_000_000_000,
			PidsLimit:         &pidsLimit,
		},
		NetworkMode:  container.NetworkMode(config.NetworkName),
		SecurityOpt:  []string{"no-new-privileges:true"},
		CapDrop:      []string{"ALL"},
		PortBindings: portBindings,
	}

	envList := append([]string{}, config.Env...)
	envList = append(envList,
		fmt.Sprintf("PORT=%d", config.AppPort),
		"HOST=0.0.0.0",
		fmt.Sprintf("SOLV_APP_RUN_COMMAND=%s", config.Command),
	)
	if strings.TrimSpace(config.SetupScript) != "" {
		envList = append(envList, fmt.Sprintf("SOLV_SETUP_SCRIPT=%s", config.SetupScript))
	}

	// El setup de la plantilla se ejecuta una vez por copia; si falla, el error
	// queda en los logs del contenedor en lugar de arrancar una app a medias.
	script := "cd /home/workspace && "
	if strings.TrimSpace(config.SetupScript) != "" {
		script += `if ! sh -c "$SOLV_SETUP_SCRIPT" > /tmp/solv_review_setup.log 2>&1; then cat /tmp/solv_review_setup.log; exit 1; fi && `
	}
	script += `exec sh -c "$SOLV_APP_RUN_COMMAND"`

	containerConfig := &container.Config{
		Image:        config.Image,
		Labels:       mergeTestRunLabels(config.Labels),
		Env:          envList,
		WorkingDir:   "/home/workspace",
		Cmd:          []string{"/bin/sh", "-c", script},
		ExposedPorts: nat.PortSet{nat.Port(appPort): struct{}{}},
	}

	cached, _, inspectErr := c.InspectLocalImage(ctx, config.Image)
	if inspectErr != nil {
		return "", fmt.Errorf("failed to inspect image %q: %w", config.Image, inspectErr)
	}
	if !cached {
		log.Printf("[SingleFlight] Image %q not found locally. Initiating single-flight pull...", config.Image)
		if err := c.PullImage(ctx, config.Image); err != nil {
			return "", err
		}
		log.Printf("Image %q pulled successfully.", config.Image)
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, config.ContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to create review environment container %q: %w", config.ContainerName, err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = c.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true, RemoveVolumes: true})
		return "", fmt.Errorf("failed to start review environment container %q: %w", config.ContainerName, err)
	}

	if config.AttachSatellitesNet {
		if netErr := c.attachContainerToSatellitesNetwork(ctx, resp.ID); netErr != nil {
			_ = c.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
			return "", fmt.Errorf("failed to attach review container %q to %s: %w", config.ContainerName, satellitesNetworkName, netErr)
		}
	}

	return resp.ID, nil
}

func (c *Client) UpdateContainerMemory(ctx context.Context, containerID string, newMemoryMB int64) error {
	updateConfig := container.UpdateConfig{
		Resources: container.Resources{
			Memory: newMemoryMB * 1024 * 1024,
		},
	}
	_, err := c.cli.ContainerUpdate(ctx, containerID, updateConfig)
	if err != nil {
		return fmt.Errorf("failed to update memory via ContainerUpdate for container %s to %d MB: %w", containerID, newMemoryMB, err)
	}
	log.Printf("[QoS Auto-Bursting] Scaled UP container %s memory limit to %d MB in-place", containerID, newMemoryMB)
	return nil
}

func (c *Client) GetContainerMetrics(ctx context.Context, containerID string) (*domain.ContainerMetrics, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return &domain.ContainerMetrics{IsRunning: false}, nil
		}
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	metrics := &domain.ContainerMetrics{
		IsRunning: inspect.State.Running,
		OOMKilled: inspect.State.OOMKilled,
		ExitCode:  inspect.State.ExitCode,
	}

	if !inspect.State.Running {
		return metrics, nil
	}

	stats, err := c.cli.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		return metrics, nil
	}
	defer func() { _ = stats.Body.Close() }()

	var s container.StatsResponse
	if err := json.NewDecoder(stats.Body).Decode(&s); err != nil {
		return metrics, nil
	}

	metrics.MemoryUsageBytes = domain.Uint64ToInt64(s.MemoryStats.Usage)
	metrics.MemoryLimitBytes = domain.Uint64ToInt64(s.MemoryStats.Limit)

	// Cálculo exacto del delta de CPU: ((cpuDelta / systemDelta) * onlineCPUs) * 100.0
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage - s.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(s.CPUStats.SystemUsage - s.PreCPUStats.SystemUsage)
	onlineCPUs := float64(s.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
	}
	if onlineCPUs == 0 {
		onlineCPUs = 1.0
	}

	if cpuDelta > 0.0 && systemDelta > 0.0 {
		metrics.CPUPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}

	for _, netStats := range s.Networks {
		metrics.RxBytes += netStats.RxBytes
		metrics.TxBytes += netStats.TxBytes
	}

	return metrics, nil
}

func (c *Client) HibernateContainer(ctx context.Context, containerID string) error {
	timeout := 5
	stopOpts := container.StopOptions{Timeout: &timeout}

	err := c.cli.ContainerStop(ctx, containerID, stopOpts)
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to gracefully stop container %q during hibernation: %w", containerID, err)
	}

	removeOpts := container.RemoveOptions{
		Force: true,
	}
	err = c.cli.ContainerRemove(ctx, containerID, removeOpts)
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to remove container %q during hibernation: %w", containerID, err)
	}

	return nil
}

// StopAndRemoveContainer detiene y elimina el contenedor respetando estrictamente el volumen persistente.
func (c *Client) StopAndRemoveContainer(ctx context.Context, containerID string) error {
	timeout := 5
	stopOpts := container.StopOptions{Timeout: &timeout}

	err := c.cli.ContainerStop(ctx, containerID, stopOpts)
	if err != nil && !errdefs.IsNotFound(err) {
		log.Printf("Warning: ContainerStop returned error for %s: %v", containerID, err)
	}

	removeOpts := container.RemoveOptions{
		Force:         true,
		RemoveVolumes: false, // Regla de Oro ADR-001: Jamás eliminar el volumen del estudiante
	}
	err = c.cli.ContainerRemove(ctx, containerID, removeOpts)
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to remove container %q: %w", containerID, err)
	}

	return nil
}

// PauseContainer congela atómicamente el contenedor a nivel de cgroups (cgroup.freeze), llevando CPU a 0%.
func (c *Client) PauseContainer(ctx context.Context, containerID string) error {
	err := c.cli.ContainerPause(ctx, containerID)
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to pause container %q: %w", containerID, err)
	}
	return nil
}

// UnpauseContainer reanuda de forma instantánea (<50ms) un contenedor previamente congelado.
func (c *Client) UnpauseContainer(ctx context.Context, containerID string) error {
	err := c.cli.ContainerUnpause(ctx, containerID)
	if err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to unpause container %q: %w", containerID, err)
	}
	return nil
}

// ExecuteDryRun arranca un contenedor efímero, obtiene sus stats y devuelve el pico de RAM usado.
func (c *Client) ExecuteDryRun(ctx context.Context, image string) (int64, error) {
	hostConfig := &container.HostConfig{
		NetworkMode: "none",
	}

	containerConfig := &container.Config{
		Image:  image,
		Labels: TestRunLabels(),
		Cmd:    []string{"sh", "-c", "echo test && sleep 1"},
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return 0, fmt.Errorf("dry-run create failed: %w", err)
	}
	containerID := resp.ID

	defer func() {
		_ = c.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return 0, fmt.Errorf("dry-run start failed: %w", err)
	}

	stats, err := c.cli.ContainerStats(ctx, containerID, true)
	if err != nil {
		return 0, fmt.Errorf("dry-run stats failed: %w", err)
	}
	defer func() { _ = stats.Body.Close() }()

	decoder := json.NewDecoder(stats.Body)
	var maxRAM int64

	for {
		var s container.StatsResponse
		if err := decoder.Decode(&s); err != nil {
			break
		}
		usage := domain.Uint64ToInt64(s.MemoryStats.Usage)
		if usage > maxRAM {
			maxRAM = usage
		}
	}

	return maxRAM, nil
}

// isPracticeEnvironmentManagedContainer indica si un contenedor pertenece al ciclo de vida de un
// laboratorio. Los motores satélite (mysql, mariadb, mongodb, redis y el PostgreSQL de
// aula) también llevan la etiqueta solv.managed, pero su vida la gobierna el orquestador
// de satélites: si el colector de zombies los listara, los destruiría a los 30 segundos.
func isPracticeEnvironmentManagedContainer(names []string, labels map[string]string) bool {
	if labels["solv.satellite"] != "" {
		return false
	}
	// Los previews de SpeedGrader tienen su propio TTL y su propio volumen
	// efímero. Si el collector los listara, los borraría como huérfanos aunque
	// el docente los esté usando.
	if labels["solv.review_environment"] == "true" {
		return false
	}
	for _, name := range names {
		trimmed := strings.TrimPrefix(name, "/")
		if strings.HasPrefix(trimmed, "solv-environment-") {
			return true
		}
	}
	return labels["solv.managed"] == "true" || labels["traefik.enable"] == "true"
}

func (c *Client) ListAllManagedContainers(ctx context.Context) ([]string, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list docker containers: %w", err)
	}

	var managedIDs []string
	for _, cnt := range containers {
		if isPracticeEnvironmentManagedContainer(cnt.Names, cnt.Labels) {
			managedIDs = append(managedIDs, cnt.ID)
		}
	}

	return managedIDs, nil
}

func (c *Client) RunSemgrepScanOnVolume(ctx context.Context, volumeName string) ([]byte, error) {
	semgrepImage := domain.SemgrepImage

	_, _, err := c.cli.ImageInspectWithRaw(ctx, semgrepImage)
	if err != nil {
		if errdefs.IsNotFound(err) {
			log.Printf("[Semgrep] Image %q not found. Pulling...", semgrepImage)
			out, pullErr := c.cli.ImagePull(ctx, semgrepImage, image.PullOptions{})
			if pullErr != nil {
				return nil, fmt.Errorf("failed to pull semgrep image: %w", pullErr)
			}
			defer func() { _ = out.Close() }()
			_, _ = io.Copy(io.Discard, out)
		}
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/src:ro", volumeName), // Montaje de Solo Lectura estricto (ADR / Ticket 2)
		},
		AutoRemove: false,
	}

	containerConfig := &container.Config{
		Image:  semgrepImage,
		Labels: TestRunLabels(),
		Cmd:    []string{"semgrep", "scan", "--json", "--config", "auto", "/src"},
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create semgrep scanner container: %w", err)
	}
	containerID := resp.ID

	// Garantizar eliminación limpia e inmediata del contenedor efímero
	defer func() {
		_ = c.cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start semgrep container: %w", err)
	}

	// Esperar finalización de la ejecución de Semgrep
	statusCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("error waiting for semgrep container: %w", err)
		}
	case <-statusCh:
	}

	// Capturar Stdout del contenedor con la salida JSON
	logsOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: false,
	}
	outStream, err := c.cli.ContainerLogs(ctx, containerID, logsOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to read semgrep container logs: %w", err)
	}
	defer func() { _ = outStream.Close() }()

	outputBytes, err := io.ReadAll(outStream)
	if err != nil {
		return nil, fmt.Errorf("failed to parse semgrep log output stream: %w", err)
	}

	// Si Docker antepone cabeceras de stream (stdcopy), limpiar o retornar cuerpo
	if len(outputBytes) > 8 && (outputBytes[0] == 1 || outputBytes[0] == 2) {
		outputBytes = outputBytes[8:]
	}

	return outputBytes, nil
}

func (c *Client) GetContainerLogs(ctx context.Context, containerID string, tailLines int) (string, error) {
	if tailLines <= 0 {
		tailLines = 100
	}
	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tailLines),
		Timestamps: true,
	}

	outStream, err := c.cli.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve docker container logs for %s: %w", containerID, err)
	}
	defer func() { _ = outStream.Close() }()

	outputBytes, err := io.ReadAll(outStream)
	if err != nil {
		return "", fmt.Errorf("failed to read log stream: %w", err)
	}

	cleaned := cleanMuxLogHeaders(outputBytes)
	return string(cleaned), nil
}

func cleanMuxLogHeaders(raw []byte) []byte {
	var result []byte
	for len(raw) > 0 {
		if len(raw) < 8 {
			result = append(result, raw...)
			break
		}
		if raw[0] == 1 || raw[0] == 2 {
			size := int(raw[4])<<24 | int(raw[5])<<16 | int(raw[6])<<8 | int(raw[7])
			raw = raw[8:]
			if size > len(raw) {
				size = len(raw)
			}
			result = append(result, raw[:size]...)
			raw = raw[size:]
		} else {
			result = append(result, raw[0])
			raw = raw[1:]
		}
	}
	return result
}

// ExecuteCommandInBackground ejecuta un comando dentro de un contenedor en ejecución sin bloquear el hilo principal.
func (c *Client) ExecuteCommandInBackground(ctx context.Context, containerID string, workDir string, cmd []string) error {
	execConfig := container.ExecOptions{
		AttachStdout: false,
		AttachStderr: false,
		WorkingDir:   workDir,
		Cmd:          cmd,
	}

	execResp, err := c.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance in container %s: %w", containerID, err)
	}

	err = c.cli.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{
		Detach: true,
	})
	if err != nil {
		return fmt.Errorf("failed to start background exec %s in container %s: %w", execResp.ID, containerID, err)
	}

	log.Printf("[Docker Exec] Background command started in container %s: %v", containerID, cmd)
	return nil
}

// resolveCompanionExtensionMount resuelve el bind mount de desarrollo para la extensión
// SOLV Companion (ADR-062). Si la carpeta existe en el host, la monta en solo lectura
// bajo la ruta que OpenVSCode Server reconoce (/home/.openvscode-server/extensions/solv.solv-companion-0.1.0).
// En producción (donde la imagen base ya incluye la extensión horneada) retorna cadena vacía.
func resolveCompanionExtensionMount() string {
	candidate := strings.TrimSpace(os.Getenv("SOLV_COMPANION_HOST_PATH"))
	if candidate == "" {
		for _, rel := range []string{"tools/solv-companion", "../tools/solv-companion", "../../tools/solv-companion"} {
			if abs, err := filepath.Abs(rel); err == nil {
				if info, err := os.Stat(abs); err == nil && info.IsDir() {
					candidate = abs
					break
				}
			}
		}
	}
	if candidate != "" {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return fmt.Sprintf("%s:/home/.openvscode-server/extensions/solv.solv-companion-0.1.0:ro", candidate)
		}
	}
	return ""
}
