package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"

	"solv-backend/internal/core/domain"
)

// Client implementa la interfaz domain.ContainerOrchestrator y domain.WorkspaceOrchestrator
type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize docker client: %w", err)
	}
	return &Client{cli: cli}, nil
}

// EnsureVolumeExists verifica y crea el volumen local si no existe (ADR-001)
func (c *Client) EnsureVolumeExists(ctx context.Context, volumeName string) error {
	_, err := c.cli.VolumeInspect(ctx, volumeName)
	if err == nil {
		return nil // El volumen ya existe
	}

	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to inspect volume %q: %w", volumeName, err)
	}

	// Crear el volumen ya que no fue encontrado
	_, err = c.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   volumeName,
		Driver: "local",
	})
	if err != nil {
		return fmt.Errorf("failed to create local volume %q: %w", volumeName, err)
	}

	return nil
}

// EnsureICCDisabledNetworkExists asegura que exista la red de Docker con la directiva ICC (Inter-Container Communication) desactivada
func (c *Client) EnsureICCDisabledNetworkExists(ctx context.Context, networkName string) error {
	_, err := c.cli.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err == nil {
		return nil // La red ya existe
	}

	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to inspect docker network %q: %w", networkName, err)
	}

	_, err = c.cli.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
		Options: map[string]string{
			"com.docker.network.bridge.enable_icc": "false",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create network with enable_icc=false %q: %w", networkName, err)
	}

	return nil
}

func (c *Client) StartContainer(ctx context.Context, config domain.LabContainerConfig) (string, error) {
	mountMode := "rw"
	if config.ReadOnly {
		mountMode = "ro"
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/workspace:%s", config.VolumeName, mountMode),
		},
		Resources: container.Resources{
			Memory: config.MemoryLimitMB * 1024 * 1024, // Conversión a bytes
		},
		NetworkMode: container.NetworkMode(config.NetworkMode),
		SecurityOpt: []string{"no-new-privileges:true"},
	}

	containerConfig := &container.Config{
		Image:  config.Image,
		Labels: config.Labels,
	}

	networkingConfig := &network.NetworkingConfig{}
	_, _, err := c.cli.ImageInspectWithRaw(ctx, config.Image)
	if err != nil {
		if errdefs.IsNotFound(err) {
			log.Printf("Image %q not found locally. Pulling from registry (this may take a while)...", config.Image)
			out, pullErr := c.cli.ImagePull(ctx, config.Image, image.PullOptions{})
			if pullErr != nil {
				return "", fmt.Errorf("failed to pull image %q: %w", config.Image, pullErr)
			}
			defer out.Close()
			_, _ = io.Copy(io.Discard, out)
			log.Printf("Image %q pulled successfully.", config.Image)
		} else {
			return "", fmt.Errorf("failed to inspect image %q: %w", config.Image, err)
		}
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, config.ContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container %q: %w", config.ContainerName, err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container %q (ID: %s): %w", config.ContainerName, resp.ID, err)
	}

	return resp.ID, nil
}

func (c *Client) StartWorkspaceContainer(ctx context.Context, config domain.WorkspaceContainerConfig) (string, error) {
	memLimit := config.MemoryLimitMB
	if memLimit <= 0 {
		memLimit = domain.DefaultBaseMemoryMB // 256MB base limit
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/home/workspace:rw", config.VolumeName),
		},
		Resources: container.Resources{
			Memory: memLimit * 1024 * 1024,
		},
		NetworkMode: container.NetworkMode(config.NetworkName),
		SecurityOpt: []string{"no-new-privileges:true"},
	}

	containerConfig := &container.Config{
		Image:  config.Image,
		Labels: config.Labels,
		Env:    config.Env,
		User:   "1000:1000",
		Cmd:    []string{"--without-connection-token", "--host", "0.0.0.0"},
	}

	_, _, err := c.cli.ImageInspectWithRaw(ctx, config.Image)
	if err != nil {
		if errdefs.IsNotFound(err) {
			log.Printf("Image %q not found. Pulling...", config.Image)
			out, pullErr := c.cli.ImagePull(ctx, config.Image, image.PullOptions{})
			if pullErr != nil {
				return "", fmt.Errorf("failed to pull image %q: %w", config.Image, pullErr)
			}
			defer out.Close()
			_, _ = io.Copy(io.Discard, out)
		} else {
			return "", fmt.Errorf("failed to inspect image %q: %w", config.Image, err)
		}
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, config.ContainerName)
	if err != nil {
		return "", fmt.Errorf("failed to create workspace container %q: %w", config.ContainerName, err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start workspace container %q: %w", config.ContainerName, err)
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
	defer stats.Body.Close()

	var s container.StatsResponse
	if err := json.NewDecoder(stats.Body).Decode(&s); err != nil {
		return metrics, nil
	}

	metrics.MemoryUsageBytes = int64(s.MemoryStats.Usage)
	metrics.MemoryLimitBytes = int64(s.MemoryStats.Limit)

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

// ExecuteDryRun arranca un contenedor efímero, obtiene sus stats y devuelve el pico de RAM usado.
func (c *Client) ExecuteDryRun(ctx context.Context, image string) (int64, error) {
	hostConfig := &container.HostConfig{
		NetworkMode: "none",
	}

	containerConfig := &container.Config{
		Image: image,
		Cmd:   []string{"sh", "-c", "echo test && sleep 1"},
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
	defer stats.Body.Close()

	decoder := json.NewDecoder(stats.Body)
	var maxRAM int64

	for {
		var s container.StatsResponse
		if err := decoder.Decode(&s); err != nil {
			break
		}
		usage := int64(s.MemoryStats.Usage)
		if usage > maxRAM {
			maxRAM = usage
		}
	}

	return maxRAM, nil
}

func (c *Client) ListAllManagedContainers(ctx context.Context) ([]string, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list docker containers: %w", err)
	}

	var managedIDs []string
	for _, cnt := range containers {
		// Filtrar contenedores gestionados por SOLV (por label o por prefijo /solv-workspace-)
		isSolvManaged := false
		for _, name := range cnt.Names {
			if len(name) > 0 && (name[0] == '/' && (len(name) > 16 && name[1:16] == "solv-workspace-") || (len(name) > 15 && name[0:15] == "solv-workspace-")) {
				isSolvManaged = true
				break
			}
		}

		if !isSolvManaged {
			if cnt.Labels != nil && (cnt.Labels["solv.managed"] == "true" || cnt.Labels["traefik.enable"] == "true") {
				isSolvManaged = true
			}
		}

		if isSolvManaged {
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
			defer out.Close()
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
		Image: semgrepImage,
		Cmd:   []string{"semgrep", "scan", "--json", "--config", "auto", "/src"},
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
	defer outStream.Close()

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

func (c *Client) GetRawClient() *client.Client {
	return c.cli
}

