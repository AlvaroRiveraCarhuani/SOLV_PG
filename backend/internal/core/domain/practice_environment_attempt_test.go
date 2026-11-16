package domain

import "testing"

func ptrString(v string) *string { return &v }

// El recuperatorio suma uno al intento cerrado. Un entorno sin intento (práctica,
// o fila anterior a la separación) empieza en 1: el intento 0 nunca es examen.
func TestNextExamAttempt(t *testing.T) {
	cases := map[int]int{
		0: 1,
		1: 2,
		2: 3,
	}
	for current, want := range cases {
		if got := NextExamAttempt(current); got != want {
			t.Errorf("desde el intento %d se esperaba %d, se obtuvo %d", current, want, got)
		}
	}
}

// La práctica conserva su volumen de siempre. El examen estrena uno por intento:
// el volumen del intento anterior queda intacto y nunca se reutiliza.
func TestPracticeEnvironmentAttemptVolumeName(t *testing.T) {
	exercise := ptrString("ex-1")

	cases := []struct {
		nombre   string
		attempt  int
		esperado string
	}{
		{"práctica", 0, "solv_workspace_student-1_subject-1_ex-1"},
		{"intento 1", 1, "solv_workspace_student-1_subject-1_ex-1_exam_1"},
		{"intento 2", 2, "solv_workspace_student-1_subject-1_ex-1_exam_2"},
	}

	for _, tc := range cases {
		t.Run(tc.nombre, func(t *testing.T) {
			if obtenido := PracticeEnvironmentAttemptVolumeName("student-1", "subject-1", exercise, tc.attempt); obtenido != tc.esperado {
				t.Fatalf("esperado %q, obtenido %q", tc.esperado, obtenido)
			}
		})
	}
}

// Sin práctica no hay volumen, tenga el intento que tenga.
func TestPracticeEnvironmentAttemptVolumeNameSinPractica(t *testing.T) {
	if obtenido := PracticeEnvironmentAttemptVolumeName("student-1", "subject-1", nil, 1); obtenido != "" {
		t.Fatalf("sin práctica el volumen debe ser vacío, vino %q", obtenido)
	}
}

func TestVolumeNameUsaElIntento(t *testing.T) {
	environment := &PracticeEnvironment{
		StudentID:   "student-1",
		SubjectID:   "subject-1",
		ExerciseID:  ptrString("ex-1"),
		ExamAttempt: 2,
	}
	if obtenido := environment.VolumeName(); obtenido != "solv_workspace_student-1_subject-1_ex-1_exam_2" {
		t.Fatalf("el volumen del entorno debe llevar el intento, vino %q", obtenido)
	}
}
