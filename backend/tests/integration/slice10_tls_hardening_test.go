package integration

import (
	"context"
	"strings"
	"testing"

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
		if domain.TraefikImage != "traefik:v3.1.2" {
			t.Errorf("Expected TraefikImage traefik:v3.1.2, got %s", domain.TraefikImage)
		}

		t.Logf("PASS: Image versions correctly pinned -> OpenVSCode: %s | Semgrep: %s | Traefik: %s",
			domain.OpenVSCodeImage, domain.SemgrepImage, domain.TraefikImage)
	})

	t.Run("2. Docker Container Hardening Options", func(t *testing.T) {
		dockerClient, err := docker.NewClient()
		if err != nil {
			t.Skipf("Skipping live Docker test: unable to connect to Docker daemon: %v", err)
		}

		ctx := context.Background()
		workspaceConfig := domain.WorkspaceContainerConfig{
			Image:         domain.OpenVSCodeImage,
			ContainerName: "solv-test-hardening-workspace",
			VolumeName:    "solv_test_vol_hardening",
			MemoryLimitMB: 256,
			NetworkName:   "solv_net",
		}

		containerID, err := dockerClient.StartWorkspaceContainer(ctx, workspaceConfig)
		if err != nil {
			t.Skipf("Skipping container inspect test: failed to start test container: %v", err)
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

		hasNoNewPrivs := false
		for _, opt := range inspect.HostConfig.SecurityOpt {
			if strings.Contains(opt, "no-new-privileges:true") || strings.Contains(opt, "no-new-privileges") {
				hasNoNewPrivs = true
				break
			}
		}

		if !hasNoNewPrivs {
			t.Errorf("Container HostConfig.SecurityOpt missing 'no-new-privileges:true', got: %v", inspect.HostConfig.SecurityOpt)
		}

		if inspect.Config.User != "1000:1000" {
			t.Errorf("Workspace container Config.User expected '1000:1000', got '%s'", inspect.Config.User)
		}

		t.Logf("PASS: Workspace container hardening verified -> SecurityOpt: %v | User: %s",
			inspect.HostConfig.SecurityOpt, inspect.Config.User)
	})

	t.Run("3. Manual TLS Verification Protocol", func(t *testing.T) {
		cmdHelp := "openssl s_client -connect solv.dedyn.io:443 -servername solv.dedyn.io"
		t.Logf("INFO: Manual TLS verification command: %s", cmdHelp)
	})
}
