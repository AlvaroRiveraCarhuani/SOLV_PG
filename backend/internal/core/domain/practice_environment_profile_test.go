package domain

import "testing"

// El perfil del entorno se deriva del propósito del ejercicio: no es un campo
// que el cliente pueda enviar. Un propósito ausente o desconocido no puede
// abrir un examen, así que cae en el perfil de práctica.
func TestProfileForPurpose(t *testing.T) {
	cases := []struct {
		nombre   string
		purpose  string
		esperado EnvironmentProfile
	}{
		{"examen", string(ExercisePurposeExam), EnvironmentProfileExam},
		{"clase", string(ExercisePurposeClass), EnvironmentProfilePractice},
		{"vacío", "", EnvironmentProfilePractice},
		{"desconocido", "taller", EnvironmentProfilePractice},
	}

	for _, tc := range cases {
		t.Run(tc.nombre, func(t *testing.T) {
			if obtenido := ProfileForPurpose(tc.purpose); obtenido != tc.esperado {
				t.Fatalf("esperado %q, obtenido %q", tc.esperado, obtenido)
			}
		})
	}
}

func TestIsExam(t *testing.T) {
	cases := []struct {
		profile  EnvironmentProfile
		esperado bool
	}{
		{EnvironmentProfileExam, true},
		{EnvironmentProfilePractice, false},
		{"", false},
		{"otro", false},
	}

	for _, tc := range cases {
		t.Run(string(tc.profile), func(t *testing.T) {
			if obtenido := tc.profile.IsExam(); obtenido != tc.esperado {
				t.Fatalf("esperado %v, obtenido %v", tc.esperado, obtenido)
			}
		})
	}
}

// El perfil declara qué cambia por momento y no fija ningún número: la memoria
// la sigue decidiendo la política del servidor.
func TestProfileSpec(t *testing.T) {
	practica := ProfileSpecFor(EnvironmentProfilePractice)
	if !practica.IdleHibernation {
		t.Error("la práctica debe poder hibernar por inactividad")
	}
	if practica.KeystrokeCapture {
		t.Error("la práctica no debe capturar tecleo")
	}
	if practica.FreshWorkspace {
		t.Error("la práctica no debe estrenar espacio en cada apertura")
	}
	if practica.FreezeOnSubmit {
		t.Error("entregar una práctica no debe congelar el entorno")
	}

	examen := ProfileSpecFor(EnvironmentProfileExam)
	if examen.IdleHibernation {
		t.Error("el examen no debe hibernar por inactividad")
	}
	if !examen.HostPressureHibernation {
		t.Error("el examen debe poder hibernar por presión del host")
	}
	if !examen.KeystrokeCapture {
		t.Error("el examen debe capturar tecleo")
	}
	if !examen.FreshWorkspace {
		t.Error("el examen debe estrenar espacio")
	}
	if !examen.FreezeOnSubmit {
		t.Error("el examen debe congelarse al entregar")
	}
}

// Un perfil desconocido no puede tener privilegios de examen: cae en práctica.
func TestProfileSpecDesconocidoEsPractica(t *testing.T) {
	spec := ProfileSpecFor("otro")
	if spec.KeystrokeCapture || spec.FreshWorkspace || spec.FreezeOnSubmit {
		t.Fatalf("un perfil desconocido no debe comportarse como examen: %+v", spec)
	}
	if !spec.IdleHibernation {
		t.Fatalf("un perfil desconocido debe comportarse como práctica: %+v", spec)
	}
}
