package memory

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
	memoryv1 "github.com/isaacwallace123/cortex/services/memory/gen/memory/v1"
)

// forwardMeta propagates selected keys from incoming gRPC metadata to the outgoing context.
func forwardMeta(ctx context.Context, keys ...string) context.Context {
	in, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	var pairs []string
	for _, k := range keys {
		if vals := in.Get(k); len(vals) > 0 {
			pairs = append(pairs, k, vals[0])
		}
	}
	if len(pairs) == 0 {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, pairs...)
}

// GRPCAdapter calls the memory service over gRPC.
type GRPCAdapter struct {
	client memoryv1.MemoryServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial memory service at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: memoryv1.NewMemoryServiceClient(conn)}, nil
}

func (a *GRPCAdapter) StoreEvent(ctx context.Context, sessionID, eventName, payload string) error {
	ctx = forwardMeta(ctx, "x-user-id", "x-workspace-id")
	_, err := a.client.StoreEvent(ctx, &memoryv1.StoreEventRequest{
		SessionId: sessionID,
		EventName: eventName,
		Payload:   payload,
	})
	if err != nil {
		return fmt.Errorf("memory.StoreEvent rpc: %w", err)
	}
	return nil
}

func (a *GRPCAdapter) GetSession(ctx context.Context, sessionID string) ([]ports.StoredEvent, error) {
	ctx = forwardMeta(ctx, "x-user-id")
	resp, err := a.client.GetSession(ctx, &memoryv1.GetSessionRequest{
		SessionId: sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("memory.GetSession rpc: %w", err)
	}

	events := make([]ports.StoredEvent, 0, len(resp.GetEvents()))
	for _, e := range resp.GetEvents() {
		events = append(events, ports.StoredEvent{
			ID:       e.GetEventId(),
			Name:     e.GetEventName(),
			Payload:  e.GetPayload(),
			StoredAt: time.Unix(e.GetStoredAt(), 0).UTC(),
		})
	}
	return events, nil
}
