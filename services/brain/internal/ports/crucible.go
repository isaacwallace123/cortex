package ports

import "context"

// CruciblePort is the Brain's interface to the Crucible sandboxed execution service.
// Vector calls RunCode for single-shot code execution steps.
type CruciblePort interface {
	// RunCode executes a command in an ephemeral sandbox and returns the output.
	RunCode(ctx context.Context, runtime, command string, timeoutSecs int) (CrucibleResult, error)
}

// CrucibleResult holds the output of a Crucible RunCode call.
type CrucibleResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int32
	DurationMs int64
}
