package application

import (
	"context"
	"fmt"
	"log/slog"

	cortexlog "github.com/isaacwallace123/cortex/services/inference/internal/log"
	"github.com/isaacwallace123/cortex/services/inference/internal/domain"
	"github.com/isaacwallace123/cortex/services/inference/internal/ports"
)

// CompleteStreamUseCase streams tokens from the configured LLM provider.
type CompleteStreamUseCase struct {
	provider     ports.LLMProviderPort
	defaultModel string
	log          *slog.Logger
}

func NewCompleteStreamUseCase(provider ports.LLMProviderPort, defaultModel string) *CompleteStreamUseCase {
	return &CompleteStreamUseCase{provider: provider, defaultModel: defaultModel, log: cortexlog.Inference()}
}

func (uc *CompleteStreamUseCase) Execute(ctx context.Context, in CompleteInput) (<-chan domain.GenerationChunk, error) {
	if in.Prompt == "" {
		return nil, fmt.Errorf("prompt must not be empty")
	}
	model := in.Model
	if model == "" {
		model = uc.defaultModel
	}
	uc.log.Info("[inference] streaming",
		slog.String("model", model),
		slog.Int("prompt_len", len(in.Prompt)),
	)
	return uc.provider.CompleteStream(ctx, domain.GenerationRequest{
		Model:       model,
		Prompt:      in.Prompt,
		Temperature: in.Temperature,
		MaxTokens:   in.MaxTokens,
	})
}
