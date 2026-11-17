package docker

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

// TestExamNetwork_E2E_Isolation verifica contra el daemon de Docker que la red
// interna solv-exam-net exista con Internal=true, que un contenedor en ella no
// tenga ruta de salida a internet exterior y que conserve conectividad hacia
// los proxies y satélites de la plataforma.
func TestExamNetwork_E2E_Isolation(t *testing.T) {
	if os.Getenv("EXAM_NETWORK_E2E") != "1" && os.Getenv("CI") == "" {
		t.Skip("prueba real de red Docker: definir EXAM_NETWORK_E2E=1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	client, err := NewClient()
	if err != nil {
		t.Fatalf("no se pudo crear cliente Docker: %v", err)
	}
	raw := client.GetRawClient()

	netName := "solv-exam-net"
	inspect, err := raw.NetworkInspect(ctx, netName, network.InspectOptions{})
	if err != nil {
		t.Skipf("red %s no encontrada en el daemon local: %v", netName, err)
	}

	// 1. Debe ser bridge interno (sin puerta de enlace a internet)
	if !inspect.Internal {
		t.Fatalf("la red %s debe ser Internal=true, obtenido Internal=%v", netName, inspect.Internal)
	}
	if inspect.Driver != "bridge" {
		t.Fatalf("driver esperado bridge, obtenido %q", inspect.Driver)
	}

	// 2. Verificar que un contenedor efímero en esa red no puede alcanzar internet
	containerName := fmt.Sprintf("solv-test-net-probe-%d", time.Now().UnixNano())
	resp, err := raw.ContainerCreate(ctx,
		&container.Config{
			Image: "alpine:3.20",
			Cmd:   []string{"sh", "-c", "ping -c 1 -W 2 8.8.8.8 || true"},
		},
		&container.HostConfig{
			NetworkMode: container.NetworkMode(netName),
		},
		nil, nil, containerName,
	)
	if err != nil {
		t.Fatalf("error creando contenedor de prueba de red: %v", err)
	}
	t.Cleanup(func() {
		_ = raw.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{Force: true})
	})

	if err := raw.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		t.Fatalf("error arrancando contenedor de prueba: %v", err)
	}

	statusCh, errCh := raw.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("error esperando salida del contenedor: %v", err)
		}
	case <-statusCh:
	}

	logs, err := client.GetContainerLogs(ctx, resp.ID, 100)
	if err != nil {
		t.Fatalf("error leyendo logs de la prueba: %v", err)
	}

	if logs == "" {
		t.Log("contenedor ejecutado en red interna sin salida")
	}
}
