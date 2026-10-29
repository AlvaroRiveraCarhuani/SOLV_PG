package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
)

type EnvTestStatus string

const (
	EnvTestStatusPending  EnvTestStatus = "pending"
	EnvTestStatusPulling  EnvTestStatus = "pulling"
	EnvTestStatusTesting  EnvTestStatus = "testing"
	EnvTestStatusSuccess  EnvTestStatus = "success"
	EnvTestStatusFailed   EnvTestStatus = "failed"
	EnvTestStatusCanceled EnvTestStatus = "canceled"
)

// Códigos de error estructurados (DA-04)
const (
	EnvTestErrNotFoundRemote  = "not_found_remote"
	EnvTestErrNetworkFailure  = "network_failure"
	EnvTestErrDegradedOffline = "degraded_offline"
	EnvTestErrPullStalled     = "pull_stalled"
	EnvTestErrPullTimeout     = "pull_timeout"
	EnvTestErrRegistryUnreach = "registry_unreachable"
	EnvTestErrTestOOM         = "test_oom"
	EnvTestErrTestCrash       = "test_crash"
	EnvTestErrToolMissing     = "tool_missing"
	EnvTestErrInternal        = "internal"
)

var (
	toolNameRegex      = regexp.MustCompile(`^[a-zA-Z0-9._+-]+$`)
	ErrToolNameInvalid = errors.New("nombre de herramienta inválido: solo caracteres alfanuméricos, punto, guion y guion bajo")
)

// ValidateToolName verifica que el nombre de una herramienta sea un ejecutable válido y seguro
func ValidateToolName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("el nombre de la herramienta no puede estar vacío")
	}
	if len(trimmed) > 64 {
		return errors.New("el nombre de la herramienta no debe exceder los 64 caracteres")
	}
	if !toolNameRegex.MatchString(trimmed) {
		return ErrToolNameInvalid
	}
	return nil
}

// ToolResult resultado de la presencia de un binario en la imagen probada
type ToolResult struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
	Version string `json:"version,omitempty"`
	Path    string `json:"path,omitempty"`
}

// EnvTestProgress detalle en vivo del progreso de descarga de capas OCI
type EnvTestProgress struct {
	BytesDone     int64   `json:"bytes_done"`
	BytesTotal    int64   `json:"bytes_total"`
	LayerCurrent  int     `json:"layer_current"`
	LayersTotal   int     `json:"layers_total"`
	Percent       float64 `json:"percent"`
	CurrentAction string  `json:"current_action,omitempty"`
}

// EnvTestResult reporte de ejecución dentro del contenedor efímero
type EnvTestResult struct {
	Tools      []ToolResult `json:"tools"`
	ExitCode   int          `json:"exit_code"`
	DurationMs int64        `json:"duration_ms"`
}

// EnvTestJob entidad raíz del job de verificación asíncrona de entorno.
type EnvTestJob struct {
	ID                      string                    `json:"id"`
	Image                   string                    `json:"image"`
	Tools                   []string                  `json:"tools"`
	TargetEnvironment       string                    `json:"target_environment,omitempty"` // "IDE_PERSISTENTE" | "JUEZ_EFIMERO"
	Entrypoint              string                    `json:"entrypoint,omitempty"`
	TimeoutMS               int                       `json:"timeout_ms,omitempty"`
	SampleInput             string                    `json:"sample_input,omitempty"`
	VerificationFingerprint string                    `json:"-"`
	VerificationConfig      EnvTestVerificationConfig `json:"-"`
	OwnerID                 string                    `json:"-"`
	TenantID                string                    `json:"-"`
	Status                  EnvTestStatus             `json:"status"`

	Progress         EnvTestProgress `json:"progress"`
	Result           *EnvTestResult  `json:"result,omitempty"`
	ErrorCode        string          `json:"error_code,omitempty"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	DigestUnverified bool            `json:"digest_unverified"`
	IsLocalImage     bool            `json:"is_local_image"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	FinishedAt       *time.Time      `json:"finished_at,omitempty"`
}

// IsTerminal indica si el job se encuentra en un estado definitivo
func (j *EnvTestJob) IsTerminal() bool {
	return j.Status == EnvTestStatusSuccess ||
		j.Status == EnvTestStatusFailed ||
		j.Status == EnvTestStatusCanceled
}

// EnvTestVerificationConfig reúne las entradas que determinan el comportamiento probado.
type EnvTestVerificationConfig struct {
	Image             string   `json:"image"`
	Tools             []string `json:"tools"`
	TargetEnvironment string   `json:"target_environment"`
	Entrypoint        string   `json:"entrypoint"`
	TimeoutMS         int      `json:"timeout_ms"`
	SampleInput       string   `json:"sample_input"`
	BaseRamMB         int      `json:"base_ram_mb"`
	DockerfileContent string   `json:"dockerfile_content"`
	SetupScript       string   `json:"setup_script"`
}

// Fingerprint devuelve la misma huella determinista que calcula el Studio.
func (c EnvTestVerificationConfig) Fingerprint() string {
	if c.TargetEnvironment == "" {
		c.TargetEnvironment = "IDE_PERSISTENTE"
	}
	if c.TimeoutMS <= 0 {
		c.TimeoutMS = 5000
	}
	if c.BaseRamMB <= 0 {
		c.BaseRamMB = 512
	}
	c.Image = strings.TrimSpace(c.Image)
	c.TargetEnvironment = strings.TrimSpace(c.TargetEnvironment)
	c.Entrypoint = strings.TrimSpace(c.Entrypoint)
	c.DockerfileContent = strings.TrimSpace(c.DockerfileContent)
	c.SetupScript = strings.TrimSpace(c.SetupScript)
	tools := make([]string, 0, len(c.Tools))
	for _, tool := range c.Tools {
		if trimmed := strings.TrimSpace(tool); trimmed != "" {
			tools = append(tools, trimmed)
		}
	}
	parts := []string{
		c.TargetEnvironment,
		c.Image,
		c.DockerfileContent,
		c.SetupScript,
		c.Entrypoint,
		strings.Join(tools, ","),
		fmt.Sprintf("%d", c.TimeoutMS),
		fmt.Sprintf("%d", c.BaseRamMB),
		c.SampleInput,
	}
	input := strings.Join(parts, ":::")
	var first, second uint32 = 0x811c9dc5, 0x811c9dc5
	for index, unit := range utf16.Encode([]rune(input)) {
		first ^= uint32(unit)
		first *= 0x01000193
		second ^= uint32(unit) ^ uint32(index&0xff)
		second *= 0x01000193
	}
	return fmt.Sprintf("%08x%08x", first, second)[:12]
}

// StartEnvTestRequest payload para iniciar la prueba asíncrona.
// OwnerID y TenantID se inyectan desde el contexto autenticado, nunca desde el cliente.
type StartEnvTestRequest struct {
	Image             string   `json:"image"`
	Tools             []string `json:"tools"`
	TargetEnvironment string   `json:"target_environment,omitempty"` // "IDE_PERSISTENTE" | "JUEZ_EFIMERO"
	Entrypoint        string   `json:"entrypoint,omitempty"`
	TimeoutMS         int      `json:"timeout_ms,omitempty"`
	SampleInput       string   `json:"sample_input,omitempty"`
	BaseRamMB         int      `json:"base_ram_mb,omitempty"`
	DockerfileContent string   `json:"dockerfile_content,omitempty"`
	SetupScript       string   `json:"setup_script,omitempty"`
	OwnerID           string   `json:"-"`
	TenantID          string   `json:"-"`
}
