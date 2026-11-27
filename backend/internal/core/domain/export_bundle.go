package domain

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

// El paquete de exportación convierte el volumen de una práctica en un proyecto
// reproducible fuera del aula: el código del alumno más un Dockerfile, un
// compose.yml con sus servicios satélite, un .env.example sin credenciales
// reales y un README que explica qué configuraba SOLV por él.
//
// Todo se decide con el árbol de archivos y los metadatos de la plantilla: este
// archivo no habla con Docker, con HTTP ni con la base de datos.
const (
	// ExportBundleMaxFiles acota cuántos archivos entran en el paquete.
	ExportBundleMaxFiles = 600
	// ExportBundleMaxFileBytes acota un archivo suelto.
	ExportBundleMaxFileBytes = 2 * 1024 * 1024
	// ExportBundleMaxWorkspaceBytes acota el proyecto leído del volumen.
	ExportBundleMaxWorkspaceBytes = 24 * 1024 * 1024
	// ExportBundleMaxZipBytes protege al host del tamaño del .zip en memoria.
	ExportBundleMaxZipBytes = 32 * 1024 * 1024
	// ExportBundleDefaultAppPort es el puerto donde SOLV publica la aplicación
	// cuando la práctica no declara uno.
	ExportBundleDefaultAppPort = 3000
	// exportBundleLocalDir es la carpeta donde viven los archivos generados
	// cuando el alumno ya tiene los suyos propios en la raíz del proyecto.
	exportBundleLocalDir = "solv-local"
)

var (
	// ErrExportBundleEmpty indica que el volumen no tiene archivos que exportar.
	ErrExportBundleEmpty = errors.New("el entorno no tiene archivos para exportar")
	// ErrExportBundleTooLarge indica que el proyecto no entra en los topes del
	// paquete. Se devuelve en vez de recortar: un .zip incompleto no es
	// reproducible y el alumno no lo sabría.
	ErrExportBundleTooLarge = errors.New("el proyecto excede el tamaño máximo para exportar")
)

// ExportBundleFile es un archivo listo para viajar en el .zip.
type ExportBundleFile struct {
	Path    string
	Content []byte
}

// ExportBundleInput reúne el árbol del alumno y lo que la plantilla declara
// sobre su entorno. Es deliberadamente plano: el generador no resuelve
// repositorios.
type ExportBundleInput struct {
	// EnvironmentName y SubjectName solo se usan para el README.
	EnvironmentName string
	SubjectName     string
	// TemplateName describe el stack ("Laboratorio: Node.js 20 + Angular").
	TemplateName string
	// DockerfileContent es el Dockerfile con el que se construyó la imagen del
	// aula; puede traer capas del editor web que no sirven en local.
	DockerfileContent string
	// TemplateImage es la imagen de la plantilla, usada solo si no hay
	// DockerfileContent.
	TemplateImage string
	// Services son los motores satélite declarados por la plantilla.
	Services []ServiceRequirement
	// DependencyProfile declara cómo instala dependencias la familia del stack.
	DependencyProfile *DependencyProfile
	// AppRunCommand y AppPort los declara el docente para la aplicación del
	// proyecto. Sin ellos el compose levanta una consola dentro del contenedor.
	AppRunCommand string
	AppPort       int
	// Files es el árbol leído del volumen del alumno.
	Files []PracticeEnvironmentVolumeFile
}

// ExportBundle es el resultado: el .zip en memoria y lo que se generó, para que
// el transporte arme el nombre del archivo y el alumno sepa qué recibió.
type ExportBundle struct {
	Content        []byte
	WorkspaceFiles int
	// GeneratedPaths son las rutas (dentro del .zip) de los archivos que no
	// venían del volumen.
	GeneratedPaths []string
	// LocalDir es la carpeta donde quedaron los archivos generados. Vacía
	// significa que conviven con el proyecto en la raíz.
	LocalDir string
}

// BuildExportBundle arma el paquete. No sobrescribe trabajo del alumno: si ya
// tiene Dockerfile, compose, .env.example o README, los archivos de SOLV van a
// una carpeta propia y el README lo explica.
func BuildExportBundle(in ExportBundleInput) (*ExportBundle, error) {
	workspace, err := normalizeExportWorkspaceFiles(in.Files)
	if err != nil {
		return nil, err
	}
	if len(workspace) == 0 {
		return nil, ErrExportBundleEmpty
	}

	localDir := exportLocalDir(workspace)
	dockerfile, removedLayers := exportLocalDockerfile(in)

	files := []ExportBundleFile{
		{Path: "Dockerfile", Content: []byte(dockerfile)},
		{Path: "compose.yml", Content: []byte(exportComposeYAML(in, localDir))},
		{Path: ".env.example", Content: []byte(exportEnvExample(in))},
		{Path: "README.md", Content: []byte(exportReadme(in, localDir, len(workspace), removedLayers))},
	}

	bundle := &ExportBundle{WorkspaceFiles: len(workspace), LocalDir: localDir}
	entries := make([]ExportBundleFile, 0, len(workspace)+len(files))
	for _, file := range workspace {
		entries = append(entries, file)
	}
	for _, file := range files {
		file.Path = path.Join(localDir, file.Path)
		entries = append(entries, file)
		bundle.GeneratedPaths = append(bundle.GeneratedPaths, file.Path)
	}
	if len(entries) > ExportBundleMaxFiles {
		return nil, fmt.Errorf("%w: %d archivos superan el máximo de %d", ErrExportBundleTooLarge, len(entries), ExportBundleMaxFiles)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })

	content, err := zipExportBundle(entries)
	if err != nil {
		return nil, err
	}
	bundle.Content = content
	return bundle, nil
}

// normalizeExportWorkspaceFiles descarta rutas que no pueden viajar en un .zip:
// absolutas, con `..`, con caracteres de control, dentro de carpetas que se
// regeneran o por encima de los topes. Un archivo descartado no rompe la
// exportación; el README dice cuántos archivos viajaron.
func normalizeExportWorkspaceFiles(files []PracticeEnvironmentVolumeFile) ([]ExportBundleFile, error) {
	seen := make(map[string]bool, len(files))
	out := make([]ExportBundleFile, 0, len(files))
	var total int64

	for _, file := range files {
		cleaned, ok := cleanExportPath(file.Path)
		if !ok || seen[cleaned] {
			continue
		}
		if len(file.Content) > ExportBundleMaxFileBytes {
			return nil, fmt.Errorf("%w: %s supera %d MB", ErrExportBundleTooLarge, cleaned, ExportBundleMaxFileBytes/(1024*1024))
		}
		total += int64(len(file.Content))
		if total > ExportBundleMaxWorkspaceBytes {
			return nil, fmt.Errorf("%w: el proyecto supera %d MB", ErrExportBundleTooLarge, ExportBundleMaxWorkspaceBytes/(1024*1024))
		}
		seen[cleaned] = true
		content := make([]byte, len(file.Content))
		copy(content, file.Content)
		out = append(out, ExportBundleFile{Path: cleaned, Content: content})
	}
	return out, nil
}

// cleanExportPath normaliza una ruta del volumen y rechaza lo que podría
// escribir fuera del .zip al descomprimirlo.
func cleanExportPath(raw string) (string, bool) {
	candidate := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	candidate = strings.TrimPrefix(candidate, "./")
	if candidate == "" || strings.HasPrefix(candidate, "/") || strings.Contains(candidate, "\x00") {
		return "", false
	}
	clean := path.Clean(candidate)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	for _, segment := range strings.Split(clean, "/") {
		if segment == "" || segment == ".." || segment == "." {
			return "", false
		}
		switch segment {
		case ".git", "node_modules", "__pycache__", ".venv", ".solv_backups",
			// Cachés de la preparación de dependencias y metadatos del aula:
			// no son trabajo del alumno y no deben viajar en el paquete.
			".solv-cache", ".solv-home", ".solv_workspace":
			return "", false
		}
	}
	return clean, true
}

// exportLocalDir elige dónde escribir los archivos generados. Sin colisión van
// a la raíz, junto al proyecto; con colisión se usa una carpeta libre para no
// pisar el Dockerfile, el compose o el README del alumno.
func exportLocalDir(workspace []ExportBundleFile) string {
	reserved := []string{"Dockerfile", "compose.yml", "compose.yaml", "docker-compose.yml", ".env.example", "README.md"}
	if !exportPathsCollide(workspace, "", reserved) {
		return ""
	}
	for i := 1; i <= 9; i++ {
		folder := exportBundleLocalDir
		if i > 1 {
			folder = fmt.Sprintf("%s-%d", exportBundleLocalDir, i)
		}
		if !exportPathsCollide(workspace, folder, reserved) {
			return folder
		}
	}
	return exportBundleLocalDir + "-export"
}

func exportPathsCollide(workspace []ExportBundleFile, folder string, names []string) bool {
	for _, file := range workspace {
		if folder != "" && (file.Path == folder || strings.HasPrefix(file.Path, folder+"/")) {
			return true
		}
		for _, name := range names {
			if file.Path == path.Join(folder, name) {
				return true
			}
		}
	}
	return false
}

// zipExportBundle escribe el .zip de forma determinista: mismo árbol, mismos
// bytes. La fecha se fija en cero para que dos exportaciones iguales coincidan.
func zipExportBundle(files []ExportBundleFile) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, file := range files {
		header := &zip.FileHeader{Name: file.Path, Method: zip.Deflate}
		header.SetMode(0o644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return nil, fmt.Errorf("no se pudo escribir %s en el paquete: %w", file.Path, err)
		}
		if _, err := entry.Write(file.Content); err != nil {
			return nil, fmt.Errorf("no se pudo escribir %s en el paquete: %w", file.Path, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("no se pudo cerrar el paquete: %w", err)
	}
	if buffer.Len() > ExportBundleMaxZipBytes {
		return nil, fmt.Errorf("%w: el paquete supera %d MB", ErrExportBundleTooLarge, ExportBundleMaxZipBytes/(1024*1024))
	}
	return buffer.Bytes(), nil
}

// SanitizeExportFilename deja un nombre de archivo seguro y legible a partir
// del título de la práctica.
func SanitizeExportFilename(text string) string {
	normalized := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u", "Ü", "u", "Ñ", "n",
	).Replace(strings.TrimSpace(text))

	var out strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(normalized) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && out.Len() > 0 {
				out.WriteByte('-')
				lastDash = true
			}
		}
		if out.Len() >= 60 {
			break
		}
	}
	return strings.Trim(out.String(), "-")
}

// exportLocalDockerfile deriva el Dockerfile del aula a uno que sirve en una
// computadora personal. Quita las capas del editor web (OpenVSCode Server y sus
// extensiones), que no aportan nada al proyecto y agrandan la imagen, y agrega
// la instalación de dependencias del stack para que la app arranque sola.
func exportLocalDockerfile(in ExportBundleInput) (string, []string) {
	var kept []string
	var removed []string
	for _, line := range strings.Split(in.DockerfileContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			// Los comentarios del constructor no describen una capa: el README
			// tiene su propio encabezado.
			continue
		}
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "openvscode-server") ||
			strings.Contains(lower, "code-server") ||
			strings.Contains(lower, "--install-extension") {
			removed = append(removed, trimmed)
			continue
		}
		kept = append(kept, trimmed)
	}

	hasFrom := false
	for _, line := range kept {
		if strings.HasPrefix(strings.ToUpper(line), "FROM ") {
			hasFrom = true
			break
		}
	}
	if !hasFrom {
		base := strings.TrimSpace(in.TemplateImage)
		if base == "" || IsBuiltStackImageTag(base) {
			// La imagen del aula se construyó en el nodo y no existe en la
			// computadora del alumno. Se deja una base honesta y el README
			// explica cómo cambiarla por la de su lenguaje.
			base = "ubuntu:24.04"
		}
		kept = append([]string{"FROM " + base}, kept...)
	}

	lines := []string{"# Dockerfile para tu computadora, generado por SOLV desde el entorno del aula."}
	lines = append(lines, kept...)
	if profile := in.DependencyProfile; profile != nil && !profile.IsEmpty() && strings.TrimSpace(profile.Install) != "" {
		lines = append(lines, "", "# Instala las dependencias que el aula preparaba por ti.", "RUN "+strings.TrimSpace(profile.Install))
	}
	lines = append(lines, "", "WORKDIR /home/workspace", "COPY . /home/workspace")
	return strings.Join(lines, "\n") + "\n", removed
}

// exportServiceSpec describe cómo se levanta un motor satélite en la
// computadora del alumno. Las credenciales son locales y de ejemplo: nunca se
// reutilizan las del aula.
type exportServiceSpec struct {
	Alias       string
	Engine      string
	Image       string
	Port        int
	DataDir     string
	Env         []string
	Healthcheck string
	VolumeName  string
	EnvExample  []string
}

func exportServiceSpecFor(binding ServiceRequirement) (exportServiceSpec, bool) {
	engine := CanonicalServiceEngine(binding.Engine)
	alias := sanitizeComposeName(binding.EffectiveEnvAlias())
	if alias == "" {
		alias = engine
	}

	switch engine {
	case "postgres":
		return exportServiceSpec{
			Alias:  alias,
			Engine: engine,
			Image:  "postgres:16-alpine",
			Port:   5432,
			Env: []string{
				"POSTGRES_USER: ${POSTGRES_USER:-solv}",
				"POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-solv}",
				"POSTGRES_DB: ${POSTGRES_DB:-app}",
			},
			Healthcheck: `pg_isready -U ${POSTGRES_USER:-solv} -d ${POSTGRES_DB:-app}`,
			DataDir:     "/var/lib/postgresql/data",
			VolumeName:  "solv-db-data",
			EnvExample: []string{
				"DATABASE_URL=postgresql://solv:solv@" + alias + ":5432/app?sslmode=disable",
				"DB_PORT=5432",
				"POSTGRES_USER=solv",
				"POSTGRES_PASSWORD=solv",
				"POSTGRES_DB=app",
			},
		}, true
	case "mysql":
		return exportServiceSpec{
			Alias:  alias,
			Engine: engine,
			Image:  "mysql:8.4",
			Port:   3306,
			Env: []string{
				"MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-solv}",
				"MYSQL_DATABASE: ${MYSQL_DATABASE:-app}",
			},
			Healthcheck: `mysqladmin ping -h 127.0.0.1 -uroot -p$${MYSQL_ROOT_PASSWORD}`,
			DataDir:     "/var/lib/mysql",
			VolumeName:  "solv-db-data",
			EnvExample: []string{
				"DATABASE_URL=mysql://solv:solv@" + alias + ":3306/app",
				"DB_PORT=3306",
				"MYSQL_ROOT_PASSWORD=solv",
				"MYSQL_DATABASE=app",
			},
		}, true
	case "mariadb":
		return exportServiceSpec{
			Alias:  alias,
			Engine: engine,
			Image:  "mariadb:11.4",
			Port:   3306,
			Env: []string{
				"MARIADB_ROOT_PASSWORD: ${MARIADB_ROOT_PASSWORD:-solv}",
				"MARIADB_DATABASE: ${MARIADB_DATABASE:-app}",
			},
			Healthcheck: `healthcheck.sh --connect --innodb_initialized`,
			DataDir:     "/var/lib/mysql",
			VolumeName:  "solv-db-data",
			EnvExample: []string{
				"DATABASE_URL=mysql://solv:solv@" + alias + ":3306/app",
				"DB_PORT=3306",
				"MARIADB_ROOT_PASSWORD=solv",
				"MARIADB_DATABASE=app",
			},
		}, true
	case "mongodb":
		return exportServiceSpec{
			Alias:  alias,
			Engine: engine,
			Image:  "mongo:7.0",
			Port:   27017,
			Env: []string{
				"MONGO_INITDB_ROOT_USERNAME: ${MONGO_INITDB_ROOT_USERNAME:-solv}",
				"MONGO_INITDB_ROOT_PASSWORD: ${MONGO_INITDB_ROOT_PASSWORD:-solv}",
			},
			Healthcheck: `mongosh --quiet --eval "db.adminCommand('ping').ok"`,
			DataDir:     "/data/db",
			VolumeName:  "solv-db-data",
			EnvExample: []string{
				"MONGODB_URI=mongodb://solv:solv@" + alias + ":27017/app?authSource=admin",
				"DB_PORT=27017",
				"MONGO_INITDB_ROOT_USERNAME=solv",
				"MONGO_INITDB_ROOT_PASSWORD=solv",
			},
		}, true
	case "redis":
		return exportServiceSpec{
			Alias:       alias,
			Engine:      engine,
			Image:       "redis:7.2-alpine",
			Port:        6379,
			Env:         nil,
			Healthcheck: `redis-cli ping`,
			DataDir:     "/data",
			VolumeName:  "solv-cache-data",
			EnvExample: []string{
				"REDIS_URL=redis://" + alias + ":6379/0",
				"CACHE_PORT=6379",
			},
		}, true
	default:
		// sqlite vive en un archivo del proyecto y 'none' no declara nada.
		return exportServiceSpec{}, false
	}
}

// managedExportServices ordena los servicios (base principal primero) y
// descarta motores embebidos o desconocidos.
func managedExportServices(services []ServiceRequirement) []exportServiceSpec {
	var out []exportServiceSpec
	seen := make(map[string]bool, len(services))
	for _, binding := range services {
		spec, ok := exportServiceSpecFor(binding)
		if !ok || seen[spec.Alias] {
			continue
		}
		seen[spec.Alias] = true
		out = append(out, spec)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return exportServiceRank(out[i]) < exportServiceRank(out[j])
	})
	return out
}

func exportServiceRank(spec exportServiceSpec) int {
	if spec.Engine == "redis" {
		return 1
	}
	return 0
}

func sanitizeComposeName(raw string) string {
	var out strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			out.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && out.Len() > 0 {
				out.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(out.String(), "-")
}

// exportDeclaredAppPort devuelve el puerto web que declaró el docente. Cero
// significa que la práctica no publica una aplicación web: ni el compose ni el
// README pueden inventar un 3000 que nadie escucha.
func exportDeclaredAppPort(in ExportBundleInput) int {
	if in.AppPort > 0 && in.AppPort <= 65535 {
		return in.AppPort
	}
	return 0
}

// exportRuntime describe el lenguaje del proyecto para que el README use
// ejemplos de SU stack y no una lista de otros lenguajes. Se deduce de la
// imagen base del Dockerfile y del nombre de la plantilla.
type exportRuntime struct {
	Label      string
	Manifest   string
	RunCommand string
}

func detectExportRuntime(in ExportBundleInput) exportRuntime {
	haystack := strings.ToLower(strings.Join([]string{
		in.DockerfileContent,
		in.TemplateImage,
		in.TemplateName,
	}, "\n"))
	switch {
	case strings.Contains(haystack, "golang"), strings.Contains(haystack, "go1."), strings.Contains(haystack, "/go:"):
		return exportRuntime{Label: "Go", Manifest: "go.mod", RunCommand: "go run ."}
	case strings.Contains(haystack, "python"), strings.Contains(haystack, "flask"), strings.Contains(haystack, "django"):
		return exportRuntime{Label: "Python", Manifest: "requirements.txt", RunCommand: "python app.py"}
	case strings.Contains(haystack, "node:"), strings.Contains(haystack, "nodejs"), strings.Contains(haystack, "npm "), strings.Contains(haystack, "angular"):
		return exportRuntime{Label: "Node.js", Manifest: "package.json", RunCommand: "npm start"}
	case strings.Contains(haystack, "openjdk"), strings.Contains(haystack, "jdk"), strings.Contains(haystack, "maven"), strings.Contains(haystack, "mvn "):
		return exportRuntime{Label: "Java", Manifest: "pom.xml", RunCommand: "mvn -q -DskipTests package"}
	case strings.Contains(haystack, "dotnet"), strings.Contains(haystack, ".net "):
		return exportRuntime{Label: ".NET", Manifest: "*.csproj", RunCommand: "dotnet run"}
	case strings.Contains(haystack, "gcc"), strings.Contains(haystack, "g++"), strings.Contains(haystack, "clang"), strings.Contains(haystack, "cmake"):
		return exportRuntime{Label: "C/C++", Manifest: "Makefile o CMakeLists.txt", RunCommand: "gcc main.c -o app && ./app"}
	default:
		return exportRuntime{}
	}
}

// exportComposeYAML arma un compose autónomo: la aplicación con su Dockerfile y
// los motores satélite con volúmenes propios. Ninguna URL apunta al aula.
func exportComposeYAML(in ExportBundleInput, localDir string) string {
	appPort := exportDeclaredAppPort(in)
	runtime := detectExportRuntime(in)
	consoleMode := strings.TrimSpace(in.AppRunCommand) == ""
	context := "."
	dockerfile := "Dockerfile"
	if localDir != "" {
		context = ".."
		dockerfile = localDir + "/Dockerfile"
	}
	services := managedExportServices(in.Services)

	runHint := "docker compose up --build"
	if consoleMode {
		runHint = "docker compose build && docker compose run --rm app sh"
		if appPort > 0 {
			runHint = "docker compose build && docker compose run --rm --service-ports app sh"
		}
	}

	var b strings.Builder
	b.WriteString("# Levanta tu entorno de práctica en tu computadora.\n")
	fmt.Fprintf(&b, "# Uso: cp .env.example .env && %s\n", runHint)
	b.WriteString("services:\n")
	b.WriteString("  app:\n")
	b.WriteString("    build:\n")
	fmt.Fprintf(&b, "      context: %s\n", context)
	fmt.Fprintf(&b, "      dockerfile: %s\n", dockerfile)
	b.WriteString("    env_file: .env\n")
	if appPort > 0 {
		b.WriteString("    # Puerto de la aplicación que declaró tu docente.\n")
		b.WriteString("    ports:\n")
		fmt.Fprintf(&b, "      - \"${APP_PORT:-%d}:${APP_PORT:-%d}\"\n", appPort, appPort)
	} else {
		b.WriteString("    # Esta práctica no declara un puerto web: el contenedor no publica puertos.\n")
	}
	b.WriteString("    stdin_open: true\n")
	b.WriteString("    tty: true\n")
	if !consoleMode {
		fmt.Fprintf(&b, "    command: [\"sh\", \"-lc\", %q]\n", strings.TrimSpace(in.AppRunCommand))
	} else {
		b.WriteString("    # Tu docente no fijó un comando de arranque: el contenedor abre una consola interactiva.\n")
		if runtime.RunCommand != "" {
			fmt.Fprintf(&b, "    # Escribe dentro el comando de tu proyecto (por ejemplo `%s`).\n", runtime.RunCommand)
		} else {
			b.WriteString("    # Escribe dentro el comando de arranque de tu proyecto.\n")
		}
		b.WriteString("    command: [\"sh\"]\n")
	}
	if len(services) > 0 {
		b.WriteString("    depends_on:\n")
		for _, spec := range services {
			fmt.Fprintf(&b, "      %s:\n        condition: service_healthy\n", spec.Alias)
		}
	}

	for _, spec := range services {
		fmt.Fprintf(&b, "\n  %s:\n", spec.Alias)
		fmt.Fprintf(&b, "    image: %s\n", spec.Image)
		if len(spec.Env) > 0 {
			b.WriteString("    environment:\n")
			for _, env := range spec.Env {
				fmt.Fprintf(&b, "      %s\n", env)
			}
		}
		fmt.Fprintf(&b, "    ports:\n      - \"${DB_PORT:-%d}:%d\"\n", spec.Port, spec.Port)
		fmt.Fprintf(&b, "    volumes:\n      - %s:%s\n", spec.VolumeName, spec.DataDir)
		b.WriteString("    healthcheck:\n")
		fmt.Fprintf(&b, "      test: [\"CMD-SHELL\", %q]\n", spec.Healthcheck)
		b.WriteString("      interval: 5s\n      timeout: 5s\n      retries: 20\n")
	}

	if len(services) > 0 {
		b.WriteString("\nvolumes:\n")
		for _, spec := range services {
			fmt.Fprintf(&b, "  %s:\n", spec.VolumeName)
		}
	}
	return b.String()
}

// exportEnvExample arma el .env.example. Solo lleva credenciales locales de
// ejemplo: el aula nunca presta las suyas.
func exportEnvExample(in ExportBundleInput) string {
	appPort := exportDeclaredAppPort(in)
	services := managedExportServices(in.Services)

	var b strings.Builder
	b.WriteString("# Archivo de ejemplo: cópialo a .env y ajusta lo que necesites.\n")
	b.WriteString("# Estas credenciales son de tu computadora, no del servidor de la institución.\n")
	b.WriteString("# .env no se comparte ni se sube a un repositorio.\n\n")
	if appPort > 0 {
		fmt.Fprintf(&b, "# Puerto donde tu aplicación escucha (el aula la publicaba en %d).\nAPP_PORT=%d\n", appPort, appPort)
	} else {
		b.WriteString("# Esta práctica no declara un puerto web: no hay APP_PORT que configurar.\n")
	}
	b.WriteString("\n# Servicios de tu entorno (levantados por compose.yml).\n")
	for _, spec := range services {
		for _, line := range spec.EnvExample {
			b.WriteString(line + "\n")
		}
	}
	if len(services) == 0 {
		b.WriteString("# Esta práctica no usaba servicios externos.\n")
	}
	return b.String()
}

// exportReadme arma la guía pedagógica en tres bloques: qué incluye, cómo
// correrlo con Docker y cómo correrlo directo en la computadora.
func exportReadme(in ExportBundleInput, localDir string, fileCount int, removedLayers []string) string {
	appPort := exportDeclaredAppPort(in)
	services := managedExportServices(in.Services)
	runtime := detectExportRuntime(in)
	consoleMode := strings.TrimSpace(in.AppRunCommand) == ""
	name := strings.TrimSpace(in.EnvironmentName)
	if name == "" {
		name = "Proyecto de práctica"
	}

	prefix := ""
	if localDir != "" {
		prefix = localDir + "/"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", name)
	b.WriteString("Este paquete lo generó SOLV desde tu espacio de práctica: incluye tu código y lo necesario para correrlo en tu propia computadora, sin depender del servidor de la institución.\n\n")

	b.WriteString("## Qué incluye este entorno\n\n")
	fmt.Fprintf(&b, "- **Práctica:** %s\n", name)
	if subject := strings.TrimSpace(in.SubjectName); subject != "" {
		fmt.Fprintf(&b, "- **Materia:** %s\n", subject)
	}
	if stack := strings.TrimSpace(in.TemplateName); stack != "" {
		fmt.Fprintf(&b, "- **Entorno del aula:** %s\n", stack)
	}
	fmt.Fprintf(&b, "- **Archivos de tu proyecto:** %d (sin `.git`, `node_modules`, `__pycache__`, `.venv` ni las cachés `.solv-cache` y `.solv-home` que SOLV creaba por ti)\n", fileCount)
	if base := exportBaseImage(in); base != "" {
		fmt.Fprintf(&b, "- **Imagen base:** `%s`\n", base)
	}
	if len(services) > 0 {
		b.WriteString("- **Servicios que SOLV levantaba por ti:**\n")
		for _, spec := range services {
			fmt.Fprintf(&b, "  - %s local, en `%s:%d` (con volumen propio)\n", exportEngineLabel(spec.Engine), spec.Alias, spec.Port)
		}
	} else {
		b.WriteString("- **Servicios:** esta práctica no usaba base de datos ni caché externas.\n")
	}
	if len(services) > 0 {
		b.WriteString("- **Variables que SOLV configuraba por ti:**\n")
		for _, spec := range services {
			fmt.Fprintf(&b, "  - `%s` (apuntaba a la base aislada de tu usuario en el aula; el `.env.example` la apunta a tus servicios locales)\n", exportURIVariable(spec.Engine))
		}
	}
	if len(removedLayers) > 0 {
		b.WriteString("\n### Qué se quitó del entorno del aula\n\n")
		b.WriteString("El Dockerfile del aula traía capas del editor web de SOLV. No hacen falta para ejecutar tu proyecto, así que el Dockerfile de este paquete las omite:\n\n")
		for _, layer := range removedLayers {
			fmt.Fprintf(&b, "- `%s`\n", compactForReadme(layer))
		}
	}
	if localDir != "" {
		fmt.Fprintf(&b, "\n### Tus archivos quedaron intactos\n\nYa tenías un `Dockerfile`, `compose.yml`, `.env.example` o `README.md` propio. No se sobrescribió nada: los archivos de SOLV están en `%s/` y debes ejecutar los comandos de abajo dentro de esa carpeta.\n", localDir)
	}

	b.WriteString("\n## Opción A — Correrlo con Docker\n\n")
	b.WriteString("Necesitas Docker Desktop (Windows/macOS) o Docker Engine con el plugin Compose (Linux).\n\n")
	composeCommand := "docker compose up --build"
	if consoleMode {
		composeCommand = "docker compose build && docker compose run --rm app sh"
		if appPort > 0 {
			composeCommand = "docker compose build && docker compose run --rm --service-ports app sh"
		}
	}
	b.WriteString("```sh\n")
	if localDir != "" {
		fmt.Fprintf(&b, "cd %s\n", strings.TrimSuffix(prefix, "/"))
	}
	fmt.Fprintf(&b, "cp .env.example .env\n%s\n```\n\n", composeCommand)
	if !consoleMode {
		fmt.Fprintf(&b, "La aplicación arranca sola con el comando que declaró tu docente: `%s`.", strings.TrimSpace(in.AppRunCommand))
		if appPort > 0 {
			fmt.Fprintf(&b, " Ábrela en http://localhost:%d.", appPort)
		}
		b.WriteString("\n")
	} else {
		b.WriteString("Ese comando abre una consola interactiva dentro del contenedor, con tu proyecto en `/home/workspace`. Escribe ahí el comando de arranque de tu proyecto")
		if runtime.RunCommand != "" {
			fmt.Fprintf(&b, " (por ejemplo `%s`)", runtime.RunCommand)
		}
		b.WriteString(".")
		if appPort > 0 {
			fmt.Fprintf(&b, " Cuando escuche en el puerto %d, ábrela en http://localhost:%d.", appPort, appPort)
		}
		b.WriteString(" Si prefieres que arranque solo, agrega `command: [\"sh\", \"-lc\", \"tu comando\"]` al servicio `app` de `compose.yml`.\n")
	}
	b.WriteString("\nPara apagarlo: `docker compose down`.")
	if len(services) > 0 {
		b.WriteString(" Tus datos de base quedan en el volumen de Docker; `docker compose down -v` los borra.")
	}
	b.WriteString("\n")

	b.WriteString("\n## Opción B — Correrlo directo en tu PC, sin Docker\n\n")
	b.WriteString("1. **Instala el runtime del entorno.** ")
	if base := exportBaseImage(in); base != "" {
		if runtime.Label != "" {
			fmt.Fprintf(&b, "El aula usaba la imagen `%s`; instala en tu computadora la versión equivalente de %s.\n", base, runtime.Label)
		} else {
			fmt.Fprintf(&b, "El aula usaba la imagen `%s`; instala en tu computadora la versión equivalente de ese lenguaje.\n", base)
		}
	} else if runtime.Label != "" {
		fmt.Fprintf(&b, "Instala en tu computadora la versión de %s que usa tu proyecto.\n", runtime.Label)
	} else {
		b.WriteString("Instala el runtime que usa tu proyecto (Node.js, Python, JDK, Go, ...).\n")
	}
	if profile := in.DependencyProfile; profile != nil && !profile.IsEmpty() && strings.TrimSpace(profile.Install) != "" {
		b.WriteString("2. **Instala las dependencias del proyecto** desde la carpeta del proyecto:\n\n")
		fmt.Fprintf(&b, "   ```sh\n   %s\n   ```\n\n", strings.TrimSpace(profile.Install))
		if verify := strings.TrimSpace(profile.Verify); verify != "" {
			fmt.Fprintf(&b, "   Comprobación: `%s`\n\n", verify)
		}
	} else if runtime.Manifest != "" {
		fmt.Fprintf(&b, "2. **Instala las dependencias del proyecto** con el gestor que corresponda; revisa `%s`.\n\n", runtime.Manifest)
	} else {
		b.WriteString("2. **Instala las dependencias del proyecto** con el gestor que corresponda (revisa si hay `package.json`, `requirements.txt`, `pom.xml` o `go.mod`).\n\n")
	}
	if len(services) > 0 {
		b.WriteString("3. **Prepara los servicios en tu computadora.** Sin Docker necesitas el motor instalado y una base propia:\n\n")
		for _, spec := range services {
			switch spec.Engine {
			case "postgres":
				b.WriteString("   ```sh\n   psql -U postgres -c \"CREATE USER solv WITH PASSWORD 'solv';\"\n   psql -U postgres -c \"CREATE DATABASE app OWNER solv;\"\n   ```\n\n")
			case "mysql", "mariadb":
				b.WriteString("   ```sh\n   mysql -u root -p -e \"CREATE DATABASE app; CREATE USER 'solv'@'localhost' IDENTIFIED BY 'solv'; GRANT ALL ON app.* TO 'solv'@'localhost';\"\n   ```\n\n")
			case "mongodb":
				b.WriteString("   Crea la base `app` en tu MongoDB local y un usuario `solv` con permiso de lectura/escritura sobre ella.\n\n")
			case "redis":
				b.WriteString("   Inicia tu Redis local (`redis-server`). No necesita configuración adicional.\n\n")
			}
			fmt.Fprintf(&b, "   Después ajusta `%s` en tu `.env` para que apunte a `localhost:%d` en lugar del nombre del servicio de Docker.\n\n", exportURIVariable(spec.Engine), spec.Port)
		}
		b.WriteString("4. **Arranca tu aplicación.** ")
	} else {
		b.WriteString("3. **Arranca tu aplicación.** ")
	}
	if command := strings.TrimSpace(in.AppRunCommand); command != "" {
		fmt.Fprintf(&b, "El comando declarado por tu docente es `%s`.\n", command)
	} else if runtime.RunCommand != "" {
		fmt.Fprintf(&b, "El comando típico de un proyecto %s es `%s`.\n", runtime.Label, runtime.RunCommand)
	} else {
		b.WriteString("Usa el comando de arranque de tu proyecto (el que indique su README).\n")
	}
	b.WriteString("\n---\n\nSi algo no arranca, compara tu versión del runtime con la imagen base del aula y revisa que las variables del `.env` apunten a tus servicios locales.\n")
	return b.String()
}

func exportBaseImage(in ExportBundleInput) string {
	for _, line := range strings.Split(in.DockerfileContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), "FROM ") {
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(trimmed, "FROM "), "from "))
		}
	}
	image := strings.TrimSpace(in.TemplateImage)
	if image == "" || IsBuiltStackImageTag(image) {
		return ""
	}
	return image
}

func exportURIVariable(engine string) string {
	switch engine {
	case "mongodb":
		return "MONGODB_URI"
	case "redis":
		return "REDIS_URL"
	default:
		return "DATABASE_URL"
	}
}

func exportEngineLabel(engine string) string {
	switch engine {
	case "postgres":
		return "PostgreSQL"
	case "mysql":
		return "MySQL"
	case "mariadb":
		return "MariaDB"
	case "mongodb":
		return "MongoDB"
	case "redis":
		return "Redis"
	default:
		return engine
	}
}

func compactForReadme(layer string) string {
	const max = 160
	if len(layer) <= max {
		return layer
	}
	return layer[:max] + "…"
}
