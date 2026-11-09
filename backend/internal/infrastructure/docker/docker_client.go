package docker

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"golang.org/x/sync/singleflight"
)

// Client implementa la interfaz domain.ContainerOrchestrator y domain.PracticeEnvironmentOrchestrator
type Client struct {
	cli       *client.Client
	pullGroup singleflight.Group
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize docker client: %w", err)
	}
	return &Client{cli: cli}, nil
}

// TestRunLabels devuelve las etiquetas del run de pruebas actual. Sin
// SOLV_TEST_RUN_ID no etiqueta nada: el despliegue no cambia su comportamiento.
// La limpieza externa solo borra recursos con esta etiqueta exacta.
func TestRunLabels() map[string]string {
	runID := strings.TrimSpace(os.Getenv("SOLV_TEST_RUN_ID"))
	if runID == "" {
		return nil
	}
	return map[string]string{
		"solv.test.run":     runID,
		"solv.test.scope":   "integration",
		"solv.test.started": time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func mergeTestRunLabels(labels map[string]string) map[string]string {
	testLabels := TestRunLabels()
	if len(testLabels) == 0 {
		return labels
	}
	merged := make(map[string]string, len(labels)+len(testLabels))
	for key, value := range labels {
		merged[key] = value
	}
	for key, value := range testLabels {
		merged[key] = value
	}
	return merged
}

// EnsureVolumeExists verifica y crea el volumen local si no existe (ADR-001)
func (c *Client) EnsureVolumeExists(ctx context.Context, volumeName string) error {
	_, err := c.ensureVolumeExists(ctx, volumeName)
	return err
}

// ensureVolumeExists informa si el volumen fue creado por esta llamada, para
// que el llamador pueda revertirlo si el resto de la operación falla.
func (c *Client) ensureVolumeExists(ctx context.Context, volumeName string) (bool, error) {
	_, err := c.cli.VolumeInspect(ctx, volumeName)
	if err == nil {
		return false, nil // El volumen ya existe
	}

	if !errdefs.IsNotFound(err) {
		return false, fmt.Errorf("failed to inspect volume %q: %w", volumeName, err)
	}

	// Crear el volumen ya que no fue encontrado
	_, err = c.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   volumeName,
		Driver: "local",
		Labels: TestRunLabels(),
	})
	if err != nil {
		return false, fmt.Errorf("failed to create local volume %q: %w", volumeName, err)
	}

	return true, nil
}

// RemoveVolume elimina un volumen nombrado y su contenido. Solo lo usan los
// previews efímeros: el volumen del alumno nunca pasa por acá.
func (c *Client) RemoveVolume(ctx context.Context, volumeName string) error {
	if strings.TrimSpace(volumeName) == "" {
		return fmt.Errorf("volume name is required")
	}
	if err := c.cli.VolumeRemove(ctx, volumeName, true); err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to remove volume %q: %w", volumeName, err)
	}
	return nil
}

// GetRawClient expone el cliente Docker subyacente para uso directo del SDK
func (c *Client) GetRawClient() *client.Client {
	return c.cli
}
