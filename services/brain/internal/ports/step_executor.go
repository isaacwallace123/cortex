package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
)

// StepExecutor is the common interface Vector uses to dispatch a plan step
// to any concrete executor (Shell, Forge, and future executors).
type StepExecutor interface {
	Execute(ctx context.Context, sessionID, planID string, step domain.Step) (domain.ExecutionResult, error)
}
