package comparators

import "strings"

// CheckerSpec describes how to invoke the teacher checker. The sidecar image
// itself is owned by administration (checker_sidecar_image profile); this
// package only defines the invocation and evaluation contract, without
// running any container.
type CheckerSpec struct {
	Command       []string
	TimeoutMs     int
	ReadOnlyPaths []string
}

// ParseCheckerSpec validates the checker invocation contract. A missing
// command is rejected with CodeCheckerInvalid.
func ParseCheckerSpec(p map[string]any) (CheckerSpec, error) {
	command, ok := paramStrings(p, "command")
	if !ok || len(command) == 0 {
		return CheckerSpec{}, &ComparatorError{Code: CodeCheckerInvalid, Field: "command", Detail: "checker command is required"}
	}
	spec := CheckerSpec{Command: command}
	if v, ok := paramFloat(p, "timeout_ms"); ok {
		spec.TimeoutMs = int(v)
	}
	if paths, ok := paramStrings(p, "read_only_paths"); ok {
		spec.ReadOnlyPaths = paths
	}
	return spec, nil
}

// CheckerInvocation binds the input, expected and output paths for one run.
// Paths are always mounted read-only; the checker decides AC/WA through its
// stdout.
type CheckerInvocation struct {
	InputPath     string
	ExpectedPath  string
	OutputPath    string
	ReadOnly      bool
	ReadOnlyPaths []string
	TimeoutMs     int
}

// NewCheckerInvocation builds a read-only invocation for the checker.
func NewCheckerInvocation(inputPath, expectedPath, outputPath string, timeoutMs int) CheckerInvocation {
	return CheckerInvocation{
		InputPath:     inputPath,
		ExpectedPath:  expectedPath,
		OutputPath:    outputPath,
		ReadOnly:      true,
		ReadOnlyPaths: []string{inputPath, expectedPath, outputPath},
		TimeoutMs:     timeoutMs,
	}
}

// CheckerOutcome is the raw sidecar result handed to the evaluator.
type CheckerOutcome struct {
	Stdout   string
	ExitCode int
	TimedOut bool
	Crashed  bool
}

// ParseCheckerOutput maps the checker stdout to a verdict: first line AC or
// WA plus an optional message on the following lines.
func ParseCheckerOutput(stdout string) Result {
	lines := strings.SplitN(strings.TrimSpace(stdout), "\n", 2)
	head := strings.ToUpper(strings.TrimSpace(lines[0]))
	message := ""
	if len(lines) > 1 {
		message = strings.TrimSpace(lines[1])
	}
	if head == string(VerdictAC) {
		return Result{Verdict: VerdictAC, Message: message}
	}
	if message == "" {
		message = "checker rejected the output"
	}
	return Result{Verdict: VerdictWA, Message: message}
}

// EvaluateCheckerOutcome applies the contract: checker crash, timeout or
// non-zero exit is VE; otherwise the stdout decides AC/WA.
func EvaluateCheckerOutcome(outcome CheckerOutcome) Result {
	if outcome.TimedOut {
		return Result{Verdict: VerdictVE, Message: "checker timed out"}
	}
	if outcome.Crashed {
		return Result{Verdict: VerdictVE, Message: "checker crashed"}
	}
	if outcome.ExitCode != 0 {
		return Result{Verdict: VerdictVE, Message: "checker exited with error"}
	}
	return ParseCheckerOutput(outcome.Stdout)
}
