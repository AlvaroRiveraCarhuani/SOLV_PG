//go:build integration

package services

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"

	"solv-backend/internal/core/domain"
)

// auditDockerWorker abre el daemon real. La prueba se omite si Docker no está
// disponible, salvo que la suite lo declare obligatorio.
func auditDockerWorker(t *testing.T) *TemplateAuditWorker {
	t.Helper()
	if testing.Short() {
		t.Skip("prueba con Docker omitida con -short")
	}
	if os.Getenv("SOLV_SKIP_DOCKER_TESTS") == "1" {
		t.Skip("prueba con Docker omitida por SOLV_SKIP_DOCKER_TESTS")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de Docker: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		if os.Getenv("SOLV_REQUIRE_DOCKER") == "1" {
			t.Fatalf("Docker es obligatorio y no responde: %v", err)
		}
		t.Skipf("Docker no disponible: %v", err)
	}
	return NewTemplateAuditWorker(cli, nil)
}

func auditImageName(t *testing.T, envName, fallback string) string {
	t.Helper()
	if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
		return value
	}
	return fallback
}

// TestTemplateAuditWorkerSmokeFailsWhenToolIsMissing comprueba que el smoke use
// el contenedor real y detecte un binario ausente. Es el caso que degrada una
// plantilla aprobada.
func TestTemplateAuditWorkerSmokeFailsWhenToolIsMissing(t *testing.T) {
	worker := auditDockerWorker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	image := auditImageName(t, "SOLV_AUDIT_SMOKE_IMAGE", "alpine:3.20")
	if err := worker.ensureImageAvailable(ctx, image); err != nil {
		if os.Getenv("SOLV_REQUIRE_DOCKER") == "1" {
			t.Fatalf("no se pudo preparar la imagen %s: %v", image, err)
		}
		t.Skipf("no se pudo preparar la imagen %s: %v", image, err)
	}

	item := &domain.AdminTemplateReviewItem{
		DockerImage:   image,
		BaseRamMB:     128,
		ToolsDeclared: []string{"solv-herramienta-inexistente"},
	}
	status, output := worker.runSmokeTest(ctx, item)
	if status != "failed" {
		t.Fatalf("el smoke debía fallar con una herramienta ausente, obtuve status=%q output=%q", status, output)
	}
	if !strings.Contains(output, "Faltan las siguientes herramientas") {
		t.Fatalf("el motivo debía nombrar las herramientas ausentes, obtuve %q", output)
	}

	finalStatus, reason := resolveAuditOutcome("APROBADA", status, output, "passed", 0)
	if finalStatus != "APROBADA" {
		t.Fatalf("una plantilla publicada conserva APROBADA tras un smoke fallido, obtuve %q", finalStatus)
	}
	if !strings.Contains(reason, "solv-herramienta-inexistente") {
		t.Fatalf("el motivo de degradación debía conservar el detalle del smoke, obtuve %q", reason)
	}
}

// TestTemplateAuditWorkerTrivyFindsCriticalCVE ejecuta el escáner real contra
// una imagen antigua. Descarga la base CVE y la imagen de Trivy, así que queda
// detrás de una variable explícita. La imagen se puede cambiar con
// SOLV_TRIVY_TEST_IMAGE.
func TestTemplateAuditWorkerTrivyFindsCriticalCVE(t *testing.T) {
	if os.Getenv("SOLV_TRIVY_INTEGRATION") != "1" {
		t.Skip("definir SOLV_TRIVY_INTEGRATION=1 para ejecutar el escaneo real de Trivy")
	}
	worker := auditDockerWorker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	image := auditImageName(t, "SOLV_TRIVY_TEST_IMAGE", "debian:10-slim")
	if err := worker.ensureImageAvailable(ctx, image); err != nil {
		t.Fatalf("no se pudo preparar la imagen vulnerable %s: %v", image, err)
	}
	if err := worker.ensureImageAvailable(ctx, TrivyDockerImage); err != nil {
		t.Fatalf("no se pudo preparar el escáner %s: %v", TrivyDockerImage, err)
	}
	if err := worker.UpdateTrivyDB(ctx); err != nil {
		t.Fatalf("no se pudo sincronizar la base CVE: %v", err)
	}

	status, critical, high, report := worker.runTrivyScan(ctx, image)
	if status != "failed" || critical == 0 {
		t.Fatalf(
			"Trivy debía encontrar CVEs críticos en %s: status=%q critical=%d high=%d report=%s",
			image, status, critical, high, auditReportTail(report, 2000),
		)
	}

	finalStatus, reason := resolveAuditOutcome("APROBADA", "passed", "", status, critical)
	if finalStatus != "APROBADA" {
		t.Fatalf("una plantilla publicada conserva APROBADA con CVEs críticos, obtuve %q", finalStatus)
	}
	if !strings.Contains(reason, "vulnerabilidad") {
		t.Fatalf("el motivo debía nombrar las vulnerabilidades críticas, obtuve %q", reason)
	}
}

// TestTemplateAuditWorkerPrunesStaleDBDownloads deja un contenedor de descarga
// con la etiqueta del worker y comprueba que el barrido de arranque lo
// elimina. Es el caso que dejaba descargas vivas, sin dueño y con un núcleo
// ocupado, cuando el backend se reiniciaba a mitad de la descarga.
func TestTemplateAuditWorkerPrunesStaleDBDownloads(t *testing.T) {
	worker := auditDockerWorker(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := worker.ensureTrivyImage(ctx); err != nil {
		if os.Getenv("SOLV_REQUIRE_DOCKER") == "1" {
			t.Fatalf("no se pudo preparar el escáner %s: %v", TrivyDockerImage, err)
		}
		t.Skipf("no se pudo preparar el escáner %s: %v", TrivyDockerImage, err)
	}

	stale, err := worker.dockerCli.ContainerCreate(ctx, &container.Config{
		Image:      TrivyDockerImage,
		Entrypoint: []string{"sleep", "600"},
		Labels:     map[string]string{auditContainerLabel: trivyDBContainerRole},
	}, &container.HostConfig{}, nil, nil, "")
	if err != nil {
		t.Fatalf("no se pudo crear el contenedor huérfano de prueba: %v", err)
	}
	t.Cleanup(func() {
		_ = worker.dockerCli.ContainerRemove(context.Background(), stale.ID, container.RemoveOptions{Force: true})
	})
	if err := worker.dockerCli.ContainerStart(ctx, stale.ID, container.StartOptions{}); err != nil {
		t.Fatalf("no se pudo arrancar el contenedor huérfano de prueba: %v", err)
	}

	worker.pruneStaleDBDownloads(ctx)

	if _, err := worker.dockerCli.ContainerInspect(ctx, stale.ID); err == nil || !errdefs.IsNotFound(err) {
		t.Fatalf("el contenedor etiquetado debía eliminarse en el barrido; la inspección devolvió %v", err)
	}
}

func auditReportTail(report []byte, max int) string {
	text := strings.TrimSpace(string(report))
	if len(text) <= max {
		return text
	}
	return "..." + text[len(text)-max:]
}
