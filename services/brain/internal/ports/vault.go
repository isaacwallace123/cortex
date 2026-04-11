package ports

import "context"

// KnowledgeEntry is a retrieved Vault fragment injected into planning context.
type KnowledgeEntry struct {
	EntryID   string
	Content   string
	Source    string
	CreatedAt int64 // unix millis
}

// VaultPort lets Brain store and retrieve long-term knowledge.
// Store is called by Vector after meaningful executions.
// Search is called by Axiom before planning to inject relevant past knowledge.
type VaultPort interface {
	Store(ctx context.Context, userID, workspaceID, content, source string) error
	Search(ctx context.Context, userID, workspaceID, query string, limit int) ([]KnowledgeEntry, error)
}
