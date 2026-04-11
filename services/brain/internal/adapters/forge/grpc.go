package forge

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
	forgev1 "github.com/isaacwallace123/cortex/services/forge/gen/forge/v1"
)

// GRPCAdapter calls the Forge filesystem executor service over gRPC.
type GRPCAdapter struct {
	client forgev1.ForgeServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial forge service at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: forgev1.NewForgeServiceClient(conn)}, nil
}

func (a *GRPCAdapter) Execute(ctx context.Context, sessionID, planID string, step domain.Step) (domain.ExecutionResult, error) {
	resp, err := a.client.ExecuteStep(ctx, &forgev1.ForgeStepRequest{
		SessionId: sessionID,
		PlanId:    planID,
		StepIndex: int32(step.Index),
		Command:   step.Command,
	})
	if err != nil {
		return domain.ExecutionResult{}, fmt.Errorf("forge.ExecuteStep: %w", err)
	}

	exitCode := 0
	if !resp.GetOk() {
		exitCode = 1
	}

	return domain.ExecutionResult{
		StepIndex:  step.Index,
		Stdout:     resp.GetOutput(),
		ExitCode:   exitCode,
		DurationMs: resp.GetDurationMs(),
	}, nil
}
