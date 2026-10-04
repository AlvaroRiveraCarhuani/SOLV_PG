package comparators

import "strings"

// ExactParams holds the exact comparator flags.
type ExactParams struct {
	TrimLines bool
}

// ParseExactParams reads the exact comparator flags.
func ParseExactParams(p map[string]any) (ExactParams, error) {
	return ExactParams{TrimLines: paramBool(p, "trim_lines")}, nil
}

// CompareExact matches texts exactly after trimming both sides. With
// TrimLines every line is trimmed before the comparison.
func CompareExact(expected, actual string, params ExactParams) Result {
	if params.TrimLines {
		expected = trimLines(expected)
		actual = trimLines(actual)
	}
	if strings.TrimSpace(expected) == strings.TrimSpace(actual) {
		return Result{Verdict: VerdictAC}
	}
	return Result{Verdict: VerdictWA, Message: "output differs"}
}

func trimLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return strings.Join(lines, "\n")
}
