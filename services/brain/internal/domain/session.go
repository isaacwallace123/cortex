package domain

import "time"

// Session represents an active interaction context between a user and Cortex.
// A session scopes all tasks, plans, and decisions for one user intent lifecycle.
type Session struct {
	ID        string
	CreatedAt time.Time
}

func NewSession(id string) Session {
	return Session{
		ID:        id,
		CreatedAt: time.Now().UTC(),
	}
}
