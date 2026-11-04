package docker

import "testing"

func TestPracticeEnvironmentVolumeBind(t *testing.T) {
	tests := []struct {
		name     string
		readOnly bool
		want     string
	}{
		{name: "trabajo normal", readOnly: false, want: "solv_workspace_s1_sub1:/home/workspace:rw"},
		{name: "examen cerrado", readOnly: true, want: "solv_workspace_s1_sub1:/home/workspace:ro"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := environmentVolumeBind("solv_workspace_s1_sub1", tt.readOnly); got != tt.want {
				t.Fatalf("montaje inesperado: got %q want %q", got, tt.want)
			}
		})
	}
}
