package domain_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"solv-backend/internal/core/domain"
)

func volumeFile(path, content string) domain.PracticeEnvironmentVolumeFile {
	return domain.PracticeEnvironmentVolumeFile{Path: path, Content: []byte(content)}
}

func readBundle(t *testing.T, content []byte) map[string]string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("el paquete no es un zip válido: %v", err)
	}
	files := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		handle, err := file.Open()
		if err != nil {
			t.Fatalf("no se pudo leer %s: %v", file.Name, err)
		}
		body, err := io.ReadAll(handle)
		_ = handle.Close()
		if err != nil {
			t.Fatalf("no se pudo leer %s: %v", file.Name, err)
		}
		files[file.Name] = string(body)
	}
	return files
}

// composeFile devuelve el compose generado, viva en la raíz o en solv-local.
func composeFile(files map[string]string) string {
	for name, body := range files {
		if strings.HasSuffix(name, "compose.yml") {
			return body
		}
	}
	return ""
}

func TestBuildExportBundle(t *testing.T) {
	const dockerfileWithEditor = "# Dockerfile generado deterministamente por el Constructor de Entornos SOLV\n" +
		"FROM node:20-bookworm-slim\n" +
		"RUN npm install -g --no-audit --no-fund @angular/cli\n" +
		"COPY --from=gitpod/openvscode-server:latest /home/.openvscode-server /home/.openvscode-server\n" +
		"RUN /home/.openvscode-server/bin/openvscode-server --install-extension dbaeumer.vscode-eslint\n"

	postgres := domain.ServiceRequirement{Role: domain.ServiceRolePrimaryDatabase, Engine: "postgres"}
	mysql := domain.ServiceRequirement{Role: domain.ServiceRolePrimaryDatabase, Engine: "mysql"}
	redis := domain.ServiceRequirement{Role: domain.ServiceRoleCache, Engine: "redis"}

	tests := []struct {
		name       string
		input      domain.ExportBundleInput
		assertions func(t *testing.T, files map[string]string, bundle *domain.ExportBundle)
		wantErr    error
	}{
		{
			name: "sin colisiones escribe los archivos en la raíz",
			input: domain.ExportBundleInput{
				EnvironmentName:   "Práctica 3 — Listas enlazadas",
				SubjectName:       "Estructuras de Datos",
				TemplateName:      "Laboratorio: Node.js 20 + Angular + PostgreSQL",
				DockerfileContent: dockerfileWithEditor,
				Services:          []domain.ServiceRequirement{postgres},
				AppRunCommand:     "npm start",
				AppPort:           4200,
				Files: []domain.PracticeEnvironmentVolumeFile{
					volumeFile("src/app.ts", "console.log('hola')"),
					volumeFile("package.json", `{"name":"practica"}`),
				},
			},
			assertions: func(t *testing.T, files map[string]string, bundle *domain.ExportBundle) {
				if bundle.LocalDir != "" {
					t.Fatalf("esperaba archivos en la raíz, obtuve carpeta %q", bundle.LocalDir)
				}
				for _, name := range []string{"Dockerfile", "compose.yml", ".env.example", "README.md", "src/app.ts", "package.json"} {
					if _, ok := files[name]; !ok {
						t.Fatalf("falta %s en el paquete: %v", name, keys(files))
					}
				}
				if strings.Contains(files["Dockerfile"], "openvscode-server") {
					t.Errorf("el Dockerfile local no debe traer el editor del aula:\n%s", files["Dockerfile"])
				}
				if !strings.Contains(files["Dockerfile"], "FROM node:20-bookworm-slim") {
					t.Errorf("el Dockerfile debe conservar la imagen base:\n%s", files["Dockerfile"])
				}
				if !strings.Contains(files["Dockerfile"], "WORKDIR /home/workspace") || !strings.Contains(files["Dockerfile"], "COPY . /home/workspace") {
					t.Errorf("el Dockerfile debe dejar el proyecto en la carpeta de trabajo:\n%s", files["Dockerfile"])
				}
				if !strings.Contains(files["compose.yml"], "postgres:16-alpine") {
					t.Errorf("compose debe levantar PostgreSQL local:\n%s", files["compose.yml"])
				}
				if strings.Contains(files["compose.yml"], "solv-sat-postgres") {
					t.Errorf("compose no debe apuntar al satélite del aula:\n%s", files["compose.yml"])
				}
				if !strings.Contains(files["compose.yml"], "4200") {
					t.Errorf("compose debe publicar el puerto declarado por la práctica:\n%s", files["compose.yml"])
				}
				if !strings.Contains(files["compose.yml"], "npm start") {
					t.Errorf("compose debe arrancar la app con el comando del docente:\n%s", files["compose.yml"])
				}
				if !strings.Contains(files[".env.example"], "DATABASE_URL=postgresql://solv:solv@db:5432/app?sslmode=disable") {
					t.Errorf(".env.example debe traer la URL local del servicio:\n%s", files[".env.example"])
				}
				readme := files["README.md"]
				for _, section := range []string{"## Qué incluye este entorno", "## Opción A — Correrlo con Docker", "## Opción B — Correrlo directo en tu PC, sin Docker"} {
					if !strings.Contains(readme, section) {
						t.Errorf("el README debe tener la sección %q:\n%s", section, readme)
					}
				}
				if !strings.Contains(readme, "Archivos de tu proyecto:** 2") {
					t.Errorf("el README debe decir cuántos archivos viajaron:\n%s", readme)
				}
			},
		},
		{
			name: "no sobrescribe un README ni compose propios del alumno",
			input: domain.ExportBundleInput{
				EnvironmentName: "Práctica de APIs",
				Files: []domain.PracticeEnvironmentVolumeFile{
					volumeFile("README.md", "# Mi proyecto\n\nNotas del alumno."),
					volumeFile("compose.yml", "services:\n  miproyecto:\n    image: nginx\n"),
					volumeFile("main.py", "print('hola')"),
				},
			},
			assertions: func(t *testing.T, files map[string]string, bundle *domain.ExportBundle) {
				if bundle.LocalDir != "solv-local" {
					t.Fatalf("esperaba los generados en solv-local, obtuve %q", bundle.LocalDir)
				}
				if !strings.HasPrefix(files["README.md"], "# Mi proyecto") {
					t.Errorf("el README del alumno se sobrescribió: %q", files["README.md"])
				}
				if !strings.Contains(files["compose.yml"], "nginx") {
					t.Errorf("el compose del alumno se sobrescribió: %q", files["compose.yml"])
				}
				for _, name := range []string{"solv-local/Dockerfile", "solv-local/compose.yml", "solv-local/.env.example", "solv-local/README.md"} {
					if _, ok := files[name]; !ok {
						t.Errorf("falta %s: %v", name, keys(files))
					}
				}
				if !strings.Contains(files["solv-local/compose.yml"], "context: ..") {
					t.Errorf("el compose en solv-local debe construir desde la raíz del proyecto:\n%s", files["solv-local/compose.yml"])
				}
				if !strings.Contains(files["solv-local/README.md"], "cd solv-local") {
					t.Errorf("el README generado debe explicar que los comandos corren en solv-local:\n%s", files["solv-local/README.md"])
				}
			},
		},
		{
			name: "descarta rutas que escapan del paquete",
			input: domain.ExportBundleInput{
				EnvironmentName: "Práctica",
				Files: []domain.PracticeEnvironmentVolumeFile{
					volumeFile("../fuera.txt", "no"),
					volumeFile("/etc/passwd", "no"),
					volumeFile("node_modules/lib.js", "no"),
					volumeFile(".solv-cache/go-mod/cache/download/rsc.io/x.zip", "no"),
					volumeFile(".solv-home/run/mutex", "no"),
					volumeFile(".solv_workspace/init.json", "no"),
					volumeFile("./src/index.js", "ok"),
				},
			},
			assertions: func(t *testing.T, files map[string]string, bundle *domain.ExportBundle) {
				if bundle.WorkspaceFiles != 1 {
					t.Fatalf("esperaba 1 archivo del volumen, obtuve %d", bundle.WorkspaceFiles)
				}
				for name := range files {
					if strings.HasPrefix(name, "/") || strings.Contains(name, "..") ||
						strings.Contains(name, "node_modules") || strings.Contains(name, ".solv-") || strings.Contains(name, ".solv_") {
						t.Fatalf("el paquete incluye una ruta insegura: %q", name)
					}
				}
				if _, ok := files["src/index.js"]; !ok {
					t.Fatalf("falta el archivo válido: %v", keys(files))
				}
			},
		},
		{
			name: "MySQL y Redis se levantan con sus propios nombres y credenciales locales",
			input: domain.ExportBundleInput{
				EnvironmentName: "Práctica con caché",
				Services:        []domain.ServiceRequirement{mysql, redis},
				Files:           []domain.PracticeEnvironmentVolumeFile{volumeFile("app.php", "<?php")},
			},
			assertions: func(t *testing.T, files map[string]string, bundle *domain.ExportBundle) {
				compose := files["compose.yml"]
				if !strings.Contains(compose, "mysql:8.4") || !strings.Contains(compose, "redis:7.2-alpine") {
					t.Errorf("compose debe levantar MySQL y Redis:\n%s", compose)
				}
				if !strings.Contains(compose, "$${MYSQL_ROOT_PASSWORD}") {
					t.Errorf("el healthcheck de MySQL debe escapar la variable para el contenedor:\n%s", compose)
				}
				if strings.Contains(files[".env.example"], "solv-sat") {
					t.Errorf(".env.example no debe nombrar satélites del aula:\n%s", files[".env.example"])
				}
				if !strings.Contains(files["README.md"], "REDIS_URL") {
					t.Errorf("el README debe nombrar la variable de Redis:\n%s", files["README.md"])
				}
			},
		},
		{
			name: "sin Dockerfile de la plantilla usa la imagen declarada",
			input: domain.ExportBundleInput{
				EnvironmentName: "Práctica Python",
				TemplateImage:   "python:3.12-slim-bookworm",
				Files:           []domain.PracticeEnvironmentVolumeFile{volumeFile("main.py", "print(1)")},
			},
			assertions: func(t *testing.T, files map[string]string, bundle *domain.ExportBundle) {
				if !strings.HasPrefix(files["Dockerfile"], "# Dockerfile para tu computadora") {
					t.Fatalf("Dockerfile inesperado:\n%s", files["Dockerfile"])
				}
				if !strings.Contains(files["Dockerfile"], "FROM python:3.12-slim-bookworm") {
					t.Errorf("debe usar la imagen de la plantilla:\n%s", files["Dockerfile"])
				}
				if strings.Contains(files["README.md"], "cd ") {
					t.Errorf("sin carpeta propia el README no debe pedir cd:\n%s", files["README.md"])
				}
			},
		},
		{
			name: "una imagen construida en el nodo no se usa como FROM",
			input: domain.ExportBundleInput{
				EnvironmentName: "Práctica",
				TemplateImage:   "solv-local/stack:abc123",
				Files:           []domain.PracticeEnvironmentVolumeFile{volumeFile("main.go", "package main")},
			},
			assertions: func(t *testing.T, files map[string]string, bundle *domain.ExportBundle) {
				if strings.Contains(files["Dockerfile"], "solv-local/stack") {
					t.Errorf("la imagen del nodo no existe en la PC del alumno:\n%s", files["Dockerfile"])
				}
				if !strings.Contains(files["Dockerfile"], "FROM ubuntu:24.04") {
					t.Errorf("se esperaba una base explícita:\n%s", files["Dockerfile"])
				}
			},
		},
		{
			name: "Go sin base de datos: README del stack, sin puerto inventado y consola interactiva",
			input: domain.ExportBundleInput{
				EnvironmentName:   "Agente Go",
				DockerfileContent: "FROM golang:1.26-bookworm\nWORKDIR /home/workspace\n",
				Files: []domain.PracticeEnvironmentVolumeFile{
					volumeFile("main.go", "package main\n"),
					volumeFile("go.mod", "module agente\n"),
				},
			},
			assertions: func(t *testing.T, files map[string]string, bundle *domain.ExportBundle) {
				readme := files["README.md"]
				compose := composeFile(files)
				for _, want := range []string{
					"go run .",
					"`go.mod`",
					"docker compose build && docker compose run --rm app",
				} {
					if !strings.Contains(readme, want) {
						t.Errorf("el README debe mencionar %q:\n%s", want, readme)
					}
				}
				for _, unwanted := range []string{"localhost", "docker compose down -v", "Instalá", "Arrancá", "Escribí", "Necesitás", "debés"} {
					if strings.Contains(readme, unwanted) {
						t.Errorf("el README no debe contener %q:\n%s", unwanted, readme)
					}
				}
				if strings.Contains(compose, "ports:") {
					t.Errorf("sin app_port el compose no debe publicar puertos:\n%s", compose)
				}
				for _, want := range []string{"stdin_open: true", "tty: true", "docker compose run --rm app sh"} {
					if !strings.Contains(compose, want) {
						t.Errorf("el compose debe permitir la consola interactiva (%q):\n%s", want, compose)
					}
				}
				if strings.Contains(files[".env.example"], "APP_PORT=") {
					t.Errorf("sin app_port no debe declararse APP_PORT:\n%s", files[".env.example"])
				}
			},
		},
		{
			name: "Go con PostgreSQL y app_port: publica el puerto y conserva los datos",
			input: domain.ExportBundleInput{
				EnvironmentName:   "Agente Go con base",
				DockerfileContent: "FROM golang:1.26-bookworm\n",
				Services:          []domain.ServiceRequirement{postgres},
				AppPort:           8080,
				Files:             []domain.PracticeEnvironmentVolumeFile{volumeFile("main.go", "package main\n")},
			},
			assertions: func(t *testing.T, files map[string]string, bundle *domain.ExportBundle) {
				readme := files["README.md"]
				compose := composeFile(files)
				for _, want := range []string{
					"http://localhost:8080",
					"docker compose down -v",
					"go run .",
					"docker compose build && docker compose run --rm --service-ports app",
				} {
					if !strings.Contains(readme, want) {
						t.Errorf("el README debe mencionar %q:\n%s", want, readme)
					}
				}
				if !strings.Contains(compose, "\n    ports:\n      - \"${APP_PORT:-8080}:${APP_PORT:-8080}\"\n") {
					t.Errorf("el puerto del servicio app debe ir indentado bajo ports:\n%s", compose)
				}
				if !strings.Contains(compose, "postgres:16-alpine") {
					t.Errorf("compose debe levantar PostgreSQL:\n%s", compose)
				}
			},
		},
		{
			name: "volumen vacío no produce paquete",
			input: domain.ExportBundleInput{
				EnvironmentName: "Práctica sin archivos",
			},
			wantErr: domain.ErrExportBundleEmpty,
		},
		{
			name: "un archivo por encima del tope corta la exportación",
			input: domain.ExportBundleInput{
				EnvironmentName: "Práctica pesada",
				Files: []domain.PracticeEnvironmentVolumeFile{
					{Path: "big.bin", Content: bytes.Repeat([]byte("x"), domain.ExportBundleMaxFileBytes+1)},
				},
			},
			wantErr: domain.ErrExportBundleTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle, err := domain.BuildExportBundle(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("esperaba %v, obtuve %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			files := readBundle(t, bundle.Content)
			var composeDoc map[string]any
			if err := yaml.Unmarshal([]byte(composeFile(files)), &composeDoc); err != nil {
				t.Fatalf("compose.yml no es YAML válido: %v\n%s", err, composeFile(files))
			}
			if _, ok := composeDoc["services"]; !ok {
				t.Fatalf("compose.yml no declara services: \n%s", composeFile(files))
			}
			tt.assertions(t, files, bundle)
		})
	}
}

func TestSanitizeExportFilename(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "Práctica 3 — Listas enlazadas", want: "practica-3-listas-enlazadas"},
		{in: "  Árboles / Grafos  ", want: "arboles-grafos"},
		{in: "", want: ""},
		{in: "!!!", want: ""},
	}
	for _, tt := range tests {
		if got := domain.SanitizeExportFilename(tt.in); got != tt.want {
			t.Errorf("SanitizeExportFilename(%q) = %q, esperaba %q", tt.in, got, tt.want)
		}
	}
}

func keys(files map[string]string) []string {
	out := make([]string, 0, len(files))
	for name := range files {
		out = append(out, name)
	}
	return out
}
