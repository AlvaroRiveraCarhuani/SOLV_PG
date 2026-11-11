package domain

import "testing"

func TestWorkspaceDependencyTreeValidate(t *testing.T) {
	valid := WorkspaceDependencyTree{
		RootPath: "/var/lib/solv/deps/abc",
		Key:      "clave",
		Paths:    []string{"node_modules", ".venv"},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("un árbol válido no debía fallar: %v", err)
	}

	cases := map[string]WorkspaceDependencyTree{
		"raíz relativa":       {RootPath: "debs/abc", Key: "k", Paths: []string{"node_modules"}},
		"raíz con ..":         {RootPath: "/var/lib/../etc", Key: "k", Paths: []string{"node_modules"}},
		"sin clave":           {RootPath: "/var/lib/solv/deps/abc", Paths: []string{"node_modules"}},
		"sin rutas":           {RootPath: "/var/lib/solv/deps/abc", Key: "k"},
		"ruta que sale":       {RootPath: "/var/lib/solv/deps/abc", Key: "k", Paths: []string{"../fuera"}},
		"ruta absoluta":       {RootPath: "/var/lib/solv/deps/abc", Key: "k", Paths: []string{"/etc"}},
		"diario del volumen":  {RootPath: "/var/lib/solv/deps/abc", Key: "k", Paths: []string{".solv_workspace"}},
		"ruta sin normalizar": {RootPath: "/var/lib/solv/deps/abc", Key: "k", Paths: []string{"a/./b"}},
	}
	for name, tree := range cases {
		if err := tree.Validate(); err == nil {
			t.Fatalf("%s debía rechazarse", name)
		}
	}
}
