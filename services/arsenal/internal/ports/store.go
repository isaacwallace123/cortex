// Package ports defines the storage interface for Arsenal.
package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/arsenal/internal/domain"
)

// ToolStore is the persistence port for tool definitions.
type ToolStore interface {
	// Upsert inserts or replaces a tool by name.
	Upsert(ctx context.Context, tool *domain.Tool) error

	// Get returns a single tool by name, or nil if not found.
	Get(ctx context.Context, name string) (*domain.Tool, error)

	// List returns all tools, optionally filtered by executor and/or tag.
	// Empty strings mean no filter.
	List(ctx context.Context, executor, tag string) ([]*domain.Tool, error)

	// Delete removes a tool by name. Returns true if a row was deleted.
	Delete(ctx context.Context, name string) (bool, error)
}
