package crucible

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

// NoopAdapter is returned when CRUCIBLE_ADDR is not configured.
// It lets the system boot but returns a clear error for any crucible step.
type NoopAdapter struct{}

func NewNoopAdapter() *NoopAdapter { return &NoopAdapter{} }

func (a *NoopAdapter) RunCode(_ context.Context, runtime, _ string, _ int) (ports.CrucibleResult, error) {
	return ports.CrucibleResult{}, fmt.Errorf("crucible executor not available (CRUCIBLE_ADDR not set); runtime=%s", runtime)
}
