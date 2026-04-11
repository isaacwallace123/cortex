package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/isaacwallace123/cortex/services/shell/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/shell/internal/log"
	"github.com/isaacwallace123/cortex/services/shell/internal/ports"
)

const defaultTimeout = 30 * time.Second

// ExecuteStepUseCase runs a single approved shell command.
type ExecuteStepUseCase struct {
	runner ports.RunnerPort
	log    *slog.Logger
}

func NewExecuteStepUseCase(runner ports.RunnerPort) *ExecuteStepUseCase {
	return &ExecuteStepUseCase{runner: runner, log: cortexlog.Shell()}
}

type ExecuteStepInput struct {
	SessionID string
	PlanID    string
	StepIndex int
	Command   string
}

type ExecuteStepOutput struct {
	Result domain.RunResult
}

func (uc *ExecuteStepUseCase) Execute(ctx context.Context, in ExecuteStepInput) (ExecuteStepOutput, error) {
	if in.Command == "" {
		return ExecuteStepOutput{}, fmt.Errorf("command must not be empty")
	}

	uc.log.Info("[Shell] executing step",
		slog.String("session_id", in.SessionID),
		slog.String("plan_id", in.PlanID),
		slog.Int("step", in.StepIndex),
		slog.String("command", in.Command),
	)

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	result, err := uc.runner.Run(ctx, in.Command)
	if err != nil {
		return ExecuteStepOutput{}, fmt.Errorf("runner failed: %w", err)
	}

	uc.log.Info("[Shell] step complete",
		slog.String("session_id", in.SessionID),
		slog.Int("step", in.StepIndex),
		slog.Int("exit_code", result.ExitCode),
		slog.Int64("duration_ms", result.DurationMs),
	)

	return ExecuteStepOutput{Result: result}, nil
}
