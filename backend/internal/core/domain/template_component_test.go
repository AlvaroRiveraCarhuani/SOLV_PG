package domain

import "testing"

func TestValidatePackageNames(t *testing.T) {
	validCases := [][]string{
		{"fastapi", "uvicorn"},
		{"@angular/cli", "typescript", "vite@5.4.0"},
		{"build-essential", "libpq-dev", "python3.12-venv"},
		{"django>=5.0"},
	}

	for _, pkgs := range validCases {
		if err := ValidatePackageNames(pkgs); err != nil {
			t.Fatalf("expected valid packages %v, got error: %v", pkgs, err)
		}
	}

	invalidCases := [][]string{
		{"flask; rm -rf /"},
		{"numpy && curl evil.com"},
		{"pandas | sh"},
		{"$(whoami)"},
		{"`id`"},
		{"pkg with spaces"},
		{""},
	}

	for _, pkgs := range invalidCases {
		if err := ValidatePackageNames(pkgs); err == nil {
			t.Fatalf("expected error for invalid package list %v, got nil", pkgs)
		}
	}
}

func TestSynthesizeAssistedBuildLayer(t *testing.T) {
	layer, err := SynthesizeAssistedBuildLayer(InstallModePip, []string{"fastapi", "uvicorn"}, "")
	if err != nil {
		t.Fatalf("unexpected error for pip: %v", err)
	}
	if layer != "RUN pip install --no-cache-dir fastapi uvicorn" {
		t.Fatalf("unexpected pip layer: %s", layer)
	}

	npmLayer, err := SynthesizeAssistedBuildLayer(InstallModeNpm, []string{"@angular/cli"}, "")
	if err != nil {
		t.Fatalf("unexpected error for npm: %v", err)
	}
	if npmLayer != "RUN npm install -g --no-audit --no-fund @angular/cli" {
		t.Fatalf("unexpected npm layer: %s", npmLayer)
	}

	aptLayer, err := SynthesizeAssistedBuildLayer(InstallModeApt, []string{"default-mysql-client"}, "")
	if err != nil {
		t.Fatalf("unexpected error for apt: %v", err)
	}
	expectedApt := "RUN apt-get update && apt-get install -y --no-install-recommends default-mysql-client && rm -rf /var/lib/apt/lists/*"
	if aptLayer != expectedApt {
		t.Fatalf("unexpected apt layer: %s", aptLayer)
	}
}

func TestComputeStackSignatureDeterministic(t *testing.T) {
	sig1 := ComputeStackSignature("python:3.12-slim-bookworm", []string{
		"RUN pip install --no-cache-dir fastapi uvicorn",
		"RUN npm install -g --no-audit --no-fund @angular/cli",
	})
	sig2 := ComputeStackSignature("python:3.12-slim-bookworm", []string{
		"RUN pip install --no-cache-dir fastapi uvicorn",
		"RUN npm install -g --no-audit --no-fund @angular/cli",
	})
	if sig1 != sig2 {
		t.Fatalf("expected deterministic signature, got %s vs %s", sig1, sig2)
	}
	if len(sig1) != 12 {
		t.Fatalf("expected 12-char hex signature, got %q (len=%d)", sig1, len(sig1))
	}

	tag := FormatBuiltStackImageTag(sig1)
	if tag != "solv-local/stack:"+sig1 {
		t.Fatalf("unexpected built image tag: %s", tag)
	}
}
