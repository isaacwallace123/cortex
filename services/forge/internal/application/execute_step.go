package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/isaacwallace123/cortex/services/forge/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/forge/internal/log"
	"github.com/isaacwallace123/cortex/services/forge/internal/ports"
)

// ExecuteStepUseCase performs a single filesystem operation on behalf of Vector.
type ExecuteStepUseCase struct {
	executor ports.FSPort
	log      *slog.Logger
}

func NewExecuteStepUseCase(executor ports.FSPort) *ExecuteStepUseCase {
	return &ExecuteStepUseCase{executor: executor, log: cortexlog.Forge()}
}

type ExecuteStepInput struct {
	SessionID string
	PlanID    string
	StepIndex int
	Command   string
}

type ExecuteStepOutput struct {
	Result domain.OpResult
}

func (uc *ExecuteStepUseCase) Execute(ctx context.Context, in ExecuteStepInput) (ExecuteStepOutput, error) {
	if in.Command == "" {
		return ExecuteStepOutput{}, fmt.Errorf("command must not be empty")
	}

	uc.log.Info("[Forge] executing step",
		slog.String("session_id", in.SessionID),
		slog.String("plan_id", in.PlanID),
		slog.Int("step", in.StepIndex),
		slog.String("command", in.Command),
	)

	result, err := uc.executor.Execute(ctx, in.Command)
	if err != nil {
		return ExecuteStepOutput{}, fmt.Errorf("fs executor failed: %w", err)
	}

	uc.log.Info("[Forge] step complete",
		slog.String("session_id", in.SessionID),
		slog.Int("step", in.StepIndex),
		slog.Bool("ok", result.OK),
		slog.Int64("duration_ms", result.DurationMs),
	)

	return ExecuteStepOutput{Result: result}, nil
}
