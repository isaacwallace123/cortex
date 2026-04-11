package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/policy/internal/domain"
	"github.com/isaacwallace123/cortex/services/policy/internal/ports"
)

// ListApprovalsUseCase returns pending approvals, optionally scoped to a user.
type ListApprovalsUseCase struct {
	store ports.ApprovalStore
}

func NewListApprovalsUseCase(store ports.ApprovalStore) *ListApprovalsUseCase {
	return &ListApprovalsUseCase{store: store}
}

type ListApprovalsOutput struct {
	Approvals []domain.Approval
}

func (uc *ListApprovalsUseCase) Execute(ctx context.Context, userID string) (ListApprovalsOutput, error) {
	approvals, err := uc.store.ListPending(ctx, userID)
	if err != nil {
		return ListApprovalsOutput{}, fmt.Errorf("list approvals: %w", err)
	}
	return ListApprovalsOutput{Approvals: approvals}, nil
}
