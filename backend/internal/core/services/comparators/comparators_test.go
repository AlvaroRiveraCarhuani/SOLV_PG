package comparators

import (
	"errors"
	"testing"
)

func assertVerdict(t *testing.T, got Result, want Verdict) {
	t.Helper()
	if got.Verdict != want {
		t.Fatalf("verdict = %q, want %q (message %q)", got.Verdict, want, got.Message)
	}
}

func TestCompareDispatchesRegisteredComparators(t *testing.T) {
	for _, id := range []string{IDExact, IDUnordered, IDFloat} {
		got, err := Compare(Spec{ID: id}, "abc", "abc")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", id, err)
		}
		assertVerdict(t, got, VerdictAC)
	}
}

func TestCompareUnknownComparator(t *testing.T) {
	_, err := Compare(Spec{ID: "nope"}, "a", "a")
	var cerr *ComparatorError
	if !errors.As(err, &cerr) || cerr.Code != CodeUnknownComparator {
		t.Fatalf("expected comparator_unknown, got %v", err)
	}
}

func TestCompareCustomWithoutRunnerNeedsEvaluator(t *testing.T) {
	_, err := Compare(Spec{ID: IDCustom}, "a", "a")
	var cerr *ComparatorError
	if !errors.As(err, &cerr) || cerr.Code != CodeCheckerInvalid {
		t.Fatalf("expected checker_invalid, got %v", err)
	}
}

func TestDefaultSpecKeepsTrimSpaceBehavior(t *testing.T) {
	// Default {"id":"exact"} preserves the current runner behavior until P3
	// removes the global TrimSpace.
	got, err := Compare(DefaultSpec(), "  hola mundo  \n", "hola mundo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertVerdict(t, got, VerdictAC)

	got, err = Compare(DefaultSpec(), "a\nb", "a\nc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertVerdict(t, got, VerdictWA)
}

func TestCompareExactCases(t *testing.T) {
	cases := []struct {
		name     string
		params   map[string]any
		expected string
		actual   string
		want     Verdict
	}{
		{"trims both sides", nil, "  hi\n", "hi", VerdictAC},
		{"inner difference is WA", nil, "a b", "a  b", VerdictWA},
		{"case differs is WA", nil, "Hi", "hi", VerdictWA},
		{"empty both is AC", nil, "", "   \n", VerdictAC},
		{"trim_lines off keeps indent", nil, "  x", "x\ny", VerdictWA},
		{"trim_lines on ignores indent", map[string]any{"trim_lines": true}, "  x\n  y ", "x\ny", VerdictAC},
		{"trim_lines on still catches content", map[string]any{"trim_lines": true}, "x\nz", "x\ny", VerdictWA},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Compare(Spec{ID: IDExact, Params: tc.params}, tc.expected, tc.actual)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertVerdict(t, got, tc.want)
		})
	}
}

func TestCompareUnorderedCases(t *testing.T) {
	cases := []struct {
		name     string
		params   map[string]any
		expected string
		actual   string
		want     Verdict
	}{
		{"line permutation is AC", nil, "a\nb\nc", "c\na\nb", VerdictAC},
		{"missing line is WA", nil, "a\nb\nc", "a\nb", VerdictWA},
		{"extra line is WA", nil, "a\nb", "a\nb\nc", VerdictWA},
		{"duplicate missed by default is WA", nil, "a\na\nb", "a\nb", VerdictWA},
		{"different sets with flag is WA", map[string]any{"ignore_duplicates": true}, "a\na\nb", "a\nc\nc", VerdictWA},
		{"duplicate kept with flag same multiset is AC", map[string]any{"ignore_duplicates": true}, "a\na\nb", "b\na\na", VerdictAC},
		{"case differs is WA", nil, "Hello", "hello", VerdictWA},
		{"ignore_case is AC", map[string]any{"ignore_case": true}, "Hello\nWorld", "world\nhello", VerdictAC},
		{"tokens split ignores layout", map[string]any{"split": "tokens"}, "a b\nc", "c a  b", VerdictAC},
		{"tokens split catches value", map[string]any{"split": "tokens"}, "a b c", "a b d", VerdictWA},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Compare(Spec{ID: IDUnordered, Params: tc.params}, tc.expected, tc.actual)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertVerdict(t, got, tc.want)
		})
	}
}

func TestCompareUnorderedBadSplitIsRejected(t *testing.T) {
	_, err := Compare(Spec{ID: IDUnordered, Params: map[string]any{"split": "chars"}}, "a", "a")
	var cerr *ComparatorError
	if !errors.As(err, &cerr) || cerr.Code != CodeParamOutOfRange {
		t.Fatalf("expected comparator_param_out_of_range, got %v", err)
	}
}
