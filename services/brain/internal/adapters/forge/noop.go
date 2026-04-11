package forge

import (
	"context"

	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
)

// NoopAdapter silently skips filesystem steps when Forge is not configured.
type NoopAdapter struct{}

func NewNoopAdapter() *NoopAdapter { return &NoopAdapter{} }

func (a *NoopAdapter) Execute(_ context.Context, _, _ string, step domain.Step) (domain.ExecutionResult, error) {
	return domain.ExecutionResult{
		StepIndex:  step.Index,
		Skipped:    true,
		SkipReason: "forge executor not configured",
	}, nil
}
