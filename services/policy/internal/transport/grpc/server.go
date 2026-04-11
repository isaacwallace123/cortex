package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	policyv1 "github.com/isaacwallace123/cortex/services/policy/gen/policy/v1"
	"github.com/isaacwallace123/cortex/services/policy/internal/application"
	"github.com/isaacwallace123/cortex/services/policy/internal/domain"
)

// metaVal returns the first value of a gRPC incoming-metadata key, or "".
func metaVal(ctx context.Context, key string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(key); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

// metaVals returns all values of a gRPC incoming-metadata key.
func metaVals(ctx context.Context, key string) []string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		return md.Get(key)
	}
	return nil
}

// Server implements the gRPC PolicyService (Aegis).
type Server struct {
	policyv1.UnimplementedPolicyServiceServer

	evaluatePlan   *application.EvaluatePlanUseCase
	submitApproval *application.SubmitApprovalUseCase
	listApprovals  *application.ListApprovalsUseCase
}

func NewServer(
	evaluatePlan *application.EvaluatePlanUseCase,
	submitApproval *application.SubmitApprovalUseCase,
	listApprovals *application.ListApprovalsUseCase,
) *Server {
	return &Server{
		evaluatePlan:   evaluatePlan,
		submitApproval: submitApproval,
		listApprovals:  listApprovals,
	}
}

func (s *Server) EvaluatePlan(ctx context.Context, req *policyv1.EvaluatePlanRequest) (*policyv1.EvaluatePlanResponse, error) {
	// Accept user_id and roles from either the proto fields or gRPC metadata
	// (Brain sends them via metadata; future callers may use proto fields).
	userID := req.GetUserId()
	if userID == "" {
		userID = metaVal(ctx, "x-user-id")
	}
	roles := req.GetRoles()
	if len(roles) == 0 {
		roles = metaVals(ctx, "x-roles")
	}

	steps := make([]domain.StepInput, 0, len(req.GetSteps()))
	for _, st := range req.GetSteps() {
		steps = append(steps, domain.StepInput{
			Index:       int(st.GetIndex()),
			Description: st.GetDescription(),
			Executor:    st.GetExecutor(),
			Command:     st.GetCommand(),
		})
	}

	out, err := s.evaluatePlan.Execute(ctx, application.EvaluatePlanInput{
		SessionID: req.GetSessionId(),
		PlanID:    req.GetPlanId(),
		Steps:     steps,
		UserID:    userID,
		Roles:     roles,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "evaluate_plan: %v", err)
	}

	decisions := make([]*policyv1.StepDecision, 0, len(out.Decisions))
	for _, d := range out.Decisions {
		decisions = append(decisions, &policyv1.StepDecision{
			StepIndex:  int32(d.StepIndex),
			Verdict:    string(d.Verdict),
			Reason:     d.Reason,
			ApprovalId: d.ApprovalID,
		})
	}
	return &policyv1.EvaluatePlanResponse{Decisions: decisions}, nil
}

func (s *Server) SubmitApproval(ctx context.Context, req *policyv1.SubmitApprovalRequest) (*policyv1.SubmitApprovalResponse, error) {
	userID := req.GetUserId()
	if userID == "" {
		userID = metaVal(ctx, "x-user-id")
	}

	out, err := s.submitApproval.Execute(ctx, application.SubmitApprovalInput{
		ApprovalID: req.GetApprovalId(),
		UserID:     userID,
		Decision:   req.GetDecision(),
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "submit_approval: %v", err)
	}
	return &policyv1.SubmitApprovalResponse{Ok: true, Verdict: out.Verdict}, nil
}

func (s *Server) ListPendingApprovals(ctx context.Context, req *policyv1.ListPendingApprovalsRequest) (*policyv1.ListPendingApprovalsResponse, error) {
	userID := req.GetUserId()
	if userID == "" {
		userID = metaVal(ctx, "x-user-id")
	}

	out, err := s.listApprovals.Execute(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_pending_approvals: %v", err)
	}

	approvals := make([]*policyv1.Approval, 0, len(out.Approvals))
	for _, a := range out.Approvals {
		pa := &policyv1.Approval{
			ApprovalId:  a.ApprovalID,
			SessionId:   a.SessionID,
			PlanId:      a.PlanID,
			StepIndex:   int32(a.StepIndex),
			Description: a.Description,
			Executor:    a.Executor,
			Command:     a.Command,
			Reason:      a.Reason,
			UserId:      a.UserID,
			Status:      string(a.Status),
			CreatedAt:   a.CreatedAt.Unix(),
		}
		if a.ResolvedAt != nil {
			pa.ResolvedAt = a.ResolvedAt.Unix()
		}
		approvals = append(approvals, pa)
	}
	return &policyv1.ListPendingApprovalsResponse{Approvals: approvals}, nil
}
