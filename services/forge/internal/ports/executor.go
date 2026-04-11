package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/forge/internal/domain"
)

// FSPort executes sandboxed filesystem operations.
type FSPort interface {
	Execute(ctx context.Context, command string) (domain.OpResult, error)
}
