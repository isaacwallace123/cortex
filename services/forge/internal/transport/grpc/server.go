package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	forgev1 "github.com/isaacwallace123/cortex/services/forge/gen/forge/v1"
	"github.com/isaacwallace123/cortex/services/forge/internal/application"
)

// Server implements the gRPC ForgeService.
type Server struct {
	forgev1.UnimplementedForgeServiceServer
	executeStep *application.ExecuteStepUseCase
}

func NewServer(executeStep *application.ExecuteStepUseCase) *Server {
	return &Server{executeStep: executeStep}
}

func (s *Server) ExecuteStep(ctx context.Context, req *forgev1.ForgeStepRequest) (*forgev1.ForgeStepResponse, error) {
	out, err := s.executeStep.Execute(ctx, application.ExecuteStepInput{
		SessionID: req.GetSessionId(),
		PlanID:    req.GetPlanId(),
		StepIndex: int(req.GetStepIndex()),
		Command:   req.GetCommand(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "forge.ExecuteStep: %v", err)
	}

	return &forgev1.ForgeStepResponse{
		Output:     out.Result.Output,
		Ok:         out.Result.OK,
		DurationMs: out.Result.DurationMs,
	}, nil
}
