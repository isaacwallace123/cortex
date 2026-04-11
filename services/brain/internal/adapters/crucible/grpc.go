package crucible

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	cruciblev1 "github.com/isaacwallace123/cortex/services/crucible/gen/crucible/v1"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

// GRPCAdapter connects Brain to Crucible over gRPC.
type GRPCAdapter struct {
	inner cruciblev1.CrucibleServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial crucible at %s: %w", addr, err)
	}
	return &GRPCAdapter{inner: cruciblev1.NewCrucibleServiceClient(conn)}, nil
}

func (a *GRPCAdapter) RunCode(ctx context.Context, runtime, command string, timeoutSecs int) (ports.CrucibleResult, error) {
	resp, err := a.inner.RunCode(ctx, &cruciblev1.RunCodeRequest{
		Runtime:     runtime,
		Command:     command,
		TimeoutSecs: int32(timeoutSecs),
	})
	if err != nil {
		return ports.CrucibleResult{}, fmt.Errorf("crucible.RunCode: %w", err)
	}
	return ports.CrucibleResult{
		Stdout:     resp.Stdout,
		Stderr:     resp.Stderr,
		ExitCode:   resp.ExitCode,
		DurationMs: resp.DurationMs,
	}, nil
}
