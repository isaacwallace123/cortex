package domain

type TaskResult struct {
	TaskID     string
	Stdout     string
	Stderr     string
	ExitCode   int32
	DurationMs int64
}
