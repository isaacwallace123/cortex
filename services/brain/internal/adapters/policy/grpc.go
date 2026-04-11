package policy

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
	policyv1 "github.com/isaacwallace123/cortex/services/policy/gen/policy/v1"
)

// metaVals returns all values for a gRPC incoming-metadata key.
func metaVals(ctx context.Context, key string) []string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		return md.Get(key)
	}
	return nil
}

// GRPCAdapter calls the Aegis policy service over gRPC.
type GRPCAdapter struct {
	client policyv1.PolicyServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial policy service at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: policyv1.NewPolicyServiceClient(conn)}, nil
}

func (a *GRPCAdapter) EvaluatePlan(ctx context.Context, sessionID, planID string, steps []domain.Step) ([]domain.StepDecision, error) {
	// Extract caller identity from incoming gRPC metadata (set by Brain's gRPC server
	// which received them from the API gateway as x-user-id / x-roles).
	userID := ""
	var roles []string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-user-id"); len(vals) > 0 {
			userID = vals[0]
		}
		roles = md.Get("x-roles")
	}

	protoSteps := make([]*policyv1.PlanStep, 0, len(steps))
	for _, s := range steps {
		protoSteps = append(protoSteps, &policyv1.PlanStep{
			Index:       int32(s.Index),
			Description: s.Description,
			Executor:    s.Executor,
			Command:     s.Command,
		})
	}

	resp, err := a.client.EvaluatePlan(ctx, &policyv1.EvaluatePlanRequest{
		SessionId: sessionID,
		PlanId:    planID,
		Steps:     protoSteps,
		UserId:    userID,
		Roles:     roles,
	})
	if err != nil {
		return nil, fmt.Errorf("policy.EvaluatePlan: %w", err)
	}

	decisions := make([]domain.StepDecision, 0, len(resp.GetDecisions()))
	for _, d := range resp.GetDecisions() {
		decisions = append(decisions, domain.StepDecision{
			StepIndex:  int(d.GetStepIndex()),
			Verdict:    d.GetVerdict(),
			Reason:     d.GetReason(),
			ApprovalID: d.GetApprovalId(),
		})
	}
	return decisions, nil
}
