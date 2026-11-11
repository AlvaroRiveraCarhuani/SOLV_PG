package docker

import (
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

func TestApplyOfflineFlags(t *testing.T) {
	cases := []struct{ command, flags, want string }{
		{"npm ci", "", "npm ci"},
		{"mvn dependency:go-offline", "-o", "mvn dependency:go-offline -o"},
		{"npm ci", "npm_config_offline=true", "npm_config_offline=true npm ci"},
		{"go mod download", "GOFLAGS=-mod=mod GOPROXY=off", "GOFLAGS=-mod=mod GOPROXY=off go mod download"},
		{"pip install -r requirements.txt", "PIP_NO_INDEX=1", "PIP_NO_INDEX=1 pip install -r requirements.txt"},
	}
	for _, tc := range cases {
		if got := applyOfflineFlags(tc.command, tc.flags); got != tc.want {
			t.Fatalf("applyOfflineFlags(%q, %q) = %q, se esperaba %q", tc.command, tc.flags, got, tc.want)
		}
	}
}

func TestIsTransientPreparationFailure(t *testing.T) {
	transient := []string{
		"npm ERR! network request failed: ETIMEDOUT",
		"dial tcp: lookup registry.npmjs.org: no such host",
		"read tcp 10.0.0.2:443: connection reset by peer",
		"Get https://pypi.org/simple/: 503 Service Unavailable",
		"toomanyrequests: too many requests",
	}
	for _, message := range transient {
		if !isTransientPreparationFailure(message) {
			t.Fatalf("debería considerarse transitorio: %q", message)
		}
	}

	deterministic := []string{
		"error: no se encontró el paquete 'pandas==99.0'",
		"npm ERR! Missing script: build",
		"gcc: fatal error: main.c: No such file or directory",
		"permission denied",
		"no space left on device",
	}
	for _, message := range deterministic {
		if isTransientPreparationFailure(message) {
			t.Fatalf("no debería considerarse transitorio: %q", message)
		}
	}
}

// TestRuntimeDirScriptCreatesUserDirectories fija el paso que evita que la CLI
// de .NET muera antes de restaurar: el directorio de ejecución se crea dentro
// del árbol y como el usuario del contenedor, no desde el host.
func TestRuntimeDirScriptCreatesUserDirectories(t *testing.T) {
	got := runtimeDirScript("/home/workspace/.solv-home")
	want := "mkdir -p '/home/workspace/.solv-home' '/home/workspace/.solv-home/run'; chmod 700 '/home/workspace/.solv-home/run'"
	if got != want {
		t.Fatalf("runtimeDirScript() = %q, se esperaba %q", got, want)
	}
}

// TestPreparationEnvDeclaresRuntimeDir comprueba que el entorno de la
// preparación declare XDG_RUNTIME_DIR y que el perfil conserve la última palabra.
func TestPreparationEnvDeclaresRuntimeDir(t *testing.T) {
	request := domain.DependencyPreparationRequest{
		ProjectPath: "/home/workspace",
		CacheEnv:    map[string]string{"NUGET_PACKAGES": "/home/workspace/.solv-cache/nuget"},
	}
	env := preparationEnv(request, "/home/workspace/.solv-home")
	if !hasEnv(env, "XDG_RUNTIME_DIR=/home/workspace/.solv-home/run") {
		t.Fatalf("el entorno debía declarar el directorio de ejecución: %v", env)
	}

	request.CacheEnv["XDG_RUNTIME_DIR"] = "/home/workspace/.solv-cache/run"
	env = preparationEnv(request, "/home/workspace/.solv-home")
	if countEnv(env, "XDG_RUNTIME_DIR=") != 1 || !hasEnv(env, "XDG_RUNTIME_DIR=/home/workspace/.solv-cache/run") {
		t.Fatalf("el perfil debía conservar su propio XDG_RUNTIME_DIR: %v", env)
	}
}

func hasEnv(env []string, want string) bool {
	for _, entry := range env {
		if entry == want {
			return true
		}
	}
	return false
}

func countEnv(env []string, prefix string) int {
	count := 0
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			count++
		}
	}
	return count
}

func TestRemapCachePathKeepsSharedCacheInsideTree(t *testing.T) {
	if got := remapCachePath("/home/workspace/proyecto", "/opt/solv/cache/m2"); got != "/home/workspace/proyecto/.solv-cache/m2" {
		t.Fatalf("la caché compartida debe remapearse dentro del árbol, obtuvo %q", got)
	}
	if got := remapCachePath("/home/workspace/proyecto", "/tmp/pip"); got != "/tmp/pip" {
		t.Fatalf("una ruta ajena a la caché no debe tocarse, obtuvo %q", got)
	}
}

func TestShellQuoteEscapesSingleQuotes(t *testing.T) {
	if got := shellQuote("/home/workspace/proyecto"); got != "'/home/workspace/proyecto'" {
		t.Fatalf("la ruta debe ir entre comillas simples, obtuvo %q", got)
	}
	if got := shellQuote("/home/con'quote"); got != `'/home/con'\''quote'` {
		t.Fatalf("las comillas simples deben escaparse, obtuvo %q", got)
	}
}
