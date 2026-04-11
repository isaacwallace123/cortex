package courier

import (
	"context"

	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

// NoopAdapter satisfies CourierPort when Courier is unavailable.
type NoopAdapter struct{}

func NewNoopAdapter() *NoopAdapter { return &NoopAdapter{} }

func (NoopAdapter) ListAgents(_ context.Context) ([]ports.RemoteAgent, error) {
	return nil, nil
}

func (NoopAdapter) Dispatch(_ context.Context, _, _ string, step domain.Step) (domain.ExecutionResult, error) {
	return domain.ExecutionResult{
		StepIndex:  step.Index,
		Skipped:    true,
		SkipReason: "courier not available",
	}, nil
}
