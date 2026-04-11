package application

import (
	"context"

	"github.com/isaacwallace123/cortex/services/catalyst/internal/domain"
	"github.com/isaacwallace123/cortex/services/catalyst/internal/ports"
)

type GetSkillUseCase struct {
	store ports.SkillStore
}

func NewGetSkillUseCase(store ports.SkillStore) *GetSkillUseCase {
	return &GetSkillUseCase{store: store}
}

func (uc *GetSkillUseCase) Execute(ctx context.Context, name string) (*domain.Skill, error) {
	return uc.store.Get(ctx, name)
}
