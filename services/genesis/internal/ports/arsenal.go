package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/genesis/internal/domain"
)

// ArsenalPort registers generated tools in the Arsenal tool registry.
type ArsenalPort interface {
	RegisterTool(ctx context.Context, tool domain.GeneratedTool) error
}
