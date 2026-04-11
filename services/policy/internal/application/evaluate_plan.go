package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/isaacwallace123/cortex/services/policy/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/policy/internal/log"
	"github.com/isaacwallace123/cortex/services/policy/internal/ports"
)

// EvaluatePlanUseCase runs each step of a plan through the Aegis policy engine.
// Pending steps create persisted approval records so decisions survive restarts.
type EvaluatePlanUseCase struct {
	evaluator ports.EvaluatorPort
	store     ports.ApprovalStore
	telemetry ports.TelemetryPort
	log       *slog.Logger
}

func NewEvaluatePlanUseCase(evaluator ports.EvaluatorPort, store ports.ApprovalStore, telemetry ports.TelemetryPort) *EvaluatePlanUseCase {
	return &EvaluatePlanUseCase{
		evaluator: evaluator,
		store:     store,
		telemetry: telemetry,
		log:       cortexlog.Aegis(),
	}
}

type EvaluatePlanInput struct {
	SessionID string
	PlanID    string
	Steps     []domain.StepInput
	UserID    string
	Roles     []string
}

type EvaluatePlanOutput struct {
	Decisions []domain.StepDecision
}

func (uc *EvaluatePlanUseCase) Execute(ctx context.Context, in EvaluatePlanInput) (EvaluatePlanOutput, error) {
	if in.PlanID == "" {
		return EvaluatePlanOutput{}, fmt.Errorf("plan_id must not be empty")
	}

	// Stamp every step with the caller identity for role-based exemptions.
	for i := range in.Steps {
		in.Steps[i].UserID = in.UserID
		in.Steps[i].Roles = in.Roles
	}

	uc.log.Info("[Aegis] evaluating plan",
		slog.String("session_id", in.SessionID),
		slog.String("plan_id", in.PlanID),
		slog.String("user_id", in.UserID),
		slog.Int("steps", len(in.Steps)),
	)

	decisions, err := uc.evaluator.Evaluate(ctx, in.SessionID, in.PlanID, in.Steps)
	if err != nil {
		return EvaluatePlanOutput{}, fmt.Errorf("policy evaluation failed: %w", err)
	}

	// For every pending decision, persist an approval record.
	denied := 0
	for i, d := range decisions {
		switch d.Verdict {
		case domain.VerdictPending:
			step := in.Steps[i]
			approval, err := uc.store.Create(ctx, domain.Approval{
				SessionID:   in.SessionID,
				PlanID:      in.PlanID,
				StepIndex:   d.StepIndex,
				Description: step.Description,
				Executor:    step.Executor,
				Command:     step.Command,
				Reason:      d.Reason,
				UserID:      in.UserID,
			})
			if err != nil {
				uc.log.Warn("[Aegis] failed to persist approval record",
					slog.Int("step", d.StepIndex),
					slog.String("error", err.Error()),
				)
			} else {
				decisions[i].ApprovalID = approval.ApprovalID
			}
			denied++
			_ = uc.telemetry.Emit(ctx, ports.Event{
				Name: "aegis.step.pending",
				Payload: map[string]any{
					"session_id": in.SessionID,
					"plan_id":    in.PlanID,
					"step_index": d.StepIndex,
					"reason":     d.Reason,
				},
			})
		case domain.VerdictDeny:
			denied++
			_ = uc.telemetry.Emit(ctx, ports.Event{
				Name: "aegis.step.denied",
				Payload: map[string]any{
					"session_id": in.SessionID,
					"plan_id":    in.PlanID,
					"step_index": d.StepIndex,
					"reason":     d.Reason,
				},
			})
		}
	}

	uc.log.Info("[Aegis] plan evaluated",
		slog.String("session_id", in.SessionID),
		slog.String("plan_id", in.PlanID),
		slog.Int("total", len(decisions)),
		slog.Int("denied_or_pending", denied),
	)

	return EvaluatePlanOutput{Decisions: decisions}, nil
}
