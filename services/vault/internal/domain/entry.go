package domain

import "time"

// Entry is a single knowledge fragment stored in the Vault.
type Entry struct {
	ID          string
	UserID      string
	WorkspaceID string
	Content     string
	Source      string
	Tags        []string
	CreatedAt   time.Time
}
