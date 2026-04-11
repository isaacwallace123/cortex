package ports

import (
	"context"
	"time"
)

// StoredEvent is a persisted session event returned by the memory service.
type StoredEvent struct {
	ID        string
	Name      string
	Payload   string
	StoredAt  time.Time
}

// MemoryPort is how brain persists and retrieves session events.
type MemoryPort interface {
	StoreEvent(ctx context.Context, sessionID, eventName, payload string) error
	GetSession(ctx context.Context, sessionID string) ([]StoredEvent, error)
}
