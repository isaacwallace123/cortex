package ports

import "context"

// StepToExecute carries the parameters for a single expanded plan step.
type StepToExecute struct {
	Index       int
	Description string
	Executor    string
	Command     string
	Agent       string
}

// StepExecutionResult is the result of a single executed step.
type StepExecutionResult struct {
	StepIndex int
	Stdout    string
	Stderr    string
	ExitCode  int32
	Skipped   bool
}

// BrainPort is the execution gateway — Catalyst delegates step execution to
// Brain's ExecutePlan RPC, which owns the executor routing logic (Shell/Forge/
// Atlas/Crucible/Courier/Web/Infer).
type BrainPort interface {
	ExecuteSteps(ctx context.Context, sessionID, planID string, steps []StepToExecute) ([]StepExecutionResult, error)
}
