package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
)

// PolicyPort is brain's interface to Aegis (the policy service).
// Vector calls this after Axiom creates a plan to get per-step verdicts.
type PolicyPort interface {
	EvaluatePlan(ctx context.Context, sessionID, planID string, steps []domain.Step) ([]domain.StepDecision, error)
}
