package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/shell/internal/domain"
)

// RunnerPort is the interface for executing shell commands.
type RunnerPort interface {
	Run(ctx context.Context, command string) (domain.RunResult, error)
}
