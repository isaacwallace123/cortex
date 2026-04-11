package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/nerva/internal/domain"
)

// StorePort persists events for replay and audit.
type StorePort interface {
	Save(ctx context.Context, event domain.Event) error
}
