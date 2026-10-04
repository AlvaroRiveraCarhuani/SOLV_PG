package comparators

import "testing"

// Frozen fixtures from spec §8 (D-EJ-08). These cases pin the observable
// behavior of the comparators and must not change without a spec update.
func TestFrozenFloatSmallEpsilon(t *testing.T) {
	// 0.30000000004 vs 0.3 = AC in all three float modes.
	for _, mode := range []string{FloatAbsolute, FloatRelative, FloatHybrid} {
		got, err := Compare(Spec{ID: IDFloat, Params: map[string]any{"mode": mode}}, "0.3", "0.30000000004")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", mode, err)
		}
		assertVerdict(t, got, VerdictAC)
	}
}

func TestFrozenFloatTwoMicro(t *testing.T) {
	// 2.000002 vs 2.0 = WA in absolute, AC in relative and hybrid.
	got, err := Compare(Spec{ID: IDFloat, Params: map[string]any{"mode": "absolute"}}, "2.0", "2.000002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertVerdict(t, got, VerdictWA)
	for _, mode := range []string{FloatRelative, FloatHybrid} {
		got, err := Compare(Spec{ID: IDFloat, Params: map[string]any{"mode": mode}}, "2.0", "2.000002")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", mode, err)
		}
		assertVerdict(t, got, VerdictAC)
	}
}

func TestFrozenUnorderedPermutation(t *testing.T) {
	// Line permutation = AC in unordered.
	got, err := Compare(Spec{ID: IDUnordered}, "alpha\nbeta\ngamma", "gamma\nalpha\nbeta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertVerdict(t, got, VerdictAC)
}

func TestFrozenUnorderedDuplicateFlag(t *testing.T) {
	// Same permutation minus one duplicate: WA with ignore_duplicates=false,
	// AC with ignore_duplicates=true.
	got, err := Compare(Spec{ID: IDUnordered}, "x\nx\ny", "x\ny")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertVerdict(t, got, VerdictWA)
	got, err = Compare(Spec{ID: IDUnordered, Params: map[string]any{"ignore_duplicates": true}}, "x\nx\ny", "x\ny")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertVerdict(t, got, VerdictAC)
}
