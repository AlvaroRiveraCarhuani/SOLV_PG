package docker

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/nat"

	"solv-backend/internal/core/domain"
)

const (
	// examNetworkName es la red interna donde vive el contenedor de examen. Sin
	// puerta de enlace no hay ruta por defecto y, por lo tanto, no hay salida.
	examNetworkName = "solv-exam-net"
	// examSatellitesNetworkName conecta el examen con los motores gestionados
	// sin devolverle salida: la red normal de satélites sí tiene gateway, así
	// que usarla como segunda red anularía el corte.
	examSatellitesNetworkName = "solv-exam-satellites-net"
	// examProxyImage reenvía el puerto del editor desde la red interna hacia la
	// red de Traefik. Versión fija: una imagen :latest no es reproducible.
	examProxyImage = "alpine/socat:1.8.0.0"
	// examProxySuffix forma el nombre del proxy a partir del contenedor.
	examProxySuffix = "-proxy"
	// examNoEgressLabel marca el contenedor para la regla de defensa del host.
	examNoEgressLabel = "solv.exam.no-egress"
	// examProxyLabel identifica el proxy de un entorno y permite retirarlo.
	examProxyLabel = "solv.exam.proxy"
)

// ensureExamNetwork asegura la red interna canónica del examen.
func (c *Client) ensureExamNetwork(ctx context.Context) error {
	return c.ensureInternalNetwork(ctx, examNetworkName, "exam-no-egress")
}

// ensureInternalNetwork crea una red bridge con Internal=true si no existe.
func (c *Client) ensureInternalNetwork(ctx context.Context, name, purpose string) error {
	if _, err := c.cli.NetworkInspect(ctx, name, network.InspectOptions{}); err == nil {
		return nil
	} else if !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to inspect internal network %q: %w", name, err)
	}

	_, err := c.cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver:   "bridge",
		Internal: true,
		Labels:   mergeTestRunLabels(map[string]string{"solv.managed": "true", "solv.purpose": purpose}),
	})
	if err != nil {
		return fmt.Errorf("failed to create internal network %q: %w", name, err)
	}
	log.Printf("[Exam Network] Red interna %q creada (sin puerta de enlace)", name)
	return nil
}

// attachExamToSatellitesNetwork conecta el examen y los motores gestionados a
// una red interna propia. El contenedor del examen nunca entra a la red normal
// de satélites: esa red tiene puerta de enlace y le devolvería la salida.
func (c *Client) attachExamToSatellitesNetwork(ctx context.Context, examContainerID string) error {
	if err := c.ensureInternalNetwork(ctx, examSatellitesNetworkName, "exam-satellites"); err != nil {
		return err
	}
	if err := c.connectNetwork(ctx, examSatellitesNetworkName, examContainerID); err != nil {
		return err
	}

	satellites, err := c.cli.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", "solv.satellite")),
	})
	if err != nil {
		return fmt.Errorf("failed to list managed satellites: %w", err)
	}
	for _, satellite := range satellites {
		if err := c.connectNetwork(ctx, examSatellitesNetworkName, satellite.ID); err != nil {
			return fmt.Errorf("failed to attach satellite %s to the exam network: %w", satellite.ID[:12], err)
		}
	}
	return nil
}

func (c *Client) connectNetwork(ctx context.Context, networkName, containerID string) error {
	if err := c.cli.NetworkConnect(ctx, networkName, containerID, nil); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("failed to connect container %s to %s: %w", containerID, networkName, err)
	}
	return nil
}

// startExamEditorProxy publica el editor de un contenedor que vive en la red
// interna. El proxy está en las dos redes: alcanza al examen por nombre y
// Traefik lo alcanza a él. Es la pieza que permite cortar la salida sin dejar
// al alumno sin editor.
func (c *Client) startExamEditorProxy(ctx context.Context, config domain.PracticeEnvironmentContainerConfig, hostPort, examContainerID string) error {
	proxyNetwork, err := c.resolveProxyNetwork(ctx)
	if err != nil {
		return err
	}
	proxyImageCached, _, inspectErr := c.InspectLocalImage(ctx, examProxyImage)
	if inspectErr != nil {
		return fmt.Errorf("failed to inspect exam proxy image %q: %w", examProxyImage, inspectErr)
	}

	proxyName := config.ContainerName + examProxySuffix
	labels := make(map[string]string, len(config.Labels)+2)
	for key, value := range config.Labels {
		labels[key] = value
	}
	labels["traefik.docker.network"] = proxyNetwork
	labels[examProxyLabel] = config.ContainerName

	hostPortValue := strings.TrimSpace(hostPort)
	if hostPortValue == "" {
		hostPortValue = "0"
	}

	hostConfig := &container.HostConfig{
		NetworkMode: container.NetworkMode(examNetworkName),
		SecurityOpt: []string{"no-new-privileges:true"},
		PortBindings: nat.PortMap{
			nat.Port(environmentServerPort): []nat.PortBinding{
				{HostIP: "127.0.0.1", HostPort: hostPortValue},
			},
		},
	}
	containerConfig := &container.Config{
		Image:        examProxyImage,
		Labels:       mergeTestRunLabels(labels),
		ExposedPorts: nat.PortSet{nat.Port(environmentServerPort): struct{}{}},
		Cmd: []string{
			"TCP-LISTEN:3000,fork,reuseaddr",
			fmt.Sprintf("TCP:%s:3000", config.ContainerName),
		},
	}

	if !proxyImageCached {
		if err := c.PullImage(ctx, examProxyImage); err != nil {
			return fmt.Errorf("failed to pull exam proxy image %q: %w", examProxyImage, err)
		}
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, proxyName)
	if err != nil {
		return fmt.Errorf("failed to create exam editor proxy %q: %w", proxyName, err)
	}
	if err := c.connectNetwork(ctx, proxyNetwork, resp.ID); err != nil {
		_ = c.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
		return err
	}
	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = c.cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
		return fmt.Errorf("failed to start exam editor proxy %q: %w", proxyName, err)
	}
	log.Printf("[Exam Network] Editor de %s publicado por proxy en %s", examContainerID[:12], proxyNetwork)
	return nil
}

// resolveProxyNetwork elige la red donde Traefik descubre servicios. El compose
// del nodo usa solv_net; instalaciones anteriores, solv-traefik-net.
func (c *Client) resolveProxyNetwork(ctx context.Context) (string, error) {
	for _, candidate := range []string{"solv-traefik-net", "solv_net"} {
		if _, err := c.cli.NetworkInspect(ctx, candidate, network.InspectOptions{}); err == nil {
			return candidate, nil
		} else if !errdefs.IsNotFound(err) {
			return "", fmt.Errorf("failed to inspect traefik network %q: %w", candidate, err)
		}
	}
	return "", fmt.Errorf("no traefik network found (solv-traefik-net or solv_net)")
}

// removeExamProxy retira el proxy asociado a un entorno antes de borrar el
// contenedor principal. Sin esto, un proxy huérfano seguiría publicando un
// puerto que ya no responde.
func (c *Client) removeExamProxy(ctx context.Context, containerID string) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil || inspect.ContainerJSONBase == nil {
		return
	}
	name := strings.TrimPrefix(inspect.Name, "/")
	if name == "" {
		return
	}
	_ = c.cli.ContainerRemove(context.Background(), name+examProxySuffix, container.RemoveOptions{Force: true})
}

// examNoEgressLabels devuelve la etiqueta que el script de firewall lee para
// derivar la regla de defensa del contenedor.
func examNoEgressLabels() map[string]string {
	return map[string]string{examNoEgressLabel: "true"}
}

// containerLabelsForPracticeEnvironment prepara las etiquetas del contenedor
// principal. En exámenes sin salida retira las etiquetas traefik.* para que
// Traefik enrute únicamente al contenedor -proxy y no intente alcanzar la IP
// interna aislada del examen.
func containerLabelsForPracticeEnvironment(config domain.PracticeEnvironmentContainerConfig) map[string]string {
	if !config.ExamNoEgress {
		return config.Labels
	}
	labels := make(map[string]string, len(config.Labels)+1)
	for key, value := range config.Labels {
		if strings.HasPrefix(key, "traefik.") {
			continue
		}
		labels[key] = value
	}
	for key, value := range examNoEgressLabels() {
		labels[key] = value
	}
	return labels
}
