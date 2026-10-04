package comparators

import (
	"errors"
	"testing"
)

func TestParseCheckerOutput(t *testing.T) {
	cases := []struct {
		name   string
		stdout string
		want   Verdict
	}{
		{"AC alone", "AC", VerdictAC},
		{"AC with message", "AC\nall good", VerdictAC},
		{"WA with message", "WA\nexpected 3 got 4", VerdictWA},
		{"lowercase accepted", "ac", VerdictAC},
		{"empty is WA", "", VerdictWA},
		{"unknown token is WA", "MAYBE", VerdictWA},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertVerdict(t, ParseCheckerOutput(tc.stdout), tc.want)
		})
	}
	if got := ParseCheckerOutput("WA\nexpected 3"); got.Message == "" {
		t.Fatalf("checker message must be preserved")
	}
}

func TestEvaluateCheckerOutcome(t *testing.T) {
	cases := []struct {
		name    string
		outcome CheckerOutcome
		want    Verdict
	}{
		{"clean AC", CheckerOutcome{Stdout: "AC", ExitCode: 0}, VerdictAC},
		{"clean WA", CheckerOutcome{Stdout: "WA\nbad", ExitCode: 0}, VerdictWA},
		{"timeout is VE", CheckerOutcome{Stdout: "AC", ExitCode: 0, TimedOut: true}, VerdictVE},
		{"crash is VE", CheckerOutcome{Stdout: "AC", ExitCode: 0, Crashed: true}, VerdictVE},
		{"nonzero exit is VE", CheckerOutcome{Stdout: "WA", ExitCode: 1}, VerdictVE},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertVerdict(t, EvaluateCheckerOutcome(tc.outcome), tc.want)
		})
	}
}

func TestCheckerInvocationIsReadOnly(t *testing.T) {
	inv := NewCheckerInvocation("/in/input.txt", "/in/expected.txt", "/in/output.txt", 2000)
	if !inv.ReadOnly || len(inv.ReadOnlyPaths) != 3 {
		t.Fatalf("checker paths must be read-only: %+v", inv)
	}
	if _, err := ParseCheckerSpec(map[string]any{}); err == nil {
		t.Fatalf("checker without command must be rejected")
	} else {
		var cerr *ComparatorError
		if !errors.As(err, &cerr) || cerr.Code != CodeCheckerInvalid {
			t.Fatalf("expected checker_invalid, got %v", err)
		}
	}
	if _, err := ParseCheckerSpec(map[string]any{
		"command": []any{"python3", "/checker.py"}, "timeout_ms": float64(2000),
		"read_only_paths": []any{"/in/input.txt"},
	}); err != nil {
		t.Fatalf("valid checker spec rejected: %v", err)
	}
}
