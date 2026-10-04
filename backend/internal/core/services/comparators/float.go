package comparators

import (
	"math"
	"strconv"
	"strings"
)

// Float comparison modes from D-EJ-02.
const (
	FloatAbsolute = "absolute"
	FloatRelative = "relative"
	FloatHybrid   = "hybrid"
)

// Default tolerances and admitted ranges from D-EJ-02.
const (
	DefaultEps   = 1e-6
	DefaultAtol  = 1e-9
	DefaultRtol  = 1e-6
	MinTolerance = 1e-12
	MaxTolerance = 1e-2
)

// FloatParams holds the float comparator settings. Eps is the fallback
// tolerance for modes whose own parameter was left unset.
type FloatParams struct {
	Mode string
	Eps  float64
	Atol float64
	Rtol float64
}

// DefaultFloatParams returns the spec defaults.
func DefaultFloatParams() FloatParams {
	return FloatParams{Mode: FloatHybrid, Eps: DefaultEps, Atol: DefaultAtol, Rtol: DefaultRtol}
}

// ParseFloatParams reads and validates the float comparator settings.
// Tolerances outside [1e-12, 1e-2] are rejected with CodeParamOutOfRange.
func ParseFloatParams(p map[string]any) (FloatParams, error) {
	params := DefaultFloatParams()
	if raw := paramString(p, "mode"); raw != "" {
		if raw != FloatAbsolute && raw != FloatRelative && raw != FloatHybrid {
			return params, &ComparatorError{Code: CodeParamOutOfRange, Field: "mode", Detail: raw}
		}
		params.Mode = raw
	}
	for _, field := range []struct {
		key string
		set func(float64)
	}{
		{"eps", func(v float64) { params.Eps = v }},
		{"atol", func(v float64) { params.Atol = v }},
		{"rtol", func(v float64) { params.Rtol = v }},
	} {
		if v, ok := paramFloat(p, field.key); ok {
			if v < MinTolerance || v > MaxTolerance {
				return params, &ComparatorError{Code: CodeParamOutOfRange, Field: field.key, Detail: strconv.FormatFloat(v, 'g', -1, 64)}
			}
			field.set(v)
		}
	}
	return params, nil
}

// CompareFloat compares both sides token by token. Numeric pairs follow
// |a-b| <= atol + rtol*|expected| restricted to the active mode; any pair
// with a non-numeric token or a token count mismatch is WA.
func CompareFloat(expected, actual string, params FloatParams) Result {
	want := strings.Fields(expected)
	got := strings.Fields(actual)
	if len(want) != len(got) {
		return Result{Verdict: VerdictWA, Message: "token count differs"}
	}
	for i := range want {
		a, errA := strconv.ParseFloat(want[i], 64)
		b, errB := strconv.ParseFloat(got[i], 64)
		if errA != nil || errB != nil {
			if want[i] != got[i] {
				return Result{Verdict: VerdictWA, Message: "token differs"}
			}
			continue
		}
		if !floatClose(a, b, params) {
			return Result{Verdict: VerdictWA, Message: "number differs beyond tolerance"}
		}
	}
	return Result{Verdict: VerdictAC}
}

func floatClose(expected, actual float64, params FloatParams) bool {
	diff := math.Abs(actual - expected)
	atol := params.Atol
	if atol == 0 {
		atol = params.Eps
	}
	rtol := params.Rtol
	if rtol == 0 {
		rtol = params.Eps
	}
	switch params.Mode {
	case FloatAbsolute:
		return diff <= atol
	case FloatRelative:
		if expected == 0 {
			return diff <= atol
		}
		return diff <= rtol*math.Abs(expected)
	default:
		return diff <= atol+rtol*math.Abs(expected)
	}
}
