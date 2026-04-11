package courier

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
	courierv1 "github.com/isaacwallace123/cortex/services/courier/gen/courier/v1"
)

// GRPCAdapter implements CourierPort by calling the Courier service over gRPC.
type GRPCAdapter struct {
	client courierv1.CourierServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial courier at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: courierv1.NewCourierServiceClient(conn)}, nil
}

func (a *GRPCAdapter) ListAgents(ctx context.Context) ([]ports.RemoteAgent, error) {
	resp, err := a.client.ListAgents(ctx, &courierv1.ListAgentsRequest{})
	if err != nil {
		return nil, fmt.Errorf("courier.ListAgents: %w", err)
	}
	agents := make([]ports.RemoteAgent, 0, len(resp.GetAgents()))
	for _, info := range resp.GetAgents() {
		agents = append(agents, ports.RemoteAgent{
			AgentID:      info.GetAgentId(),
			DeviceName:   info.GetDeviceName(),
			Hostname:     info.GetHostname(),
			OS:           info.GetOs(),
			Arch:         info.GetArch(),
			Capabilities: info.GetCapabilities(),
			Status:       info.GetStatus(),
		})
	}
	return agents, nil
}

func (a *GRPCAdapter) Dispatch(ctx context.Context, _, _ string, step domain.Step) (domain.ExecutionResult, error) {
	resp, err := a.client.Dispatch(ctx, &courierv1.DispatchRequest{
		AgentId: step.AgentID,
		Command: step.Command,
	})
	if err != nil {
		return domain.ExecutionResult{StepIndex: step.Index, ExitCode: -1, Stderr: err.Error()}, fmt.Errorf("courier.Dispatch: %w", err)
	}
	return domain.ExecutionResult{
		StepIndex:  step.Index,
		Stdout:     resp.GetStdout(),
		Stderr:     resp.GetStderr(),
		ExitCode:   int(resp.GetExitCode()),
		DurationMs: resp.GetDurationMs(),
	}, nil
}
