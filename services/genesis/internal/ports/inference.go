package ports

import "context"

// InferencePort calls the LLM to generate text completions.
type InferencePort interface {
	Complete(ctx context.Context, prompt string, temperature float32) (string, error)
}
