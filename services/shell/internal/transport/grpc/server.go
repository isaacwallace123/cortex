package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	shellv1 "github.com/isaacwallace123/cortex/services/shell/gen/shell/v1"
	"github.com/isaacwallace123/cortex/services/shell/internal/application"
)

// Server implements the gRPC ShellService.
type Server struct {
	shellv1.UnimplementedShellServiceServer
	executeStep *application.ExecuteStepUseCase
}

func NewServer(executeStep *application.ExecuteStepUseCase) *Server {
	return &Server{executeStep: executeStep}
}

func (s *Server) ExecuteStep(ctx context.Context, req *shellv1.ExecuteStepRequest) (*shellv1.ExecuteStepResponse, error) {
	out, err := s.executeStep.Execute(ctx, application.ExecuteStepInput{
		SessionID: req.GetSessionId(),
		PlanID:    req.GetPlanId(),
		StepIndex: int(req.GetStepIndex()),
		Command:   req.GetCommand(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "execute_step: %v", err)
	}
	return &shellv1.ExecuteStepResponse{
		Stdout:     out.Result.Stdout,
		Stderr:     out.Result.Stderr,
		ExitCode:   int32(out.Result.ExitCode),
		DurationMs: out.Result.DurationMs,
	}, nil
}
