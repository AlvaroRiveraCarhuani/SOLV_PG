//go:build integration

package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"

	"solv-backend/internal/core/domain"
)

// dependencyFamilyCase describe una familia tal como la declara el catálogo: los
// archivos del proyecto, el comando que instala, el que comprueba sin red, el
// entorno del aula y las rutas que tienen que quedar dentro del árbol.
type dependencyFamilyCase struct {
	name         string
	image        string
	files        map[string]string
	install      string
	verify       string
	offlineFlags string
	env          map[string]string
	paths        []string
	guard        string
}

func dependencyTestImage(variable, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(variable)); value != "" {
		return value
	}
	return fallback
}

// dependencyFamilyCases cubre las cuatro familias que antes usaban rutas fuera
// del proyecto. El proyecto se escribe en /home/workspace, la misma ruta que ve
// el alumno, y cada almacén queda en .solv-cache dentro del árbol.
func dependencyFamilyCases() []dependencyFamilyCase {
	return []dependencyFamilyCase{
		{
			name:  "java-maven",
			image: dependencyTestImage("SOLV_TEST_JAVA_IMAGE", "maven:3.9-eclipse-temurin-21"),
			files: map[string]string{
				"pom.xml": `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>solv.test</groupId>
  <artifactId>preparacion</artifactId>
  <version>1.0.0</version>
  <properties>
    <maven.compiler.source>21</maven.compiler.source>
    <maven.compiler.target>21</maven.compiler.target>
  </properties>
  <dependencies>
    <dependency>
      <groupId>org.apache.commons</groupId>
      <artifactId>commons-lang3</artifactId>
      <version>3.14.0</version>
    </dependency>
  </dependencies>
</project>
`,
				"src/main/java/solv/test/App.java": `package solv.test;

import org.apache.commons.lang3.StringUtils;

public final class App {
    public static String saludo() {
        return StringUtils.capitalize("hola");
    }
}
`,
			},
			install:      "mvn -B -q dependency:go-offline",
			verify:       "mvn -q -DskipTests package",
			offlineFlags: "-o",
			env:          map[string]string{"MAVEN_OPTS": "-Dmaven.repo.local=/home/workspace/.solv-cache/m2"},
			paths:        []string{".solv-cache/m2"},
			guard:        ".solv-cache/m2/org/apache/commons/commons-lang3",
		},
		{
			name:  "go-modules",
			image: dependencyTestImage("SOLV_TEST_GO_IMAGE", "golang:1.22-alpine"),
			files: map[string]string{
				"go.mod": `module solv.test/preparacion

go 1.22

require rsc.io/quote v1.5.2
`,
				"main.go": `package main

import (
	"fmt"

	"rsc.io/quote"
)

func main() {
	fmt.Println(quote.Hello())
}
`,
			},
			install:      "go mod tidy && go mod download",
			verify:       "go build ./...",
			offlineFlags: "GOFLAGS=-mod=mod GOPROXY=off",
			env: map[string]string{
				"GOMODCACHE": "/home/workspace/.solv-cache/go-mod",
				"GOPATH":     "/home/workspace/.solv-cache/go",
			},
			paths: []string{".solv-cache/go-mod", ".solv-cache/go"},
			guard: ".solv-cache/go-mod/cache/download/rsc.io",
		},
		{
			name:  "rust-cargo",
			image: dependencyTestImage("SOLV_TEST_RUST_IMAGE", "rust:1-alpine"),
			files: map[string]string{
				"Cargo.toml": `[package]
name = "preparacion"
version = "0.1.0"
edition = "2021"

[dependencies]
itoa = "1"
`,
				"src/main.rs": `fn main() {
    let mut buffer = itoa::Buffer::new();
    println!("{}", buffer.format(42));
}
`,
			},
			install:      "cargo fetch",
			verify:       "cargo check",
			offlineFlags: "--offline",
			env:          map[string]string{"CARGO_HOME": "/home/workspace/.solv-cache/cargo"},
			paths:        []string{".solv-cache/cargo"},
			guard:        ".solv-cache/cargo/registry",
		},
		{
			name: "dotnet-nuget",
			// La misma imagen que declara el catálogo para dotnet-8. La CLI crea un
			// mutex con nombre antes de restaurar y lo respalda en XDG_RUNTIME_DIR:
			// el runner crea ese directorio dentro del árbol y lo exporta.
			image: dependencyTestImage("SOLV_TEST_DOTNET_IMAGE", "mcr.microsoft.com/dotnet/sdk:8.0-bookworm-slim"),
			files: map[string]string{
				"preparacion.csproj": `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net8.0</TargetFramework>
    <Nullable>enable</Nullable>
    <ImplicitUsings>enable</ImplicitUsings>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
  </ItemGroup>
</Project>
`,
				"Program.cs": `using Newtonsoft.Json;

Console.WriteLine(JsonConvert.SerializeObject(new { ok = true }));
`,
			},
			install: "dotnet restore",
			verify:  "dotnet build --no-restore",
			env: map[string]string{
				"NUGET_PACKAGES":                        "/home/workspace/.solv-cache/nuget",
				"DOTNET_CLI_TELEMETRY_OPTOUT":           "1",
				"DOTNET_SYSTEM_GLOBALIZATION_INVARIANT": "1",
			},
			paths: []string{".solv-cache/nuget", "obj"},
			guard: "obj/project.assets.json",
		},
	}
}

// TestDependencyPreparationFamiliesIntegration ejercita con Docker real las
// cuatro familias que antes declaraban rutas fuera del proyecto: instala con red,
// comprueba sin red y verifica que el almacén y el entorno del aula apunten a la
// misma ruta dentro del árbol.
func TestDependencyPreparationFamiliesIntegration(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de Docker: %v", err)
	}
	runner := NewDependencyPreparationDockerRunner(cli)
	user := fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())

	for _, family := range dependencyFamilyCases() {
		family := family
		t.Run(family.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			ensureDependencyFamilyImage(ctx, t, cli, family.image)

			dir := t.TempDir()
			// El caché de módulos de Go deja directorios de solo lectura: sin
			// devolverlos a escritura, el borrado del temporal falla y el test
			// termina en rojo aunque la preparación haya funcionado.
			t.Cleanup(func() { makeDependencyTreeWritable(dir) })
			for name, content := range family.files {
				path := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("no se pudo crear %s: %v", filepath.Dir(name), err)
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("no se pudo escribir %s: %v", name, err)
				}
			}

			request := domain.DependencyPreparationRequest{
				TenantID:       "tenant-integration",
				VersionID:      "version-" + family.name,
				ComponentSlug:  family.name,
				BaseImage:      family.image,
				ProfileVersion: 1,
				ProjectPath:    "/home/workspace",
				User:           user,
				Install:        family.install,
				Verify:         family.verify,
				Network:        "build",
				OfflineFlags:   family.offlineFlags,
				CacheEnv:       family.env,
			}
			if err := request.Validate(); err != nil {
				t.Fatalf("la solicitud de %s no es válida: %v", family.name, err)
			}

			logPath := filepath.Join(t.TempDir(), "preparacion.log")
			logFile, err := os.Create(logPath)
			if err != nil {
				t.Fatalf("no se pudo crear el log: %v", err)
			}
			defer func() { _ = logFile.Close() }()

			startedAt := time.Now()
			if err := runner.RunInstall(ctx, request, dir, logFile); err != nil {
				content, _ := os.ReadFile(logPath)
				t.Fatalf("la instalación de %s falló: %v\n%s", family.name, err, tail(string(content), 3000))
			}
			for _, dependencyPath := range family.paths {
				if _, err := os.Stat(filepath.Join(dir, dependencyPath)); err != nil {
					t.Fatalf("%s no dejó %s dentro del árbol: %v", family.name, dependencyPath, err)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, family.guard)); err != nil {
				t.Fatalf("%s no dejó el contenido esperado en %s: %v", family.name, family.guard, err)
			}
			// El entorno del aula tiene que apuntar a algo que viaja en el árbol.
			for name, value := range family.env {
				if !strings.HasPrefix(value, "/home/workspace/") {
					continue
				}
				delivered := strings.TrimPrefix(value, "/home/workspace/")
				if _, err := os.Stat(filepath.Join(dir, delivered)); err != nil {
					t.Fatalf("%s: %s apunta a %s, que no se entregó con el árbol", family.name, name, value)
				}
			}

			if err := runner.RunVerify(ctx, request, dir, logFile); err != nil {
				content, _ := os.ReadFile(logPath)
				t.Fatalf("la comprobación sin red de %s falló: %v\n%s", family.name, err, tail(string(content), 3000))
			}
			t.Logf("%s preparado y comprobado sin red en %s", family.name, time.Since(startedAt).Round(time.Second))
		})
	}
}

// makeDependencyTreeWritable devuelve permisos de escritura a los directorios
// antes de borrar el temporal.
func makeDependencyTreeWritable(root string) {
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		_ = os.Chmod(path, 0o755)
		return nil
	})
}

func ensureDependencyFamilyImage(ctx context.Context, t *testing.T, cli *client.Client, ref string) {
	t.Helper()
	if _, _, err := cli.ImageInspectWithRaw(ctx, ref); err == nil {
		return
	}
	reader, err := cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		t.Fatalf("no se pudo descargar %s: %v", ref, err)
	}
	defer func() { _ = reader.Close() }()
	if _, err := io.Copy(io.Discard, reader); err != nil {
		t.Fatalf("la descarga de %s no terminó: %v", ref, err)
	}
}
