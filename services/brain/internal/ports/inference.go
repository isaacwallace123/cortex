package ports

import "context"

// InferencePort abstracts the LLM access layer.
// Brain calls this to reason about input and produce plans.
type InferencePort interface {
	// Complete sends a prompt and returns the full text response.
	// temperature controls randomness: 0.0 = deterministic, 1.0 = very random.
	Complete(ctx context.Context, prompt string, temperature float32) (string, error)

	// CompleteStream sends a prompt and streams tokens as they arrive.
	// The returned channel is closed when generation finishes or ctx is cancelled.
	// temperature controls randomness: 0.0 = deterministic, 1.0 = very random.
	CompleteStream(ctx context.Context, prompt string, temperature float32) (<-chan string, error)
}
