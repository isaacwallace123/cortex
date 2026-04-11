package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/policy/internal/domain"
)

// ApprovalStore persists and resolves pending approvals.
type ApprovalStore interface {
	// Create persists a new pending approval and returns it with ApprovalID set.
	Create(ctx context.Context, a domain.Approval) (domain.Approval, error)

	// Resolve marks an approval as approved or denied.
	Resolve(ctx context.Context, approvalID, userID string, status domain.ApprovalStatus) error

	// Get returns a single approval by ID.
	Get(ctx context.Context, approvalID string) (domain.Approval, error)

	// ListPending returns all approvals with status=pending, optionally filtered by userID.
	ListPending(ctx context.Context, userID string) ([]domain.Approval, error)
}
