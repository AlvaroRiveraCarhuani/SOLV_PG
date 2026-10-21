package domain_test

import (
	"testing"

	"solv-backend/internal/core/domain"
)

func TestTemplateSupportsEnvironment(t *testing.T) {
	tests := []struct {
		name         string
		templateEnv  string
		requestedEnv string
		expected     bool
	}{
		{
			name:         "dual template supports persistent ide",
			templateEnv:  domain.TargetEnvironmentAmbos,
			requestedEnv: domain.TargetEnvironmentIDEPersistente,
			expected:     true,
		},
		{
			name:         "dual template supports ephemeral judge",
			templateEnv:  domain.TargetEnvironmentAmbos,
			requestedEnv: domain.TargetEnvironmentJuezEfimero,
			expected:     true,
		},
		{
			name:         "dual template rejects unknown environment",
			templateEnv:  domain.TargetEnvironmentAmbos,
			requestedEnv: "CUSTOM_UNKNOWN",
			expected:     false,
		},
		{
			name:         "ide template matches persistent ide",
			templateEnv:  domain.TargetEnvironmentIDEPersistente,
			requestedEnv: domain.TargetEnvironmentIDEPersistente,
			expected:     true,
		},
		{
			name:         "ide template rejects ephemeral judge",
			templateEnv:  domain.TargetEnvironmentIDEPersistente,
			requestedEnv: domain.TargetEnvironmentJuezEfimero,
			expected:     false,
		},
		{
			name:         "judge template matches ephemeral judge",
			templateEnv:  domain.TargetEnvironmentJuezEfimero,
			requestedEnv: domain.TargetEnvironmentJuezEfimero,
			expected:     true,
		},
		{
			name:         "judge template rejects persistent ide",
			templateEnv:  domain.TargetEnvironmentJuezEfimero,
			requestedEnv: domain.TargetEnvironmentIDEPersistente,
			expected:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.TemplateSupportsEnvironment(tc.templateEnv, tc.requestedEnv)
			if got != tc.expected {
				t.Fatalf("TemplateSupportsEnvironment(%q, %q) = %v; want %v", tc.templateEnv, tc.requestedEnv, got, tc.expected)
			}
		})
	}
}

func TestResolveJudgeRamMB(t *testing.T) {
	tests := []struct {
		name      string
		baseRamMB int
		expected  int
	}{
		{
			name:      "below 1024 returns standard 128 MB judge limit",
			baseRamMB: 512,
			expected:  128,
		},
		{
			name:      "exactly 1024 returns high 256 MB judge limit",
			baseRamMB: 1024,
			expected:  256,
		},
		{
			name:      "large 2048 returns high 256 MB judge limit",
			baseRamMB: 2048,
			expected:  256,
		},
		{
			name:      "small 256 returns standard 128 MB judge limit",
			baseRamMB: 256,
			expected:  128,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.ResolveJudgeRamMB(tc.baseRamMB)
			if got != tc.expected {
				t.Fatalf("ResolveJudgeRamMB(%d) = %d; want %d", tc.baseRamMB, got, tc.expected)
			}
		})
	}
}
