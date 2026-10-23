package docker

import (
	"strings"
	"testing"
)

// TestPureRuntimeWorkspaceCmd_IdempotentSetupSeed valida el contrato del shell
// de arranque: el setup de la plantilla se ejecuta una sola vez por volumen,
// sellado con /home/workspace/.solv_initialized, y su salida queda en el log
// de diagnóstico. Una reactivación tras hibernar no debe reejecutarlo.
func TestPureRuntimeWorkspaceCmd_IdempotentSetupSeed(t *testing.T) {
	if !strings.Contains(pureRuntimeWorkspaceCmd, `[ ! -f /home/workspace/.solv_initialized ]`) {
		t.Errorf("esperado chequeo del sello .solv_initialized antes de ejecutar el setup")
	}
	if !strings.Contains(pureRuntimeWorkspaceCmd, `touch /home/workspace/.solv_initialized`) {
		t.Errorf("esperado sellado del volumen tras ejecutar el setup")
	}
	if !strings.Contains(pureRuntimeWorkspaceCmd, `SOLV_SETUP_SCRIPT`) {
		t.Errorf("esperado referencia a la variable SOLV_SETUP_SCRIPT")
	}
	if !strings.Contains(pureRuntimeWorkspaceCmd, `> /home/workspace/.solv_setup.log 2>&1`) {
		t.Errorf("esperado volcado de la salida del setup a .solv_setup.log")
	}
}

// TestPureRuntimeWorkspaceCmd_PreparePermissions y DirectorioBase valida que el
// volumen siempre queda montado con el directorio y los permisos que OpenVSCode
// necesita para escribir como el usuario 1000.
func TestPureRuntimeWorkspaceCmd_PreparePermissions(t *testing.T) {
	if !strings.Contains(pureRuntimeWorkspaceCmd, `mkdir -p /home/workspace`) {
		t.Errorf("esperado mkdir del directorio base del proyecto")
	}
	if !strings.Contains(pureRuntimeWorkspaceCmd, `chmod 0777 /home/workspace`) {
		t.Errorf("esperado chmod sobre el directorio base del proyecto")
	}
}

// TestPureRuntimeWorkspaceCmd_OpensProjectFolderByDefault valida que el editor
// abre directo en el proyecto del estudiante sin intervención manual.
func TestPureRuntimeWorkspaceCmd_OpensProjectFolderByDefault(t *testing.T) {
	if !strings.Contains(pureRuntimeWorkspaceCmd, `--default-folder /home/workspace`) {
		t.Errorf("esperado --default-folder /home/workspace en el arranque de OpenVSCode")
	}
}

// TestPureRuntimeWorkspaceCmd_FallbackWithoutOpenVSCode valida que una imagen
// sin el binario de OpenVSCode no muere en arranque: duerme indefinidamente y
// conserva el volumen, en lugar de fallar con "executable file not found".
func TestPureRuntimeWorkspaceCmd_FallbackWithoutOpenVSCode(t *testing.T) {
	if !strings.Contains(pureRuntimeWorkspaceCmd, `[ -x /home/.openvscode-server/bin/openvscode-server ]`) {
		t.Errorf("esperado chequeo de existencia del binario OpenVSCode")
	}
	if !strings.Contains(pureRuntimeWorkspaceCmd, `exec sleep infinity`) {
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
