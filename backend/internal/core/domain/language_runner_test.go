package domain

import (
	"errors"
	"testing"
)

func TestCanonicalJudgeLanguage(t *testing.T) {
	cases := map[string]string{
		"Python3":  "python",
		"py":       "python",
		"JS":       "javascript",
		"node":     "javascript",
		"c++":      "cpp",
		"c#":       "csharp",
		"cs":       "csharp",
		"Golang":   "go",
		"rs":       "rust",
		"java":     "java",
		"julia":    "julia",
		"  Ruby  ": "ruby",
	}
	for input, want := range cases {
		if got := CanonicalJudgeLanguage(input); got != want {
			t.Errorf("CanonicalJudgeLanguage(%q) = %q, quería %q", input, got, want)
		}
	}
}

func TestNormalizeLanguageRunners_CanonicalizaYOrdena(t *testing.T) {
	runners, err := NormalizeLanguageRunners([]LanguageRunner{
		{Language: "c++", Entrypoint: "g++ -O2 solution.cpp -o solution && ./solution"},
		{Language: "Python3", Entrypoint: "python3 solution.py"},
		{Language: "js", Entrypoint: "node solution.js"},
	})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	want := []string{"python", "javascript", "cpp"}
	if len(runners) != len(want) {
		t.Fatalf("esperaba %d ejecutores, obtuve %d", len(want), len(runners))
	}
	for i, language := range want {
		if runners[i].Language != language {
			t.Errorf("posición %d: esperaba %q, obtuve %q", i, language, runners[i].Language)
		}
	}
}

func TestNormalizeLanguageRunners_Rechazos(t *testing.T) {
	t.Run("lenguaje repetido tras canonizar", func(t *testing.T) {
		_, err := NormalizeLanguageRunners([]LanguageRunner{
			{Language: "cpp", Entrypoint: "g++ solution.cpp -o solution && ./solution"},
			{Language: "c++", Entrypoint: "g++ solution.cpp -o solution && ./solution"},
		})
		if !errors.Is(err, ErrLanguageRunnerDuplicate) {
			t.Fatalf("esperaba ErrLanguageRunnerDuplicate, obtuve %v", err)
		}
	})

	t.Run("sin entrypoint", func(t *testing.T) {
		_, err := NormalizeLanguageRunners([]LanguageRunner{{Language: "python", Entrypoint: "  "}})
		if !errors.Is(err, ErrLanguageRunnerEntrypointRequired) {
			t.Fatalf("esperaba ErrLanguageRunnerEntrypointRequired, obtuve %v", err)
		}
	})

	t.Run("lenguaje fuera del catálogo sin imagen", func(t *testing.T) {
		_, err := NormalizeLanguageRunners([]LanguageRunner{{Language: "julia", Entrypoint: "julia solution.jl"}})
		if !errors.Is(err, ErrLanguageRunnerImageRequired) {
			t.Fatalf("esperaba ErrLanguageRunnerImageRequired, obtuve %v", err)
		}
	})

	t.Run("imagen latest", func(t *testing.T) {
		_, err := NormalizeLanguageRunners([]LanguageRunner{{
			Language:    "julia",
			Entrypoint:  "julia solution.jl",
			DockerImage: "julia:latest",
		}})
		if !errors.Is(err, ErrLanguageRunnerImageUnpinned) {
			t.Fatalf("esperaba ErrLanguageRunnerImageUnpinned, obtuve %v", err)
		}
	})

	t.Run("tiempo fuera de rango", func(t *testing.T) {
		_, err := NormalizeLanguageRunners([]LanguageRunner{{
			Language:   "python",
			Entrypoint: "python3 solution.py",
			TimeoutMS:  10,
		}})
		if !errors.Is(err, ErrLanguageRunnerTimeoutRange) {
			t.Fatalf("esperaba ErrLanguageRunnerTimeoutRange, obtuve %v", err)
		}
	})
}

func TestNormalizeLanguageRunners_LenguajePropioConImagen(t *testing.T) {
	runners, err := NormalizeLanguageRunners([]LanguageRunner{{
		Language:    "Julia",
		Entrypoint:  "julia solution.jl",
		DockerImage: "julia:1.11-bookworm",
	}})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if len(runners) != 1 || runners[0].Language != "julia" || runners[0].DockerImage != "julia:1.11-bookworm" {
		t.Fatalf("ejecutor inesperado: %+v", runners)
	}
}

func TestResolveLanguageRunner(t *testing.T) {
	runners := []LanguageRunner{
		{Language: "python", Entrypoint: "python3 solution.py"},
		{Language: "cpp", Entrypoint: "g++ solution.cpp -o solution && ./solution", DockerImage: "solv/runner:1"},
	}

	runner, ok := ResolveLanguageRunner(runners, "C++")
	if !ok || runner.Entrypoint != "g++ solution.cpp -o solution && ./solution" || runner.DockerImage != "solv/runner:1" {
		t.Fatalf("no resolvió el ejecutor de c++: ok=%v runner=%+v", ok, runner)
	}

	if _, ok := ResolveLanguageRunner(runners, "java"); ok {
		t.Fatal("java no está declarado y no debía resolver")
	}
}

func TestTemplateSummarySupportsLanguage(t *testing.T) {
	summary := &TemplateSummary{
		LanguageRunners: []LanguageRunner{{Language: "python", Entrypoint: "python3 solution.py"}},
	}
	if !summary.SupportsLanguage("PYTHON") {
		t.Error("esperaba que el entorno declare python")
	}
	if summary.SupportsLanguage("go") {
		t.Error("go no está declarado en el entorno")
	}

	var nilSummary *TemplateSummary
	if nilSummary.SupportsLanguage("python") {
		t.Error("un entorno ausente no declara lenguajes")
	}
}
