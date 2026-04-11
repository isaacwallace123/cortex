package atlas

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	atlasv1 "github.com/isaacwallace123/cortex/services/atlas/gen/atlas/v1"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

// GRPCAdapter connects Brain to Atlas over gRPC.
type GRPCAdapter struct {
	inner atlasv1.AtlasServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial atlas at %s: %w", addr, err)
	}
	return &GRPCAdapter{inner: atlasv1.NewAtlasServiceClient(conn)}, nil
}

func (a *GRPCAdapter) Execute(ctx context.Context, command string, timeoutSecs int) (ports.AtlasResult, error) {
	resp, err := a.inner.Execute(ctx, &atlasv1.ExecuteRequest{
		Command:     command,
		TimeoutSecs: int32(timeoutSecs),
	})
	if err != nil {
		return ports.AtlasResult{}, fmt.Errorf("atlas.Execute: %w", err)
	}
	return ports.AtlasResult{
		Stdout:     resp.Stdout,
		Stderr:     resp.Stderr,
		ExitCode:   resp.ExitCode,
		DurationMs: resp.DurationMs,
	}, nil
}
