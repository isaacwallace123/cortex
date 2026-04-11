package rules

import (
	"context"

	"github.com/isaacwallace123/cortex/services/policy/internal/domain"
)

// StubEvaluator is the default Aegis adapter — it allows all steps.
// Replace with a rule-based or human-in-the-loop evaluator for production.
type StubEvaluator struct{}

func NewStubEvaluator() *StubEvaluator { return &StubEvaluator{} }

func (e *StubEvaluator) Evaluate(_ context.Context, _, _ string, steps []domain.StepInput) ([]domain.StepDecision, error) {
	decisions := make([]domain.StepDecision, 0, len(steps))
	for _, s := range steps {
		decisions = append(decisions, domain.StepDecision{
			StepIndex: s.Index,
			Verdict:   domain.VerdictAllow,
			Reason:    "stub: all steps allowed",
		})
	}
	return decisions, nil
}
