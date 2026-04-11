package domain

import "time"

// ApprovalStatus tracks the lifecycle of a pending approval.
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalDenied   ApprovalStatus = "denied"
)

// Approval is a persisted record for a step that requires human sign-off.
// Created when Aegis issues a VerdictPending; resolved when the user
// calls SubmitApproval with "approved" or "denied".
type Approval struct {
	ApprovalID  string
	SessionID   string
	PlanID      string
	StepIndex   int
	Description string
	Executor    string
	Command     string
	Reason      string
	UserID      string         // who triggered the step
	Status      ApprovalStatus
	CreatedAt   time.Time
	ResolvedAt  *time.Time // nil while pending
}
