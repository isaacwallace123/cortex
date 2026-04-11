package ports

import "context"

// AtlasPort is Brain's interface to the Atlas Kubernetes execution service.
// Vector routes steps with executor="atlas" through this port.
type AtlasPort interface {
	Execute(ctx context.Context, command string, timeoutSecs int) (AtlasResult, error)
}

// AtlasResult holds structured output from an Atlas kubectl execution.
type AtlasResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int32
	DurationMs int64
}
