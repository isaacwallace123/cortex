package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/genesis/internal/domain"
)

// ProposalStore persists generation proposals so the history is queryable.
type ProposalStore interface {
	Save(ctx context.Context, p domain.Proposal) (domain.Proposal, error)
	List(ctx context.Context) ([]domain.Proposal, error)
}
