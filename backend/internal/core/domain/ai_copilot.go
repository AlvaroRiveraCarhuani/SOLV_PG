package domain

// DTOs del Copiloto Estructurado para Componentes Tecnológicos (Admin) y
// Ejercicios con sello determinista del Juez (Docente). Ambos casos de uso se
// resuelven sobre AISettingsService.CompleteJSON y devuelven JSON saneado,
// nunca la salida cruda del LLM.

// AISuggestComponentRequestDTO es el payload de POST /api/v1/admin/ai/suggest-component.
type AISuggestComponentRequestDTO struct {
	Prompt        string         `json:"prompt"`                   // ej. "PyTorch CPU y scikit-learn para Inteligencia Artificial" (3..600 chars)
	Layer         ComponentLayer `json:"layer"`                    // opcional: "runtime" | "addon" | "database" (default "addon")
	TargetRuntime string         `json:"target_runtime,omitempty"` // ej. "python", "node", "cpp"
}

// AISuggestComponentResponseDTO devuelve el DTO ya normalizado con el
// build_layer_snippet sintetizado por SynthesizeAssistedBuildLayer.
type AISuggestComponentResponseDTO struct {
	Component CreateTemplateComponentDTO `json:"component"`
	Rationale string                     `json:"rationale"` // explicación técnica en español
	LatencyMS int64                      `json:"latency_ms"`
}

// AIGenerateExerciseRequestDTO es el payload de POST /api/v1/teacher/ai/generate-exercise.
type AIGenerateExerciseRequestDTO struct {
	Prompt           string `json:"prompt"`                       // consigna o tema pedagógico (5..1500 chars)
	Language         string `json:"language"`                     // "python", "cpp", "java", "javascript", "go", "rust", "c", "csharp"
	Difficulty       string `json:"difficulty"`                   // "easy" | "medium" | "hard" (default "medium")
	PublicCasesCount int    `json:"public_cases_count"`           // default 2, rango 1..5
	HiddenCasesCount int    `json:"hidden_cases_count"`           // default 3, rango 1..10
	VerifyWithRunner *bool  `json:"verify_with_runner,omitempty"` // nil o true ejecuta validación determinista
}

// AIGeneratedTestCaseDTO es un caso de prueba generado y, cuando aplica,
// sellado con la salida real del runner.
type AIGeneratedTestCaseDTO struct {
	OrderIndex       int                `json:"order_index"`
	Input            string             `json:"input"`
	ExpectedOutput   string             `json:"expected_output"`
	Visibility       TestCaseVisibility `json:"visibility"` // "example" | "public" | "hidden"
	Weight           float64            `json:"weight"`
	VerifiedByRunner bool               `json:"verified_by_runner"`
}

// AIRepairTemplateRequestDTO es el payload de POST /api/v1/admin/ai/repair-template:
// la radiografía fresca del fallo del contenedor, sin historial de chat.
type AIRepairTemplateRequestDTO struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	TargetEnvironment string   `json:"target_environment"`
	DockerImage       string   `json:"docker_image"`
	DockerfileContent string   `json:"dockerfile_content"`
	SetupScript       string   `json:"setup_script"`
	ToolsDeclared     []string `json:"tools_declared"`
	Entrypoint        string   `json:"entrypoint"`
	BaseRamMB         int      `json:"base_ram_mb"`
	ErrorMessage      string   `json:"error_message"`
	FailedTools       []string `json:"failed_tools"`
}

// AIRepairTemplateResponseDTO devuelve el manifiesto técnico reparado y la
// causa raíz explicada en español por el motor.
type AIRepairTemplateResponseDTO struct {
	DockerImage       string   `json:"docker_image"`
	DockerfileContent string   `json:"dockerfile_content"`
	SetupScript       string   `json:"setup_script"`
	ToolsDeclared     []string `json:"tools_declared"`
	Entrypoint        string   `json:"entrypoint"`
	BaseRamMB         int      `json:"base_ram_mb"`
	Rationale         string   `json:"rationale"`
	LatencyMS         int64    `json:"latency_ms"`
}

// AISuggestTemplateRequestDTO es el payload de POST /api/v1/admin/ai/suggest-template:
// el administrador describe los requisitos del entorno en lenguaje natural, sin
// tocar el constructor visual todavía (flujo propuesta -> revisión -> aplicar).
type AISuggestTemplateRequestDTO struct {
	Prompt            string `json:"prompt"`                       // ej. "Entorno para estructuras de datos con C++17, CMake y Sanitizers"
	TargetEnvironment string `json:"target_environment,omitempty"` // JUEZ_EFIMERO | IDE_PERSISTENTE | AMBOS
}

// AIMissingComponentDTO representa una tecnología o complemento detectado por la IA
// que fue solicitado por el usuario pero no existe actualmente en el catálogo del tenant.
type AIMissingComponentDTO struct {
	Name     string `json:"name"`               // Nombre reconocible de la herramienta (ej: PyTorch, Celery)
	Layer    string `json:"layer"`              // runtime | addon | database
	Category string `json:"category,omitempty"` // backend | web | framework | orm | toolkit | heavy
	Reason   string `json:"reason"`             // Justificación técnica breve de por qué se necesita
}

// AISuggestTemplateResponseDTO propone el stack completo de una plantilla con
// slugs canónicos de ecosistema o del catálogo. El frontend resuelve la
// disponibilidad real contra el catálogo del tenant y muestra las acciones
// para los componentes faltantes: el backend no aplica nada.
type AISuggestTemplateResponseDTO struct {
	Name              string                  `json:"name"`
	Description       string                  `json:"description"`
	RuntimeSlug       string                  `json:"runtime_slug"`
	WebPlatformSlug   string                  `json:"web_platform_slug,omitempty"`
	AddonSlugs        []string                `json:"addon_slugs,omitempty"`
	DatabaseSlug      string                  `json:"database_slug,omitempty"`
	SetupScript       string                  `json:"setup_script,omitempty"`
	BaseRamMB         int                     `json:"base_ram_mb"`
	Rationale         string                  `json:"rationale"` // explicación técnica en español
	MissingComponents []AIMissingComponentDTO `json:"missing_components,omitempty"`
	LatencyMS         int64                   `json:"latency_ms"`
}

// AIGenerateExerciseResponseDTO es el borrador de ejercicio devuelto al docente.
type AIGenerateExerciseResponseDTO struct {
	Title               string                   `json:"title"`
	Description         string                   `json:"description"` // enunciado en Markdown
	Difficulty          string                   `json:"difficulty"`
	Language            string                   `json:"language"`
	Boilerplate         string                   `json:"boilerplate"`
	ReferenceSolution   string                   `json:"reference_solution"` // lee de stdin, escribe en stdout
	TestCases           []AIGeneratedTestCaseDTO `json:"test_cases"`
	RunnerVerifiedAll   bool                     `json:"runner_verified_all"`
	VerificationWarning string                   `json:"verification_warning,omitempty"`
	LatencyMS           int64                    `json:"latency_ms"`
}
