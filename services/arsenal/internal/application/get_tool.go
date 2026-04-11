package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/arsenal/internal/domain"
	"github.com/isaacwallace123/cortex/services/arsenal/internal/ports"
)

type GetToolUseCase struct {
	store ports.ToolStore
}

func NewGetToolUseCase(store ports.ToolStore) *GetToolUseCase {
	return &GetToolUseCase{store: store}
}

func (uc *GetToolUseCase) Execute(ctx context.Context, name string) (*domain.Tool, error) {
	t, err := uc.store.Get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get tool %q: %w", name, err)
	}
	return t, nil
}
