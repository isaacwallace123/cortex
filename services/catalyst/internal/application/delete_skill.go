package application

import (
	"context"

	"github.com/isaacwallace123/cortex/services/catalyst/internal/ports"
)

type DeleteSkillUseCase struct {
	store ports.SkillStore
}

func NewDeleteSkillUseCase(store ports.SkillStore) *DeleteSkillUseCase {
	return &DeleteSkillUseCase{store: store}
}

func (uc *DeleteSkillUseCase) Execute(ctx context.Context, name string) (bool, error) {
	return uc.store.Delete(ctx, name)
}
