package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/arsenal/internal/domain"
	"github.com/isaacwallace123/cortex/services/arsenal/internal/ports"
)

type ListToolsUseCase struct {
	store ports.ToolStore
}

func NewListToolsUseCase(store ports.ToolStore) *ListToolsUseCase {
	return &ListToolsUseCase{store: store}
}

func (uc *ListToolsUseCase) Execute(ctx context.Context, executor, tag string) ([]*domain.Tool, error) {
	tools, err := uc.store.List(ctx, executor, tag)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}
	return tools, nil
}
