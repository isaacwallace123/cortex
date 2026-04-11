package beacon

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

// NoopAdapter is used when Beacon is not configured. It returns an error
// so Vector can surface a meaningful skip reason to the user.
type NoopAdapter struct{}

func NewNoopAdapter() NoopAdapter { return NoopAdapter{} }

func (NoopAdapter) Search(_ context.Context, _ string, _ int) ([]ports.SearchResult, string, error) {
	return nil, "", fmt.Errorf("beacon not configured (BEACON_ADDR not set)")
}

func (NoopAdapter) Fetch(_ context.Context, _ string, _ int) (ports.FetchResult, error) {
	return ports.FetchResult{Error: "beacon not configured (BEACON_ADDR not set)"}, nil
}
