package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/arsenal/internal/domain"
	"github.com/isaacwallace123/cortex/services/arsenal/internal/ports"
)

type RegisterToolUseCase struct {
	store ports.ToolStore
}

func NewRegisterToolUseCase(store ports.ToolStore) *RegisterToolUseCase {
	return &RegisterToolUseCase{store: store}
}

func (uc *RegisterToolUseCase) Execute(ctx context.Context, t *domain.Tool) (*domain.Tool, error) {
	if t.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}
	if t.Executor == "" {
		return nil, fmt.Errorf("tool executor is required")
	}
	if t.SafetyLevel == "" {
		t.SafetyLevel = domain.SafetyLevelSafe
	}
	if t.Version == "" {
		t.Version = "1.0.0"
	}
	if err := uc.store.Upsert(ctx, t); err != nil {
		return nil, fmt.Errorf("upsert tool %q: %w", t.Name, err)
	}
	return t, nil
}
