package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/policy/internal/domain"
)

// EvaluatorPort is the policy engine interface.
// Aegis uses this to evaluate plan steps against configured rules.
type EvaluatorPort interface {
	Evaluate(ctx context.Context, sessionID, planID string, steps []domain.StepInput) ([]domain.StepDecision, error)
}
