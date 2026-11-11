package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// DependencyMaterialize describe dónde deja el gestor de paquetes el resultado de
// la preparación del proyecto.
type DependencyMaterialize string

const (
	// DependencyMaterializeTree deja el resultado dentro del proyecto
	// (node_modules, .venv, vendor/). Viaja con el snapshot y se enlaza al volumen.
	DependencyMaterializeTree DependencyMaterialize = "tree"
	// DependencyMaterializeCache deja el resultado fuera del proyecto
	// (~/.m2, GOMODCACHE, CARGO_HOME, NUGET_PACKAGES). Se comparte por el nodo.
	DependencyMaterializeCache DependencyMaterialize = "cache"
	// DependencyMaterializeNone indica que la familia no resuelve dependencias de
	// proyecto (un Makefile o un archivo único no declaran paquetes externos).
	DependencyMaterializeNone DependencyMaterialize = "none"
)

// DependencyProfile declara, por familia tecnológica, cómo se materializan las
// dependencias del proyecto: qué manifiestos mira, con qué comando instala, dónde
// deja el resultado, cómo comprueba que quedó usable y qué modificadores la
// obligan a resolver sin red. Es dato del catálogo, no código del backend.
type DependencyProfile struct {
	Version            int                   `json:"version"`
	Detect             []string              `json:"detect,omitempty"`
	Lockfiles          []string              `json:"lockfiles,omitempty"`
	Install            string                `json:"install,omitempty"`
	InstallWithoutLock string                `json:"install_without_lock,omitempty"`
	Materialize        DependencyMaterialize `json:"materialize,omitempty"`
	DependencyPaths    []string              `json:"dependency_paths,omitempty"`
	// CacheEnv es el entorno de la preparación: dónde guarda el gestor lo que
	// descarga. No se aplica en el aula.
	CacheEnv map[string]string `json:"cache_env,omitempty"`
	// RuntimeEnv es el entorno del aula: las variables que hacen que el alumno
	// encuentre el almacén en la misma ruta donde se preparó. Solo tiene sentido
	// con materialize tree, porque solo ahí hay una ruta entregada.
	RuntimeEnv   map[string]string `json:"runtime_env,omitempty"`
	Verify       string            `json:"verify,omitempty"`
	Network      string            `json:"network,omitempty"`
	OfflineFlags string            `json:"offline_flags,omitempty"`
	// User es el usuario efectivo dentro del contenedor, en formato uid:gid o
	// solo uid. Vacío deja el valor por defecto del nodo. El árbol preparado se
	// crea con ese dueño: sin él, el contenedor no puede escribir en el árbol.
	User string `json:"user,omitempty"`
	// Home es el directorio del usuario. Una ruta bajo /opt/solv/cache se
	// remapea dentro del árbol mientras la caché compartida no exista.
	Home string `json:"home,omitempty"`
}

var (
	ErrInvalidDependencyProfile = errors.New("perfil de dependencias inválido")
	envVarNameRegex             = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	manifestPatternRegex        = regexp.MustCompile(`^[A-Za-z0-9._*?\[\]-]+$`)
	// El usuario se valida numérico porque el worker lo usa como uid:gid del
	// dueño del árbol; un nombre exigiría resolverlo dentro de cada imagen.
	preparationUserRegex = regexp.MustCompile(`^[0-9]+(:[0-9]+)?$`)
)

// IsEmpty indica si el componente no declara preparación de dependencias.
func (p DependencyProfile) IsEmpty() bool {
	return p.Version == 0 &&
		len(p.Detect) == 0 &&
		len(p.Lockfiles) == 0 &&
		strings.TrimSpace(p.Install) == "" &&
		strings.TrimSpace(p.InstallWithoutLock) == "" &&
		p.Materialize == "" &&
		len(p.DependencyPaths) == 0 &&
		len(p.CacheEnv) == 0 &&
		len(p.RuntimeEnv) == 0 &&
		strings.TrimSpace(p.Verify) == "" &&
		strings.TrimSpace(p.Network) == "" &&
		strings.TrimSpace(p.OfflineFlags) == "" &&
		strings.TrimSpace(p.User) == "" &&
		strings.TrimSpace(p.Home) == ""
}

func hasControlChars(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func (p DependencyProfile) validateCommand(field, value string, required bool) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if required {
			return fmt.Errorf("%w: %s es obligatorio", ErrInvalidDependencyProfile, field)
		}
		return nil
	}
	if hasControlChars(value) {
		return fmt.Errorf("%w: %s no admite saltos de línea ni caracteres de control", ErrInvalidDependencyProfile, field)
	}
	return nil
}

// Validate comprueba que el perfil sea ejecutable sin ambigüedad. Un perfil vacío
// es válido: significa que la familia no prepara dependencias de proyecto.
func (p DependencyProfile) Validate() error {
	if p.IsEmpty() {
		return nil
	}
	if p.Version < 1 {
		return fmt.Errorf("%w: version debe ser mayor o igual a 1", ErrInvalidDependencyProfile)
	}

	switch p.Materialize {
	case DependencyMaterializeTree, DependencyMaterializeCache, DependencyMaterializeNone:
	case "":
		return fmt.Errorf("%w: materialize es obligatorio (%s, %s o %s)",
			ErrInvalidDependencyProfile, DependencyMaterializeTree, DependencyMaterializeCache, DependencyMaterializeNone)
	default:
		return fmt.Errorf("%w: materialize desconocido: %q", ErrInvalidDependencyProfile, p.Materialize)
	}

	switch p.Network {
	case "", "build", "none":
	default:
		return fmt.Errorf("%w: network debe ser build o none", ErrInvalidDependencyProfile)
	}

	if len(p.Detect) == 0 {
		return fmt.Errorf("%w: detect necesita al menos un manifiesto", ErrInvalidDependencyProfile)
	}
	for _, pattern := range p.Detect {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" || len(trimmed) > 120 || !manifestPatternRegex.MatchString(trimmed) {
			return fmt.Errorf("%w: manifiesto no válido: %q", ErrInvalidDependencyProfile, pattern)
		}
	}
	for _, pattern := range p.Lockfiles {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" || len(trimmed) > 120 || !manifestPatternRegex.MatchString(trimmed) {
			return fmt.Errorf("%w: lockfile no válido: %q", ErrInvalidDependencyProfile, pattern)
		}
	}

	requiresInstall := p.Materialize != DependencyMaterializeNone
	if err := p.validateCommand("install", p.Install, requiresInstall); err != nil {
		return err
	}
	if err := p.validateCommand("install_without_lock", p.InstallWithoutLock, false); err != nil {
		return err
	}
	if err := p.validateCommand("verify", p.Verify, requiresInstall); err != nil {
		return err
	}
	if err := p.validateCommand("offline_flags", p.OfflineFlags, false); err != nil {
		return err
	}

	if requiresInstall && len(p.DependencyPaths) == 0 {
		return fmt.Errorf("%w: dependency_paths es obligatorio cuando materialize es %s", ErrInvalidDependencyProfile, p.Materialize)
	}
	for _, rawPath := range p.DependencyPaths {
		path := strings.TrimSpace(rawPath)
		if path == "" || hasControlChars(path) {
			return fmt.Errorf("%w: ruta de dependencias no válida: %q", ErrInvalidDependencyProfile, rawPath)
		}
		if strings.Contains(path, "..") {
			return fmt.Errorf("%w: la ruta %q no puede salir del proyecto", ErrInvalidDependencyProfile, rawPath)
		}
		switch p.Materialize {
		case DependencyMaterializeTree:
			if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "~") {
				return fmt.Errorf("%w: con materialize tree la ruta %q debe ser relativa al proyecto", ErrInvalidDependencyProfile, rawPath)
			}
		case DependencyMaterializeCache:
			if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "~/") {
				return fmt.Errorf("%w: con materialize cache la ruta %q debe ser absoluta o empezar con ~/", ErrInvalidDependencyProfile, rawPath)
			}
		}
	}

	if user := strings.TrimSpace(p.User); user != "" && !preparationUserRegex.MatchString(user) {
		return fmt.Errorf("%w: user debe ser uid o uid:gid numérico", ErrInvalidDependencyProfile)
	}
	if home := strings.TrimSpace(p.Home); home != "" {
		if hasControlChars(home) || len(home) > 200 || strings.Contains(home, "..") {
			return fmt.Errorf("%w: home no válido: %q", ErrInvalidDependencyProfile, home)
		}
		if !strings.HasPrefix(home, "/") && !strings.HasPrefix(home, "~/") {
			return fmt.Errorf("%w: home debe ser absoluto o empezar con ~/", ErrInvalidDependencyProfile)
		}
	}

	for _, group := range []struct {
		field string
		vars  map[string]string
	}{
		{"cache_env", p.CacheEnv},
		{"runtime_env", p.RuntimeEnv},
	} {
		for name, value := range group.vars {
			if !envVarNameRegex.MatchString(name) {
				return fmt.Errorf("%w: nombre de variable no válido en %s: %q",
					ErrInvalidDependencyProfile, group.field, name)
			}
			if strings.TrimSpace(value) == "" || hasControlChars(value) {
				return fmt.Errorf("%w: valor no válido para %s en %s",
					ErrInvalidDependencyProfile, name, group.field)
			}
		}
	}

	return nil
}

// Value serializa el perfil para almacenarlo como JSONB.
func (p DependencyProfile) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan lee el perfil desde JSONB.
func (p *DependencyProfile) Scan(value interface{}) error {
	if value == nil {
		*p = DependencyProfile{}
		return nil
	}
	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return fmt.Errorf("perfil de dependencias: tipo inesperado %T", value)
	}
	if len(raw) == 0 {
		*p = DependencyProfile{}
		return nil
	}
	return json.Unmarshal(raw, p)
}
