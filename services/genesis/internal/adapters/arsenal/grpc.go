// Package arsenal provides a gRPC adapter for ArsenalPort.
package arsenal

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	arsenalv1 "github.com/isaacwallace123/cortex/services/arsenal/gen/arsenal/v1"
	"github.com/isaacwallace123/cortex/services/genesis/internal/domain"
)

// GRPCAdapter registers generated tools in the Arsenal tool registry.
type GRPCAdapter struct {
	client arsenalv1.ArsenalServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial arsenal at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: arsenalv1.NewArsenalServiceClient(conn)}, nil
}

func (a *GRPCAdapter) RegisterTool(ctx context.Context, tool domain.GeneratedTool) error {
	_, err := a.client.RegisterTool(ctx, &arsenalv1.RegisterToolRequest{
		Tool: &arsenalv1.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			Executor:    tool.Executor,
			Example:     tool.Example,
			SafetyLevel: tool.SafetyLevel,
			Tags:        tool.Tags,
			Enabled:     true,
			Version:     tool.Version,
		},
	})
	if err != nil {
		return fmt.Errorf("arsenal.RegisterTool rpc: %w", err)
	}
	return nil
}
