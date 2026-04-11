package inference

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/isaacwallace123/cortex/pkg/observe"
	inferencev1 "github.com/isaacwallace123/cortex/services/inference/gen/inference/v1"
)

// GRPCAdapter calls the inference service over gRPC.
// This replaces the StubAdapter for production use.
type GRPCAdapter struct {
	client inferencev1.InferenceServiceClient
}

// NewGRPCAdapter dials the inference service and returns a ready adapter.
// addr is the gRPC address, e.g. "localhost:9090".
func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial inference service at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: inferencev1.NewInferenceServiceClient(conn)}, nil
}

func (a *GRPCAdapter) Complete(ctx context.Context, prompt string, temperature float32) (string, error) {
	resp, err := a.client.Complete(ctx, &inferencev1.CompleteRequest{
		Prompt:      prompt,
		Temperature: temperature,
	})
	if err != nil {
		return "", fmt.Errorf("inference.Complete rpc: %w", err)
	}
	return resp.GetText(), nil
}

func (a *GRPCAdapter) CompleteStream(ctx context.Context, prompt string, temperature float32) (<-chan string, error) {
	stream, err := a.client.CompleteStream(ctx, &inferencev1.CompleteRequest{
		Prompt:      prompt,
		Temperature: temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("inference.CompleteStream rpc: %w", err)
	}

	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		for {
			chunk, err := stream.Recv()
			if err != nil {
				return
			}
			if chunk.GetToken() != "" {
				select {
				case <-ctx.Done():
					return
				case ch <- chunk.GetToken():
				}
			}
			if chunk.GetDone() {
				return
			}
		}
	}()
	return ch, nil
}
