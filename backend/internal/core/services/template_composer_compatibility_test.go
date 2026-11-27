package services

import (
	"context"
	"errors"
	"testing"

	"solv-backend/internal/core/domain"
)

func compatibilityStub() *stubComponentSource {
	stub := newStubComponentSource()
	extra := []*domain.TemplateComponent{
		{
			ID: "ad-node-tools", Slug: "typescript-tools", Layer: domain.ComponentLayerAddon,
			SubCategory: "toolkit", Name: "TypeScript + Linters", IconKey: "typescript",
			CompatibleRuntimes: []string{"node", "web"},
			BuildLayerSnippet:  "RUN npm install -g typescript",
			IsActive:           true,
		},
		{
			ID: "ad-any-tool", Slug: "any-tool", Layer: domain.ComponentLayerAddon,
			SubCategory: "toolkit", Name: "Herramienta Universal", IconKey: "docker",
			CompatibleRuntimes: []string{"any"},
			IsActive:           true,
		},
		{
			ID: "ad-free-tool", Slug: "free-tool", Layer: domain.ComponentLayerAddon,
			SubCategory: "toolkit", Name: "Herramienta Sin Restricción", IconKey: "docker",
			CompatibleRuntimes: []string{},
			IsActive:           true,
		},
		{
			ID: "ad-universal-tool", Slug: "universal-tool", Layer: domain.ComponentLayerAddon,
			SubCategory: "toolkit", Name: "Herramienta Universal 2", IconKey: "docker",
			CompatibleRuntimes: []string{"universal"},
			IsActive:           true,
		},
		{
			ID: "ad-cpp-debug", Slug: "cpp-debug-valgrind", Layer: domain.ComponentLayerAddon,
			SubCategory: "toolkit", Name: "Depuración y Memoria (GDB + Valgrind)", IconKey: "cpp",
			CompatibleRuntimes: []string{"c", "cpp"},
			IsActive:           true,
		},
	}
	for _, item := range extra {
		stub.items[item.ID] = item
		stub.items[item.Slug] = item
	}
	return stub
}

func TestComposeCompatibleRuntimesValidation(t *testing.T) {
	cases := []struct {
		name    string
		env     string
		cfg     domain.StackConfig
		wantErr bool
	}{
		{
			name:    "addon incompatible con el runtime activo",
			env:     "IDE_PERSISTENTE",
			cfg:     domain.StackConfig{RuntimeID: "rt-python", AddonIDs: []string{"ad-spring-maven"}},
			wantErr: true,
		},
		{
			name: "addon sin runtimes declarados aplica a cualquiera",
			env:  "IDE_PERSISTENTE",
			cfg:  domain.StackConfig{RuntimeID: "rt-python", AddonIDs: []string{"ad-free-tool"}},
		},
		{
			name: "addon universal aplica a cualquiera",
			env:  "IDE_PERSISTENTE",
			cfg:  domain.StackConfig{RuntimeID: "rt-python", AddonIDs: []string{"ad-universal-tool"}},
		},
		{
			name: "addon any aplica a cualquiera",
			env:  "JUEZ_EFIMERO",
			cfg:  domain.StackConfig{RuntimeID: "rt-python", AddonIDs: []string{"ad-any-tool"}},
		},
		{
			name: "full-stack Python Angular con addon Node",
			env:  "IDE_PERSISTENTE",
			cfg:  domain.StackConfig{RuntimeID: "rt-python", WebPlatformID: "rt-angular", AddonIDs: []string{"ad-node-tools"}},
		},
		{
			name: "C con addon de debug C y C++",
			env:  "IDE_PERSISTENTE",
			cfg:  domain.StackConfig{RuntimeID: "rt-cpp", AddonIDs: []string{"ad-cpp-debug"}},
		},
		{
			name: "addon inexistente se omite sin error",
			env:  "IDE_PERSISTENTE",
			cfg:  domain.StackConfig{RuntimeID: "rt-python", AddonIDs: []string{"ad-inexistente"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			composer := NewTemplateComposerService(compatibilityStub())
			plan, err := composer.Compose(context.Background(), "tenant", tc.env, tc.cfg)
			if tc.wantErr {
				if !errors.Is(err, domain.ErrIncompatibleAddon) {
					t.Fatalf("esperaba ErrIncompatibleAddon, obtuve plan=%v err=%v", plan, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("no esperaba error, obtuve %v", err)
			}
			if plan == nil {
				t.Fatal("esperaba un plan compuesto")
			}
		})
	}
}

func TestEnvironmentRequestCreateRejectsIncompatibleAddon(t *testing.T) {
	compRepo := &fakeComponentRepo{comps: map[string]*domain.TemplateComponent{
		"python-312": {
			ID: "python-312", Slug: "python-312", Name: "Python 3.12", Layer: domain.ComponentLayerRuntime,
			BaseImage: "python:3.12-slim-bookworm", IconKey: "python", CompatibleRuntimes: []string{"python"},
		},
		"spring-maven": {
			ID: "spring-maven", Slug: "spring-maven", Name: "Maven + Gestión Java", Layer: domain.ComponentLayerAddon,
			IconKey: "spring", CompatibleRuntimes: []string{"java"},
		},
	}}
	svc := NewEnvironmentRequestService(&fakeEnvironmentRequestRepo{}, compRepo)
	_, err := svc.Create(context.Background(), "tenant", "teacher-a", domain.CreateEnvironmentRequestDTO{
		Description: "Necesito un entorno para practicar APIs en clase.",
		Uses:        []string{"practice"},
		StackConfig: domain.StackConfig{RuntimeID: "python-312", AddonIDs: []string{"spring-maven"}},
	})
	if !errors.Is(err, domain.ErrIncompatibleAddon) {
		t.Fatalf("esperaba ErrIncompatibleAddon, obtuve %v", err)
	}
}
