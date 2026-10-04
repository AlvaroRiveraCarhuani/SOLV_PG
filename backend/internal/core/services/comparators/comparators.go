package comparators

import "fmt"

// Verdicts emitted by comparators. VE belongs to the custom checker contract
// (checker crash or timeout); the remaining verdicts stay AC/WA.
type Verdict string

const (
	VerdictAC Verdict = "AC"
	VerdictWA Verdict = "WA"
	VerdictVE Verdict = "VE"
)

// Comparator identifiers from D-EJ-02.
const (
	IDExact     = "exact"
	IDUnordered = "unordered"
	IDFloat     = "float"
	IDCustom    = "custom"
)

// Machine error codes for comparator failures.
const (
	CodeUnknownComparator = "comparator_unknown"
	CodeParamOutOfRange   = "comparator_param_out_of_range"
	CodeCheckerInvalid    = "checker_invalid"
)

// Result is the outcome of a single comparison.
type Result struct {
	Verdict Verdict
	Message string
}

// Spec identifies a comparator plus its parameters.
type Spec struct {
	ID     string
	Params map[string]any
}

// DefaultSpec preserves the current runner behavior (exact match with
// TrimSpace) until P3 removes the global TrimSpace.
func DefaultSpec() Spec { return Spec{ID: IDExact} }

// ComparatorError carries a stable machine code plus the affected field.
type ComparatorError struct {
	Code   string
	Field  string
	Detail string
}

func (e *ComparatorError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Field, e.Detail)
	}
	return fmt.Sprintf("%s (%s)", e.Code, e.Detail)
}

// Compare dispatches to the comparator named by spec. Custom has no local
// execution path (the teacher checker runs in a sidecar): use
// EvaluateCheckerOutcome with the sidecar result instead.
func Compare(spec Spec, expected, actual string) (Result, error) {
	switch spec.ID {
	case IDExact:
		params, err := ParseExactParams(spec.Params)
		if err != nil {
			return Result{}, err
		}
		return CompareExact(expected, actual, params), nil
	case IDUnordered:
		params, err := ParseUnorderedParams(spec.Params)
		if err != nil {
			return Result{}, err
		}
		return CompareUnordered(expected, actual, params), nil
	case IDFloat:
		params, err := ParseFloatParams(spec.Params)
		if err != nil {
			return Result{}, err
		}
		return CompareFloat(expected, actual, params), nil
	case IDCustom:
		return Result{}, &ComparatorError{Code: CodeCheckerInvalid, Detail: "custom needs EvaluateCheckerOutcome with the sidecar result"}
	default:
		return Result{}, &ComparatorError{Code: CodeUnknownComparator, Field: "id", Detail: spec.ID}
	}
}

func paramBool(p map[string]any, key string) bool {
	v, ok := p[key].(bool)
	return ok && v
}

func paramString(p map[string]any, key string) string {
	v, _ := p[key].(string)
	return v
}

func paramFloat(p map[string]any, key string) (float64, bool) {
	switch v := p[key].(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}

func paramStrings(p map[string]any, key string) ([]string, bool) {
	if s, ok := p[key].([]string); ok {
		return s, true
	}
	raw, ok := p[key].([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}
