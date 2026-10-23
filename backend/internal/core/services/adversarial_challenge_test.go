package services

import (
	"context"
	"strings"
	"testing"

	"solv-backend/internal/core/domain"
)

// makeAdversarialCatalog creates an extensive fixture catalog to stress test all edge cases.
func makeAdversarialCatalog() *stubComponentSource {
	items := []*domain.TemplateComponent{
		// 1. Python Runtimes
		{
			ID:                 "rt-python-debian",
			Slug:               "python-312-debian",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Python 3.12 (Debian)",
			IconKey:            "python",
			BaseImage:          "python:3.12-slim-bookworm",
			CompatibleRuntimes: []string{"python"},
			JudgeEntrypoint:    "python3 solution.py",
			ToolsProvided:      []string{"python3", "pip"},
			RamDeltaMB:         512,
			IsActive:           true,
		},
		{
			ID:                 "rt-python-alpine",
			Slug:               "python-312-alpine",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Python 3.12 (Alpine)",
			IconKey:            "python",
			BaseImage:          "python:3.12-alpine3.19",
			CompatibleRuntimes: []string{"python"},
			JudgeEntrypoint:    "python3 solution.py",
			ToolsProvided:      []string{"python3"},
			RamDeltaMB:         256,
			IsActive:           true,
		},
		// 2. Go Runtimes
		{
			ID:                 "rt-go-debian",
			Slug:               "golang-122-debian",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Go 1.22 (Debian)",
			IconKey:            "go",
			BaseImage:          "golang:1.22-bookworm",
			CompatibleRuntimes: []string{"go"},
			JudgeEntrypoint:    "go run main.go",
			ToolsProvided:      []string{"go"},
			RamDeltaMB:         512,
			IsActive:           true,
		},
		{
			ID:                 "rt-go-alpine",
			Slug:               "golang-122-alpine",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Go 1.22 (Alpine)",
			IconKey:            "go",
			BaseImage:          "golang:1.22-alpine",
			CompatibleRuntimes: []string{"go"},
			JudgeEntrypoint:    "go run main.go",
			ToolsProvided:      []string{"go"},
			RamDeltaMB:         256,
			IsActive:           true,
		},
		// 3. Node Runtimes (Backend)
		{
			ID:                 "rt-node-backend",
			Slug:               "node-20-backend",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Node.js 20 LTS",
			IconKey:            "node",
			BaseImage:          "node:20-bookworm-slim",
			CompatibleRuntimes: []string{"node", "javascript"},
			JudgeEntrypoint:    "node solution.js",
			ToolsProvided:      []string{"node", "npm"},
			RamDeltaMB:         512,
			IsActive:           true,
		},
		{
			ID:                 "rt-node-alpine",
			Slug:               "node-20-alpine",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Node.js 20 (Alpine)",
			IconKey:            "node",
			BaseImage:          "node:20-alpine",
			CompatibleRuntimes: []string{"node"},
			ToolsProvided:      []string{"node", "npm"},
			RamDeltaMB:         384,
			IsActive:           true,
		},
		// 4. Java Runtime
		{
			ID:                 "rt-java-21",
			Slug:               "java-21",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Java 21 (Temurin)",
			IconKey:            "java",
			BaseImage:          "eclipse-temurin:21-jdk-jammy",
			CompatibleRuntimes: []string{"java"},
			JudgeEntrypoint:    "javac Solution.java && java -Xmx256m Solution",
			ToolsProvided:      []string{"java", "javac"},
			RamDeltaMB:         1024,
			IsActive:           true,
		},
		// 5. C/C++ Runtime
		{
			ID:                 "rt-cpp-gcc",
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
		// 6. Rust Runtime (inferJudgeEntrypoint testing)
		{
			ID:                 "rt-rust-slim",
			Slug:               "rust-176",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "backend",
			Name:               "Rust 1.76",
			IconKey:            "rust",
			BaseImage:          "rust:1.76-slim-bookworm",
			CompatibleRuntimes: []string{"rust"},
			ToolsProvided:      []string{"rustc", "cargo"},
			RamDeltaMB:         512,
			IsActive:           true,
		},
		// 7. Web Platforms
		{
			ID:                 "web-angular-standard",
			Slug:               "angular-18",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "web",
			Name:               "Angular 18",
			IconKey:            "angular",
			BaseImage:          "node:20-bookworm-slim",
			CompatibleRuntimes: []string{"node", "web"},
			BuildLayerSnippet:  "RUN npm install -g --no-audit --no-fund @angular/cli@18",
			ToolsProvided:      []string{"node", "npm", "ng"},
			RamDeltaMB:         256,
			IsActive:           true,
		},
		{
			ID:                 "web-vue-custom-node",
			Slug:               "vue-3-custom",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "web",
			Name:               "Vue 3 (Node 22)",
			IconKey:            "vue",
			BaseImage:          "node:22-bookworm-slim",
			CompatibleRuntimes: []string{"node", "web"},
			BuildLayerSnippet:  "RUN npm install -g --no-audit --no-fund create-vue",
			ToolsProvided:      []string{"node", "npm"},
			RamDeltaMB:         256,
			IsActive:           true,
		},
		{
			ID:                 "web-react-alpine",
			Slug:               "react-alpine",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "web",
			Name:               "React (Alpine Base)",
			IconKey:            "react",
			BaseImage:          "node:20-alpine",
			CompatibleRuntimes: []string{"node", "web"},
			BuildLayerSnippet:  "RUN npm install -g --no-audit --no-fund create-react-app",
			ToolsProvided:      []string{"node", "npm"},
			RamDeltaMB:         256,
			IsActive:           true,
		},
		{
			ID:                 "web-static-html",
			Slug:               "static-html",
			Layer:              domain.ComponentLayerRuntime,
			SubCategory:        "web",
			Name:               "Static Web",
			IconKey:            "html5",
			BaseImage:          "nginx:1.25-alpine",
			CompatibleRuntimes: []string{"web"},
			BuildLayerSnippet:  "",
			ToolsProvided:      []string{"nginx"},
			RamDeltaMB:         128,
			IsActive:           true,
		},
		// 8. Addons
		{
			ID:                "ad-git-explicit",
			Slug:              "git-tools",
			Layer:             domain.ComponentLayerAddon,
			SubCategory:       "tools",
			Name:              "Git VCS Tools",
			IconKey:           "git",
			BuildLayerSnippet: "RUN apt-get update && apt-get install -y git",
			ToolsProvided:     []string{"git", "git-lfs"},
			RamDeltaMB:        32,
			IsActive:          true,
		},
		{
			ID:                "ad-fastapi",
			Slug:              "fastapi",
			Layer:             domain.ComponentLayerAddon,
			SubCategory:       "framework",
			Name:              "FastAPI",
			IconKey:           "fastapi",
			InstallMode:       domain.InstallModePip,
			PackageNames:      []string{"fastapi", "uvicorn"},
			BuildLayerSnippet: "RUN pip install --no-cache-dir fastapi uvicorn",
			ToolsProvided:     []string{"python3", "pip"},
			RamDeltaMB:        64,
			IsActive:          true,
		},
		{
			ID:                "ad-heavy-datascience",
			Slug:              "datascience-heavy",
			Layer:             domain.ComponentLayerAddon,
			SubCategory:       "heavy",
			Name:              "Data Science Heavy Bundle",
			IconKey:           "python",
			BuildLayerSnippet: "RUN pip install torch pandas scikit-learn",
			ToolsProvided:     []string{"python3", "pip"},
			RamDeltaMB:        512,
			IsActive:          true,
		},
		// 9. Databases
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
			ID:            "db-redis",
			Slug:          "redis-7",
			Layer:         domain.ComponentLayerDatabase,
			SubCategory:   "cache",
			Name:          "Redis Cache",
			IconKey:       "redis",
			ServiceEngine: "redis",
			ToolsProvided: []string{"redis-cli"},
			RamDeltaMB:    64,
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

// -------------------------------------------------------------
// ADVERSARIAL CHALLENGE TEST 1: Solo Frontend Edge Cases
// -------------------------------------------------------------
func TestAdversarial_SoloFrontendTopologies(t *testing.T) {
	composer := NewTemplateComposerService(makeAdversarialCatalog())
	ctx := context.Background()

	t.Run("Standard Angular as sole primary", func(t *testing.T) {
		plan, err := composer.Compose(ctx, "default", "IDE_PERSISTENTE", domain.StackConfig{
			Mode:             "visual",
			RuntimeID:        "",
			WebPlatformID:    "web-angular-standard",
			WorkspaceFolders: []string{"src", "public"},
		})
		if err != nil {
			t.Fatalf("unexpected error composing standard solo frontend: %v", err)
		}
		if plan.BaseImage != "node:20-bookworm-slim" {
			t.Errorf("expected BaseImage 'node:20-bookworm-slim', got %q", plan.BaseImage)
		}
		if !plan.NeedsLocalBuild {
			t.Errorf("expected NeedsLocalBuild=true for Angular cli snippet")
		}
		if !strings.Contains(plan.DockerfileContent, "@angular/cli@18") {
			t.Errorf("expected Dockerfile to contain angular snippet, got:\n%s", plan.DockerfileContent)
		}
		if strings.Contains(plan.DockerfileContent, "COPY --from=") && !strings.Contains(plan.DockerfileContent, "openvscode-server") {
			t.Errorf("unexpected redundant COPY --from multi-stage copy in solo frontend:\n%s", plan.DockerfileContent)
		}
		if strings.Contains(plan.DockerfileContent, "apk add") {
			t.Errorf("unexpected apk add in solo frontend:\n%s", plan.DockerfileContent)
		}
		if plan.SuggestedName != "Laboratorio: Angular 18" {
			t.Errorf("expected suggested name 'Laboratorio: Angular 18', got %q", plan.SuggestedName)
		}
		if plan.RecommendedRamMB != 1024 {
			t.Errorf("expected 1024 MB for Solo Frontend, got %d", plan.RecommendedRamMB)
		}
		if !strings.Contains(plan.SetupScript, "mkdir -p /home/workspace/src /home/workspace/public") {
			t.Errorf("expected workspace directories in setup script, got %q", plan.SetupScript)
		}
	})

	t.Run("Solo Frontend with Custom Node Base Image", func(t *testing.T) {
		plan, err := composer.Compose(ctx, "default", "IDE_PERSISTENTE", domain.StackConfig{
			Mode:          "visual",
			RuntimeID:     "",
			WebPlatformID: "web-vue-custom-node",
		})
		if err != nil {
			t.Fatalf("unexpected error composing vue solo frontend: %v", err)
		}
		if plan.BaseImage != "node:22-bookworm-slim" {
			t.Errorf("expected custom BaseImage 'node:22-bookworm-slim', got %q", plan.BaseImage)
		}
		if !strings.Contains(plan.DockerfileContent, "FROM node:22-bookworm-slim") {
			t.Errorf("expected Dockerfile FROM node:22-bookworm-slim, got:\n%s", plan.DockerfileContent)
		}
		if strings.Contains(plan.DockerfileContent, "COPY --from=") && !strings.Contains(plan.DockerfileContent, "openvscode-server") {
			t.Errorf("unexpected redundant COPY --from in solo frontend with custom node image:\n%s", plan.DockerfileContent)
		}
	})

	t.Run("Solo Frontend with coupled Database (e.g. Postgres)", func(t *testing.T) {
		plan, err := composer.Compose(ctx, "default", "IDE_PERSISTENTE", domain.StackConfig{
			Mode:          "visual",
			RuntimeID:     "",
			WebPlatformID: "web-angular-standard",
			DatabaseID:    "db-postgres",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(plan.ServicesConfig.Services) != 1 || plan.ServicesConfig.Services[0].Engine != "postgres" {
			t.Errorf("expected coupled postgres service in solo frontend with DB, got %+v", plan.ServicesConfig)
		}
		// RAM: 210 (editor) + 50 (pg) + 640 (active web) = 900 -> 1024 MB
		if plan.RecommendedRamMB != 1024 {
			t.Errorf("expected 1024 MB, got %d", plan.RecommendedRamMB)
		}
		if plan.SuggestedName != "Laboratorio: Angular 18 + PostgreSQL (Aula)" {
			t.Errorf("expected composite suggested name, got %q", plan.SuggestedName)
		}
	})

	t.Run("Solo Frontend with Heavy Addon", func(t *testing.T) {
		plan, err := composer.Compose(ctx, "default", "IDE_PERSISTENTE", domain.StackConfig{
			Mode:          "visual",
			RuntimeID:     "",
			WebPlatformID: "web-angular-standard",
			AddonIDs:      []string{"ad-heavy-datascience"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// RAM: 210 (editor) + 1280 (heavy process) = 1490 -> 2048 MB
		if plan.RecommendedRamMB != 2048 {
			t.Errorf("expected 2048 MB for solo frontend with heavy addon, got %d", plan.RecommendedRamMB)
		}
	})
}

// -------------------------------------------------------------
// ADVERSARIAL CHALLENGE TEST 2: Multi-Distro & Multi-Stage Node Injection
// -------------------------------------------------------------
func TestAdversarial_FullStackMultiDistroInjection(t *testing.T) {
	composer := NewTemplateComposerService(makeAdversarialCatalog())
	ctx := context.Background()

	cases := []struct {
		name              string
		runtimeID         string
		webID             string
		expectFrom        string
		expectSnippet     string
		forbidSnippet     string
		expectRecommended int
	}{
		{
			name:              "Python Debian + Angular standard -> Multi-Stage COPY from default node",
			runtimeID:         "rt-python-debian",
			webID:             "web-angular-standard",
			expectFrom:        "FROM python:3.12-slim-bookworm",
			expectSnippet:     "COPY --from=node:20-bookworm-slim /usr/local /usr/local",
			forbidSnippet:     "apk add",
			expectRecommended: 1024,
		},
		{
			name:              "Python Debian + Vue custom node 22 -> Multi-Stage COPY from web.BaseImage",
			runtimeID:         "rt-python-debian",
			webID:             "web-vue-custom-node",
			expectFrom:        "FROM python:3.12-slim-bookworm",
			expectSnippet:     "COPY --from=node:22-bookworm-slim /usr/local /usr/local",
			forbidSnippet:     "apk add",
			expectRecommended: 1024,
		},
		{
			name:              "Python Alpine + Angular standard -> RUN apk add nodejs npm",
			runtimeID:         "rt-python-alpine",
			webID:             "web-angular-standard",
			expectFrom:        "FROM python:3.12-alpine3.19",
			expectSnippet:     "RUN apk add --no-cache nodejs npm",
			forbidSnippet:     "COPY --from=",
			expectRecommended: 1024,
		},
		{
			name:              "Go Alpine + Angular standard -> RUN apk add nodejs npm",
			runtimeID:         "rt-go-alpine",
			webID:             "web-angular-standard",
			expectFrom:        "FROM golang:1.22-alpine",
			expectSnippet:     "RUN apk add --no-cache nodejs npm",
			forbidSnippet:     "COPY --from=",
			expectRecommended: 1024,
		},
		{
			name:              "Go Debian + Angular standard -> Multi-Stage COPY",
			runtimeID:         "rt-go-debian",
			webID:             "web-angular-standard",
			expectFrom:        "FROM golang:1.22-bookworm",
			expectSnippet:     "COPY --from=node:20-bookworm-slim /usr/local /usr/local",
			forbidSnippet:     "apk add",
			expectRecommended: 1024,
		},
		{
			name:              "Node Backend + Angular standard -> NO redundant node injection",
			runtimeID:         "rt-node-backend",
			webID:             "web-angular-standard",
			expectFrom:        "FROM node:20-bookworm-slim",
			expectSnippet:     "RUN npm install -g --no-audit --no-fund @angular/cli@18",
			forbidSnippet:     "COPY --from=",
			expectRecommended: 1024,
		},
		{
			name:              "Node Alpine Backend + Angular standard -> NO redundant injection (node present)",
			runtimeID:         "rt-node-alpine",
			webID:             "web-angular-standard",
			expectFrom:        "FROM node:20-alpine",
			expectSnippet:     "RUN npm install -g --no-audit --no-fund @angular/cli@18",
			forbidSnippet:     "apk add --no-cache nodejs",
			expectRecommended: 1024,
		},
		{
			name:              "Java 21 Temurin + Angular -> Multi-Stage COPY and 2048 MB RAM",
			runtimeID:         "rt-java-21",
			webID:             "web-angular-standard",
			expectFrom:        "FROM eclipse-temurin:21-jdk-jammy",
			expectSnippet:     "COPY --from=node:20-bookworm-slim /usr/local /usr/local",
			forbidSnippet:     "apk add",
			expectRecommended: 2048, // 210 + 1280 = 1490 -> 2048 MB
		},
		{
			name:              "C/C++ GCC 13 + Angular -> Multi-Stage COPY and 1024 MB RAM",
			runtimeID:         "rt-cpp-gcc",
			webID:             "web-angular-standard",
			expectFrom:        "FROM gcc:13.2-bookworm",
			expectSnippet:     "COPY --from=node:20-bookworm-slim /usr/local /usr/local",
			forbidSnippet:     "apk add",
			expectRecommended: 1024,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := composer.Compose(ctx, "default", "IDE_PERSISTENTE", domain.StackConfig{
				Mode:          "visual",
				RuntimeID:     tc.runtimeID,
				WebPlatformID: tc.webID,
			})
			if err != nil {
				t.Fatalf("unexpected error composing: %v", err)
			}
			if !strings.Contains(plan.DockerfileContent, tc.expectFrom) {
				t.Errorf("expected Dockerfile to contain %q, got:\n%s", tc.expectFrom, plan.DockerfileContent)
			}
			if !strings.Contains(plan.DockerfileContent, tc.expectSnippet) {
				t.Errorf("expected Dockerfile to contain %q, got:\n%s", tc.expectSnippet, plan.DockerfileContent)
			}
			if tc.forbidSnippet != "" {
				lines := strings.Split(plan.DockerfileContent, "\n")
				for _, l := range lines {
					if strings.Contains(l, tc.forbidSnippet) && !strings.Contains(l, "openvscode-server") {
						t.Errorf("forbidden snippet %q found in Dockerfile line %q:\n%s", tc.forbidSnippet, l, plan.DockerfileContent)
					}
				}
			}
			if plan.RecommendedRamMB != tc.expectRecommended {
				t.Errorf("expected %d MB, got %d MB", tc.expectRecommended, plan.RecommendedRamMB)
			}
		})
	}
}

// -------------------------------------------------------------
// ADVERSARIAL CHALLENGE TEST 3: Judge Environment Constraints & Oracles
// -------------------------------------------------------------
func TestAdversarial_JudgeEnvironmentStress(t *testing.T) {
	composer := NewTemplateComposerService(makeAdversarialCatalog())
	ctx := context.Background()

	t.Run("Rejects Empty RuntimeID even if WebPlatformID is provided in JUEZ_EFIMERO", func(t *testing.T) {
		_, err := composer.Compose(ctx, "default", "JUEZ_EFIMERO", domain.StackConfig{
			RuntimeID:     "",
			WebPlatformID: "web-angular-standard",
		})
		if err == nil {
			t.Fatal("expected error when composing judge template without runtime_id, got nil")
		}
		if !strings.Contains(err.Error(), "runtime_id is required") {
			t.Errorf("expected error message 'runtime_id is required', got: %v", err)
		}
	})

	t.Run("Rejects Empty RuntimeID even if WebPlatformID is provided in JUEZ_VIRTUAL", func(t *testing.T) {
		_, err := composer.Compose(ctx, "default", "JUEZ_VIRTUAL", domain.StackConfig{
			RuntimeID:     "",
			WebPlatformID: "web-angular-standard",
		})
		if err == nil {
			t.Fatal("expected error in JUEZ_VIRTUAL without runtime_id, got nil")
		}
	})

	t.Run("Strips Web, DB, and Workspaces in Judge Environment", func(t *testing.T) {
		plan, err := composer.Compose(ctx, "default", "JUEZ_EFIMERO", domain.StackConfig{
			RuntimeID:        "rt-python-debian",
			WebPlatformID:    "web-angular-standard",
			DatabaseID:       "db-postgres",
			WorkspaceFolders: []string{"src", "eval"},
			AddonIDs:         []string{"ad-fastapi"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(plan.SuggestedName, "Evaluador: ") {
			t.Errorf("expected SuggestedName prefix 'Evaluador: ', got %q", plan.SuggestedName)
		}
		if strings.Contains(plan.SuggestedName, "Angular") {
			t.Errorf("judge SuggestedName should NOT contain web platform: %q", plan.SuggestedName)
		}
		if strings.Contains(plan.SuggestedName, "PostgreSQL") {
			t.Errorf("judge SuggestedName should NOT contain database: %q", plan.SuggestedName)
		}
		if len(plan.ServicesConfig.Services) != 0 {
			t.Errorf("judge environment must have 0 coupled database services, got %+v", plan.ServicesConfig)
		}
		if strings.Contains(plan.SetupScript, "mkdir -p") {
			t.Errorf("judge environment must not create IDE workspace folders, got %q", plan.SetupScript)
		}
		if strings.Contains(plan.DockerfileContent, "COPY --from=") || strings.Contains(plan.DockerfileContent, "@angular/cli") {
			t.Errorf("judge environment must strip web build layer snippets, got:\n%s", plan.DockerfileContent)
		}
		if plan.DefaultEntrypoint != "python3 solution.py" {
			t.Errorf("expected judge entrypoint 'python3 solution.py', got %q", plan.DefaultEntrypoint)
		}
		// RAM: 32 + 224 = 256 MB
		if plan.RecommendedRamMB != 256 {
			t.Errorf("expected 256 MB for Python Judge, got %d", plan.RecommendedRamMB)
		}
	})

	t.Run("Infer Judge Entrypoint for all language families without explicit entrypoint", func(t *testing.T) {
		planRust, err := composer.Compose(ctx, "default", "JUEZ_EFIMERO", domain.StackConfig{
			RuntimeID: "rt-rust-slim",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedRust := "rustc -O solution.rs -o solution && ./solution"
		if planRust.DefaultEntrypoint != expectedRust {
			t.Errorf("expected inferred entrypoint %q for Rust, got %q", expectedRust, planRust.DefaultEntrypoint)
		}
	})

	t.Run("Judge RAM Presets for C/C++, Python, Java, and Heavy Addons", func(t *testing.T) {
		// C/C++: 32 + 96 = 128 MB
		planCpp, _ := composer.Compose(ctx, "default", "JUEZ_EFIMERO", domain.StackConfig{RuntimeID: "rt-cpp-gcc"})
		if planCpp.RecommendedRamMB != 128 {
			t.Errorf("expected 128 MB for C/C++ Judge, got %d", planCpp.RecommendedRamMB)
		}

		// Java: 32 + 480 = 512 MB
		planJava, _ := composer.Compose(ctx, "default", "JUEZ_EFIMERO", domain.StackConfig{RuntimeID: "rt-java-21"})
		if planJava.RecommendedRamMB != 512 {
			t.Errorf("expected 512 MB for Java Judge, got %d", planJava.RecommendedRamMB)
		}

		// Python + Heavy Addon: 32 + 480 = 512 MB
		planPyHeavy, _ := composer.Compose(ctx, "default", "JUEZ_EFIMERO", domain.StackConfig{
			RuntimeID: "rt-python-debian",
			AddonIDs:  []string{"ad-heavy-datascience"},
		})
		if planPyHeavy.RecommendedRamMB != 512 {
			t.Errorf("expected 512 MB for Python Judge with Heavy Addon, got %d", planPyHeavy.RecommendedRamMB)
		}
	})
}

// -------------------------------------------------------------
// ADVERSARIAL CHALLENGE TEST 4: Ghost Git Absence & Tool Declarations
// -------------------------------------------------------------
func TestAdversarial_GhostGitAbsenceAndToolUnion(t *testing.T) {
	composer := NewTemplateComposerService(makeAdversarialCatalog())
	ctx := context.Background()

	t.Run("No git when no selected component provides git", func(t *testing.T) {
		testConfigs := []domain.StackConfig{
			{RuntimeID: "rt-python-debian"},
			{RuntimeID: "rt-python-alpine"},
			{RuntimeID: "rt-go-debian"},
			{WebPlatformID: "web-angular-standard"},
			{RuntimeID: "rt-python-debian", WebPlatformID: "web-angular-standard", DatabaseID: "db-postgres"},
		}

		for idx, cfg := range testConfigs {
			plan, err := composer.Compose(ctx, "default", "IDE_PERSISTENTE", cfg)
			if err != nil {
				t.Fatalf("case %d failed: %v", idx, err)
			}
			for _, tool := range plan.ToolsDeclared {
				if tool == "git" {
					t.Fatalf("case %d: found implicit/ghost git in ToolsDeclared: %+v", idx, plan.ToolsDeclared)
				}
			}
		}
	})

	t.Run("Git is included if and only if explicitly provided by a component", func(t *testing.T) {
		plan, err := composer.Compose(ctx, "default", "IDE_PERSISTENTE", domain.StackConfig{
			RuntimeID: "rt-python-debian",
			AddonIDs:  []string{"ad-git-explicit"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		hasGit := false
		hasGitLfs := false
		for _, tool := range plan.ToolsDeclared {
			if tool == "git" {
				hasGit = true
			}
			if tool == "git-lfs" {
				hasGitLfs = true
			}
		}
		if !hasGit || !hasGitLfs {
			t.Errorf("expected ToolsDeclared to contain git and git-lfs when explicit addon is added, got %+v", plan.ToolsDeclared)
		}
	})
}

// -------------------------------------------------------------
// ADVERSARIAL CHALLENGE TEST 5: Determinism & Hash Invariance
// -------------------------------------------------------------
func TestAdversarial_DeterministicHashInvariance(t *testing.T) {
	composer := NewTemplateComposerService(makeAdversarialCatalog())
	ctx := context.Background()

	// Addon IDs passed in reverse order must result in identical hash & order
	cfg1 := domain.StackConfig{
		RuntimeID: "rt-python-debian",
		AddonIDs:  []string{"ad-fastapi", "ad-git-explicit"},
	}
	cfg2 := domain.StackConfig{
		RuntimeID: "rt-python-debian",
		AddonIDs:  []string{"ad-git-explicit", "ad-fastapi"},
	}

	plan1, err1 := composer.Compose(ctx, "default", "IDE_PERSISTENTE", cfg1)
	plan2, err2 := composer.Compose(ctx, "default", "IDE_PERSISTENTE", cfg2)

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected compose errors: %v / %v", err1, err2)
	}

	if plan1.SignatureHash != plan2.SignatureHash {
		t.Errorf("signature hash is not invariant to addon order: %s vs %s", plan1.SignatureHash, plan2.SignatureHash)
	}
	if plan1.ResolvedImage != plan2.ResolvedImage {
		t.Errorf("resolved image tag mismatch: %s vs %s", plan1.ResolvedImage, plan2.ResolvedImage)
	}
}
