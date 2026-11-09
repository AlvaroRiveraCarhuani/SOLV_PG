package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/errdefs"
	"solv-backend/internal/infrastructure/database"
	dockerinfra "solv-backend/internal/infrastructure/docker"
)

// TestMain fija el DSN de tests para todo el paquete. Varias pruebas leen
// DATABASE_URL directamente; sin esta normalización podrían apuntar a la base
// de desarrollo o a la del despliegue cuando el runner exporta DATABASE_URL.
func TestMain(m *testing.M) {
	dsn := getTestDSN()
	if err := validateTestDSN(dsn); err != nil {
		fmt.Fprintf(os.Stderr, "tests de integración: %v\n", err)
		os.Exit(2)
	}
	if requireInfra() {
		dbInstance, err := database.NewPostgresDB(dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tests de integración: PostgreSQL obligatorio no disponible: %v\n", err)
			os.Exit(2)
		}
		_ = dbInstance.GetDB().Close()
	}
	if os.Getenv("SOLV_REQUIRE_DOCKER") == "1" {
		ensureTestRunID()
		dockerClient, err := dockerinfra.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "tests de integración: Docker obligatorio no disponible: %v\n", err)
			os.Exit(2)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, pingErr := dockerClient.GetRawClient().Ping(ctx)
		cancel()
		if pingErr != nil {
			fmt.Fprintf(os.Stderr, "tests de integración: Docker obligatorio no responde: %v\n", pingErr)
			os.Exit(2)
		}
	}
	if err := os.Setenv("DATABASE_URL", dsn); err != nil {
		fmt.Fprintf(os.Stderr, "tests de integración: no se pudo fijar DATABASE_URL: %v\n", err)
		os.Exit(2)
	}
	if err := os.Setenv("TEST_DB_DSN", dsn); err != nil {
		fmt.Fprintf(os.Stderr, "tests de integración: no se pudo fijar TEST_DB_DSN: %v\n", err)
		os.Exit(2)
	}
	code := m.Run()
	// La limpieza se ejecuta también cuando alguna prueba falla. Un SIGKILL no
	// puede atenderse desde el proceso; para ese caso queda la limpieza externa
	// por etiqueta y el paso if: always() del workflow.
	if err := cleanupTestDockerResources(); err != nil {
		fmt.Fprintf(os.Stderr, "tests de integración: limpieza Docker falló: %v\n", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func cleanupTestDockerResources() error {
	runID := os.Getenv("SOLV_TEST_RUN_ID")
	if runID == "" {
		return nil
	}
	dockerClient, err := dockerinfra.NewClient()
	if err != nil {
		return fmt.Errorf("no se pudo crear el cliente Docker: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	label := filters.NewArgs(filters.Arg("label", "solv.test.run="+runID))
	var cleanupErrors []error

	containers, err := dockerClient.GetRawClient().ContainerList(ctx, container.ListOptions{All: true, Filters: label})
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("listar contenedores: %w", err))
	} else {
		for _, item := range containers {
			if err := dockerClient.GetRawClient().ContainerRemove(ctx, item.ID, container.RemoveOptions{Force: true, RemoveVolumes: true}); err != nil && !errdefs.IsNotFound(err) {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("eliminar contenedor %s: %w", item.ID, err))
			}
		}
	}
	volumes, err := dockerClient.GetRawClient().VolumeList(ctx, volume.ListOptions{Filters: label})
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("listar volúmenes: %w", err))
	} else {
		for _, item := range volumes.Volumes {
			if err := dockerClient.GetRawClient().VolumeRemove(ctx, item.Name, true); err != nil && !errdefs.IsNotFound(err) {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("eliminar volumen %s: %w", item.Name, err))
			}
		}
	}
	networks, err := dockerClient.GetRawClient().NetworkList(ctx, network.ListOptions{Filters: label})
	if err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("listar redes: %w", err))
	} else {
		for _, item := range networks {
			if err := dockerClient.GetRawClient().NetworkRemove(ctx, item.ID); err != nil && !errdefs.IsNotFound(err) {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("eliminar red %s: %w", item.ID, err))
			}
		}
	}
	return errors.Join(cleanupErrors...)
}
