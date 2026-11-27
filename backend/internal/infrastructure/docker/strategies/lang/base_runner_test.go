package lang

import (
	"os"
	"path/filepath"
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

func TestCreateEvalTempDir_UsesEnvAndFallback(t *testing.T) {
	customBase := filepath.Join(t.TempDir(), "nested-eval-tmp")
	t.Setenv(evalTempDirEnv, customBase)

	dir, err := createEvalTempDir("solv-eval-*")
	if err != nil {
		t.Fatalf("createEvalTempDir with env failed: %v", err)
	}
	if filepath.Dir(dir) != customBase {
		_ = os.RemoveAll(dir)
		t.Fatalf("expected parent dir %q, got %q", customBase, filepath.Dir(dir))
	}
	if err := os.WriteFile(filepath.Join(dir, "solution.py"), []byte("print(1)\n"), 0644); err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("failed to write test file inside eval dir: %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	t.Setenv(evalTempDirEnv, "")
	fallbackDir, err := createEvalTempDir("solv-eval-*")
	if err != nil {
		t.Fatalf("createEvalTempDir fallback failed: %v", err)
	}
	defer func() { _ = os.RemoveAll(fallbackDir) }()

	if filepath.Dir(fallbackDir) != filepath.Clean(os.TempDir()) {
		t.Errorf("expected fallback parent %q, got %q", filepath.Clean(os.TempDir()), filepath.Dir(fallbackDir))
	}
}
