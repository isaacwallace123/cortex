package ports

import "context"

// Event is the standard telemetry envelope for the policy service.
// Name follows the dot-separated convention: subsystem.noun.verb
// e.g. "aegis.plan.evaluated", "aegis.step.denied"
type Event struct {
	Name    string
	Payload map[string]any
}

// TelemetryPort abstracts event emission.
// The concrete adapter may log, publish to a queue, or both.
type TelemetryPort interface {
	Emit(ctx context.Context, event Event) error
}
