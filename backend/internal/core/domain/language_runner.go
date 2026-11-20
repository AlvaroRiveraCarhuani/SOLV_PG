package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrLanguageRunnerLanguageRequired   = errors.New("language_runners: cada ejecutor necesita un lenguaje")
	ErrLanguageRunnerEntrypointRequired = errors.New("language_runners: cada lenguaje necesita su comando de compilación y ejecución")
	ErrLanguageRunnerDuplicate          = errors.New("language_runners: el lenguaje está declarado más de una vez")
	ErrLanguageRunnerImageRequired      = errors.New("language_runners: un lenguaje fuera del catálogo del juez necesita su imagen Docker")
	ErrLanguageRunnerImageUnpinned      = errors.New("language_runners: la imagen del ejecutor debe tener tag fijo o digest, nunca :latest")
	ErrLanguageRunnerTimeoutRange       = errors.New("language_runners: el tiempo límite por lenguaje debe estar entre 100 ms y 60000 ms")
	ErrLanguageRunnerMemoryRange        = errors.New("language_runners: la memoria por lenguaje debe estar entre 64 MB y 4096 MB")
	ErrLanguageRunnerLimit              = errors.New("language_runners: un entorno admite como máximo 12 lenguajes")

	// ErrLanguageNotSupportedByEnvironment rechaza una entrega cuyo lenguaje no
	// está declarado por el entorno de evaluación.
	ErrLanguageNotSupportedByEnvironment = errors.New("el entorno de evaluación no declara ese lenguaje")
)

// judgeCatalogOrder es el orden estable en el que se presentan los lenguajes
// del catálogo del juez. El orden no es alfabético a propósito: agrupa primero
// los lenguajes con más uso en clase.
var judgeCatalogOrder = []string{"python", "javascript", "cpp", "c", "csharp", "java", "go", "rust", "php"}

var judgeCatalogLanguages = func() map[string]bool {
	catalog := make(map[string]bool, len(judgeCatalogOrder))
	for _, language := range judgeCatalogOrder {
		catalog[language] = true
	}
	return catalog
}()

// CanonicalJudgeLanguage normaliza alias a la clave canónica del juez
// (c++ -> cpp, c#/cs -> csharp, py/python3 -> python, js/node -> javascript,
// golang -> go, rs -> rust). Un lenguaje fuera del catálogo se devuelve en
// minúsculas sin espacios, para que un administrador pueda declararlo con su
// propia imagen.
func CanonicalJudgeLanguage(language string) string {
	cleaned := strings.ToLower(strings.TrimSpace(language))
	switch cleaned {
	case "py", "python3":
		return "python"
	case "js", "node", "nodejs", "typescript", "ts":
		return "javascript"
	case "c++", "cplusplus", "cc":
		return "cpp"
	case "c#", "cs", "dotnet", ".net":
		return "csharp"
	case "golang":
		return "go"
	case "rs":
		return "rust"
	default:
		return cleaned
	}
}

// IsJudgeCatalogLanguage indica si el lenguaje tiene una estrategia de
// ejecución de primera clase en el motor del juez.
func IsJudgeCatalogLanguage(language string) bool {
	return judgeCatalogLanguages[CanonicalJudgeLanguage(language)]
}

// JudgeCatalogLanguages devuelve el catálogo canónico en su orden estable.
func JudgeCatalogLanguages() []string {
	out := make([]string, len(judgeCatalogOrder))
	copy(out, judgeCatalogOrder)
	return out
}

// LanguageRunner es el ejecutor de un lenguaje dentro de un entorno de
// evaluación. Un entorno multilingüe declara un ejecutor por lenguaje; el juez
// elige el que corresponde al lenguaje de la entrega.
//
// DockerImage vacía significa "usa la imagen del catálogo del juez para este
// lenguaje", de modo que agregar un lenguaje no obliga a construir una imagen
// que contenga todos los runtimes.
type LanguageRunner struct {
	Language      string `json:"language" db:"language"`
	Entrypoint    string `json:"entrypoint" db:"entrypoint"`
	DockerImage   string `json:"docker_image,omitempty" db:"docker_image"`
	TimeoutMS     int    `json:"timeout_ms,omitempty" db:"timeout_ms"`
	MemoryLimitMB int    `json:"memory_limit_mb,omitempty" db:"memory_limit_mb"`
}

// EffectiveTimeoutMS resuelve el tiempo límite del ejecutor con su respaldo.
func (r LanguageRunner) EffectiveTimeoutMS(fallback int) int {
	if r.TimeoutMS > 0 {
		return r.TimeoutMS
	}
	return fallback
}

// EffectiveMemoryMB resuelve la memoria del ejecutor con su respaldo.
func (r LanguageRunner) EffectiveMemoryMB(fallback int) int {
	if r.MemoryLimitMB > 0 {
		return r.MemoryLimitMB
	}
	return fallback
}

// NormalizeLanguageRunners valida y deja deterministas los ejecutores de un
// entorno. Es la única puerta de validación del contrato: el repositorio y el
// juez confían en su salida.
func NormalizeLanguageRunners(runners []LanguageRunner) ([]LanguageRunner, error) {
	if len(runners) == 0 {
		return []LanguageRunner{}, nil
	}
	if len(runners) > 12 {
		return nil, ErrLanguageRunnerLimit
	}

	normalized := make([]LanguageRunner, 0, len(runners))
	seen := make(map[string]bool, len(runners))

	for _, raw := range runners {
		language := CanonicalJudgeLanguage(raw.Language)
		if language == "" {
			return nil, ErrLanguageRunnerLanguageRequired
		}
		if seen[language] {
			return nil, fmt.Errorf("%w: %s", ErrLanguageRunnerDuplicate, language)
		}
		seen[language] = true

		entrypoint := strings.TrimSpace(raw.Entrypoint)
		if entrypoint == "" {
			return nil, fmt.Errorf("%w: %s", ErrLanguageRunnerEntrypointRequired, language)
		}
		if strings.ContainsAny(entrypoint, "\r\n") {
			return nil, fmt.Errorf("%w: %s", ErrLanguageRunnerEntrypointRequired, language)
		}

		image := strings.TrimSpace(raw.DockerImage)
		if image != "" {
			if strings.HasSuffix(strings.ToLower(image), ":latest") {
				return nil, fmt.Errorf("%w: %s", ErrLanguageRunnerImageUnpinned, image)
			}
			if strings.ContainsAny(image, " \t") {
				return nil, fmt.Errorf("%w: %s", ErrLanguageRunnerImageUnpinned, image)
			}
		} else if !IsJudgeCatalogLanguage(language) {
			return nil, fmt.Errorf("%w: %s", ErrLanguageRunnerImageRequired, language)
		}

		if raw.TimeoutMS != 0 && (raw.TimeoutMS < 100 || raw.TimeoutMS > 60000) {
			return nil, fmt.Errorf("%w: %s", ErrLanguageRunnerTimeoutRange, language)
		}
		if raw.MemoryLimitMB != 0 && (raw.MemoryLimitMB < 64 || raw.MemoryLimitMB > 4096) {
			return nil, fmt.Errorf("%w: %s", ErrLanguageRunnerMemoryRange, language)
		}

		normalized = append(normalized, LanguageRunner{
			Language:      language,
			Entrypoint:    entrypoint,
			DockerImage:   image,
			TimeoutMS:     raw.TimeoutMS,
			MemoryLimitMB: raw.MemoryLimitMB,
		})
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		return languageRank(normalized[i].Language) < languageRank(normalized[j].Language)
	})
	return normalized, nil
}

func languageRank(language string) string {
	for idx, catalogLanguage := range judgeCatalogOrder {
		if catalogLanguage == language {
			// El prefijo numérico conserva el orden del catálogo y deja el
			// resto en orden alfabético detrás.
			return fmt.Sprintf("%02d-%s", idx, language)
		}
	}
	return "99-" + language
}

// ResolveLanguageRunner busca el ejecutor del lenguaje de una entrega.
// Devuelve false cuando el entorno no declara ese lenguaje.
func ResolveLanguageRunner(runners []LanguageRunner, language string) (LanguageRunner, bool) {
	key := CanonicalJudgeLanguage(language)
	if key == "" {
		return LanguageRunner{}, false
	}
	for _, runner := range runners {
		if CanonicalJudgeLanguage(runner.Language) == key {
			return runner, true
		}
	}
	return LanguageRunner{}, false
}

// IsLanguageRunnerValidationError reconoce los rechazos del contrato de
// ejecutores, para que la entrega HTTP responda 422 y no 500.
func IsLanguageRunnerValidationError(err error) bool {
	targets := []error{
		ErrLanguageRunnerLanguageRequired,
		ErrLanguageRunnerEntrypointRequired,
		ErrLanguageRunnerDuplicate,
		ErrLanguageRunnerImageRequired,
		ErrLanguageRunnerImageUnpinned,
		ErrLanguageRunnerTimeoutRange,
		ErrLanguageRunnerMemoryRange,
		ErrLanguageRunnerLimit,
	}
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// LanguageRunnerLanguages devuelve las claves canónicas declaradas, en el orden
// determinista que dejó la normalización.
func LanguageRunnerLanguages(runners []LanguageRunner) []string {
	languages := make([]string, 0, len(runners))
	for _, runner := range runners {
		languages = append(languages, runner.Language)
	}
	return languages
}
