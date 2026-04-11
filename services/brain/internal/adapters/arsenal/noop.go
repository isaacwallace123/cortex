package arsenal

import (
	"context"

	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

// NoopAdapter returns an empty tool list. Used when Arsenal is unavailable.
// Brain falls back to its built-in hardcoded executor rules.
type NoopAdapter struct{}

func (NoopAdapter) ListTools(_ context.Context) ([]ports.RegistryTool, error) {
	return nil, nil
}
