package docker

import "testing"

// PostgreSQL necesita SIGINT (apagado fast). Con la señal por defecto el stop espera a
// los clientes conectados y termina en SIGKILL, que es lo que se midió en el nodo.
func TestManagedSatelliteStopSignal(t *testing.T) {
	cases := map[string]string{
		"postgres": "SIGINT",
		"mysql":    "",
		"mariadb":  "",
		"mongodb":  "",
		"redis":    "",
	}
	for engine, want := range cases {
		if got := managedSatelliteStopSignal(engine); got != want {
			t.Errorf("managedSatelliteStopSignal(%q) = %q, esperado %q", engine, got, want)
		}
	}
}
