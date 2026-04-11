package domain

import "time"

// Workspace is a named, isolated environment belonging to a user.
// It scopes memory, chats, projects, and Forge file operations.
type Workspace struct {
	WorkspaceID string
	UserID      string
	Name        string
	Description string
	RootPath    string    // absolute path for Forge scoping; empty = default
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
