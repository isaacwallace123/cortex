package application

import (
	"context"

	"github.com/isaacwallace123/cortex/services/catalyst/internal/domain"
	"github.com/isaacwallace123/cortex/services/catalyst/internal/ports"
)

type ListSkillsUseCase struct {
	store ports.SkillStore
}

func NewListSkillsUseCase(store ports.SkillStore) *ListSkillsUseCase {
	return &ListSkillsUseCase{store: store}
}

func (uc *ListSkillsUseCase) Execute(ctx context.Context, tag string) ([]domain.Skill, error) {
	return uc.store.List(ctx, tag)
}
