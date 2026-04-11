package ports

import "context"

// Event is the standard telemetry envelope for Cortex.
// All significant internal actions emit one of these.
type Event struct {
	// Name follows the dot-separated convention: subsystem.noun.verb
	// e.g. "prism.input.parsed", "axiom.plan.created", "vector.step.executed"
	Name    string
	Payload map[string]any
}

// TelemetryPort abstracts event emission.
// The concrete adapter may log, publish to a queue, or both.
type TelemetryPort interface {
	Emit(ctx context.Context, event Event) error
}
