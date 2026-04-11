package domain

import "time"

// Event is the canonical Nerva event envelope.
type Event struct {
	ID        string
	Subsystem string
	Name      string    // dot-separated: "prism.input.parsed"
	SessionID string    // empty for non-session-scoped events
	Payload   string    // JSON string
	CreatedAt time.Time
}
