// Package brain provides a gRPC adapter for the BrainPort.
// It calls Brain.ExecutePlan to delegate step execution to the routing layer
// (Vector) that already knows how to dispatch to Shell/Forge/Atlas/etc.
package brain

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	brainv1 "github.com/isaacwallace123/cortex/services/brain/gen/brain/v1"
	"github.com/isaacwallace123/cortex/services/catalyst/internal/ports"
)

// GRPCAdapter implements BrainPort via the Brain gRPC service.
type GRPCAdapter struct {
	client brainv1.BrainServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial brain at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: brainv1.NewBrainServiceClient(conn)}, nil
}

func (a *GRPCAdapter) ExecuteSteps(ctx context.Context, sessionID, planID string, steps []ports.StepToExecute) ([]ports.StepExecutionResult, error) {
	protoSteps := make([]*brainv1.Step, 0, len(steps))
	for _, s := range steps {
		protoSteps = append(protoSteps, &brainv1.Step{
			Index:       int32(s.Index),
			Description: s.Description,
			Executor:    s.Executor,
			Command:     s.Command,
			AgentId:     s.Agent,
		})
	}

	resp, err := a.client.ExecutePlan(ctx, &brainv1.ExecutePlanRequest{
		SessionId: sessionID,
		PlanId:    planID,
		Steps:     protoSteps,
	})
	if err != nil {
		return nil, fmt.Errorf("brain.ExecutePlan: %w", err)
	}

	results := make([]ports.StepExecutionResult, 0, len(resp.GetResults()))
	for _, r := range resp.GetResults() {
		results = append(results, ports.StepExecutionResult{
			StepIndex: int(r.GetStepIndex()),
			Stdout:    r.GetStdout(),
			Stderr:    r.GetStderr(),
			ExitCode:  r.GetExitCode(),
			Skipped:   r.GetSkipped(),
		})
	}
	return results, nil
}
