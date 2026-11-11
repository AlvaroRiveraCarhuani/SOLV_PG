package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func validNodeProfile() DependencyProfile {
	return DependencyProfile{
		Version:         1,
		Detect:          []string{"package.json"},
		Lockfiles:       []string{"pnpm-lock.yaml", "package-lock.json"},
		Install:         "corepack enable && (pnpm install --frozen-lockfile || npm ci)",
		Materialize:     DependencyMaterializeTree,
		DependencyPaths: []string{"node_modules"},
		CacheEnv:        map[string]string{"npm_config_cache": "/opt/solv/cache/npm"},
		Verify:          "npm ls --depth=0 --offline",
		Network:         "build",
		OfflineFlags:    "npm_config_offline=true",
	}
}

func TestDependencyProfileEmptyIsValid(t *testing.T) {
	if err := (DependencyProfile{}).Validate(); err != nil {
		t.Fatalf("un perfil vacío es válido, obtuvo: %v", err)
	}
	if !(DependencyProfile{}).IsEmpty() {
		t.Fatal("el perfil vacío debería informarse como vacío")
	}
	if validNodeProfile().IsEmpty() {
		t.Fatal("un perfil declarado no es vacío")
	}
}

func TestDependencyProfileValidateAcceptsFamilyProfiles(t *testing.T) {
	cases := map[string]DependencyProfile{
		"node tree": validNodeProfile(),
		"python tree": {
			Version:         1,
			Detect:          []string{"requirements.txt", "pyproject.toml"},
			Lockfiles:       []string{"uv.lock", "poetry.lock"},
			Install:         "python3 -m venv .venv && .venv/bin/pip install -r requirements.txt",
			Materialize:     DependencyMaterializeTree,
			DependencyPaths: []string{".venv"},
			Verify:          ".venv/bin/python -m pip check",
			Network:         "build",
			OfflineFlags:    "PIP_NO_INDEX=1",
		},
		"java cache": {
			Version:         1,
			Detect:          []string{"pom.xml", "build.gradle"},
			Install:         "mvn -B -q dependency:go-offline",
			Materialize:     DependencyMaterializeCache,
			DependencyPaths: []string{"/opt/solv/cache/m2"},
			CacheEnv:        map[string]string{"MAVEN_OPTS": "-Dmaven.repo.local=/opt/solv/cache/m2"},
			Verify:          "mvn -o -q -DskipTests package",
			Network:         "build",
			OfflineFlags:    "-o",
		},
		"dotnet glob manifest": {
			Version:         1,
			Detect:          []string{"*.csproj", "*.sln"},
			Lockfiles:       []string{"packages.lock.json"},
			Install:         "dotnet restore",
			Materialize:     DependencyMaterializeCache,
			DependencyPaths: []string{"/opt/solv/cache/nuget"},
			Verify:          "dotnet build --no-restore",
			Network:         "build",
		},
		"c con makefile sin manifiesto": {
			Version:     1,
			Detect:      []string{"CMakeLists.txt", "conanfile.txt"},
			Install:     "cmake -S . -B build",
			Materialize: DependencyMaterializeNone,
			Network:     "none",
		},
	}

	for name, profile := range cases {
		t.Run(name, func(t *testing.T) {
			if err := profile.Validate(); err != nil {
				t.Fatalf("se esperaba un perfil válido, obtuvo: %v", err)
			}
		})
	}
}

func TestDependencyProfileValidateRejects(t *testing.T) {
	cases := map[string]struct {
		profile DependencyProfile
		want    string
	}{
		"sin versión": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.Version = 0
				return p
			}(),
			want: "version",
		},
		"materialize desconocido": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.Materialize = "volumen"
				return p
			}(),
			want: "materialize",
		},
		"sin manifiestos": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.Detect = nil
				return p
			}(),
			want: "detect",
		},
		"sin install": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.Install = ""
				return p
			}(),
			want: "install",
		},
		"sin verify": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.Verify = ""
				return p
			}(),
			want: "verify",
		},
		"install con salto de línea": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.Install = "npm ci\nrm -rf /"
				return p
			}(),
			want: "control",
		},
		"ruta con escape de proyecto": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.DependencyPaths = []string{"../vecino"}
				return p
			}(),
			want: "..",
		},
		"ruta absoluta en tree": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.DependencyPaths = []string{"/etc/passwd"}
				return p
			}(),
			want: "relativa",
		},
		"ruta relativa en cache": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.Materialize = DependencyMaterializeCache
				p.DependencyPaths = []string{"vendor"}
				return p
			}(),
			want: "absoluta",
		},
		"tree sin rutas": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.DependencyPaths = nil
				return p
			}(),
			want: "dependency_paths",
		},
		"variable de entorno inválida": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.CacheEnv = map[string]string{"npm cache": "/tmp"}
				return p
			}(),
			want: "variable",
		},
		"manifiesto con ruta": {
			profile: func() DependencyProfile {
				p := validNodeProfile()
				p.Detect = []string{"sub/package.json"}
				return p
			}(),
			want: "manifiesto",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.profile.Validate()
			if err == nil {
				t.Fatalf("se esperaba error que mencione %q", tc.want)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("el error debería mencionar %q, obtuvo: %v", tc.want, err)
			}
		})
	}
}

func TestDependencyProfileScanAndValueRoundTrip(t *testing.T) {
	original := validNodeProfile()
	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value falló: %v", err)
	}

	var decoded DependencyProfile
	if err := decoded.Scan(value); err != nil {
		t.Fatalf("Scan falló: %v", err)
	}
	if decoded.Version != original.Version || decoded.Materialize != original.Materialize || decoded.Verify != original.Verify {
		t.Fatalf("el perfil decodificado no coincide: %+v", decoded)
	}

	var fromString DependencyProfile
	raw, _ := json.Marshal(original)
	if err := fromString.Scan(string(raw)); err != nil {
		t.Fatalf("Scan desde string falló: %v", err)
	}
	if fromString.Install != original.Install {
		t.Fatalf("el comando de instalación cambió al decodificar: %q", fromString.Install)
	}

	var fromNil DependencyProfile
	if err := fromNil.Scan(nil); err != nil {
		t.Fatalf("Scan nil falló: %v", err)
	}
	if !fromNil.IsEmpty() {
		t.Fatal("un valor nulo debe producir un perfil vacío")
	}
}
