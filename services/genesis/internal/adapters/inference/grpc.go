// Package inference provides a gRPC adapter for InferencePort.
package inference

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	inferencev1 "github.com/isaacwallace123/cortex/services/inference/gen/inference/v1"
)

// GRPCAdapter calls the Inference service to generate completions.
type GRPCAdapter struct {
	client inferencev1.InferenceServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial inference at %s: %w", addr, err)
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
