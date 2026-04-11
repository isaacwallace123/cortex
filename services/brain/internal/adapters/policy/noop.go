package policy

import (
	"context"

	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
)

// NoopAdapter allows all steps when no policy service is configured.
type NoopAdapter struct{}

func NewNoopAdapter() *NoopAdapter { return &NoopAdapter{} }

func (a *NoopAdapter) EvaluatePlan(_ context.Context, _, _ string, steps []domain.Step) ([]domain.StepDecision, error) {
	decisions := make([]domain.StepDecision, 0, len(steps))
	for _, s := range steps {
		decisions = append(decisions, domain.StepDecision{
			StepIndex: s.Index,
			Verdict:   "allow",
			Reason:    "noop: policy service not configured",
		})
	}
	return decisions, nil
}
