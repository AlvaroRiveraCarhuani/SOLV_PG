package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

// stubGenerationExerciseRepo implementa solo la lectura que usa la generación:
// el resto del contrato queda sin usar y no debe invocarse.
type stubGenerationExerciseRepo struct {
	domain.ExerciseRepository
	byID map[string]*domain.Exercise
}

func (s *stubGenerationExerciseRepo) GetByIDAndTenant(_ context.Context, id, tenantID string) (*domain.Exercise, error) {
	exercise, ok := s.byID[id]
	if !ok || exercise == nil || exercise.TenantID != tenantID {
		return nil, sql.ErrNoRows
	}
	return exercise, nil
}

func newIDEGenerationService(t *testing.T, content string) (*WorkspaceProjectService, *stubGenerationExerciseRepo) {
	t.Helper()
	server := newCopilotLLMServer(t, content)
	exerciseRepo := &stubGenerationExerciseRepo{byID: map[string]*domain.Exercise{
		"ex-ide": {
			ID:              "ex-ide",
			TenantID:        "t-1",
			Title:           "Práctica de proyecto",
			EnvironmentType: string(domain.EnvironmentTypeIDEPersistente),
		},
		"ex-juez": {
			ID:              "ex-juez",
			TenantID:        "t-1",
			Title:           "Práctica de juez",
			EnvironmentType: string(domain.EnvironmentTypeJuezEfimero),
		},
	}}
	// repo nil: la generación no debe tocar persistencia.
	svc := NewWorkspaceProjectService(nil, false)
	svc.SetAccessRepositories(exerciseRepo, nil)
	svc.SetAISettingsService(newEnabledCopilotSettings(t, server))
	return svc, exerciseRepo
}

func generationServer(t *testing.T, raw string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": raw}},
			},
		})
	}))
	t.Cleanup(server.Close)
	return server
}

func generatedProposalJSON(t *testing.T, files [][2]string, summary string) string {
	t.Helper()
	type file struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	payload := map[string]any{"summary": summary}
	entries := make([]file, 0, len(files))
	for _, pair := range files {
		entries = append(entries, file{Path: pair[0], Content: pair[1]})
	}
	payload["files"] = entries
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("no se pudo serializar la propuesta: %v", err)
	}
	return string(encoded)
}

// TestWorkspaceGenerateProposal_NormalizesFilesWithoutPersisting comprueba el
// camino feliz: la propuesta se normaliza y ordena igual que una importación y
// nunca se guarda un borrador.
func TestWorkspaceGenerateProposal_NormalizesFilesWithoutPersisting(t *testing.T) {
	raw := generatedProposalJSON(t, [][2]string{
		{"src/main.py", "print('hola')\r\n"},
		{"README.md", "# Proyecto\n\nDescripción.\n"},
	}, "Proyecto mínimo en Python con su README.")
	svc, _ := newIDEGenerationService(t, "```json\n"+raw+"\n```")

	proposal, err := svc.GenerateProposal(context.Background(), "t-1", "u-1", "ex-ide", GenerateWorkspaceProjectRequest{
		Description:  "Un proyecto mínimo en Python que imprima un saludo.",
		Technology:   "Python 3.12",
		Requirements: "Incluir README con instrucciones de lectura.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proposal.FileCount != 2 || len(proposal.Files) != 2 {
		t.Fatalf("esperaba 2 archivos, got %d", len(proposal.Files))
	}
	if proposal.Files[0].Path != "README.md" || proposal.Files[1].Path != "src/main.py" {
		t.Errorf("las rutas deben quedar ordenadas: %#v", []string{proposal.Files[0].Path, proposal.Files[1].Path})
	}
	if proposal.Files[1].Content != "print('hola')\n" {
		t.Errorf("el contenido debía canonicalizarse sin CRLF, got %q", proposal.Files[1].Content)
	}
	if proposal.TotalBytes == 0 || strings.TrimSpace(proposal.Summary) == "" {
		t.Errorf("la propuesta debe informar resumen y tamaño: %#v", proposal)
	}
}

// TestWorkspaceGenerateProposal_RejectsUntrustedResponses cubre las respuestas
// que el contrato no admite: rutas inseguras, duplicados, binarios, campos
// desconocidos y objetos vacíos.
func TestWorkspaceGenerateProposal_RejectsUntrustedResponses(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{name: "traversal", raw: generatedProposalJSON(t, [][2]string{{"../etc/passwd", "x\n"}}, "resumen")},
		{name: "absoluta", raw: generatedProposalJSON(t, [][2]string{{"/etc/passwd", "x\n"}}, "resumen")},
		{name: "separador_ambiguo", raw: generatedProposalJSON(t, [][2]string{{`src\main.py`, "x\n"}}, "resumen")},
		{name: "no_normalizada", raw: generatedProposalJSON(t, [][2]string{{"src//main.py", "x\n"}}, "resumen")},
		{name: "segmento_punto", raw: generatedProposalJSON(t, [][2]string{{"src/./main.py", "x\n"}}, "resumen")},
		{name: "duplicada", raw: generatedProposalJSON(t, [][2]string{{"main.py", "a\n"}, {"main.py", "b\n"}}, "resumen")},
		{name: "binario", raw: generatedProposalJSON(t, [][2]string{{"main.py", "a\x00b"}}, "resumen")},
		{name: "sin_archivos", raw: generatedProposalJSON(t, nil, "resumen")},
		{name: "sin_resumen", raw: generatedProposalJSON(t, [][2]string{{"main.py", "a\n"}}, "  ")},
		{name: "json_invalido", raw: `{"files":[{"path":"main.py","content":"a\n"}`},
		{name: "campo_desconocido", raw: `{"files":[{"path":"main.py","content":"a\n"}],"summary":"s","docker_image":"node:latest"}`},
		{name: "contenido_extra", raw: generatedProposalJSON(t, [][2]string{{"main.py", "a\n"}}, "resumen") + "\n\n{}"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newIDEGenerationService(t, tc.raw)
			_, err := svc.GenerateProposal(context.Background(), "t-1", "u-1", "ex-ide", GenerateWorkspaceProjectRequest{
				Description: "Un proyecto cualquiera para el laboratorio.",
			})
			if err == nil {
				t.Fatalf("esperaba rechazo de la propuesta")
			}
			var vErr *AISettingsValidationError
			if !errors.As(err, &vErr) || vErr.Code != "ai_response_invalid" {
				t.Fatalf("esperaba AISettingsValidationError ai_response_invalid, got %v", err)
			}
		})
	}
}

// TestWorkspaceGenerateProposal_EnforcesImportLimits confirma que la
// generación usa los mismos topes que la importación de Fase 2.
func TestWorkspaceGenerateProposal_EnforcesImportLimits(t *testing.T) {
	oversized := strings.Repeat("a", maxWorkspaceProjectFileBytes+1)
	svc, _ := newIDEGenerationService(t, generatedProposalJSON(t, [][2]string{{"big.txt", oversized}}, "resumen"))
	_, err := svc.GenerateProposal(context.Background(), "t-1", "u-1", "ex-ide", GenerateWorkspaceProjectRequest{
		Description: "Un proyecto con un archivo enorme.",
	})
	if err == nil {
		t.Fatalf("esperaba rechazo por tamaño de archivo")
	}
	var vErr *AISettingsValidationError
	if !errors.As(err, &vErr) || vErr.Code != "ai_response_invalid" {
		t.Fatalf("esperaba ai_response_invalid, got %v", err)
	}

	manyFiles := make([][2]string, 0, maxWorkspaceProjectFiles+1)
	for i := 0; i <= maxWorkspaceProjectFiles; i++ {
		manyFiles = append(manyFiles, [2]string{fmt.Sprintf("archivo-%03d.txt", i), "a\n"})
	}
	svcMany, _ := newIDEGenerationService(t, generatedProposalJSON(t, manyFiles, "resumen"))
	// Cada ruta es distinta; se espera el rechazo por cantidad de archivos.
	if _, err := svcMany.GenerateProposal(context.Background(), "t-1", "u-1", "ex-ide", GenerateWorkspaceProjectRequest{
		Description: "Un proyecto con demasiados archivos.",
	}); err == nil {
		t.Fatalf("esperaba rechazo por cantidad de archivos")
	}
}

// TestWorkspaceGenerateProposal_OnlyForIDEPersistentExercise verifica que el
// juez efímero y los ejercicios de otro tenant no lleguen al proveedor.
func TestWorkspaceGenerateProposal_OnlyForIDEPersistentExercise(t *testing.T) {
	svc, _ := newIDEGenerationService(t, generatedProposalJSON(t, [][2]string{{"main.py", "a\n"}}, "resumen"))

	if _, err := svc.GenerateProposal(context.Background(), "t-1", "u-1", "ex-juez", GenerateWorkspaceProjectRequest{
		Description: "Un proyecto para el juez.",
	}); !errors.Is(err, domain.ErrWorkspaceProjectNotIDEPersistent) {
		t.Fatalf("esperaba ErrWorkspaceProjectNotIDEPersistent, got %v", err)
	}

	if _, err := svc.GenerateProposal(context.Background(), "t-2", "u-1", "ex-ide", GenerateWorkspaceProjectRequest{
		Description: "Un proyecto de otro instituto.",
	}); !errors.Is(err, domain.ErrWorkspaceProjectNotFound) {
		t.Fatalf("esperaba ErrWorkspaceProjectNotFound para otro tenant, got %v", err)
	}
}

// TestWorkspaceGenerateProposal_RequiresConfiguredProvider evita simular una
// generación exitosa cuando la IA institucional no está disponible.
func TestWorkspaceGenerateProposal_RequiresConfiguredProvider(t *testing.T) {
	exerciseRepo := &stubGenerationExerciseRepo{byID: map[string]*domain.Exercise{
		"ex-ide": {ID: "ex-ide", TenantID: "t-1", Title: "Práctica", EnvironmentType: string(domain.EnvironmentTypeIDEPersistente)},
	}}

	withoutProvider := NewWorkspaceProjectService(nil, false)
	withoutProvider.SetAccessRepositories(exerciseRepo, nil)
	_, err := withoutProvider.GenerateProposal(context.Background(), "t-1", "u-1", "ex-ide", GenerateWorkspaceProjectRequest{
		Description: "Un proyecto cualquiera.",
	})
	var vErr *AISettingsValidationError
	if !errors.As(err, &vErr) || vErr.Code != "ai_provider_unavailable" {
		t.Fatalf("esperaba ai_provider_unavailable, got %v", err)
	}

	disabled := NewWorkspaceProjectService(nil, false)
	disabled.SetAccessRepositories(exerciseRepo, nil)
	disabled.SetAISettingsService(NewAISettingsService(newTestAISettingsRepo(`{}`)))
	if _, err := disabled.GenerateProposal(context.Background(), "t-1", "u-1", "ex-ide", GenerateWorkspaceProjectRequest{
		Description: "Un proyecto cualquiera.",
	}); !errors.Is(err, domain.ErrAIDisabled) {
		t.Fatalf("esperaba domain.ErrAIDisabled, got %v", err)
	}
}

// TestWorkspaceGenerateProposal_ReportsProviderTimeout usa el mismo error
// tipado que el resto del módulo de IA para que el handler responda 504.
func TestWorkspaceGenerateProposal_ReportsProviderTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(500 * time.Millisecond):
		}
	}))
	t.Cleanup(server.Close)

	exerciseRepo := &stubGenerationExerciseRepo{byID: map[string]*domain.Exercise{
		"ex-ide": {ID: "ex-ide", TenantID: "t-1", Title: "Práctica", EnvironmentType: string(domain.EnvironmentTypeIDEPersistente)},
	}}
	svc := NewWorkspaceProjectService(nil, false)
	svc.SetAccessRepositories(exerciseRepo, nil)
	svc.SetAISettingsService(newEnabledCopilotSettings(t, server))

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	_, err := svc.GenerateProposal(ctx, "t-1", "u-1", "ex-ide", GenerateWorkspaceProjectRequest{
		Description: "Un proyecto que tardará más de lo permitido.",
	})
	if !errors.Is(err, domain.ErrAITimeout) {
		t.Fatalf("esperaba domain.ErrAITimeout, got %v", err)
	}
}

// TestWorkspaceGenerateProposal_ValidatesTeacherInput acota lo que el docente
// puede enviar antes de gastar una invocación del proveedor.
func TestWorkspaceGenerateProposal_ValidatesTeacherInput(t *testing.T) {
	svc, _ := newIDEGenerationService(t, generatedProposalJSON(t, [][2]string{{"main.py", "a\n"}}, "resumen"))

	_, err := svc.GenerateProposal(context.Background(), "t-1", "u-1", "ex-ide", GenerateWorkspaceProjectRequest{
		Description: "   ",
	})
	var vErr *AISettingsValidationError
	if !errors.As(err, &vErr) || vErr.Code != "generation_description_required" {
		t.Fatalf("esperaba generation_description_required, got %v", err)
	}

	_, err = svc.GenerateProposal(context.Background(), "t-1", "u-1", "ex-ide", GenerateWorkspaceProjectRequest{
		Description: strings.Repeat("a", 2001),
	})
	if !errors.As(err, &vErr) || vErr.Code != "generation_input_too_large" {
		t.Fatalf("esperaba generation_input_too_large, got %v", err)
	}
}
