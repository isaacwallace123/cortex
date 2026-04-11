package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	memoryv1 "github.com/isaacwallace123/cortex/services/memory/gen/memory/v1"
)

// PlanSummary is a compact plan record returned by ListPlans.
type PlanSummary struct {
	PlanID    string
	SessionID string
	Intent    string
	StepCount int
	CreatedAt time.Time
}

// PlanStep is a single step with its Aegis verdict as stored at plan creation time.
type PlanStep struct {
	Index       int32
	Description string
	Executor    string
	Command     string
	Verdict     string
}

// PlanDetail is a full plan with steps, reconstructed from the stored event payload.
type PlanDetail struct {
	PlanID    string
	SessionID string
	Intent    string
	Steps     []PlanStep
}

// Client is a typed gRPC client for the Echo memory service.
type Client struct {
	inner memoryv1.MemoryServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial memory at %s: %w", addr, err)
	}
	return &Client{inner: memoryv1.NewMemoryServiceClient(conn)}, nil
}

// ListPlans returns plan summaries for a session in chronological order.
func (c *Client) ListPlans(ctx context.Context, sessionID string) ([]PlanSummary, error) {
	resp, err := c.inner.ListPlans(ctx, &memoryv1.ListPlansRequest{SessionId: sessionID})
	if err != nil {
		return nil, fmt.Errorf("memory.ListPlans: %w", err)
	}
	plans := make([]PlanSummary, 0, len(resp.GetPlans()))
	for _, p := range resp.GetPlans() {
		plans = append(plans, PlanSummary{
			PlanID:    p.GetPlanId(),
			SessionID: p.GetSessionId(),
			Intent:    p.GetIntent(),
			StepCount: int(p.GetStepCount()),
			CreatedAt: time.Unix(p.GetCreatedAt(), 0).UTC(),
		})
	}
	return plans, nil
}

// GetPlanDetail fetches the full plan (steps + verdicts) by scanning session events
// for the axiom.plan.created event that matches planID.
func (c *Client) GetPlanDetail(ctx context.Context, sessionID, planID string) (PlanDetail, error) {
	resp, err := c.inner.GetSession(ctx, &memoryv1.GetSessionRequest{SessionId: sessionID})
	if err != nil {
		return PlanDetail{}, fmt.Errorf("memory.GetSession: %w", err)
	}

	for _, e := range resp.GetEvents() {
		if e.GetEventName() != "axiom.plan.created" {
			continue
		}
		var payload struct {
			PlanID string `json:"plan_id"`
			Intent string `json:"intent"`
			Steps  []struct {
				Index       int32  `json:"index"`
				Description string `json:"description"`
				Executor    string `json:"executor"`
				Command     string `json:"command"`
				Verdict     string `json:"verdict"`
			} `json:"steps"`
		}
		if err := json.Unmarshal([]byte(e.GetPayload()), &payload); err != nil {
			continue
		}
		if payload.PlanID != planID {
			continue
		}
		steps := make([]PlanStep, 0, len(payload.Steps))
		for _, s := range payload.Steps {
			steps = append(steps, PlanStep{
				Index:       s.Index,
				Description: s.Description,
				Executor:    s.Executor,
				Command:     s.Command,
				Verdict:     s.Verdict,
			})
		}
		return PlanDetail{
			PlanID:    planID,
			SessionID: sessionID,
			Intent:    payload.Intent,
			Steps:     steps,
		}, nil
	}
	return PlanDetail{}, fmt.Errorf("plan %s not found in session %s", planID, sessionID)
}
