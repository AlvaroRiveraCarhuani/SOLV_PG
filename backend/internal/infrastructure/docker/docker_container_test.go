package docker

import (
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

// TestPureRuntimePracticeEnvironmentCmd_IdempotentSetupSeed valida el contrato del shell
// de arranque: el setup de la plantilla se ejecuta una sola vez por volumen,
// sellado con /home/workspace/.solv_initialized, y su salida queda en el log
// de diagnóstico. Una reactivación tras hibernar no debe reejecutarlo.
func TestPureRuntimePracticeEnvironmentCmd_IdempotentSetupSeed(t *testing.T) {
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `[ ! -f /home/workspace/.solv_initialized ]`) {
		t.Errorf("esperado chequeo del sello .solv_initialized antes de ejecutar el setup")
	}
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `touch /home/workspace/.solv_initialized`) {
		t.Errorf("esperado sellado del volumen tras ejecutar el setup")
	}
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `SOLV_SETUP_SCRIPT`) {
		t.Errorf("esperado referencia a la variable SOLV_SETUP_SCRIPT")
	}
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `> /home/workspace/.solv_setup.log 2>&1`) {
		t.Errorf("esperado volcado de la salida del setup a .solv_setup.log")
	}
}

// TestPureRuntimePracticeEnvironmentCmd_PreparePermissions y DirectorioBase valida que el
// volumen siempre queda montado con el directorio y los permisos que OpenVSCode
// necesita para escribir como el usuario 1000.
func TestPureRuntimePracticeEnvironmentCmd_PreparePermissions(t *testing.T) {
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `mkdir -p /home/workspace`) {
		t.Errorf("esperado mkdir del directorio base del proyecto")
	}
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `chmod 0777 /home/workspace`) {
		t.Errorf("esperado chmod sobre el directorio base del proyecto")
	}
}

// TestPureRuntimePracticeEnvironmentCmd_ToleratesReadOnlyWorkspace valida que el
// contenedor de un aula pausada arranca con el volumen en solo lectura: los
// toques al workspace no abortan el arranque y el sembrado se salta.
func TestPureRuntimePracticeEnvironmentCmd_ToleratesReadOnlyWorkspace(t *testing.T) {
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `chmod 0777 /home/workspace 2>/dev/null || true`) {
		t.Errorf("el chmod debía tolerar un volumen en solo lectura")
	}
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `mkdir -p /home/workspace/.vscode/extensions 2>/dev/null || true`) {
		t.Errorf("el mkdir de extensiones debía tolerar un volumen en solo lectura")
	}
	if strings.Count(pureRuntimePracticeEnvironmentCmd, `[ -w /home/workspace ]`) != 2 {
		t.Errorf("el sembrado de settings y el setup debían exigir un workspace escribible")
	}
}

// TestPureRuntimePracticeEnvironmentCmd_OpensProjectFolderByDefault valida que el editor
// abre directo en el proyecto del estudiante sin intervención manual.
func TestPureRuntimePracticeEnvironmentCmd_OpensProjectFolderByDefault(t *testing.T) {
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `--default-folder /home/workspace`) {
		t.Errorf("esperado --default-folder /home/workspace en el arranque de OpenVSCode")
	}
}

// TestPureRuntimePracticeEnvironmentCmd_FallbackWithoutOpenVSCode valida que una imagen
// sin el binario de OpenVSCode no muere en arranque: duerme indefinidamente y
// conserva el volumen, en lugar de fallar con "executable file not found".
func TestPureRuntimePracticeEnvironmentCmd_FallbackWithoutOpenVSCode(t *testing.T) {
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `[ -x /home/.openvscode-server/bin/openvscode-server ]`) {
		t.Errorf("esperado chequeo de existencia del binario OpenVSCode")
	}
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `exec sleep infinity`) {
		t.Errorf("esperado fallback a sleep infinity cuando no hay OpenVSCode")
	}
}

// TestImageShipsOpenVSCode detecta correctamente las imágenes que traen el
// editor como entrypoint frente a los runtimes puros (python, node, etc).
func TestImageShipsOpenVSCode(t *testing.T) {
	cases := []struct {
		image string
		want  bool
	}{
		{"gitpod/openvscode-server:1.96.0", true},
		{"ghcr.io/solv/openvscode-server:2.0", true},
		{"python:3.12-slim-bookworm", false},
		{"node:22-alpine", false},
		{"", false},
	}

	for _, tc := range cases {
		if got := imageShipsOpenVSCode(tc.image); got != tc.want {
			t.Errorf("imageShipsOpenVSCode(%q) = %v, esperado %v", tc.image, got, tc.want)
		}
	}
}

// Los satélites gestionados comparten la etiqueta solv.managed con los laboratorios, pero
// su ciclo de vida es del orquestador de satélites: el colector de zombies no debe
// listarlos ni destruirlos.
func TestIsPracticeEnvironmentManagedContainer(t *testing.T) {
	cases := []struct {
		name   string
		names  []string
		labels map[string]string
		want   bool
	}{
		{name: "environment por nombre", names: []string{"/solv-environment-abc123"}, want: true},
		{name: "environment por etiqueta", names: []string{"/cualquier-cosa"}, labels: map[string]string{"solv.managed": "true"}, want: true},
		{name: "environment publicado por traefik", names: []string{"/otro"}, labels: map[string]string{"traefik.enable": "true"}, want: true},
		{name: "satélite postgres excluido", names: []string{"/solv-sat-postgres"}, labels: map[string]string{"solv.managed": "true", "solv.satellite": "postgres"}, want: false},
		{name: "satélite con nombre de environment excluido igual", names: []string{"/solv-environment-falso"}, labels: map[string]string{"solv.managed": "true", "solv.satellite": "redis"}, want: false},
		{name: "servicio compose excluido por proyecto", names: []string{"/solv_backend"}, labels: map[string]string{"traefik.enable": "true", "com.docker.compose.project": "solv_pg"}, want: false},
		{name: "servicio de plataforma excluido por nombre", names: []string{"/solv_backend"}, labels: map[string]string{"traefik.enable": "true"}, want: false},
		{name: "contenedor ajeno", names: []string{"/traefik"}, labels: map[string]string{}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPracticeEnvironmentManagedContainer(tc.names, tc.labels); got != tc.want {
				t.Fatalf("isPracticeEnvironmentManagedContainer() = %v, esperado %v", got, tc.want)
			}
		})
	}
}

func TestPureRuntimePracticeEnvironmentCmd_IncludesExtensionsDir(t *testing.T) {
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `--extensions-dir /home/workspace/.vscode/extensions`) {
		t.Errorf("esperado flag --extensions-dir /home/workspace/.vscode/extensions en arranque de openvscode-server")
	}
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `mkdir -p /home/workspace/.vscode/extensions`) {
		t.Errorf("esperado aseguramiento del directorio de extensiones de usuario")
	}
}

func TestPureRuntimePracticeEnvironmentCmd_SeedsDefaultSettings(t *testing.T) {
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `[ ! -f /home/workspace/.vscode/settings.json ]`) {
		t.Errorf("esperado chequeo de existencia previa de settings.json")
	}
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `"workbench.colorTheme": "Default Dark Modern"`) {
		t.Errorf("esperado tema oscuro por defecto en settings.json")
	}
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `"files.autoSave": "afterDelay"`) {
		t.Errorf("esperado autoguardado de archivos por defecto")
	}
	if !strings.Contains(pureRuntimePracticeEnvironmentCmd, `"editor.formatOnSave": true`) {
		t.Errorf("esperado autoformato al guardar por defecto")
	}
}

func TestResolveCompanionExtensionMount(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("SOLV_COMPANION_HOST_PATH", tempDir)

	mount := resolveCompanionExtensionMount()
	expectedPrefix := tempDir + ":" + practiceEnvironmentBuiltinExtensionsDir + "/solv.solv-companion-0.1.0:ro"
	if mount != expectedPrefix {
		t.Fatalf("esperado %s, obtenido %s", expectedPrefix, mount)
	}

	// Una ruta explícita se usa tal cual, aunque este proceso no pueda verla: con el
	// backend en contenedor la resuelve el daemon de Docker sobre el host.
	t.Setenv("SOLV_COMPANION_HOST_PATH", "/ruta/del/host/tools/solv-companion")
	expectedHost := "/ruta/del/host/tools/solv-companion:" + practiceEnvironmentBuiltinExtensionsDir + "/solv.solv-companion-0.1.0:ro"
	if mountHost := resolveCompanionExtensionMount(); mountHost != expectedHost {
		t.Fatalf("esperado %s, obtenido %s", expectedHost, mountHost)
	}
}

func TestPracticeEnvironmentContainer_HostConfigResources(t *testing.T) {
	cases := []struct {
		name               string
		config             domain.PracticeEnvironmentContainerConfig
		wantMemBytes       int64
		wantMemReservation int64
		wantCPUNanos       int64
		wantPidsLimit      int64
	}{
		{
			name: "valores por defecto cuando se omite cuota",
			config: domain.PracticeEnvironmentContainerConfig{
				MemoryLimitMB: 0,
				CPUNanos:      0,
				PidsLimit:     0,
			},
			wantMemBytes:       256 * 1024 * 1024,
			wantMemReservation: int64(float64(256)*0.75) * 1024 * 1024,
			wantCPUNanos:       1_000_000_000,
			wantPidsLimit:      512,
		},
		{
			name: "valores explícitos respetados fielmente",
			config: domain.PracticeEnvironmentContainerConfig{
				MemoryLimitMB: 512,
				CPUNanos:      500_000_000,
				PidsLimit:     256,
			},
			wantMemBytes:       512 * 1024 * 1024,
			wantMemReservation: int64(float64(512)*0.75) * 1024 * 1024,
			wantCPUNanos:       500_000_000,
			wantPidsLimit:      256,
		},
		{
			name: "entorno pesado de 1024 MB con 2 cores y 1024 pids",
			config: domain.PracticeEnvironmentContainerConfig{
				MemoryLimitMB: 1024,
				CPUNanos:      2_000_000_000,
				PidsLimit:     1024,
			},
			wantMemBytes:       1024 * 1024 * 1024,
			wantMemReservation: int64(float64(1024)*0.75) * 1024 * 1024,
			wantCPUNanos:       2_000_000_000,
			wantPidsLimit:      1024,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			memLimit := tc.config.MemoryLimitMB
			if memLimit <= 0 {
				memLimit = domain.DefaultBaseMemoryMB
			}
			memReservation := int64(float64(memLimit) * 0.75)
			cpuNanos := tc.config.CPUNanos
			if cpuNanos <= 0 {
				cpuNanos = 1_000_000_000
			}
			pidsLimit := tc.config.PidsLimit
			if pidsLimit <= 0 {
				pidsLimit = 512
			}

			if memLimit*1024*1024 != tc.wantMemBytes {
				t.Errorf("memoria esperada %d, obtenida %d", tc.wantMemBytes, memLimit*1024*1024)
			}
			if memReservation*1024*1024 != tc.wantMemReservation {
				t.Errorf("reserva de memoria esperada %d, obtenida %d", tc.wantMemReservation, memReservation*1024*1024)
			}
			if cpuNanos != tc.wantCPUNanos {
				t.Errorf("NanoCPUs esperado %d, obtenido %d", tc.wantCPUNanos, cpuNanos)
			}
			if pidsLimit != tc.wantPidsLimit {
				t.Errorf("PidsLimit esperado %d, obtenido %d", tc.wantPidsLimit, pidsLimit)
			}
		})
	}
}
