package domain

import "time"

// Event is a persisted record of something significant that happened in a session.
// Brain emits events after parsing input, creating plans, and making decisions.
// The event stream is the source of truth for session history.
type Event struct {
	ID          string
	SessionID   string
	UserID      string    // identity scope — empty means legacy/anonymous
	WorkspaceID string    // workspace scope — empty means default workspace
	Name        string    // e.g. "brain.input.parsed", "brain.plan.created"
	Payload     string    // JSON-encoded data specific to the event type
	StoredAt    time.Time
}

// PlanSummary is a projection of plan-creation events, for efficient listing.
type PlanSummary struct {
	PlanID      string
	SessionID   string
	UserID      string
	WorkspaceID string
	Intent      string
	StepCount   int
	CreatedAt   time.Time
}
