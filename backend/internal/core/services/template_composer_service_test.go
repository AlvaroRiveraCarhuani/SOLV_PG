package services

import (
	"context"
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

func TestComposeSingleRuntimeUsesDirectImage(t *testing.T) {
	composer := NewTemplateComposerService(newStubComponentSource())
	plan, err := composer.Compose(context.Background(), "default", "IDE_PERSISTENTE", domain.StackConfig{
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
	if !strings.Contains(plan.SetupScript, "mkdir -p /home/workspace/src") {
		t.Fatalf("expected setup script to create /home/workspace/src, got %q", plan.SetupScript)
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
