package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)

// DependencyPreparationStatus es el estado de la preparación de una versión del proyecto.
type DependencyPreparationStatus string

const (
	// DependencyPreparationPrepared indica que el árbol quedó listo y usable sin red.
	DependencyPreparationPrepared DependencyPreparationStatus = "prepared"
	// DependencyPreparationFailed indica que la instalación o la comprobación falló.
	DependencyPreparationFailed DependencyPreparationStatus = "failed"
	// DependencyPreparationSkipped indica que la familia no declara preparación.
	DependencyPreparationSkipped DependencyPreparationStatus = "skipped"
)

var ErrInvalidDependencyPreparation = errors.New("solicitud de preparación inválida")

// ErrDependencyPreparationTransient marca un fallo que puede resolverse
// repitiendo el intento: un corte de red, un registro lento, un daemon que no
// responde. El resto de los fallos son deterministas y no se reintentan.
var ErrDependencyPreparationTransient = errors.New("fallo transitorio de preparación")

const (
	DefaultDependencyPreparationTimeout      = 15 * time.Minute
	DefaultDependencyPreparationMaxBytes     = int64(2) << 30
	DefaultDependencyPreparationMinFreeBytes = int64(10) << 30
	DefaultDependencyPreparationMaxAttempts  = 3
	DefaultDependencyPreparationRetryBackoff = 2 * time.Second
)

// DependencyPreparationLimits son los techos de una preparación en el nodo.
// Una sola preparación corre a la vez: dos instalaciones compiten por el mismo
// disco y el mismo enlace de red.
type DependencyPreparationLimits struct {
	Timeout      time.Duration
	MaxBytes     int64
	MinFreeBytes int64
	// MaxAttempts incluye el primer intento. Con 3 hay un intento y dos reintentos.
	MaxAttempts int
	// RetryBackoff es la espera del primer reintento; cada uno duplica la anterior.
	RetryBackoff time.Duration
}

// WithDefaults completa los techos que llegan en cero.
func (l DependencyPreparationLimits) WithDefaults() DependencyPreparationLimits {
	if l.Timeout <= 0 {
		l.Timeout = DefaultDependencyPreparationTimeout
	}
	if l.MaxBytes <= 0 {
		l.MaxBytes = DefaultDependencyPreparationMaxBytes
	}
	if l.MinFreeBytes <= 0 {
		l.MinFreeBytes = DefaultDependencyPreparationMinFreeBytes
	}
	if l.MaxAttempts <= 0 {
		l.MaxAttempts = DefaultDependencyPreparationMaxAttempts
	}
	if l.RetryBackoff <= 0 {
		l.RetryBackoff = DefaultDependencyPreparationRetryBackoff
	}
	return l
}

// Validate rechaza techos negativos o absurdos.
func (l DependencyPreparationLimits) Validate() error {
	if l.Timeout <= 0 {
		return fmt.Errorf("%w: el tiempo máximo debe ser positivo", ErrInvalidDependencyPreparation)
	}
	if l.MaxBytes <= 0 {
		return fmt.Errorf("%w: el techo de bytes debe ser positivo", ErrInvalidDependencyPreparation)
	}
	if l.MinFreeBytes < 0 {
		return fmt.Errorf("%w: el mínimo libre no puede ser negativo", ErrInvalidDependencyPreparation)
	}
	if l.MaxAttempts < 1 {
		return fmt.Errorf("%w: los intentos deben ser al menos uno", ErrInvalidDependencyPreparation)
	}
	if l.RetryBackoff < 0 {
		return fmt.Errorf("%w: la espera entre intentos no puede ser negativa", ErrInvalidDependencyPreparation)
	}
	return nil
}

// DependencyPreparationRequest describe una preparación concreta: qué imagen, qué
// versión del proyecto, qué perfil y qué comandos. Todo lo que el runner necesita
// viaja acá; el runner no conoce familias tecnológicas. El árbol que produce
// contiene el proyecto de la versión y sus dependencias: se prepara una vez por
// versión y el enlace copia solo las rutas declaradas en dependency_paths.
type DependencyPreparationRequest struct {
	TenantID       string
	VersionID      string
	ComponentSlug  string
	BaseImage      string
	ImageDigest    string
	ProfileVersion int

	ProjectPath string
	User        string
	Home        string

	// Files son los archivos de la versión publicada. Se escriben en el árbol
	// antes de instalar: sin ellos el gestor corre sobre un directorio vacío y
	// el resultado no sirve.
	Files []WorkspaceSeedFile

	Install            string
	InstallWithoutLock string
	Verify             string
	HasLockfile        bool
	OfflineFlags       string
	CacheEnv           map[string]string

	// Network es "build" cuando la instalación necesita red y "none" cuando el
	// gestor resuelve desde la caché. La comprobación siempre corre sin red.
	Network string

	LockfileHash string
}

// Validate comprueba que la solicitud tenga lo mínimo para ejecutarse.
func (r DependencyPreparationRequest) Validate() error {
	if strings.TrimSpace(r.TenantID) == "" {
		return fmt.Errorf("%w: falta el tenant", ErrInvalidDependencyPreparation)
	}
	if strings.TrimSpace(r.VersionID) == "" {
		return fmt.Errorf("%w: falta la versión del proyecto", ErrInvalidDependencyPreparation)
	}
	if strings.TrimSpace(r.BaseImage) == "" {
		return fmt.Errorf("%w: falta la imagen base", ErrInvalidDependencyPreparation)
	}
	if r.ProfileVersion < 1 {
		return fmt.Errorf("%w: la versión del perfil debe ser mayor o igual a 1", ErrInvalidDependencyPreparation)
	}
	projectPath := strings.TrimSpace(r.ProjectPath)
	if projectPath == "" || !strings.HasPrefix(projectPath, "/") || filepath.Clean(projectPath) == "/" {
		return fmt.Errorf("%w: la ruta del proyecto debe ser absoluta y distinta de la raíz", ErrInvalidDependencyPreparation)
	}
	if strings.Contains(projectPath, "..") {
		return fmt.Errorf("%w: la ruta del proyecto no puede salir del volumen", ErrInvalidDependencyPreparation)
	}
	install := strings.TrimSpace(r.Install)
	if !r.HasLockfile {
		if fallback := strings.TrimSpace(r.InstallWithoutLock); fallback != "" {
			install = fallback
		}
	}
	if install == "" {
		return fmt.Errorf("%w: falta el comando de instalación", ErrInvalidDependencyPreparation)
	}
	if strings.TrimSpace(r.Verify) == "" {
		return fmt.Errorf("%w: falta la comprobación de la preparación", ErrInvalidDependencyPreparation)
	}
	switch strings.TrimSpace(r.Network) {
	case "", "build", "none":
	default:
		return fmt.Errorf("%w: network debe ser build o none", ErrInvalidDependencyPreparation)
	}
	return nil
}

// InstallCommand devuelve el comando que corresponde según haya lockfile o no.
func (r DependencyPreparationRequest) InstallCommand() string {
	if !r.HasLockfile {
		if fallback := strings.TrimSpace(r.InstallWithoutLock); fallback != "" {
			return fallback
		}
	}
	return strings.TrimSpace(r.Install)
}

// Key calcula la clave del árbol preparado: digest de la imagen, versión del
// proyecto, versión del perfil y hash de los lockfiles. La versión entra en la
// clave a propósito: el árbol contiene el proyecto de esa versión, así que no se
// comparte entre versiones. Cambiar imagen, perfil o lockfile cambia la clave de
// la misma versión y fuerza preparación nueva.
func (r DependencyPreparationRequest) Key() string {
	image := strings.TrimSpace(r.ImageDigest)
	if image == "" {
		image = strings.TrimSpace(r.BaseImage)
	}
	canonical := strings.Join([]string{
		image,
		strings.TrimSpace(r.VersionID),
		fmt.Sprintf("%d", r.ProfileVersion),
		strings.TrimSpace(r.LockfileHash),
	}, "\n")
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

// DependencyPreparationResult es el resultado de una preparación.
type DependencyPreparationResult struct {
	Status   DependencyPreparationStatus
	Key      string
	Dir      string
	Bytes    int64
	Duration time.Duration
	LogPath  string
	Attempts int
	Reused   bool
	Message  string
}

// DependencyPreparationRunnerPort ejecuta las dos pasadas de una preparación.
// La instalación puede usar red según el perfil; la comprobación nunca.
// El runner no decide rutas ni techos: recibe el directorio y escribe su salida
// en el log que le pasa el worker.
type DependencyPreparationRunnerPort interface {
	RunInstall(ctx context.Context, req DependencyPreparationRequest, dir string, logWriter io.Writer) error
	RunVerify(ctx context.Context, req DependencyPreparationRequest, dir string, logWriter io.Writer) error
}
