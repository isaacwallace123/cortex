package domain

// Verdict is the outcome of an Aegis policy evaluation for a single step.
type Verdict string

const (
	VerdictAllow   Verdict = "allow"
	VerdictDeny    Verdict = "deny"
	VerdictPending Verdict = "pending"
)

// StepInput is the plan step data Aegis needs to make a decision.
type StepInput struct {
	Index       int
	Description string
	Executor    string
	Command     string   // concrete shell command, for pattern-based rules
	UserID      string   // authenticated caller
	Roles       []string // caller's roles — used for role-based exemptions
}

// StepDecision is Aegis's ruling on a single plan step.
type StepDecision struct {
	StepIndex  int
	Verdict    Verdict
	Reason     string
	ApprovalID string // non-empty when Verdict == VerdictPending
}
