package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/isaacwallace123/cortex/services/policy/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/policy/internal/log"
	"github.com/isaacwallace123/cortex/services/policy/internal/ports"
)

// SubmitApprovalUseCase resolves a pending approval with an approve or deny decision.
type SubmitApprovalUseCase struct {
	store ports.ApprovalStore
	log   *slog.Logger
}

func NewSubmitApprovalUseCase(store ports.ApprovalStore) *SubmitApprovalUseCase {
	return &SubmitApprovalUseCase{store: store, log: cortexlog.Aegis()}
}

type SubmitApprovalInput struct {
	ApprovalID string
	UserID     string
	Decision   string // "approved" | "denied"
}

type SubmitApprovalOutput struct {
	Verdict string
}

func (uc *SubmitApprovalUseCase) Execute(ctx context.Context, in SubmitApprovalInput) (SubmitApprovalOutput, error) {
	if in.ApprovalID == "" {
		return SubmitApprovalOutput{}, fmt.Errorf("approval_id required")
	}

	var status domain.ApprovalStatus
	switch in.Decision {
	case "approved":
		status = domain.ApprovalApproved
	case "denied":
		status = domain.ApprovalDenied
	default:
		return SubmitApprovalOutput{}, fmt.Errorf("decision must be 'approved' or 'denied', got %q", in.Decision)
	}

	if err := uc.store.Resolve(ctx, in.ApprovalID, in.UserID, status); err != nil {
		return SubmitApprovalOutput{}, fmt.Errorf("resolve approval: %w", err)
	}

	uc.log.Info("[Aegis] approval resolved",
		slog.String("approval_id", in.ApprovalID),
		slog.String("user_id", in.UserID),
		slog.String("decision", in.Decision),
	)

	return SubmitApprovalOutput{Verdict: in.Decision}, nil
}
