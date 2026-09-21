package domain_test

import (
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

func TestValidateToolName(t *testing.T) {
	tests := []struct {
		name    string
		tool    string
		wantErr bool
	}{
		{"valid standard binary", "gcc", false},
		{"valid with hyphen and numbers", "python3.12", false},
		{"valid with plus", "g++", false},
		{"valid with underscore", "clang_format", false},
		{"valid complex tool", "x86_64-linux-gnu-gcc-12", false},
		{"empty name", "", true},
		{"whitespace only", "   ", true},
		{"shell injection semicolon", "gcc;rm -rf /", true},
		{"shell injection pipe", "gcc|cat /etc/passwd", true},
		{"shell injection backtick", "gcc`whoami`", true},
		{"shell injection redirection", "gcc>out", true},
		{"contains spaces", "gcc -v", true},
		{"tool name too long", string(make([]byte, 65)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateToolName(tt.tool)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToolName(%q) error = %v, wantErr %v", tt.tool, err, tt.wantErr)
			}
		})
	}
}

func TestEnvTestJob_IsTerminal(t *testing.T) {
	now := time.Now()
	job := &domain.EnvTestJob{
		ID:        "job-1",
		Status:    domain.EnvTestStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if job.IsTerminal() {
		t.Errorf("expected pending job to NOT be terminal")
	}

	job.Status = domain.EnvTestStatusPulling
	if job.IsTerminal() {
		t.Errorf("expected pulling job to NOT be terminal")
	}

	job.Status = domain.EnvTestStatusTesting
	if job.IsTerminal() {
		t.Errorf("expected testing job to NOT be terminal")
	}

	job.Status = domain.EnvTestStatusSuccess
	if !job.IsTerminal() {
		t.Errorf("expected success job to be terminal")
	}

	job.Status = domain.EnvTestStatusFailed
	if !job.IsTerminal() {
		t.Errorf("expected failed job to be terminal")
	}

	job.Status = domain.EnvTestStatusCanceled
	if !job.IsTerminal() {
		t.Errorf("expected canceled job to be terminal")
	}
}
