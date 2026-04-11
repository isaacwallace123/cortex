// Package policy provides a gRPC client for the Aegis policy service.
package policy

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	policyv1 "github.com/isaacwallace123/cortex/services/policy/gen/policy/v1"
)

// Client wraps the Aegis PolicyService gRPC client.
type Client struct {
	inner policyv1.PolicyServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial policy at %s: %w", addr, err)
	}
	return &Client{inner: policyv1.NewPolicyServiceClient(conn)}, nil
}

// Approval is a pending or resolved Aegis approval record.
type Approval struct {
	ApprovalID  string    `json:"approval_id"`
	SessionID   string    `json:"session_id"`
	PlanID      string    `json:"plan_id"`
	StepIndex   int32     `json:"step_index"`
	Description string    `json:"description"`
	Executor    string    `json:"executor"`
	Command     string    `json:"command"`
	Reason      string    `json:"reason"`
	UserID      string    `json:"user_id"`
	Status      string    `json:"status"` // "pending" | "approved" | "denied"
	CreatedAt   time.Time `json:"created_at"`
	ResolvedAt  time.Time `json:"resolved_at,omitempty"`
}

// ListPendingApprovals returns unresolved approvals. When userID is empty,
// the policy service returns all pending approvals (admin view).
func (c *Client) ListPendingApprovals(ctx context.Context, userID string) ([]Approval, error) {
	resp, err := c.inner.ListPendingApprovals(ctx, &policyv1.ListPendingApprovalsRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("policy.ListPendingApprovals: %w", err)
	}
	out := make([]Approval, 0, len(resp.GetApprovals()))
	for _, a := range resp.GetApprovals() {
		out = append(out, protoApproval(a))
	}
	return out, nil
}

// SubmitApproval resolves an approval record. decision must be "approved" or "denied".
func (c *Client) SubmitApproval(ctx context.Context, approvalID, userID, decision string) (bool, string, error) {
	resp, err := c.inner.SubmitApproval(ctx, &policyv1.SubmitApprovalRequest{
		ApprovalId: approvalID,
		UserId:     userID,
		Decision:   decision,
	})
	if err != nil {
		return false, "", fmt.Errorf("policy.SubmitApproval: %w", err)
	}
	return resp.GetOk(), resp.GetVerdict(), nil
}

func protoApproval(a *policyv1.Approval) Approval {
	if a == nil {
		return Approval{}
	}
	var resolvedAt time.Time
	if a.GetResolvedAt() != 0 {
		resolvedAt = time.Unix(a.GetResolvedAt(), 0).UTC()
	}
	return Approval{
		ApprovalID:  a.GetApprovalId(),
		SessionID:   a.GetSessionId(),
		PlanID:      a.GetPlanId(),
		StepIndex:   a.GetStepIndex(),
		Description: a.GetDescription(),
		Executor:    a.GetExecutor(),
		Command:     a.GetCommand(),
		Reason:      a.GetReason(),
		UserID:      a.GetUserId(),
		Status:      a.GetStatus(),
		CreatedAt:   time.Unix(a.GetCreatedAt(), 0).UTC(),
		ResolvedAt:  resolvedAt,
	}
}
