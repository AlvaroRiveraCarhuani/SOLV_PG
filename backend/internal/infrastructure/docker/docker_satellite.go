package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"

	"solv-backend/internal/core/domain"
)

const (
	// satellitesNetworkName es la red bridge interna donde viven los motores de base
	// de datos auto-orquestados y los laboratorios que los usan. Deliberadamente NO es
	// la red de Traefik: los contenedores de distintos alumnos no se alcanzan entre sí y
	// ningún motor queda publicado al host.
	satellitesNetworkName = "solv-satellites-net"

	// managedSatelliteReadyTimeout acota cuánto se espera a que el motor acepte
	// conexiones antes de devolver el endpoint.
	managedSatelliteReadyTimeout = 30 * time.Second

	// managedSatelliteStopTimeout es el margen de apagado ordenado antes de matar el
	// proceso del motor. El volumen nombrado sobrevive al apagado.
	managedSatelliteStopTimeout = 10

	// managedSatelliteExecTimeout acota un único comando dentro del satélite.
	managedSatelliteExecTimeout = 30 * time.Second
)

// managedSatelliteDataPath resuelve dónde persiste cada motor sus datos dentro del
// contenedor; el volumen nombrado se monta en esa ruta.
//
// Desde PostgreSQL 18 el volumen oficial es el padre /var/lib/postgresql y PGDATA vive
// en /var/lib/postgresql/<major>/docker: montar en /var/lib/postgresql/data queda fuera
// del contrato de la imagen y el entrypoint lo rechaza.
func managedSatelliteDataPath(engine string) string {
	switch engine {
	case "postgres":
		return "/var/lib/postgresql"
	case "mysql", "mariadb":
		return "/var/lib/mysql"
	case "mongodb":
		return "/data/db"
	case "redis":
		return "/data"
	default:
		return "/data"
	}
}

// managedSatelliteStopSignal elige la señal de apagado del motor. PostgreSQL atiende
// SIGTERM como apagado "smart": espera a que los clientes se desconecten y el stop por
// timeout termina en SIGKILL (exit 137). SIGINT dispara el apagado "fast", que corta las
// conexiones y cierra ordenadamente. El resto de los motores usa la señal por defecto.
func managedSatelliteStopSignal(engine string) string {
	if engine == "postgres" {
		return "SIGINT"
	}
	return ""
}

// managedSatelliteMemoryMB es el techo de memoria del contenedor cuando el llamador no
// lo declara explícitamente.
func managedSatelliteMemoryMB(engine string) int64 {
	if engine == "redis" {
		return 96
	}
	return 256
}

// satelliteNetworkName resuelve la red de satélites declarada por el llamador, con la
// red interna canónica como default.
func satelliteNetworkName(spec domain.ManagedSatelliteSpec) string {
	if name := strings.TrimSpace(spec.NetworkName); name != "" {
		return name
	}
	return satellitesNetworkName
}

// ensureSatelliteNetwork asegura la red bridge de satélites. El DNS interno de Docker
// queda habilitado por defecto, de modo que un laboratorio resuelve 'solv-sat-<engine>'
// sin conocer direcciones IP.
func (c *Client) ensureSatelliteNetwork(ctx context.Context, networkName string) error {
	_, err := c.cli.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err == nil {
		return nil
	}
	if !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to inspect satellite network %q: %w", networkName, err)
	}

	_, err = c.cli.NetworkCreate(ctx, networkName, network.CreateOptions{
		Driver: "bridge",
		Labels: map[string]string{"solv.managed": "true", "solv.purpose": "satellites"},
	})
	if err != nil {
		return fmt.Errorf("failed to create satellite network %q: %w", networkName, err)
	}
	log.Printf("[Satellites] Red interna %q creada para motores auto-orquestados", networkName)
	return nil
}

// ensureSatellitesNetworkExists asegura la red interna canónica que comparten los
// motores auto-orquestados y los laboratorios que los usan.
func (c *Client) ensureSatellitesNetworkExists(ctx context.Context) error {
	return c.ensureSatelliteNetwork(ctx, satellitesNetworkName)
}

// attachContainerToSatellitesNetwork conecta un contenedor ya iniciado a la red interna
// de satélites. Es idempotente: un contenedor ya conectado no vuelve a engancharse.
func (c *Client) attachContainerToSatellitesNetwork(ctx context.Context, containerID string) error {
	if err := c.ensureSatellitesNetworkExists(ctx); err != nil {
		return err
	}
	err := c.cli.NetworkConnect(ctx, satellitesNetworkName, containerID, nil)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("failed to connect container %s to %s: %w", containerID, satellitesNetworkName, err)
	}
	return nil
}

// EnsureManagedSatelliteRunning asegura que el contenedor satélite del motor exista y
// esté corriendo, y devuelve su endpoint dentro de la red interna.
//
// El motor se comparte entre los laboratorios del nodo: el aislamiento no es por
// contenedor sino por credenciales (base, usuario y contraseña por alumno y materia), y
// datos y volumen sobreviven a los apagados de Scale-to-Zero.
func (c *Client) EnsureManagedSatelliteRunning(ctx context.Context, spec domain.ManagedSatelliteSpec) (*domain.ManagedSatelliteEndpoint, error) {
	engine := strings.ToLower(strings.TrimSpace(spec.Engine))
	if engine == "" {
		return nil, errors.New("managed satellite spec requires an engine")
	}
	containerName := strings.TrimSpace(spec.ContainerName)
	if containerName == "" {
		containerName = "solv-sat-" + engine
	}
	networkName := satelliteNetworkName(spec)

	if err := c.ensureSatelliteNetwork(ctx, networkName); err != nil {
		return nil, err
	}

	inspect, err := c.cli.ContainerInspect(ctx, containerName)
	switch {
	case err == nil:
		if inspect.State != nil && inspect.State.Running {
			return &domain.ManagedSatelliteEndpoint{
				Engine:      engine,
				Host:        containerName,
				Port:        spec.Port,
				ContainerID: inspect.ID,
			}, nil
		}
		log.Printf("[Satellites] Satélite %s estaba detenido: reactivando por demanda", containerName)
		if startErr := c.cli.ContainerStart(ctx, inspect.ID, container.StartOptions{}); startErr != nil {
			return nil, fmt.Errorf("failed to restart managed satellite %q: %w", containerName, startErr)
		}
		if readyErr := c.waitManagedSatelliteReady(ctx, inspect.ID, spec.ReadyCmd); readyErr != nil {
			return nil, readyErr
		}
		return &domain.ManagedSatelliteEndpoint{
			Engine:      engine,
			Host:        containerName,
			Port:        spec.Port,
			ContainerID: inspect.ID,
		}, nil
	case !errdefs.IsNotFound(err):
		return nil, fmt.Errorf("failed to inspect managed satellite %q: %w", containerName, err)
	}

	// El contenedor no existe: primera creación bajo demanda.
	cached, _, inspectErr := c.InspectLocalImage(ctx, spec.Image)
	if inspectErr != nil {
		return nil, fmt.Errorf("failed to inspect image %q: %w", spec.Image, inspectErr)
	}
	if !cached {
		log.Printf("[SingleFlight] Image %q not found locally. Initiating single-flight pull...", spec.Image)
		if pullErr := c.PullImage(ctx, spec.Image); pullErr != nil {
			return nil, pullErr
		}
	}

	volumeName := strings.TrimSpace(spec.VolumeName)
	if volumeName == "" {
		volumeName = "solv_sat_data_" + engine
	}
	if err := c.EnsureVolumeExists(ctx, volumeName); err != nil {
		return nil, err
	}

	memLimitMB := spec.MemoryLimitMB
	if memLimitMB <= 0 {
		memLimitMB = managedSatelliteMemoryMB(engine)
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", volumeName, managedSatelliteDataPath(engine)),
		},
		Resources: container.Resources{
			Memory:     memLimitMB * 1024 * 1024,
			MemorySwap: memLimitMB * 1024 * 1024,
		},
		// Solo alcanzable desde la red interna: sin PortBindings el motor no se
		// publica al host y nunca queda expuesto a Internet.
		NetworkMode: container.NetworkMode(networkName),
		SecurityOpt: []string{"no-new-privileges:true"},
	}

	containerConfig := &container.Config{
		Image:  spec.Image,
		Env:    spec.Env,
		Cmd:    spec.Cmd,
		Labels: map[string]string{"solv.managed": "true", "solv.satellite": engine},
	}
	containerConfig.StopSignal = managedSatelliteStopSignal(engine)

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create managed satellite %q: %w", containerName, err)
	}
	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start managed satellite %q: %w", containerName, err)
	}

	if err := c.waitManagedSatelliteReady(ctx, resp.ID, spec.ReadyCmd); err != nil {
		return nil, err
	}

	log.Printf("[Satellites] Motor %s listo en %s (%d MB, volumen %s)", engine, containerName, memLimitMB, volumeName)
	return &domain.ManagedSatelliteEndpoint{
		Engine:      engine,
		Host:        containerName,
		Port:        spec.Port,
		ContainerID: resp.ID,
	}, nil
}

// waitManagedSatelliteReady sondea el motor dentro del contenedor con backoff hasta que
// acepta conexiones o se agota el margen de espera.
func (c *Client) waitManagedSatelliteReady(ctx context.Context, containerID string, readyCmd []string) error {
	if len(readyCmd) == 0 {
		return nil
	}

	deadline := time.Now().Add(managedSatelliteReadyTimeout)
	backoff := 500 * time.Millisecond
	var lastErr error

	for {
		if _, err := c.execInContainer(ctx, containerID, readyCmd); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("managed satellite not ready after %s: %w", managedSatelliteReadyTimeout, lastErr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 4*time.Second {
			backoff *= 2
			if backoff > 4*time.Second {
				backoff = 4 * time.Second
			}
		}
	}
}

// StopManagedSatellite detiene el contenedor del motor liberando toda su memoria en el
// host. El contenedor no se elimina: su volumen de datos permanece y la próxima sesión
// lo reactiva sin volver a crear nada.
func (c *Client) StopManagedSatellite(ctx context.Context, engine string) error {
	name := "solv-sat-" + strings.ToLower(strings.TrimSpace(engine))
	inspect, err := c.cli.ContainerInspect(ctx, name)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to inspect managed satellite %q: %w", name, err)
	}
	if inspect.State == nil || !inspect.State.Running {
		return nil
	}

	timeout := managedSatelliteStopTimeout
	if err := c.cli.ContainerStop(ctx, inspect.ID, container.StopOptions{Timeout: &timeout}); err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to stop managed satellite %q: %w", name, err)
	}
	log.Printf("[Scale-to-Zero] Satélite gestionado %s detenido; volumen de datos preservado", name)
	return nil
}

// ExecInManagedSatellite ejecuta un comando dentro del contenedor del motor y devuelve
// su salida estándar. Se usa para el aprovisionamiento idempotente por alumno.
func (c *Client) ExecInManagedSatellite(ctx context.Context, engine string, cmd []string) (string, error) {
	name := "solv-sat-" + strings.ToLower(strings.TrimSpace(engine))
	inspect, err := c.cli.ContainerInspect(ctx, name)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return "", fmt.Errorf("managed satellite %q is not provisioned", name)
		}
		return "", fmt.Errorf("failed to inspect managed satellite %q: %w", name, err)
	}
	if inspect.State == nil || !inspect.State.Running {
		return "", fmt.Errorf("managed satellite %q is not running", name)
	}
	return c.execInContainer(ctx, inspect.ID, cmd)
}

// execInContainer ejecuta un comando y demuxa stdout/stderr del stream multiplexado de
// Docker, devolviendo stdout y un error con el stderr cuando el comando falla.
func (c *Client) execInContainer(ctx context.Context, containerID string, cmd []string) (string, error) {
	execResp, err := c.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create exec %v in container %s: %w", cmd, containerID, err)
	}

	attach, err := c.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to attach exec %s: %w", execResp.ID, err)
	}

	var stdout, stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		_, copyErr := stdcopy.StdCopy(&stdout, &stderr, attach.Reader)
		done <- copyErr
	}()

	select {
	case copyErr := <-done:
		if copyErr != nil && !errors.Is(copyErr, io.EOF) {
			attach.Close()
			return stdout.String(), fmt.Errorf("failed to read exec output: %w", copyErr)
		}
	case <-time.After(managedSatelliteExecTimeout):
		attach.Close()
		return "", fmt.Errorf("exec %v timed out after %s", cmd, managedSatelliteExecTimeout)
	case <-ctx.Done():
		attach.Close()
		return "", ctx.Err()
	}
	attach.Close()

	inspect, err := c.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return stdout.String(), fmt.Errorf("failed to inspect exec %s: %w", execResp.ID, err)
	}
	if inspect.ExitCode != 0 {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		return stdout.String(), fmt.Errorf("exec %v exited with code %d: %s", cmd, inspect.ExitCode, detail)
	}

	return stdout.String(), nil
}
