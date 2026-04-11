package ports

import "context"

// InferencePort abstracts the LLM access layer.
// Brain calls this to reason about input and produce plans.
type InferencePort interface {
	// Complete sends a prompt and returns the full text response.
	Complete(ctx context.Context, prompt string) (string, error)

	// CompleteStream sends a prompt and streams tokens as they arrive.
	// The returned channel is closed when generation finishes or ctx is cancelled.
	CompleteStream(ctx context.Context, prompt string) (<-chan string, error)
}
