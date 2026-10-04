package comparators

import (
	"errors"
	"testing"
)

func floatSpec(params map[string]any) Spec { return Spec{ID: IDFloat, Params: params} }

func TestCompareFloatModes(t *testing.T) {
	cases := []struct {
		name     string
		params   map[string]any
		expected string
		actual   string
		want     Verdict
	}{
		{"tiny diff AC in absolute", map[string]any{"mode": "absolute"}, "0.3", "0.30000000004", VerdictAC},
		{"small diff WA in absolute", map[string]any{"mode": "absolute"}, "2.0", "2.000002", VerdictWA},
		{"small diff AC in relative", map[string]any{"mode": "relative"}, "2.0", "2.000002", VerdictAC},
		{"small diff AC in hybrid", map[string]any{"mode": "hybrid"}, "2.0", "2.000002", VerdictAC},
		{"default mode is hybrid", nil, "2.0", "2.000002", VerdictAC},
		{"non numeric token is WA", nil, "abc", "abd", VerdictWA},
		{"mixed token mismatch is WA", nil, "1.0 ok", "1.0 fail", VerdictWA},
		{"token count differs is WA", nil, "1.0 2.0", "1.0", VerdictWA},
		{"multiline numeric is AC", nil, "1.0\n2.0", "1.0  2.0", VerdictAC},
		{"large diff is WA everywhere", nil, "1.0", "1.5", VerdictWA},
		{"zero expected falls back to atol", map[string]any{"mode": "relative"}, "0.0", "1e-10", VerdictAC},
		{"bad mode is rejected", map[string]any{"mode": "quadratic"}, "1.0", "1.0", VerdictWA},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "bad mode is rejected" {
				_, err := Compare(floatSpec(tc.params), tc.expected, tc.actual)
				if err == nil {
					t.Fatalf("expected param error, got none")
				}
				return
			}
			got, err := Compare(floatSpec(tc.params), tc.expected, tc.actual)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertVerdict(t, got, tc.want)
		})
	}
}

func TestFloatDefaultsAndRanges(t *testing.T) {
	d := DefaultFloatParams()
	if d.Mode != FloatHybrid || d.Eps != 1e-6 || d.Atol != 1e-9 || d.Rtol != 1e-6 {
		t.Fatalf("unexpected defaults: %+v", d)
	}
	for _, tc := range []struct {
		name   string
		params map[string]any
		field  string
	}{
		{"atol too small", map[string]any{"atol": 1e-13}, "atol"},
		{"atol too big", map[string]any{"atol": 1.0}, "atol"},
		{"rtol too small", map[string]any{"rtol": 1e-13}, "rtol"},
		{"eps too big", map[string]any{"eps": 0.5}, "eps"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compare(floatSpec(tc.params), "1.0", "1.0")
			var cerr *ComparatorError
			if !errors.As(err, &cerr) || cerr.Code != CodeParamOutOfRange || cerr.Field != tc.field {
				t.Fatalf("expected out-of-range on %s, got %v", tc.field, err)
			}
		})
	}
	if _, err := Compare(floatSpec(map[string]any{"atol": 1e-12, "rtol": 1e-2}), "1.0", "1.0"); err != nil {
		t.Fatalf("boundary values must be accepted, got %v", err)
	}
}
