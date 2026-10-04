package lang

import (
	"strings"
	"testing"
	"time"

	"solv-backend/internal/core/domain"
)

func TestResolveCrashVerdict(t *testing.T) {
	cases := []struct {
		name      string
		exitCode  int64
		oomKilled bool
		want      domain.Verdict
	}{
		{"oom kill por cgroups es MLE", 137, true, domain.VerdictMLE},
		{"salida 137 sin oom es RE", 137, false, domain.VerdictRE},
		{"salida distinta de cero es RE", 1, false, domain.VerdictRE},
		{"oom con otra salida sigue MLE", 9, true, domain.VerdictMLE},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveCrashVerdict(tt.exitCode, tt.oomKilled); got != tt.want {
				t.Errorf("resolveCrashVerdict(%d, %v) = %s, want %s", tt.exitCode, tt.oomKilled, got, tt.want)
			}
		})
	}
}

func TestSanitizeBuildStderr(t *testing.T) {
	raw := "/tmp/solv-eval-abc123/solution.cpp:5: error: expected ';' before '}' token\n/tmp/solv-eval-abc123/solution.cpp:9: warning: unused"
	got := sanitizeBuildStderr(raw, "/tmp/solv-eval-abc123")
	if strings.Contains(got, "/tmp/solv-eval-abc123") {
		t.Errorf("sanitized output still has host temp path: %q", got)
	}
	if !strings.Contains(got, "error") {
		t.Errorf("sanitized output lost the compiler message: %q", got)
	}

	if got := sanitizeBuildStderr("   ", "/tmp/x"); got != "" {
		t.Errorf("blank input should sanitize to empty, got %q", got)
	}

	long := strings.Repeat("e", maxBuildStderr+500)
	if got := sanitizeBuildStderr(long, ""); len(got) > maxBuildStderr {
		t.Errorf("sanitized output exceeds cap: %d > %d", len(got), maxBuildStderr)
	}
}

func TestCheckerResultToRunResult(t *testing.T) {
	ve := checkerResultToRunResult(checkerOutcome{verdict: domain.VerdictVE, message: "checker crashed"}, time.Second)
	if ve.Verdict != domain.VerdictVE {
		t.Errorf("checker crash verdict = %s, want VE", ve.Verdict)
	}
	ac := checkerResultToRunResult(checkerOutcome{verdict: domain.VerdictAC}, time.Second)
	if ac.Verdict != domain.VerdictAC {
		t.Errorf("checker pass verdict = %s, want AC", ac.Verdict)
	}
	wa := checkerResultToRunResult(checkerOutcome{verdict: domain.VerdictWA, message: "diff"}, time.Second)
	if wa.Verdict != domain.VerdictWA {
		t.Errorf("checker mismatch verdict = %s, want WA", wa.Verdict)
	}
}
