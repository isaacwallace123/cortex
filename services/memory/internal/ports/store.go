package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/memory/internal/domain"
)

// StorePort is the persistence abstraction for memory.
// Concrete adapters: SQLite (local), PostgreSQL (production).
type StorePort interface {
	// StoreEvent persists an event and returns its assigned ID and timestamp.
	StoreEvent(ctx context.Context, event domain.Event) (domain.Event, error)

	// GetSession returns all events for a session in chronological order.
	// When userID is non-empty, results are filtered to that user.
	GetSession(ctx context.Context, sessionID, userID string) ([]domain.Event, error)

	// ListPlans returns plan summaries for a session.
	// When userID is non-empty, results are filtered to that user.
	ListPlans(ctx context.Context, sessionID, userID string) ([]domain.PlanSummary, error)
}
