package application

import (
	"context"
	"fmt"
	"time"

	"github.com/isaacwallace123/cortex/services/catalyst/internal/domain"
	"github.com/isaacwallace123/cortex/services/catalyst/internal/ports"
)

// CreateSkillUseCase registers a new skill in the Catalyst registry.
type CreateSkillUseCase struct {
	store ports.SkillStore
}

func NewCreateSkillUseCase(store ports.SkillStore) *CreateSkillUseCase {
	return &CreateSkillUseCase{store: store}
}

func (uc *CreateSkillUseCase) Execute(ctx context.Context, skill domain.Skill) (domain.Skill, error) {
	if skill.Name == "" {
		return domain.Skill{}, fmt.Errorf("skill name is required")
	}
	if len(skill.Steps) == 0 {
		return domain.Skill{}, fmt.Errorf("skill must have at least one step")
	}
	if skill.Version == "" {
		skill.Version = "1.0.0"
	}
	skill.CreatedAt = time.Now().UTC()
	return uc.store.Create(ctx, skill)
}
