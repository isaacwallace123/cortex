package domain

// Plan is a structured sequence of steps produced by Axiom (reasoning/planning).
// Each step names the action and which executor should carry it out.
type Plan struct {
	ID        string
	SessionID string
	Steps     []Step
}

// Step is a single action inside a Plan.
type Step struct {
	Index       int
	Description string
	Executor    string // e.g. "shell", "filesystem", "courier", "infer"
	Command     string // concrete command, set by Axiom during planning
	AgentID     string // for executor:courier — device_name or agent_id of the target agent
	Verdict     string // "allow" | "deny" | "pending" — set by Aegis after evaluation
	ApprovalID  string // non-empty when Verdict == "pending"; references the persisted Aegis approval
}

// StepDecision is Aegis's ruling on a single step, carried back through PolicyPort.
type StepDecision struct {
	StepIndex  int
	Verdict    string // "allow" | "deny" | "pending"
	Reason     string
	ApprovalID string // non-empty when Verdict == "pending"; references persisted approval
}

// ExecutionResult is what Shell (or another executor) returns after running a step.
type ExecutionResult struct {
	StepIndex  int
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
	Skipped    bool   // true if step was not executed (verdict deny, unsupported executor, no command)
	SkipReason string // why it was skipped
}
