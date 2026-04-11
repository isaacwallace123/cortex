package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/inference/internal/domain"
	"github.com/isaacwallace123/cortex/services/inference/internal/ports"
)

// ListModelsUseCase returns models available from the configured LLM provider.
type ListModelsUseCase struct {
	provider ports.LLMProviderPort
}

func NewListModelsUseCase(provider ports.LLMProviderPort) *ListModelsUseCase {
	return &ListModelsUseCase{provider: provider}
}

type ListModelsOutput struct {
	Models []domain.Model
}

func (uc *ListModelsUseCase) Execute(ctx context.Context) (ListModelsOutput, error) {
	models, err := uc.provider.ListModels(ctx)
	if err != nil {
		return ListModelsOutput{}, fmt.Errorf("failed to list models: %w", err)
	}
	return ListModelsOutput{Models: models}, nil
}
