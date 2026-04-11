package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/arsenal/internal/ports"
)

type DeleteToolUseCase struct {
	store ports.ToolStore
}

func NewDeleteToolUseCase(store ports.ToolStore) *DeleteToolUseCase {
	return &DeleteToolUseCase{store: store}
}

func (uc *DeleteToolUseCase) Execute(ctx context.Context, name string) (bool, error) {
	deleted, err := uc.store.Delete(ctx, name)
	if err != nil {
		return false, fmt.Errorf("delete tool %q: %w", name, err)
	}
	return deleted, nil
}
