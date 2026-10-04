package lang

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"solv-backend/internal/core/domain"
)

// Integration against the local daemon: proves ImageInspectWithRaw with the
// pinned repo@digest form works and the registry-backed comparison holds.
// Skipped with -short (no daemon in CI).
func TestRunContainerExecution_PinnedDigest(t *testing.T) {
	if testing.Short() {
		t.Skip("needs local docker daemon")
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Skipf("no docker client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cfg := domain.EvaluationRunConfig{
		Language:      "c",
		SourceCode:    "#include <stdio.h>\nint main(){printf(\"hola\\n\");return 0;}",
		MemoryLimitMB: 128,
		TimeLimitMS:   10000,
		TestCase:      domain.TestCase{Input: "", ExpectedOutput: "hola"},
	}
	res, err := runContainerExecution(ctx, cli, CImage, "solution.c", []string{"sh", "-c", "gcc -O2 /runner/solution.c -o /tmp/sol && /tmp/sol"}, cfg)
	if err != nil {
		t.Fatalf("runContainerExecution error: %v", err)
	}
	if res.Verdict != domain.VerdictAC {
		t.Fatalf("verdict = %s (%s), want AC", res.Verdict, res.ErrorDetails)
	}
	if res.ImageDigest == "" {
		t.Error("ImageDigest empty, want digest from repo@digest reference")
	}

	waCfg := cfg
	waCfg.TestCase.ExpectedOutput = "adios"
	waRes, err := runContainerExecution(ctx, cli, CImage, "solution.c", []string{"sh", "-c", "gcc -O2 /runner/solution.c -o /tmp/sol && /tmp/sol"}, waCfg)
	if err != nil {
		t.Fatalf("runContainerExecution error: %v", err)
	}
	if waRes.Verdict != domain.VerdictWA {
		t.Fatalf("verdict = %s, want WA", waRes.Verdict)
	}
}
