package application

import (
	"context"

	"github.com/isaacwallace123/cortex/services/genesis/internal/domain"
	"github.com/isaacwallace123/cortex/services/genesis/internal/ports"
)

type ListProposalsUseCase struct {
	store ports.ProposalStore
}

func NewListProposalsUseCase(store ports.ProposalStore) *ListProposalsUseCase {
	return &ListProposalsUseCase{store: store}
}

func (uc *ListProposalsUseCase) Execute(ctx context.Context) ([]domain.Proposal, error) {
	return uc.store.List(ctx)
}
