package services

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

type stubComponentSource struct {
	items map[string]*domain.TemplateComponent
}

func newStubComponentSource() *stubComponentSource {
	items := []*domain.TemplateComponent{
		{
			ID:                 "rt-python",
			Slug:               "python-312",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Python 3.12",
			IconKey:            "python",
			BaseImage:          "python:3.12-slim-bookworm",
			CompatibleRuntimes: []string{"python"},
			JudgeEntrypoint:    "python3 solution.py",
			SetupSnippet:       "echo 'export PYTHONUNBUFFERED=1' >> ~/.bashrc",
			ToolsProvided:      []string{"python3", "pip"},
			RamDeltaMB:         512,
			IsActive:           true,
		},
		{
			ID:                 "rt-angular",
			Slug:               "angular-18",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "web",
			Name:               "Angular",
			IconKey:            "angular",
			BaseImage:          "node:20-bookworm-slim",
			CompatibleRuntimes: []string{"node", "web"},
			BuildLayerSnippet:  "RUN npm install -g --no-audit --no-fund @angular/cli",
			ToolsProvided:      []string{"node", "npm", "ng"},
			RamDeltaMB:         256,
			IsActive:           true,
		},
		{
			ID:                 "ad-fastapi",
			Slug:               "fastapi",
			Layer:              domain.ComponentLayerAddon,
			SubCategory:        "framework",
			Name:               "FastAPI",
			IconKey:            "fastapi",
			CompatibleRuntimes: []string{"python"},
			InstallMode:        domain.InstallModePip,
			PackageNames:       []string{"fastapi", "uvicorn"},
			BuildLayerSnippet:  "RUN pip install --no-cache-dir fastapi uvicorn",
			ToolsProvided:      []string{"python3", "pip"},
			RamDeltaMB:         64,
			IsActive:           true,
		},
		{
			ID:                 "rt-cpp",
			Slug:               "cpp-gcc13",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "C / C++ (GCC 13)",
			IconKey:            "cpp",
			BaseImage:          "gcc:13.2-bookworm",
			CompatibleRuntimes: []string{"cpp"},
			JudgeEntrypoint:    "gcc -O2 solution.c -o solution && ./solution",
			ToolsProvided:      []string{"gcc", "g++", "make"},
			RamDeltaMB:         384,
			IsActive:           true,
		},
		{
			ID:                 "ad-flask",
			Slug:               "flask",
			Layer:              domain.ComponentLayerAddon,
			SubCategory:        "framework",
			Name:               "Flask",
			IconKey:            "flask",
			CompatibleRuntimes: []string{"python"},
			InstallMode:        domain.InstallModePip,
			PackageNames:       []string{"flask"},
			BuildLayerSnippet:  "RUN pip install --no-cache-dir flask",
			ToolsProvided:      []string{"python3", "pip"},
			RamDeltaMB:         64,
			IsActive:           true,
		},
		{
			ID:            "db-postgres",
			Slug:          "postgres-16",
			Layer:         domain.ComponentLayerDatabase,
			SubCategory:   "relational",
			Name:          "PostgreSQL (Aula)",
			IconKey:       "postgres",
			ServiceEngine: "postgres",
			ToolsProvided: []string{"psql"},
			RamDeltaMB:    50,
			IsActive:      true,
		},
		{
			ID:            "db-mysql",
			Slug:          "mysql-84",
			Layer:         domain.ComponentLayerDatabase,
			SubCategory:   "relational",
			Name:          "MySQL 8.4",
			IconKey:       "mysql",
			ServiceEngine: "mysql",
			ToolsProvided: []string{"mysql"},
			RamDeltaMB:    128,
			IsActive:      true,
		},
		{
			ID:                 "rt-alpine",
			Slug:               "python-alpine",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Python Alpine",
			IconKey:            "python",
			BaseImage:          "python:3.12-alpine",
			CompatibleRuntimes: []string{"python"},
			ToolsProvided:      []string{"python3"},
			RamDeltaMB:         256,
			IsActive:           true,
		},
		{
			ID:                 "rt-julia",
			Slug:               "julia-lang",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Julia Data Science",
			IconKey:            "julia",
			BaseImage:          "julia:1.10-bookworm",
			CompatibleRuntimes: []string{"julia"},
			ToolsProvided:      []string{"julia"},
			RamDeltaMB:         1024,
			IsActive:           true,
		},
		{
			ID:                 "rt-lua",
			Slug:               "lua-jit",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Lua JIT",
			IconKey:            "lua",
			BaseImage:          "nickblah/lua:5.4-alpine",
			CompatibleRuntimes: []string{"lua"},
			ToolsProvided:      []string{"lua"},
			RamDeltaMB:         384,
			IsActive:           true,
		},
	}
	m := make(map[string]*domain.TemplateComponent, len(items)*2)
	for _, it := range items {
		m[it.ID] = it
		m[it.Slug] = it
	}
	return &stubComponentSource{items: m}
}

func (s *stubComponentSource) GetComponent(ctx context.Context, tenantID, idOrSlug string) (*domain.TemplateComponent, error) {
	if it, ok := s.items[idOrSlug]; ok {
		return it, nil
	}
	return nil, domain.ErrComponentNotFound
}

func TestComposeSingleRuntime_IDEPersistente_InjectsOpenVSCode(t *testing.T) {
	stub := newStubComponentSource()
	rtPython, _ := stub.GetComponent(context.Background(), "default", "rt-python")
	rtPython.BaseImage = "python:3.12-slim-bookworm"
	stub.items["rt-python"] = rtPython
	composer := NewTemplateComposerService(stub)
	plan, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:             "visual",
		RuntimeID:        "rt-python",
		WorkspaceFolders: []string{"src"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedCopy := fmt.Sprintf("COPY --from=%s /home/.openvscode-server /home/.openvscode-server", domain.OpenVSCodeImage)
	if !strings.Contains(plan.DockerfileContent, expectedCopy) {
		t.Fatalf("expected openvscode-server injection, got:\n%s", plan.DockerfileContent)
	}
	if !plan.NeedsLocalBuild {
		t.Fatalf("expected needs_build=true, got %v", plan.NeedsLocalBuild)
	}
	if !strings.HasPrefix(plan.ResolvedImage, "solv-local/stack:") {
		t.Fatalf("expected tag solv-local/stack:<sha>, got %s", plan.ResolvedImage)
	}
}

func TestComposeSingleRuntimeUsesDirectImage(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())
	plan, err := composer.Compose(context.Background(), "default", "JUEZ_EFIMERO", domain.StackConfig{
		Mode:             "visual",
		RuntimeID:        "rt-python",
		WorkspaceFolders: []string{"src"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.NeedsLocalBuild {
		t.Fatalf("expected single runtime without build layers to not require local build")
	}
	if plan.ResolvedImage != "python:3.12-slim-bookworm" {
		t.Fatalf("expected direct image python:3.12-slim-bookworm, got %s", plan.ResolvedImage)
	}
	// JUEZ_EFIMERO descarta workspace folders y scripts de setup del runtime.
	// (El script de setup se limpia igual que los workspace folders en judge, por lo que
	// plan.SetupScript debe estar vacío aquí.)
	if plan.SetupScript != "" {
		t.Fatalf("expected empty setup script in JUEZ_EFIMERO, got %q", plan.SetupScript)
	}
	if plan.DefaultEntrypoint != "python3 solution.py" {
		t.Fatalf("expected judge entrypoint python3 solution.py, got %q", plan.DefaultEntrypoint)
	}
}

func TestComposeMultiLayerBuildsDeterministicTag(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())
	plan, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:             "visual",
		RuntimeID:        "rt-python",
		WebPlatformID:    "rt-angular",
		AddonIDs:         []string{"ad-fastapi"},
		DatabaseID:       "db-mysql",
		WorkspaceFolders: []string{"backend", "frontend"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !plan.NeedsLocalBuild {
		t.Fatalf("expected multi-layer composition to require local build")
	}
	if !strings.HasPrefix(plan.ResolvedImage, "solv-local/stack:") {
		t.Fatalf("expected built tag solv-local/stack:<hash>, got %s", plan.ResolvedImage)
	}
	if !strings.Contains(plan.DockerfileContent, "FROM python:3.12-slim-bookworm") {
		t.Fatalf("expected Dockerfile to start from python:3.12-slim-bookworm, got:\n%s", plan.DockerfileContent)
	}
	if !strings.Contains(plan.DockerfileContent, "@angular/cli") || !strings.Contains(plan.DockerfileContent, "fastapi") {
		t.Fatalf("expected Dockerfile to include Angular and FastAPI layers, got:\n%s", plan.DockerfileContent)
	}
	if len(plan.ServicesConfig.Services) != 1 || plan.ServicesConfig.Services[0].Engine != "mysql" {
		t.Fatalf("expected mysql satellite service, got %+v", plan.ServicesConfig.Services)
	}
	if plan.RecommendedRamMB < 1024 {
		t.Fatalf("expected at least 1024 MB recommended RAM for Python+Angular+FastAPI+MySQL, got %d", plan.RecommendedRamMB)
	}
}

func TestComposeJudgeAutoEntrypointAndStripsWebAndDB(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())
	plan, err := composer.Compose(context.Background(), "default", "JUEZ_EFIMERO", domain.StackConfig{
		Mode:          "visual",
		RuntimeID:     "rt-python",
		WebPlatformID: "rt-angular",
		DatabaseID:    "db-mysql",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.DefaultEntrypoint != "python3 solution.py" {
		t.Fatalf("expected default judge entrypoint 'python3 solution.py', got %q", plan.DefaultEntrypoint)
	}
	if len(plan.ServicesConfig.Services) != 0 {
		t.Fatalf("expected judge environment to strip coupled database services, got %+v", plan.ServicesConfig.Services)
	}
}

func TestComposeResidentProcessMemoryModelWithoutDiskAddonInflation(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())

	// 1. C/C++ sin BD -> escalón inicial ADR-045 de 384 MB
	planCpp, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:      "visual",
		RuntimeID: "rt-cpp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if planCpp.RecommendedRamMB != 384 {
		t.Fatalf("expected 384 MB recommended RAM for C/C++ IDE, got %d", planCpp.RecommendedRamMB)
	}

	// 2. Python + FastAPI + Flask (librerías en disco) -> se mantiene en 512 MB sin inflación lineal
	planPyLibs, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:      "visual",
		RuntimeID: "rt-python",
		AddonIDs:  []string{"ad-fastapi", "ad-flask"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if planPyLibs.RecommendedRamMB != 512 {
		t.Fatalf("expected 512 MB recommended RAM for Python + disk libraries, got %d", planPyLibs.RecommendedRamMB)
	}

	// 3. Python + FastAPI + PostgreSQL Aula (+50 MB residente) -> escalón ADR-045 de 768 MB
	planPyPg, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:       "visual",
		RuntimeID:  "rt-python",
		AddonIDs:   []string{"ad-fastapi"},
		DatabaseID: "db-postgres",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if planPyPg.RecommendedRamMB != 768 {
		t.Fatalf("expected 768 MB recommended RAM for Python + PostgreSQL Aula, got %d", planPyPg.RecommendedRamMB)
	}
}

func TestComposeSoloFrontendUsesWebPlatformAsPrimary(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())
	plan, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:          "visual",
		RuntimeID:     "",
		WebPlatformID: "rt-angular",
	})
	if err != nil {
		t.Fatalf("unexpected error composing solo frontend: %v", err)
	}
	if plan.BaseImage != "node:20-bookworm-slim" {
		t.Fatalf("expected solo frontend to use web platform base image node:20-bookworm-slim, got %s", plan.BaseImage)
	}
	if !plan.NeedsLocalBuild {
		t.Fatalf("expected solo frontend with angular build snippet to need local build")
	}
	if !strings.Contains(plan.DockerfileContent, "@angular/cli") {
		t.Fatalf("expected dockerfile to contain @angular/cli, got:\n%s", plan.DockerfileContent)
	}
	// El runtime IDE (node:20-bookworm-slim) tampoco trae openvscode-server, así que el
	// inyector oficial COPY los dos faults distintos no son redundantes: uno trae Node desde
	// la imagen base (ya presente) y el otro trae el editor desde gitpod/openvscode-server.
	if !strings.Contains(plan.DockerfileContent, "openvscode-server") {
		t.Fatalf("expected ide runtime without openvscode-server to still receive the editor layer, got:\n%s", plan.DockerfileContent)
	}
	if plan.SuggestedName != "Laboratorio: Angular" {
		t.Fatalf("expected suggested name 'Laboratorio: Angular', got %q", plan.SuggestedName)
	}
	if plan.RecommendedRamMB != 1024 {
		t.Fatalf("expected 1024 MB recommended RAM for Solo Frontend with IDE editor, got %d", plan.RecommendedRamMB)
	}
	for _, tool := range plan.ToolsDeclared {
		if tool == "git" {
			t.Fatalf("expected no ghost git in ToolsDeclared, got %+v", plan.ToolsDeclared)
		}
	}
	foundNode := false
	foundNg := false
	for _, tool := range plan.ToolsDeclared {
		if tool == "node" {
			foundNode = true
		}
		if tool == "ng" {
			foundNg = true
		}
	}
	if !foundNode || !foundNg {
		t.Fatalf("expected tools declared to contain node and ng, got %+v", plan.ToolsDeclared)
	}
}

func TestComposeSingleRuntime_IDEPersistente_InjectsOpenVSCode_WithExistingEditorLayer(t *testing.T) {
	stub := newStubComponentSource()
	rtPython, _ := stub.GetComponent(context.Background(), "default", "rt-python")
	rtPython.BaseImage = "python:3.12-slim-bookworm"
	// Forced duplicate-style editor layer already present in build layers.
	stub.items["rt-python"] = rtPython
	composer := NewTemplateComposerService(stub)
	plan, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:             "visual",
		RuntimeID:        "rt-python",
		WorkspaceFolders: []string{"src"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedCopy := fmt.Sprintf("COPY --from=%s /home/.openvscode-server /home/.openvscode-server", domain.OpenVSCodeImage)
	if !strings.Contains(plan.DockerfileContent, expectedCopy) {
		t.Fatalf("expected openvscode-server injection, got:\n%s", plan.DockerfileContent)
	}
	if !plan.NeedsLocalBuild {
		t.Fatalf("expected needs_build=true, got %v", plan.NeedsLocalBuild)
	}
	if !strings.HasPrefix(plan.ResolvedImage, "solv-local/stack:") {
		t.Fatalf("expected tag solv-local/stack:<sha>, got %s", plan.ResolvedImage)
	}
}

func TestComposeFullStackMultiStageNodeInjection(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())
	plan, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:          "visual",
		RuntimeID:     "rt-python",
		WebPlatformID: "rt-angular",
	})
	if err != nil {
		t.Fatalf("unexpected error composing full stack: %v", err)
	}
	expectedCopy := "COPY --from=node:20-bookworm-slim /usr/local /usr/local"
	if !strings.Contains(plan.DockerfileContent, expectedCopy) {
		t.Fatalf("expected dockerfile to contain %q, got:\n%s", expectedCopy, plan.DockerfileContent)
	}
	if !strings.Contains(plan.DockerfileContent, "@angular/cli") {
		t.Fatalf("expected dockerfile to contain @angular/cli build snippet, got:\n%s", plan.DockerfileContent)
	}
	if strings.Contains(plan.DockerfileContent, "apt-get") || strings.Contains(plan.DockerfileContent, "apk add") {
		t.Fatalf("expected deterministic multi-stage node copy without apt-get/apk, got:\n%s", plan.DockerfileContent)
	}
	if plan.RecommendedRamMB != 1024 {
		t.Fatalf("expected 1024 MB recommended RAM for full stack, got %d", plan.RecommendedRamMB)
	}
}

func TestComposeFullStackAlpineUsesApkAdd(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())
	plan, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:          "visual",
		RuntimeID:     "rt-alpine",
		WebPlatformID: "rt-angular",
	})
	if err != nil {
		t.Fatalf("unexpected error composing alpine full stack: %v", err)
	}
	expectedApk := "RUN apk add --no-cache nodejs npm"
	if !strings.Contains(plan.DockerfileContent, expectedApk) {
		t.Fatalf("expected alpine dockerfile to contain %q, got:\n%s", expectedApk, plan.DockerfileContent)
	}
	if strings.Contains(plan.DockerfileContent, "COPY --from=node") {
		t.Fatalf("expected alpine to use apk add rather than glibc COPY --from node, got:\n%s", plan.DockerfileContent)
	}
	if !strings.Contains(plan.DockerfileContent, "openvscode-server") {
		t.Fatalf("expected alpine ide runtime to receive the editor layer, got:\n%s", plan.DockerfileContent)
	}
}

func TestComposeNoGhostGitInToolsDeclared(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())
	plan, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:      "visual",
		RuntimeID: "rt-python",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, tool := range plan.ToolsDeclared {
		if tool == "git" {
			t.Fatalf("expected plan.ToolsDeclared to not contain implicit git, found %+v", plan.ToolsDeclared)
		}
	}
	expected := []string{"python3", "pip"}
	if len(plan.ToolsDeclared) != len(expected) {
		t.Fatalf("expected tools %v, got %v", expected, plan.ToolsDeclared)
	}
}

func TestComposeJudgeRejectsEmptyRuntime(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())
	_, err := composer.Compose(context.Background(), "default", "JUEZ_EFIMERO", domain.StackConfig{
		Mode:          "visual",
		RuntimeID:     "",
		WebPlatformID: "rt-angular",
	})
	if err == nil {
		t.Fatalf("expected error when composing judge template without runtime_id, got nil")
	}
	if !strings.Contains(err.Error(), "runtime_id is required") {
		t.Fatalf("expected error message mentioning 'runtime_id is required', got: %v", err)
	}
}

func TestComposeAmbosDualEnvironment(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())
	plan, err := composer.Compose(context.Background(), "default", "AMBOS", domain.StackConfig{
		Mode:          "visual",
		RuntimeID:     "rt-python",
		WebPlatformID: "rt-angular",  // Debe ser ignorado en AMBOS
		DatabaseID:    "db-postgres", // Debe ser ignorado en AMBOS
		AddonIDs:      []string{"ad-fastapi"},
	})
	if err != nil {
		t.Fatalf("unexpected error composing AMBOS template: %v", err)
	}

	if !strings.HasPrefix(plan.SuggestedName, "Entorno Dual: ") {
		t.Fatalf("expected suggested name to start with 'Entorno Dual: ', got: %q", plan.SuggestedName)
	}
	if strings.Contains(plan.SuggestedName, "Angular") || strings.Contains(plan.SuggestedName, "PostgreSQL") {
		t.Fatalf("expected dual template to not couple web platform or satellite database in name, got: %q", plan.SuggestedName)
	}
	if len(plan.ServicesConfig.Services) != 0 {
		t.Fatalf("expected no satellite services in AMBOS plan, got %+v", plan.ServicesConfig.Services)
	}
	if plan.DefaultEntrypoint != "python3 solution.py" {
		t.Fatalf("expected default entrypoint 'python3 solution.py', got %q", plan.DefaultEntrypoint)
	}
	if !strings.Contains(plan.SetupScript, "PYTHONUNBUFFERED") {
		t.Fatalf("expected dual template to preserve setup script for IDE workspace, got %q", plan.SetupScript)
	}
}

func TestComposeDataDrivenRAMCalculation(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())

	// 1. Runtime con RamDeltaMB >= 1024 (Julia) sin clave "java" -> clasifica como heavy
	planJulia, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:      "visual",
		RuntimeID: "rt-julia",
	})
	if err != nil {
		t.Fatalf("unexpected error composing julia: %v", err)
	}
	if planJulia.RecommendedRamMB != 1024 {
		t.Fatalf("expected 1024 MB for heavy runtime Julia, got %d", planJulia.RecommendedRamMB)
	}

	// Julia en modo Juez -> 512 MB
	planJuliaJudge, err := composer.Compose(context.Background(), "default", "JUEZ_EFIMERO", domain.StackConfig{
		Mode:      "visual",
		RuntimeID: "rt-julia",
	})
	if err != nil {
		t.Fatalf("unexpected error composing julia in judge mode: %v", err)
	}
	if planJuliaJudge.RecommendedRamMB != 512 {
		t.Fatalf("expected 512 MB for heavy runtime Julia in judge, got %d", planJuliaJudge.RecommendedRamMB)
	}
	if planJuliaJudge.DefaultEntrypoint != "julia solution" {
		t.Fatalf("expected entrypoint inferred from tools provided ('julia solution'), got %q", planJuliaJudge.DefaultEntrypoint)
	}

	// 2. Runtime con RamDeltaMB <= 384 (Lua) sin clave "cpp" -> clasifica como light
	planLua, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
		Mode:      "visual",
		RuntimeID: "rt-lua",
	})
	if err != nil {
		t.Fatalf("unexpected error composing lua: %v", err)
	}
	if planLua.RecommendedRamMB != 384 {
		t.Fatalf("expected 384 MB for light runtime Lua, got %d", planLua.RecommendedRamMB)
	}

	// Lua en modo Juez -> 128 MB
	planLuaJudge, err := composer.Compose(context.Background(), "default", "JUEZ_EFIMERO", domain.StackConfig{
		Mode:      "visual",
		RuntimeID: "rt-lua",
	})
	if err != nil {
		t.Fatalf("unexpected error composing lua in judge mode: %v", err)
	}
	if planLuaJudge.RecommendedRamMB != 128 {
		t.Fatalf("expected 128 MB for light runtime Lua in judge, got %d", planLuaJudge.RecommendedRamMB)
	}
	if planLuaJudge.DefaultEntrypoint != "lua solution" {
		t.Fatalf("expected entrypoint inferred from tools provided ('lua solution'), got %q", planLuaJudge.DefaultEntrypoint)
	}
}

func TestSeparateCandCppRuntimesAndLab12Composition(t *testing.T) {
	src := newStubComponentSource()
	src.items["c-gcc13"] = &domain.TemplateComponent{
		ID:                 "rt-c",
		Slug:               "c-gcc13",
		Layer:              domain.ComponentLayerRuntime,
		SubCategory:        "backend",
		Name:               "C (GCC 13)",
		IconKey:            "c",
		BaseImage:          "gcc:13.2-bookworm",
		CompatibleRuntimes: []string{"c", "cpp"},
		JudgeEntrypoint:    "gcc -O2 solution.c -o solution && ./solution",
		ToolsProvided:      []string{"gcc", "make"},
		RamDeltaMB:         384,
		IsActive:           true,
	}
	src.items["cpp-gcc13"] = &domain.TemplateComponent{
		ID:                 "rt-cpp",
		Slug:               "cpp-gcc13",
		Layer:              domain.ComponentLayerRuntime,
		SubCategory:        "backend",
		Name:               "C++ (G++ 13)",
		IconKey:            "cpp",
		BaseImage:          "gcc:13.2-bookworm",
		CompatibleRuntimes: []string{"cpp", "c"},
		JudgeEntrypoint:    "g++ -O2 solution.cpp -o solution && ./solution",
		ToolsProvided:      []string{"g++", "gcc", "make"},
		RamDeltaMB:         384,
		IsActive:           true,
	}
	src.items["cpp-debug-valgrind"] = &domain.TemplateComponent{
		ID:                 "ad-cpp-debug",
		Slug:               "cpp-debug-valgrind",
		Layer:              domain.ComponentLayerAddon,
		SubCategory:        "toolkit",
		Name:               "Depuracion y Memoria (GDB + Valgrind)",
		IconKey:            "cpp",
		CompatibleRuntimes: []string{"c", "cpp"},
		InstallMode:        domain.InstallModeApt,
		PackageNames:       []string{"gdb", "valgrind"},
		BuildLayerSnippet:  "RUN apt-get update && apt-get install -y --no-install-recommends gdb valgrind && rm -rf /var/lib/apt/lists/*",
		ToolsProvided:      []string{"gdb", "valgrind"},
		RamDeltaMB:         0,
		IsActive:           true,
	}
	src.items["cpp-cmake"] = &domain.TemplateComponent{
		ID:                 "ad-cpp-cmake",
		Slug:               "cpp-cmake",
		Layer:              domain.ComponentLayerAddon,
		SubCategory:        "toolkit",
		Name:               "Construccion CMake",
		IconKey:            "cpp",
		CompatibleRuntimes: []string{"c", "cpp"},
		InstallMode:        domain.InstallModeApt,
		PackageNames:       []string{"cmake"},
		BuildLayerSnippet:  "RUN apt-get update && apt-get install -y --no-install-recommends cmake && rm -rf /var/lib/apt/lists/*",
		ToolsProvided:      []string{"cmake"},
		RamDeltaMB:         0,
		IsActive:           true,
	}

	composer := NewTemplateComposerService(src)

	planC, err := composer.Compose(context.Background(), "default", "AMBOS", domain.StackConfig{
		Mode:      "visual",
		RuntimeID: "c-gcc13",
	})
	if err != nil {
		t.Fatalf("unexpected error composing C runtime: %v", err)
	}
	if planC.DefaultEntrypoint != "gcc -O2 solution.c -o solution && ./solution" {
		t.Fatalf("expected C entrypoint to compile solution.c with gcc, got %q", planC.DefaultEntrypoint)
	}

	planCppLab12, err := composer.Compose(context.Background(), "default", "AMBOS", domain.StackConfig{
		Mode:      "visual",
		RuntimeID: "cpp-gcc13",
		AddonIDs:  []string{"cpp-debug-valgrind", "cpp-cmake"},
	})
	if err != nil {
		t.Fatalf("unexpected error composing C++ LAB-1.2: %v", err)
	}
	if planCppLab12.DefaultEntrypoint != "g++ -O2 solution.cpp -o solution && ./solution" {
		t.Fatalf("expected C++ entrypoint to compile solution.cpp with g++, got %q", planCppLab12.DefaultEntrypoint)
	}
	if planCppLab12.RecommendedRamMB != 384 {
		t.Fatalf("expected C++ LAB-1.2 with CLI addons to keep 384 MB recommended RAM, got %d", planCppLab12.RecommendedRamMB)
	}
	for _, expectedTool := range []string{"g++", "gdb", "valgrind", "cmake"} {
		found := false
		for _, tName := range planCppLab12.ToolsDeclared {
			if tName == expectedTool {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected tool %q in ToolsDeclared %v", expectedTool, planCppLab12.ToolsDeclared)
		}
	}
}

func TestNormalizeComponentMetadataAutoInfersToolsEntrypointAndMemory(t *testing.T) {
	// 1. Addon con apt y paquetes gdb valgrind cmake sin tools_provided ni ram_delta_mb
	addonDto := domain.CreateTemplateComponentDTO{
		Name:               "Herramientas C++",
		Layer:              domain.ComponentLayerAddon,
		SubCategory:        "toolkit",
		CompatibleRuntimes: []string{"cpp"},
		InstallMode:        domain.InstallModeApt,
		PackageNames:       []string{"gdb", "valgrind", "cmake"},
	}
	normalizedAddon := domain.NormalizeCreateComponentDTO(addonDto)
	if len(normalizedAddon.ToolsProvided) != 3 {
		t.Fatalf("expected apt packages to auto-populate ToolsProvided, got %v", normalizedAddon.ToolsProvided)
	}
	if normalizedAddon.RamDeltaMB != 0 {
		t.Fatalf("expected CLI/library addon to default to 0 MB resident delta, got %d", normalizedAddon.RamDeltaMB)
	}

	// 2. Runtime C++ sin entrypoint ni RAM manual
	rtDto := domain.CreateTemplateComponentDTO{
		Name:        "C++ (G++ 13)",
		Layer:       domain.ComponentLayerRuntime,
		SubCategory: "backend",
		IconKey:     "cpp",
		BaseImage:   "gcc:13.2-bookworm",
	}
	normalizedRt := domain.NormalizeCreateComponentDTO(rtDto)
	if normalizedRt.JudgeEntrypoint != "g++ -O2 solution.cpp -o solution && ./solution" {
		t.Fatalf("expected C++ runtime to infer g++ solution.cpp entrypoint, got %q", normalizedRt.JudgeEntrypoint)
	}
	if normalizedRt.RamDeltaMB != 384 {
		t.Fatalf("expected native C++ runtime to infer 384 MB RAM profile, got %d", normalizedRt.RamDeltaMB)
	}
}

func TestSynthesizeAssistedBuildLayerFirstClassManagers(t *testing.T) {
	cases := []struct {
		mode         domain.InstallMode
		pkgs         []string
		wantContains string
	}{
		{domain.InstallModePnpm, []string{"@angular/cli@18", "typescript"}, "corepack enable && pnpm add -g @angular/cli@18 typescript"},
		{domain.InstallModeYarn, []string{"vite"}, "corepack enable && yarn global add vite"},
		{domain.InstallModeBun, []string{"typescript"}, "bun add -g typescript"},
		{domain.InstallModeUv, []string{"fastapi", "uvicorn"}, "uv pip install --system --no-cache fastapi uvicorn"},
		{domain.InstallModeCargo, []string{"serde", "tokio"}, "cargo install serde tokio"},
		{domain.InstallModeGo, []string{"golang.org/x/tools/gopls@latest"}, "go install golang.org/x/tools/gopls@latest"},
		{domain.InstallModeComposer, []string{"laravel/installer"}, "composer global require laravel/installer"},
		{domain.InstallModeDotnet, []string{"dotnet-ef"}, "dotnet tool install -g dotnet-ef"},
	}

	for _, tc := range cases {
		got, err := domain.SynthesizeAssistedBuildLayer(tc.mode, tc.pkgs, "")
		if err != nil {
			t.Fatalf("mode %s failed unexpectedly: %v", tc.mode, err)
		}
		if !strings.Contains(got, tc.wantContains) {
			t.Fatalf("mode %s expected %q in %q", tc.mode, tc.wantContains, got)
		}
	}

	// Verify shell injection is blocked for all first-class modes
	if _, err := domain.SynthesizeAssistedBuildLayer(domain.InstallModePnpm, []string{"vite; rm -rf /"}, ""); err == nil {
		t.Fatalf("expected shell injection in pnpm package name to be rejected")
	}
}
