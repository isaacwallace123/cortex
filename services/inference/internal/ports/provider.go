package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/inference/internal/domain"
)

// LLMProviderPort abstracts a concrete LLM backend (Ollama, vLLM, etc.).
// The application layer depends only on this interface.
type LLMProviderPort interface {
	// Complete sends a prompt to the model and returns the generated text.
	Complete(ctx context.Context, req domain.GenerationRequest) (domain.GenerationResponse, error)

	// CompleteStream streams generated tokens as they arrive from the provider.
	// The returned channel is closed when streaming completes or the context is cancelled.
	CompleteStream(ctx context.Context, req domain.GenerationRequest) (<-chan domain.GenerationChunk, error)

	// ListModels returns models available from this provider.
	ListModels(ctx context.Context) ([]domain.Model, error)
}
