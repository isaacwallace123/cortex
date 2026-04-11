package shell

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
	shellv1 "github.com/isaacwallace123/cortex/services/shell/gen/shell/v1"
)

// GRPCAdapter calls the Shell executor service over gRPC.
type GRPCAdapter struct {
	client shellv1.ShellServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial shell service at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: shellv1.NewShellServiceClient(conn)}, nil
}

func (a *GRPCAdapter) Execute(ctx context.Context, sessionID, planID string, step domain.Step) (domain.ExecutionResult, error) {
	resp, err := a.client.ExecuteStep(ctx, &shellv1.ExecuteStepRequest{
		SessionId: sessionID,
		PlanId:    planID,
		StepIndex: int32(step.Index),
		Command:   step.Command,
	})
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("shell.ExecuteStep: %w", err)
	}
	return domain.ExecutionResult{
		StepIndex:  step.Index,
		Stdout:     resp.GetStdout(),
		Stderr:     resp.GetStderr(),
		ExitCode:   int(resp.GetExitCode()),
		DurationMs: resp.GetDurationMs(),
	}, nil
}
