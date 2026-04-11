package shell

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
)

// NoopAdapter logs but does not execute when no shell service is configured.
type NoopAdapter struct{}

func NewNoopAdapter() *NoopAdapter { return &NoopAdapter{} }

func (a *NoopAdapter) Execute(_ context.Context, _, _ string, step domain.Step) (domain.ExecutionResult, error) {
	return domain.ExecutionResult{
		StepIndex:  step.Index,
		Skipped:    true,
		SkipReason: fmt.Sprintf("shell executor not configured (SHELL_ADDR unset), step %d skipped", step.Index),
	}, nil
}
