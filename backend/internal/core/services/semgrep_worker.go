package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"solv-backend/internal/core/domain"
)

type SemgrepWorker struct {
	repo         domain.WorkspaceRepository
	orchestrator domain.WorkspaceOrchestrator
	rulesDir     string
}

func NewSemgrepWorker(repo domain.WorkspaceRepository, orchestrator domain.WorkspaceOrchestrator, rulesDir string) *SemgrepWorker {
	if rulesDir == "" {
		rulesDir = "internal/infrastructure/semgrep/rules"
	}
	return &SemgrepWorker{
		repo:         repo,
		orchestrator: orchestrator,
		rulesDir:     rulesDir,
	}
}

// AuditWorkspace ejecuta la auditoría AST semántica sobre el volumen de un estudiante y persiste el JSONB en PostgreSQL
func (w *SemgrepWorker) AuditWorkspace(ctx context.Context, workspaceID string, volumeName string) ([]byte, error) {
	log.Printf("[Semgrep Worker] Starting AST semantic code audit on workspace %s (Volume: %s)...", workspaceID, volumeName)

	// 1. Ejecutar escaneo efímero en contenedor semgrep/semgrep en modo solo lectura (:ro)
	auditJSON, err := w.orchestrator.RunSemgrepScanOnVolume(ctx, volumeName)
	if err != nil {
		return nil, fmt.Errorf("semgrep scan failed for volume %s: %w", volumeName, err)
	}

	if len(auditJSON) == 0 {
		auditJSON = []byte("{}")
	}

	// 2. Persistir resultado de la auditoría AST en la columna JSONB de PostgreSQL
	if err := w.repo.SaveSemgrepAudit(ctx, workspaceID, auditJSON); err != nil {
		return nil, fmt.Errorf("failed to persist semgrep audit for workspace %s: %w", workspaceID, err)
	}

	log.Printf("[Semgrep Worker] AST semantic audit completed successfully for workspace %s. Persisted %d bytes to PostgreSQL JSONB.", workspaceID, len(auditJSON))
	return auditJSON, nil
}

type semgrepMatch struct {
	CheckID string `json:"check_id"`
	Start   struct {
		Line int `json:"line"`
	} `json:"start"`
	Extra struct {
		Message string `json:"message"`
	} `json:"extra"`
}

type semgrepCLIOutput struct {
	Results []semgrepMatch `json:"results"`
}

// ScanCode ejecuta el pre-chequeo con Semgrep CLI local contra el código fuente ingresado.
func (w *SemgrepWorker) ScanCode(code string, language string) (*domain.ScanResult, error) {
	langDir, fileExt := mapLanguageToRulesDirAndExt(language)
	if langDir == "" {
		// Lenguaje no soportado para pre-chequeo semgrep, retorna sin violaciones
		return &domain.ScanResult{HasViolations: false}, nil
	}

	ruleFile := filepath.Join(w.rulesDir, langDir, "forbidden.yaml")
	if _, err := os.Stat(ruleFile); os.IsNotExist(err) {
		// Archivo de reglas no existe, no bloquea
		return &domain.ScanResult{HasViolations: false}, nil
	}

	// 1. Crear archivo temporal con el código
	tmpFile, err := os.CreateTemp("", "solv-scan-*"+fileExt)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file for semgrep scan: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(code); err != nil {
		_ = tmpFile.Close()
		return nil, fmt.Errorf("failed to write code to temp file for semgrep scan: %w", err)
	}
	_ = tmpFile.Close()

	// 2. Resolver ejecutable de Semgrep (soporta PATH y $HOME/.local/bin)
	semgrepBin := findSemgrepBinary()

	// 3. Ejecutar semgrep CLI
	cmd := exec.Command(semgrepBin, "--config", ruleFile, "--json", tmpFile.Name())
	outBytes, err := cmd.Output()
	if err != nil {
		// Semgrep devuelve exit code 1 si hay hallazgos o errores, pero la salida JSON se escribe a stdout.
		// Si outBytes tiene contenido, intentamos parsear.
		if len(outBytes) == 0 {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return nil, fmt.Errorf("semgrep execution error: %s (stderr: %s)", err, string(exitErr.Stderr))
			}
			return nil, fmt.Errorf("semgrep execution error: %w", err)
		}
	}

	// 4. Parsear output JSON de Semgrep
	var cliOut semgrepCLIOutput
	if err := json.Unmarshal(outBytes, &cliOut); err != nil {
		return nil, fmt.Errorf("failed to unmarshal semgrep CLI output: %w", err)
	}

	violations := make([]domain.ScanViolation, 0, len(cliOut.Results))
	for _, match := range cliOut.Results {
		ruleID := match.CheckID
		// Limpiar prefijo de ruta larga en ruleID si existe
		if idx := strings.LastIndex(ruleID, "."); idx != -1 {
			ruleID = ruleID[idx+1:]
		}

		violations = append(violations, domain.ScanViolation{
			RuleID:  ruleID,
			Message: match.Extra.Message,
			Line:    match.Start.Line,
		})
	}

	return &domain.ScanResult{
		Violations:    violations,
		HasViolations: len(violations) > 0,
	}, nil
}

func mapLanguageToRulesDirAndExt(lang string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "python", "py":
		return "python", ".py"
	case "javascript", "js", "typescript", "ts", "node":
		return "javascript", ".js"
	case "java":
		return "java", ".java"
	case "csharp", "c#", "cs":
		return "csharp", ".cs"
	case "cpp", "c++", "c":
		return "cpp", ".cpp"
	default:
		return "", ".txt"
	}
}

func findSemgrepBinary() string {
	home, _ := os.UserHomeDir()
	localBin := filepath.Join(home, ".local", "bin", "semgrep")
	if _, err := os.Stat(localBin); err == nil {
		return localBin
	}

	if path, err := exec.LookPath("semgrep"); err == nil {
		return path
	}

	return "semgrep"
}
