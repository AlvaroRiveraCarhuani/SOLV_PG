package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
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
	EnvTestErrPullStalled     = "pull_stalled"
	EnvTestErrPullTimeout     = "pull_timeout"
	EnvTestErrRegistryUnreach = "registry_unreachable"
	EnvTestErrTestOOM         = "test_oom"
	EnvTestErrTestCrash       = "test_crash"
	EnvTestErrInternal        = "internal"
)

var (
	toolNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._+-]+$`)
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

// EnvTestJob entidad raíz del job de verificación asíncrona de entorno
type EnvTestJob struct {
	ID               string          `json:"id"`
	Image            string          `json:"image"`
	Tools             []string        `json:"tools"`
	TargetEnvironment string          `json:"target_environment,omitempty"` // "IDE_PERSISTENTE" | "JUEZ_EFIMERO"
	Entrypoint        string          `json:"entrypoint,omitempty"`
	TimeoutMS         int             `json:"timeout_ms,omitempty"`
	SampleInput       string          `json:"sample_input,omitempty"`
	Status            EnvTestStatus   `json:"status"`
	Progress          EnvTestProgress `json:"progress"`
	Result            *EnvTestResult  `json:"result,omitempty"`
	ErrorCode         string          `json:"error_code,omitempty"`
	ErrorMessage      string          `json:"error_message,omitempty"`
	DigestUnverified  bool            `json:"digest_unverified"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	FinishedAt        *time.Time      `json:"finished_at,omitempty"`
}

// IsTerminal indica si el job se encuentra en un estado definitivo
func (j *EnvTestJob) IsTerminal() bool {
	return j.Status == EnvTestStatusSuccess ||
		j.Status == EnvTestStatusFailed ||
		j.Status == EnvTestStatusCanceled
}

// StartEnvTestRequest payload para iniciar la prueba asíncrona
type StartEnvTestRequest struct {
	Image             string   `json:"image"`
	Tools             []string `json:"tools"`
	TargetEnvironment string   `json:"target_environment,omitempty"` // "IDE_PERSISTENTE" | "JUEZ_EFIMERO"
	Entrypoint        string   `json:"entrypoint,omitempty"`
	TimeoutMS         int      `json:"timeout_ms,omitempty"`
	SampleInput       string   `json:"sample_input,omitempty"`
	BaseRamMB         int      `json:"base_ram_mb,omitempty"`
}
