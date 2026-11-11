package docker

import (
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

func TestBuildDependencyLinkScriptUsesHardLinksWithCopyWithFallback(t *testing.T) {
	tree := domain.WorkspaceDependencyTree{
		RootPath: "/var/lib/solv/deps/abc",
		Key:      "clave-de-preparacion",
		Paths:    []string{"node_modules", ".venv"},
	}
	script, err := buildDependencyLinkScript(tree)
	if err != nil {
		t.Fatalf("el guion debía construirse: %v", err)
	}

	expected := []string{
		"set -eu",
		"/prepared/node_modules",
		"/prepared/.venv",
		"cp -al",
		"cp -a",
		"clave-de-preparacion",
		"test -e",
		"exit 3",
	}
	for _, fragment := range expected {
		if !strings.Contains(script, fragment) {
			t.Fatalf("el guion no contiene %q:\n%s", fragment, script)
		}
	}
	if strings.Contains(script, "reflink") {
		t.Fatal("el reflink no se declara: la caída a copia cubre el caso")
	}
}

func TestBuildDependencyLinkScriptRejectsUnsafePaths(t *testing.T) {
	tree := domain.WorkspaceDependencyTree{
		RootPath: "/var/lib/solv/deps/abc",
		Key:      "clave",
		Paths:    []string{"../../etc"},
	}
	if _, err := buildDependencyLinkScript(tree); err == nil {
		t.Fatal("una ruta que sale del volumen debe rechazarse antes de crear el contenedor")
	}
}
