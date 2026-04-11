package vault

import (
	"context"

	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

type NoopAdapter struct{}

func NewNoopAdapter() *NoopAdapter { return &NoopAdapter{} }

func (NoopAdapter) Store(_ context.Context, _, _, _, _ string) error { return nil }

func (NoopAdapter) Search(_ context.Context, _, _, _ string, _ int) ([]ports.KnowledgeEntry, error) {
	return nil, nil
}
