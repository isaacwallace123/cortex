// Package arsenal provides a gRPC adapter for the Arsenal tool registry.
package arsenal

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
	arsenalv1 "github.com/isaacwallace123/cortex/services/arsenal/gen/arsenal/v1"
)

// GRPCAdapter implements ArsenalPort via the Arsenal gRPC service.
type GRPCAdapter struct {
	client arsenalv1.ArsenalServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial arsenal at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: arsenalv1.NewArsenalServiceClient(conn)}, nil
}

func (a *GRPCAdapter) ListTools(ctx context.Context) ([]ports.RegistryTool, error) {
	resp, err := a.client.ListTools(ctx, &arsenalv1.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("arsenal.ListTools: %w", err)
	}
	tools := make([]ports.RegistryTool, 0, len(resp.GetTools()))
	for _, t := range resp.GetTools() {
		tools = append(tools, ports.RegistryTool{
			Name:        t.GetName(),
			Description: t.GetDescription(),
			Executor:    t.GetExecutor(),
			Example:     t.GetExample(),
			SafetyLevel: t.GetSafetyLevel(),
			Tags:        t.GetTags(),
		})
	}
	return tools, nil
}
