package lang

import (
	"strings"
	"testing"
)

// Runner images stay pinned by digest; mutable tags are banned in the judge.
func TestRunnerImagesPinned(t *testing.T) {
	// Built via concatenation so the ban check below never matches itself.
	banned := ":" + "latest"
	cases := []struct {
		language string
		image    string
		want     string
	}{
		{"python", PythonImage, "python:3.11-slim@sha256:db3ff2e1800a8581e2c48a27c3995339d47bdf046da21c7627accd3d51053a93"},
		{"javascript", JavaScriptImage, "node:20-alpine@sha256:fb4cd12c85ee03686f6af5362a0b0d56d50c58a04632e6c0fb8363f609372293"},
		{"c", CImage, "gcc@sha256:cde79a7114216f9a1a66509932adabdd1e8620d8c8d11be19a34ee4b22d66c91"},
		{"cpp", CppImage, "gcc@sha256:cde79a7114216f9a1a66509932adabdd1e8620d8c8d11be19a34ee4b22d66c91"},
		{"csharp", CSharpImage, "mono@sha256:34d816779b1248b5cfd095770b64ecbaf1798e2aca693a91c11a018dce9c7ad5"},
		{"java", JavaImage, "eclipse-temurin:21-alpine@sha256:1ff763083f2993d57d0bf374ab10bb3e2cb873af6c13a04458ebbd3e0337dc76"},
	}
	for _, tt := range cases {
		t.Run(tt.language, func(t *testing.T) {
			if tt.image != tt.want {
				t.Errorf("image for %s = %q, want %q", tt.language, tt.image, tt.want)
			}
			if strings.Contains(tt.image, banned) {
				t.Errorf("image for %s uses mutable tag: %q", tt.language, tt.image)
			}
			parts := strings.SplitN(tt.image, "@sha256:", 2)
			if len(parts) != 2 || len(parts[1]) != 64 {
				t.Errorf("image for %s lacks 64-hex digest: %q", tt.language, tt.image)
			}
		})
	}
}
