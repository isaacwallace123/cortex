package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/catalyst/internal/domain"
)

// SkillStore is the persistence abstraction for Catalyst skill definitions.
type SkillStore interface {
	// Create persists a new skill. Returns an error if a skill with the same
	// name already exists.
	Create(ctx context.Context, skill domain.Skill) (domain.Skill, error)

	// Get returns the skill with the given name, or nil if not found.
	Get(ctx context.Context, name string) (*domain.Skill, error)

	// List returns all skills, optionally filtered by tag.
	// An empty tag string returns all skills.
	List(ctx context.Context, tag string) ([]domain.Skill, error)

	// Delete removes the skill with the given name.
	// Returns true if the skill existed and was deleted.
	Delete(ctx context.Context, name string) (bool, error)
}
