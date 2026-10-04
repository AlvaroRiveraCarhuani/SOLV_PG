package services

import (
	"testing"

	"solv-backend/internal/core/domain"
)

func TestCanonicalLanguage(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"alias cpp con simbolos", "c++", "cpp"},
		{"alias cpp en mayusculas", "C++", "cpp"},
		{"alias csharp con simbolo", "c#", "csharp"},
		{"alias csharp corto", "cs", "csharp"},
		{"canonico cpp", "cpp", "cpp"},
		{"canonico python", "python", "python"},
		{"canonico c", "c", "c"},
		{"canonico java", "java", "java"},
		{"canonico javascript", "javascript", "javascript"},
		{"canonico csharp", "csharp", "csharp"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanonicalLanguage(tt.input); got != tt.want {
				t.Errorf("CanonicalLanguage(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestP95OverAC(t *testing.T) {
	metrics := []domain.RunMetric{
		{Language: "cpp", DurationMS: 10, Verdict: domain.VerdictAC},
		{Language: "cpp", DurationMS: 20, Verdict: domain.VerdictAC},
		{Language: "cpp", DurationMS: 30, Verdict: domain.VerdictAC},
		{Language: "cpp", DurationMS: 40, Verdict: domain.VerdictAC},
		{Language: "cpp", DurationMS: 5000, Verdict: domain.VerdictTLE},
	}
	if got := P95OverAC(metrics); got != 40 {
		t.Errorf("P95OverAC = %d, want 40 (TLE excluded, nearest-rank over 4 AC samples)", got)
	}
}

func TestP95OverAC_Empty(t *testing.T) {
	if got := P95OverAC(nil); got != 0 {
		t.Errorf("P95OverAC(nil) = %d, want 0", got)
	}
	noAC := []domain.RunMetric{{Language: "cpp", DurationMS: 99, Verdict: domain.VerdictWA}}
	if got := P95OverAC(noAC); got != 0 {
		t.Errorf("P95OverAC without AC = %d, want 0", got)
	}
}
