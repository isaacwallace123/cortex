package application

import (
	"context"
	"fmt"
	"log/slog"

	cortexlog "github.com/isaacwallace123/cortex/services/memory/internal/log"
	"github.com/isaacwallace123/cortex/services/memory/internal/domain"
	"github.com/isaacwallace123/cortex/services/memory/internal/ports"
)

type ListPlansUseCase struct {
	store ports.StorePort
	log   *slog.Logger
}

func NewListPlansUseCase(store ports.StorePort) *ListPlansUseCase {
	return &ListPlansUseCase{store: store, log: cortexlog.Echo()}
}

type ListPlansOutput struct {
	Plans []domain.PlanSummary
}

func (uc *ListPlansUseCase) Execute(ctx context.Context, sessionID, userID string) (ListPlansOutput, error) {
	if sessionID == "" {
		return ListPlansOutput{}, fmt.Errorf("session_id must not be empty")
	}

	plans, err := uc.store.ListPlans(ctx, sessionID, userID)
	if err != nil {
		return ListPlansOutput{}, fmt.Errorf("list plans: %w", err)
	}

	uc.log.Info("[Echo] plans listed",
		slog.String("session_id", sessionID),
		slog.Int("plans", len(plans)),
	)

	return ListPlansOutput{Plans: plans}, nil
}
