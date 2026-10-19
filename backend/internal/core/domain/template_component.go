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
	InstallModeNone   InstallMode = "none"
	InstallModePip    InstallMode = "pip"
	InstallModeNpm    InstallMode = "npm"
	InstallModeApt    InstallMode = "apt"
	InstallModeCustom InstallMode = "custom"
)

var (
	ErrComponentNotFound     = errors.New("template component not found")
	ErrComponentSlugConflict = errors.New("a technology component with this slug or name already exists")
	ErrComponentSystemLocked = errors.New("system technology components cannot be deleted")
	ErrInvalidPackageName    = errors.New("invalid package name: only alphanumeric characters and @_./=->< are allowed")
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
	Mode             string   `json:"mode,omitempty"`              // "visual" | "code"
	RuntimeID        string   `json:"runtime_id,omitempty"`        // ID o slug del lenguaje principal
	WebPlatformID    string   `json:"web_platform_id,omitempty"`   // ID o slug del framework web opcional (Angular/React/Vue)
	AddonIDs         []string `json:"addon_ids,omitempty"`         // IDs o slugs de los complementos seleccionados
	DatabaseID       string   `json:"database_id,omitempty"`       // ID o slug del servicio de base de datos acoplado
	WorkspaceFolders []string `json:"workspace_folders,omitempty"` // Carpetas iniciales en /home/workspace
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
	case InstallModePip, InstallModeNpm, InstallModeApt:
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
		case InstallModeNpm:
			return "RUN npm install -g --no-audit --no-fund " + joined, nil
		case InstallModeApt:
			return "RUN apt-get update && apt-get install -y --no-install-recommends " + joined + " && rm -rf /var/lib/apt/lists/*", nil
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
