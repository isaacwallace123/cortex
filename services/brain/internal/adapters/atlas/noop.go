package atlas

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

// NoopAdapter is used when ATLAS_ADDR is not configured.
type NoopAdapter struct{}

func NewNoopAdapter() *NoopAdapter { return &NoopAdapter{} }

func (a *NoopAdapter) Execute(_ context.Context, command string, _ int) (ports.AtlasResult, error) {
	return ports.AtlasResult{}, fmt.Errorf("atlas executor not available (ATLAS_ADDR not set); command=%q", command)
}
