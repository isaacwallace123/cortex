package domain

// RunResult is the outcome of a shell command executed by the Shell service.
type RunResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
}
