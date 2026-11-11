package docker

import "testing"

// TestPracticeEnvironmentRuntimeDirOptions fija el dueño del tmpfs: privado
// cuando el usuario es numérico y escribible por todos cuando la imagen usa un
// nombre que el daemon resuelve dentro del contenedor.
func TestPracticeEnvironmentRuntimeDirOptions(t *testing.T) {
	const base = "rw,exec,nosuid,nodev,size=16m"
	cases := []struct{ name, user, want string }{
		{"usuario del aula", "1000:1000", base + ",mode=0700,uid=1000,gid=1000"},
		{"solo uid", "1000", base + ",mode=0700,uid=1000,gid=0"},
		{"root por defecto", "", base + ",mode=0700,uid=0,gid=0"},
		{"usuario por nombre", "vscode", base + ",mode=1777"},
	}
	for _, tc := range cases {
		if got := practiceEnvironmentRuntimeDirOptions(tc.user); got != tc.want {
			t.Fatalf("%s: obtuve %q, se esperaba %q", tc.name, got, tc.want)
		}
	}
}

// TestWithPracticeEnvironmentRuntimeEnv comprueba que el aula declare el
// directorio de ejecución y que un perfil que ya lo declara conserve la última
// palabra, sin montar un tmpfs que no corresponde.
func TestWithPracticeEnvironmentRuntimeEnv(t *testing.T) {
	env, added := withPracticeEnvironmentRuntimeEnv([]string{"PATH=/usr/bin"})
	if !added {
		t.Fatal("debía agregar el directorio de ejecución")
	}
	if len(env) != 2 || env[1] != "XDG_RUNTIME_DIR=/run/solv" {
		t.Fatalf("entorno inesperado: %v", env)
	}

	declared := []string{"XDG_RUNTIME_DIR=/home/workspace/.solv-home/run"}
	env, added = withPracticeEnvironmentRuntimeEnv(declared)
	if added {
		t.Fatal("el perfil ya declaraba el directorio: no se monta otro")
	}
	if len(env) != 1 || env[0] != declared[0] {
		t.Fatalf("el perfil debía conservarse intacto: %v", env)
	}
}
