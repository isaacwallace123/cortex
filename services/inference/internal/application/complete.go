package application

import (
	"context"
	"fmt"
	"log/slog"

	cortexlog "github.com/isaacwallace123/cortex/services/inference/internal/log"
	"github.com/isaacwallace123/cortex/services/inference/internal/domain"
	"github.com/isaacwallace123/cortex/services/inference/internal/ports"
)

// CompleteUseCase sends a prompt to the configured LLM provider.
type CompleteUseCase struct {
	provider     ports.LLMProviderPort
	defaultModel string
	log          *slog.Logger
}

func NewCompleteUseCase(provider ports.LLMProviderPort, defaultModel string) *CompleteUseCase {
	return &CompleteUseCase{provider: provider, defaultModel: defaultModel, log: cortexlog.Inference()}
}

type CompleteInput struct {
	Model       string
	Prompt      string
	Temperature float32
	MaxTokens   int32
}

type CompleteOutput struct {
	Response domain.GenerationResponse
}

func (uc *CompleteUseCase) Execute(ctx context.Context, in CompleteInput) (CompleteOutput, error) {
	if in.Prompt == "" {
		return CompleteOutput{}, fmt.Errorf("prompt must not be empty")
	}

	model := in.Model
	if model == "" {
		model = uc.defaultModel
	}

	uc.log.Info("[inference] completing",
		slog.String("model", model),
		slog.Int("prompt_len", len(in.Prompt)),
	)

	resp, err := uc.provider.Complete(ctx, domain.GenerationRequest{
		Model:       model,
		Prompt:      in.Prompt,
		Temperature: in.Temperature,
		MaxTokens:   in.MaxTokens,
	})
	if err != nil {
		uc.log.Error("[inference] completion failed",
			slog.String("model", model),
			slog.String("error", err.Error()),
		)
		return CompleteOutput{}, fmt.Errorf("provider completion failed: %w", err)
	}

	uc.log.Info("[inference] complete",
		slog.String("model", resp.Model),
		slog.Int("input_tokens", int(resp.InputTokens)),
		slog.Int("output_tokens", int(resp.OutputTokens)),
	)

	return CompleteOutput{Response: resp}, nil
}
