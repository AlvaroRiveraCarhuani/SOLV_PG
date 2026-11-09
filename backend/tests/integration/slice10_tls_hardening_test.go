package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
	"solv-backend/internal/infrastructure/docker"
)

func TestTLSAndHardening(t *testing.T) {
	t.Run("1. Pinned Image Versions (No :latest)", func(t *testing.T) {
		if strings.Contains(domain.OpenVSCodeImage, ":latest") {
			t.Fatalf("OpenVSCodeImage must be pinned to a specific version, got: %s", domain.OpenVSCodeImage)
		}
		if strings.Contains(domain.SemgrepImage, ":latest") {
			t.Fatalf("SemgrepImage must be pinned to a specific version, got: %s", domain.SemgrepImage)
		}
		if strings.Contains(domain.TraefikImage, ":latest") {
			t.Fatalf("TraefikImage must be pinned to a specific version, got: %s", domain.TraefikImage)
		}

		if domain.OpenVSCodeImage != "gitpod/openvscode-server:1.96.0" {
			t.Errorf("Expected OpenVSCodeImage gitpod/openvscode-server:1.96.0, got %s", domain.OpenVSCodeImage)
		}
		if domain.SemgrepImage != "semgrep/semgrep:1.100.0" {
			t.Errorf("Expected SemgrepImage semgrep/semgrep:1.100.0, got %s", domain.SemgrepImage)
		}
		if domain.TraefikImage != "traefik:v3.7.8" {
			t.Errorf("Expected TraefikImage traefik:v3.7.8, got %s", domain.TraefikImage)
		}

		t.Logf("PASS: Image versions correctly pinned -> OpenVSCode: %s | Semgrep: %s | Traefik: %s",
			domain.OpenVSCodeImage, domain.SemgrepImage, domain.TraefikImage)
	})

	t.Run("2. Docker Container Hardening Options", func(t *testing.T) {
		requireDocker(t)
		dockerClient, err := docker.NewClient()
		if err != nil {
			skipOrFailInfra(t, "Docker no disponible para verificar hardening: %v", err)
		}

		ctx := context.Background()
		// Red propia del test: no depende de la red del despliegue ni de que el
		// runner haya levantado compose.yml.
		networkName := fmt.Sprintf("solv_test_hardening_%d", time.Now().UnixNano())
		if err := dockerClient.EnsureICCDisabledNetworkExists(ctx, networkName); err != nil {
			t.Fatalf("No se pudo crear la red aislada del test: %v", err)
		}
		rawClient := dockerClient.GetRawClient()
		t.Cleanup(func() {
			_ = rawClient.NetworkRemove(context.Background(), networkName)
		})

		environmentConfig := domain.PracticeEnvironmentContainerConfig{
			Image:         domain.OpenVSCodeImage,
			ContainerName: "solv-test-hardening-environment",
			VolumeName:    "solv_test_vol_hardening",
			MemoryLimitMB: 256,
			NetworkName:   networkName,
		}

		t.Cleanup(func() {
			_ = dockerClient.RemoveVolume(context.Background(), environmentConfig.VolumeName)
		})
		containerID, err := dockerClient.StartPracticeEnvironmentContainer(ctx, environmentConfig)
		if err != nil {
			t.Fatalf("No se pudo iniciar el contenedor de hardening: %v", err)
		}
		defer func() {
			_ = dockerClient.StopAndRemoveContainer(context.Background(), containerID)
		}()

		// Insccionar contenedor y validar SecurityOpt + User
		cli := dockerClient.GetRawClient()
		inspect, err := cli.ContainerInspect(ctx, containerID)
		if err != nil {
			t.Fatalf("Failed to inspect container %s: %v", containerID, err)
		}

		if !hasExactNoNewPrivilegesTrue(inspect.HostConfig.SecurityOpt) {
			t.Errorf("Container HostConfig.SecurityOpt missing 'no-new-privileges:true', got: %v", inspect.HostConfig.SecurityOpt)
		}

		if inspect.Config.User != "1000:1000" {
			t.Errorf("PracticeEnvironment container Config.User expected '1000:1000', got '%s'", inspect.Config.User)
		}

		if runID := os.Getenv("SOLV_TEST_RUN_ID"); runID != "" {
			if inspect.Config.Labels["solv.test.run"] != runID {
				t.Errorf("El contenedor de test no lleva la etiqueta solv.test.run=%s: %v", runID, inspect.Config.Labels)
			}
		}

		t.Logf("PASS: PracticeEnvironment container hardening verified -> SecurityOpt: %v | User: %s",
			inspect.HostConfig.SecurityOpt, inspect.Config.User)
	})

	t.Run("3. Manual TLS Verification Protocol", func(t *testing.T) {
		cmdHelp := "openssl s_client -connect solv.dedyn.io:443 -servername solv.dedyn.io"
		t.Logf("INFO: Manual TLS verification command: %s", cmdHelp)
	})
}

func hasExactNoNewPrivilegesTrue(options []string) bool {
	for _, option := range options {
		if option == "no-new-privileges:true" {
			return true
		}
	}
	return false
}

func TestHardeningRejectsNoNewPrivilegesFalse(t *testing.T) {
	if hasExactNoNewPrivilegesTrue([]string{"no-new-privileges:false"}) {
		t.Fatal("no-new-privileges:false fue aceptado como endurecimiento")
	}
	if hasExactNoNewPrivilegesTrue([]string{"no-new-privileges"}) {
		t.Fatal("no-new-privileges sin valor fue aceptado como endurecimiento")
	}
	if !hasExactNoNewPrivilegesTrue([]string{"no-new-privileges:true"}) {
		t.Fatal("no-new-privileges:true fue rechazado")
	}
}
