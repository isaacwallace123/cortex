package memory

import (
	"context"

	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

// NoopAdapter silently discards writes and returns empty reads.
// Used when MEMORY_ADDR is not configured.
type NoopAdapter struct{}

func NewNoopAdapter() *NoopAdapter { return &NoopAdapter{} }

func (a *NoopAdapter) StoreEvent(_ context.Context, _, _, _ string) error { return nil }

func (a *NoopAdapter) GetSession(_ context.Context, _ string) ([]ports.StoredEvent, error) {
	return nil, nil
}
