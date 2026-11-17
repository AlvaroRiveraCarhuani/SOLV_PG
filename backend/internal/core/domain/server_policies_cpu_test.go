package domain

import (
	"testing"
)

func TestServerPolicies_EffectivePracticeCPUMillis(t *testing.T) {
	tests := []struct {
		name     string
		policies ServerPolicies
		expected int64
	}{
		{
			name:     "por defecto toma el valor configurado",
			policies: ServerPolicies{PracticeCPUMillis: 1000},
			expected: 1000,
		},
		{
			name:     "valor 500 milisegundos de cpu",
			policies: ServerPolicies{PracticeCPUMillis: 500},
			expected: 500,
		},
		{
			name:     "valor 0 significa sin limite",
			policies: ServerPolicies{PracticeCPUMillis: 0},
			expected: 0,
		},
		{
			name:     "valor negativo cae a sin limite",
			policies: ServerPolicies{PracticeCPUMillis: -100},
			expected: 0,
		},
		{
			name:     "politica por defecto DefaultServerPolicies tiene 1000",
			policies: DefaultServerPolicies(),
			expected: 1000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.policies.EffectivePracticeCPUMillis()
			if got != tc.expected {
				t.Fatalf("EffectivePracticeCPUMillis() = %d; esperaba %d", got, tc.expected)
			}
		})
	}
}

func TestServerPolicies_AllowedPracticeCPUMillis(t *testing.T) {
	expected := []int{0, 500, 1000, 1500, 2000}
	if len(AllowedPracticeCPUMillis) != len(expected) {
		t.Fatalf("AllowedPracticeCPUMillis largo = %d; esperaba %d", len(AllowedPracticeCPUMillis), len(expected))
	}
	for i, v := range expected {
		if AllowedPracticeCPUMillis[i] != v {
			t.Errorf("AllowedPracticeCPUMillis[%d] = %d; esperaba %d", i, AllowedPracticeCPUMillis[i], v)
		}
	}
}
