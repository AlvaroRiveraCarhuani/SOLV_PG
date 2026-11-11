package integration

import (
	"encoding/json"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

// TestDependencyProfilesDeliverInsideTheProject comprueba las semillas que
// corrigen la entrega. La preparación corre en la misma ruta que ve el alumno y
// el volumen se enlaza desde el árbol, así que una ruta absoluta fuera del
// proyecto nunca llega al aula: el enlace la busca donde no está y el entorno
// abre sin dependencias.
func TestDependencyProfilesDeliverInsideTheProject(t *testing.T) {
	db := newWorkspaceProjectTestDB(t)
	rows := make([]struct {
		Slug    string `db:"slug"`
		Profile []byte `db:"dependency_profile"`
	}, 0)
	err := db.GetDB().Select(&rows, `
		SELECT slug, dependency_profile
		FROM template_components
		WHERE slug IN ('java-21', 'go-122', 'rust-stable', 'dotnet-8', 'c-gcc13', 'cpp-gcc13')
		ORDER BY slug`)
	if err != nil {
		t.Fatalf("no se pudieron leer los perfiles: %v", err)
	}
	if len(rows) != 6 {
		t.Fatalf("esperaba seis perfiles, obtuve %d", len(rows))
	}

	profiles := make(map[string]domain.DependencyProfile, len(rows))
	for _, row := range rows {
		var profile domain.DependencyProfile
		if err := json.Unmarshal(row.Profile, &profile); err != nil {
			t.Fatalf("el perfil de %s no es JSON válido: %v", row.Slug, err)
		}
		if err := profile.Validate(); err != nil {
			t.Fatalf("el perfil de %s no valida: %v", row.Slug, err)
		}
		for _, dependencyPath := range profile.DependencyPaths {
			if strings.HasPrefix(dependencyPath, "/") {
				t.Fatalf("%s declara la ruta absoluta %q: el enlace solo copia rutas del proyecto",
					row.Slug, dependencyPath)
			}
		}
		profiles[row.Slug] = profile
	}

	for _, slug := range []string{"java-21", "go-122", "rust-stable", "dotnet-8"} {
		profile := profiles[slug]
		if profile.Materialize != domain.DependencyMaterializeTree {
			t.Fatalf("%s debía entregarse como árbol, obtuvo %q", slug, profile.Materialize)
		}
		if !dependencyPathsHavePrefix(profile.DependencyPaths, ".solv-cache/") {
			t.Fatalf("%s debía guardar su almacén en .solv-cache, obtuvo %v", slug, profile.DependencyPaths)
		}
		if len(profile.RuntimeEnv) == 0 {
			t.Fatalf("%s debía declarar el entorno que apunta al almacén en el aula", slug)
		}
		for name, value := range profile.RuntimeEnv {
			if !strings.Contains(value, "/home/workspace/.solv-cache/") {
				t.Fatalf("%s: %s apunta fuera del almacén entregado: %q", slug, name, value)
			}
		}
	}
	if !dependencyPathPresent(profiles["dotnet-8"].DependencyPaths, "obj") {
		t.Fatal("dotnet-8 debía entregar obj: sin el archivo de activos no hay compilación")
	}
	if !strings.Contains(profiles["java-21"].Install, "pom.xml") {
		t.Fatal("java-21 debía explicar que solo resuelve Maven")
	}
	for _, slug := range []string{"c-gcc13", "cpp-gcc13"} {
		if profiles[slug].Materialize != domain.DependencyMaterializeNone {
			t.Fatalf("%s no resuelve dependencias de proyecto y debía quedar en none", slug)
		}
	}
}

func dependencyPathsHavePrefix(paths []string, prefix string) bool {
	for _, dependencyPath := range paths {
		if strings.HasPrefix(dependencyPath, prefix) {
			return true
		}
	}
	return false
}

func dependencyPathPresent(paths []string, want string) bool {
	for _, dependencyPath := range paths {
		if dependencyPath == want {
			return true
		}
	}
	return false
}
