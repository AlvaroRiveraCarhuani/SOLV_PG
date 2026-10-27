package domain

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type ComponentLayer string

const (
	ComponentLayerRuntime  ComponentLayer = "runtime"
	ComponentLayerAddon    ComponentLayer = "addon"
	ComponentLayerDatabase ComponentLayer = "database"
)

type InstallMode string

const (
	InstallModeNone     InstallMode = "none"
	InstallModePip      InstallMode = "pip"
	InstallModeUv       InstallMode = "uv"
	InstallModeNpm      InstallMode = "npm"
	InstallModePnpm     InstallMode = "pnpm"
	InstallModeYarn     InstallMode = "yarn"
	InstallModeBun      InstallMode = "bun"
	InstallModeApt      InstallMode = "apt"
	InstallModeCargo    InstallMode = "cargo"
	InstallModeGo       InstallMode = "go"
	InstallModeComposer InstallMode = "composer"
	InstallModeDotnet   InstallMode = "dotnet"
	InstallModeCustom   InstallMode = "custom"
)

var (
	ErrComponentNotFound      = errors.New("template component not found")
	ErrComponentSlugConflict  = errors.New("a technology component with this slug or name already exists")
	ErrComponentSystemLocked  = errors.New("system technology components cannot be deleted")
	ErrInvalidPackageName     = errors.New("invalid package name: only alphanumeric characters and @_./=->< are allowed")
	packageNameAllowlistRegex = regexp.MustCompile(`^[a-zA-Z0-9@_./=<>-]+$`)
)

// TemplateComponent representa un componente tecnológico del catálogo componible en el Constructor de Entornos.
type TemplateComponent struct {
	ID                 string         `db:"id" json:"id"`
	TenantID           string         `db:"tenant_id" json:"tenant_id"`
	Slug               string         `db:"slug" json:"slug"`
	Layer              ComponentLayer `db:"layer" json:"layer"`
	SubCategory        string         `db:"sub_category" json:"sub_category"` // "backend" | "web" | "framework" | "toolkit" | "relational" | "nosql"
	Name               string         `db:"name" json:"name"`
	Description        string         `db:"description" json:"description"`
	IconKey            string         `db:"icon_key" json:"icon_key"`
	BaseImage          string         `db:"base_image" json:"base_image"`
	CompatibleRuntimes []string       `db:"-" json:"compatible_runtimes"`
	InstallMode        InstallMode    `db:"install_mode" json:"install_mode"`
	PackageNames       []string       `db:"-" json:"package_names"`
	BuildLayerSnippet  string         `db:"build_layer_snippet" json:"build_layer_snippet"`
	SetupSnippet       string         `db:"setup_snippet" json:"setup_snippet"`
	JudgeEntrypoint    string         `db:"judge_entrypoint" json:"judge_entrypoint"`
	ToolsProvided      []string       `db:"-" json:"tools_provided"`
	ServiceEngine      string         `db:"service_engine" json:"service_engine"`
	RamDeltaMB         int            `db:"ram_delta_mb" json:"ram_delta_mb"`
	IsSystem           bool           `db:"is_system" json:"is_system"`
	IsActive           bool           `db:"is_active" json:"is_active"`
	SortOrder          int            `db:"sort_order" json:"sort_order"`
	CreatedAt          time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at" json:"updated_at"`
}

// CreateTemplateComponentDTO DTO para registrar un nuevo componente tecnológico en el catálogo.
type CreateTemplateComponentDTO struct {
	Name               string         `json:"name" validate:"required,min=2,max=120"`
	Slug               string         `json:"slug,omitempty"`
	Layer              ComponentLayer `json:"layer" validate:"required"`
	SubCategory        string         `json:"sub_category,omitempty"`
	Description        string         `json:"description,omitempty"`
	IconKey            string         `json:"icon_key,omitempty"`
	BaseImage          string         `json:"base_image,omitempty"`
	CompatibleRuntimes []string       `json:"compatible_runtimes,omitempty"`
	InstallMode        InstallMode    `json:"install_mode,omitempty"`
	PackageNames       []string       `json:"package_names,omitempty"`
	BuildLayerSnippet  string         `json:"build_layer_snippet,omitempty"`
	SetupSnippet       string         `json:"setup_snippet,omitempty"`
	JudgeEntrypoint    string         `json:"judge_entrypoint,omitempty"`
	ToolsProvided      []string       `json:"tools_provided,omitempty"`
	ServiceEngine      string         `json:"service_engine,omitempty"`
	RamDeltaMB         int            `json:"ram_delta_mb,omitempty"`
	SortOrder          int            `json:"sort_order,omitempty"`
}

// UpdateTemplateComponentDTO DTO para actualizar un componente tecnológico existente.
type UpdateTemplateComponentDTO struct {
	Name               string   `json:"name" validate:"required,min=2,max=120"`
	SubCategory        string   `json:"sub_category,omitempty"`
	Description        string   `json:"description,omitempty"`
	IconKey            string   `json:"icon_key,omitempty"`
	BaseImage          string   `json:"base_image,omitempty"`
	CompatibleRuntimes []string `json:"compatible_runtimes,omitempty"`
	InstallMode        string   `json:"install_mode,omitempty"`
	PackageNames       []string `json:"package_names,omitempty"`
	BuildLayerSnippet  string   `json:"build_layer_snippet,omitempty"`
	SetupSnippet       string   `json:"setup_snippet,omitempty"`
	JudgeEntrypoint    string   `json:"judge_entrypoint,omitempty"`
	ToolsProvided      []string `json:"tools_provided,omitempty"`
	ServiceEngine      string   `json:"service_engine,omitempty"`
	RamDeltaMB         *int     `json:"ram_delta_mb,omitempty"`
	IsActive           *bool    `json:"is_active,omitempty"`
	SortOrder          *int     `json:"sort_order,omitempty"`
}

// StackConfig configuración de componentes tecnológicos persistida junto con la plantilla o modelo.
type StackConfig struct {
	Mode              string   `json:"mode,omitempty"`               // "visual" | "code"
	RuntimeID         string   `json:"runtime_id,omitempty"`         // ID o slug del lenguaje principal
	WebPlatformID     string   `json:"web_platform_id,omitempty"`    // ID o slug del framework web opcional (Angular/React/Vue)
	AddonIDs          []string `json:"addon_ids,omitempty"`          // IDs o slugs de los complementos seleccionados
	DatabaseID        string   `json:"database_id,omitempty"`        // ID o slug del servicio de base de datos acoplado
	WorkspaceFolders  []string `json:"workspace_folders,omitempty"`  // Carpetas iniciales en /home/workspace
	DockerfileContent string   `json:"dockerfile_content,omitempty"` // Contenido del Dockerfile para auto-healing y reproducibilidad
}

func (sc StackConfig) Value() (driver.Value, error) {
	if sc.AddonIDs == nil {
		sc.AddonIDs = []string{}
	}
	if sc.WorkspaceFolders == nil {
		sc.WorkspaceFolders = []string{}
	}
	return json.Marshal(sc)
}

func (sc *StackConfig) Scan(value interface{}) error {
	if value == nil {
		*sc = StackConfig{AddonIDs: []string{}, WorkspaceFolders: []string{}}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan StackConfig: expected []byte, got %T", value)
	}
	if len(b) == 0 {
		*sc = StackConfig{AddonIDs: []string{}, WorkspaceFolders: []string{}}
		return nil
	}
	return json.Unmarshal(b, sc)
}

// ComposedTemplatePlan resultado determinista tras compilar una selección de componentes tecnológicos.
type ComposedTemplatePlan struct {
	SuggestedName     string         `json:"suggested_name"`
	ResolvedImage     string         `json:"resolved_image"`
	BaseImage         string         `json:"base_image"`
	NeedsLocalBuild   bool           `json:"needs_local_build"`
	SignatureHash     string         `json:"signature_hash"`
	DockerfileContent string         `json:"dockerfile_content"`
	SetupScript       string         `json:"setup_script"`
	ToolsDeclared     []string       `json:"tools_declared"`
	DefaultEntrypoint string         `json:"default_entrypoint"`
	RecommendedRamMB  int            `json:"recommended_ram_mb"`
	ServicesConfig    ServicesConfig `json:"services_config"`
}

// ValidatePackageNames verifica que cada nombre de paquete cumpla con la allowlist estricta sin metacaracteres de shell.
func ValidatePackageNames(pkgs []string) error {
	if len(pkgs) == 0 {
		return fmt.Errorf("%w: package list cannot be empty", ErrInvalidPackageName)
	}
	for _, raw := range pkgs {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || !packageNameAllowlistRegex.MatchString(trimmed) {
			return fmt.Errorf("%w: %q", ErrInvalidPackageName, raw)
		}
	}
	return nil
}

// SynthesizeAssistedBuildLayer construye la instrucción RUN limpia de Dockerfile según el gestor de paquetes elegido.
func SynthesizeAssistedBuildLayer(mode InstallMode, pkgs []string, customSnippet string) (string, error) {
	switch mode {
	case InstallModeNone, "":
		return "", nil
	case InstallModeCustom:
		return strings.TrimSpace(customSnippet), nil
	case InstallModePip, InstallModeUv, InstallModeNpm, InstallModePnpm, InstallModeYarn, InstallModeBun, InstallModeApt, InstallModeCargo, InstallModeGo, InstallModeComposer, InstallModeDotnet:
		if err := ValidatePackageNames(pkgs); err != nil {
			return "", err
		}
		var clean []string
		for _, p := range pkgs {
			clean = append(clean, strings.TrimSpace(p))
		}
		joined := strings.Join(clean, " ")
		switch mode {
		case InstallModePip:
			return "RUN pip install --no-cache-dir " + joined, nil
		case InstallModeUv:
			return "RUN uv pip install --system --no-cache " + joined, nil
		case InstallModeNpm:
			return "RUN npm install -g --no-audit --no-fund " + joined, nil
		case InstallModePnpm:
			return "RUN corepack enable && pnpm add -g " + joined, nil
		case InstallModeYarn:
			return "RUN corepack enable && yarn global add " + joined, nil
		case InstallModeBun:
			return "RUN bun add -g " + joined, nil
		case InstallModeApt:
			return "RUN apt-get update && apt-get install -y --no-install-recommends " + joined + " && rm -rf /var/lib/apt/lists/*", nil
		case InstallModeCargo:
			return "RUN cargo install " + joined, nil
		case InstallModeGo:
			return "RUN go install " + joined, nil
		case InstallModeComposer:
			return "RUN composer global require " + joined, nil
		case InstallModeDotnet:
			return "RUN dotnet tool install -g " + joined, nil
		}
	}
	return "", fmt.Errorf("unsupported install mode: %s", mode)
}

// ComputeStackSignature calcula una firma SHA-256 determinista de 12 caracteres hexadecimales sobre la imagen base y las capas.
func ComputeStackSignature(baseImage string, layers []string) string {
	var normalized []string
	normalized = append(normalized, "FROM "+strings.TrimSpace(baseImage))
	for _, l := range layers {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	canonical := strings.Join(normalized, "\n")
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])[:12]
}

// FormatBuiltStackImageTag devuelve el tag local inmutable para una imagen de entorno construida localmente en el nodo.
func FormatBuiltStackImageTag(signatureHash string) string {
	return "solv-local/stack:" + strings.TrimSpace(signatureHash)
}

// IsBuiltStackImageTag indica si una referencia de imagen pertenece al espacio de imágenes construidas localmente por SOLV.
func IsBuiltStackImageTag(imageRef string) bool {
	return strings.HasPrefix(strings.TrimSpace(imageRef), "solv-local/stack:")
}

// canonicalServiceEngines son los motores de base de datos gobernados por el catálogo.
// El catálogo no admite motores fuera de esta lista: cada uno tiene un modo de
// aprovisionamiento conocido (archivo local, host compartido de aula o contenedor
// satélite auto-orquestado).
var canonicalServiceEngines = []string{"none", "sqlite", "postgres", "mysql", "mariadb", "mongodb", "redis"}

// SupportedServiceEngines devuelve la lista canónica de motores soportados por el catálogo.
func SupportedServiceEngines() []string {
	out := make([]string, len(canonicalServiceEngines))
	copy(out, canonicalServiceEngines)
	return out
}

// IsSupportedServiceEngine indica si un motor pertenece al dominio soportado.
func IsSupportedServiceEngine(engine string) bool {
	return canonicalServiceEngine(engine) != ""
}

// canonicalServiceEngine resuelve un alias de motor a su nombre canónico ('none' cuando
// el componente declara explícitamente que no usa base de datos).
func canonicalServiceEngine(engine string) string {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "none":
		return "none"
	case "sqlite", "sqlite3":
		return "sqlite"
	case "postgres", "postgresql":
		return "postgres"
	case "mysql":
		return "mysql"
	case "mariadb":
		return "mariadb"
	case "mongodb", "mongo":
		return "mongodb"
	case "redis":
		return "redis"
	default:
		return ""
	}
}

// NormalizeServiceEngine resuelve el motor de servicio canónico a partir del valor
// declarado y, cuando viene vacío o desconocido, lo infiere del slug, nombre, icono o
// imagen del componente. Devuelve cadena vacía cuando no es reconocible.
func NormalizeServiceEngine(engine, slug, name, iconKey, baseImage string) string {
	if canonical := canonicalServiceEngine(engine); canonical != "" {
		return canonical
	}
	combined := strings.ToLower(strings.Join([]string{slug, name, iconKey, baseImage}, " "))
	switch {
	case strings.Contains(combined, "sqlite"):
		return "sqlite"
	case strings.Contains(combined, "postgres") || strings.Contains(combined, "pgsql"):
		return "postgres"
	case strings.Contains(combined, "mariadb"):
		return "mariadb"
	case strings.Contains(combined, "mysql"):
		return "mysql"
	case strings.Contains(combined, "mongo"):
		return "mongodb"
	case strings.Contains(combined, "redis"):
		return "redis"
	default:
		return ""
	}
}

// DefaultDatabaseSubCategory clasifica un motor: relacional, NoSQL/caché o sin base de datos.
func DefaultDatabaseSubCategory(engine string) string {
	switch canonicalServiceEngine(engine) {
	case "none":
		return "none"
	case "mongodb", "redis":
		return "nosql"
	default:
		return "relational"
	}
}

// DefaultDatabaseBaseImage devuelve la imagen con tag fijo del motor cuando el
// administrador no la declara. SQLite (archivo local) y PostgreSQL de aula (host
// compartido) no usan imagen propia.
func DefaultDatabaseBaseImage(engine string) string {
	switch canonicalServiceEngine(engine) {
	case "mysql":
		return "mysql:8.4"
	case "mariadb":
		return "mariadb:11.4"
	case "mongodb":
		return "mongo:7.0"
	case "redis":
		return "redis:7.2-alpine"
	default:
		return ""
	}
}

// DefaultDatabaseRamDeltaMB es la huella de memoria residente del motor cuando el
// administrador no la declara. Los motores de contenedor satélite se levantan bajo
// demanda y solo consumen RAM mientras hay laboratorios activos usándolos.
func DefaultDatabaseRamDeltaMB(engine string) int {
	switch canonicalServiceEngine(engine) {
	case "postgres":
		return 50
	case "redis":
		return 64
	case "mysql", "mariadb", "mongodb":
		return 128
	default:
		return 0
	}
}

// NormalizeCreateComponentDTO completa metadatos técnicos omitidos por el administrador (entrypoint, binarios verificables y huella de memoria) a partir de la familia tecnológica.
func NormalizeCreateComponentDTO(dto CreateTemplateComponentDTO) CreateTemplateComponentDTO {
	if dto.Layer == ComponentLayerDatabase {
		dto.ServiceEngine = NormalizeServiceEngine(dto.ServiceEngine, dto.Slug, dto.Name, dto.IconKey, dto.BaseImage)
		if dto.ServiceEngine == "" {
			dto.ServiceEngine = "postgres"
		}
		dto.SubCategory = DefaultDatabaseSubCategory(dto.ServiceEngine)
		if len(dto.CompatibleRuntimes) == 0 {
			dto.CompatibleRuntimes = []string{"universal"}
		}
		if strings.TrimSpace(dto.BaseImage) == "" {
			dto.BaseImage = DefaultDatabaseBaseImage(dto.ServiceEngine)
		}
		if dto.RamDeltaMB <= 0 {
			dto.RamDeltaMB = DefaultDatabaseRamDeltaMB(dto.ServiceEngine)
		}
		// Un motor de base de datos no instala paquetes: se aprovisiona como servicio.
		dto.InstallMode = InstallModeNone
		dto.PackageNames = nil
	}

	if dto.Layer == ComponentLayerRuntime {
		if strings.TrimSpace(dto.JudgeEntrypoint) == "" && strings.TrimSpace(dto.SubCategory) != "web" {
			dto.JudgeEntrypoint = InferRuntimeJudgeEntrypoint(dto.Name, dto.Slug, dto.IconKey, dto.BaseImage)
		}
		if len(dto.ToolsProvided) == 0 {
			dto.ToolsProvided = InferRuntimeTools(dto.Name, dto.Slug, dto.IconKey, dto.BaseImage)
		}
		if dto.RamDeltaMB <= 0 {
			dto.RamDeltaMB = InferRuntimeRamDeltaMB(dto.Name, dto.Slug, dto.IconKey, dto.BaseImage, dto.SubCategory)
		}
	}

	if len(dto.ToolsProvided) == 0 && dto.InstallMode == InstallModeApt && len(dto.PackageNames) > 0 {
		dto.ToolsProvided = InferAptVerifiableTools(dto.PackageNames)
	}

	return dto
}

// InferRuntimeJudgeEntrypoint resuelve el comando de evaluación del Juez Virtual diferenciando C puro de C++ y otros lenguajes estándar.
func InferRuntimeJudgeEntrypoint(name, slug, iconKey, baseImage string) string {
	combined := strings.ToLower(strings.TrimSpace(name + " " + slug + " " + iconKey))
	imgLower := strings.ToLower(strings.TrimSpace(baseImage))

	if strings.Contains(combined, "c++") || strings.Contains(combined, "g++") || strings.Contains(combined, "cpp") {
		return "g++ -O2 solution.cpp -o solution && ./solution"
	}
	if iconKey == "c" || strings.HasPrefix(slug, "c-gcc") || strings.HasPrefix(slug, "c-clang") || strings.Contains(combined, "lenguaje c") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(name)), "c (") {
		return "gcc -O2 solution.c -o solution && ./solution"
	}

	switch {
	case strings.Contains(imgLower, "python") || strings.Contains(combined, "python"):
		return "python3 solution.py"
	case strings.Contains(imgLower, "gcc") || strings.Contains(imgLower, "clang"):
		return "gcc -O2 solution.c -o solution && ./solution"
	case strings.Contains(imgLower, "java") || strings.Contains(imgLower, "temurin") || strings.Contains(imgLower, "openjdk"):
		return "javac Solution.java && java -Xmx256m Solution"
	case strings.Contains(imgLower, "golang") || strings.HasPrefix(imgLower, "go:") || iconKey == "go":
		return "go run main.go"
	case strings.Contains(imgLower, "node") || iconKey == "node":
		return "node solution.js"
	case strings.Contains(imgLower, "rust") || iconKey == "rust":
		return "rustc -O solution.rs -o solution && ./solution"
	case strings.Contains(imgLower, "php") || iconKey == "php":
		return "php solution.php"
	case strings.Contains(imgLower, "dotnet") || strings.Contains(combined, ".net") || strings.Contains(combined, "c#"):
		return "dotnet run"
	case strings.Contains(imgLower, "ruby") || strings.Contains(combined, "ruby"):
		return "ruby solution.rb"
	case strings.Contains(imgLower, "r-base") || iconKey == "r":
		return "Rscript solution.R"
	default:
		return ""
	}
}

// InferRuntimeTools deduce los binarios de verificación de arranque para un lenguaje principal cuando no se indicaron manualmente.
func InferRuntimeTools(name, slug, iconKey, baseImage string) []string {
	combined := strings.ToLower(strings.TrimSpace(name + " " + slug + " " + iconKey))
	imgLower := strings.ToLower(strings.TrimSpace(baseImage))

	if strings.Contains(combined, "c++") || strings.Contains(combined, "g++") || strings.Contains(combined, "cpp") {
		return []string{"g++", "gcc", "make"}
	}
	if iconKey == "c" || strings.HasPrefix(slug, "c-gcc") || strings.Contains(imgLower, "gcc") {
		return []string{"gcc", "make"}
	}
	switch {
	case strings.Contains(imgLower, "python"):
		return []string{"python3", "pip"}
	case strings.Contains(imgLower, "node"):
		return []string{"node", "npm"}
	case strings.Contains(imgLower, "temurin") || strings.Contains(imgLower, "openjdk") || strings.Contains(imgLower, "java"):
		return []string{"java", "javac"}
	case strings.Contains(imgLower, "golang"):
		return []string{"go", "gofmt"}
	case strings.Contains(imgLower, "rust"):
		return []string{"rustc", "cargo"}
	case strings.Contains(imgLower, "php"):
		return []string{"php"}
	case strings.Contains(imgLower, "dotnet"):
		return []string{"dotnet"}
	case strings.Contains(imgLower, "ruby"):
		return []string{"ruby", "gem"}
	case strings.Contains(imgLower, "r-base"):
		return []string{"R", "Rscript"}
	default:
		return nil
	}
}

// InferRuntimeRamDeltaMB determina la huella base de memoria del runtime según su arquitectura sin requerir configuración manual.
func InferRuntimeRamDeltaMB(name, slug, iconKey, baseImage, subCategory string) int {
	if strings.EqualFold(strings.TrimSpace(subCategory), "web") {
		return 256
	}
	combined := strings.ToLower(strings.TrimSpace(name + " " + slug + " " + iconKey + " " + baseImage))
	if strings.Contains(combined, "gcc") || strings.Contains(combined, "clang") || strings.Contains(combined, "c++") || iconKey == "c" || iconKey == "cpp" || strings.Contains(combined, "lua") || strings.Contains(combined, "alpine") {
		return 384
	}
	if strings.Contains(combined, "java") || strings.Contains(combined, "temurin") || strings.Contains(combined, "openjdk") || strings.Contains(combined, "dotnet") || strings.Contains(combined, "kotlin") || strings.Contains(combined, "julia") {
		return 1024
	}
	return 512
}

// InferAptVerifiableTools sincroniza los paquetes de sistema (apt/apk) hacia la lista de verificación del Smoke Test.
func InferAptVerifiableTools(pkgs []string) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(t string) {
		clean := strings.TrimSpace(t)
		if clean != "" && !seen[clean] {
			seen[clean] = true
			out = append(out, clean)
		}
	}
	for _, raw := range pkgs {
		pkg := strings.TrimSpace(raw)
		if pkg == "" {
			continue
		}
		switch pkg {
		case "build-essential":
			add("gcc")
			add("g++")
			add("make")
		case "ninja-build":
			add("ninja")
		default:
			add(pkg)
		}
	}
	return out
}
