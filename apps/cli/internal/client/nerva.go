package client

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	nervav1 "github.com/isaacwallace123/cortex/services/nerva/gen/nerva/v1"
)

// NervaClient is a streaming gRPC client for the Nerva event bus.
type NervaClient struct {
	client nervav1.NervaServiceClient
}

func NewNervaClient(addr string) (*NervaClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial nerva at %s: %w", addr, err)
	}
	return &NervaClient{client: nervav1.NewNervaServiceClient(conn)}, nil
}

// Event is a single event received from the Nerva stream.
type Event struct {
	EventID   string
	Subsystem string
	EventName string
	SessionID string
	Payload   string
}

// Subscribe opens a server-side streaming subscription to Nerva.
// filter is passed to SubscribeRequest; empty means all events.
// The returned channel is closed when the stream ends or ctx is cancelled.
func (n *NervaClient) Subscribe(ctx context.Context, filter string) (<-chan Event, <-chan error) {
	events := make(chan Event, 32)
	errc := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errc)

		stream, err := n.client.Subscribe(ctx, &nervav1.SubscribeRequest{Filter: filter})
		if err != nil {
			errc <- fmt.Errorf("subscribe: %w", err)
			return
		}

		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				if ctx.Err() != nil {
					return // cancelled — not an error
				}
				errc <- fmt.Errorf("stream recv: %w", err)
				return
			}
			events <- Event{
				EventID:   msg.EventId,
				Subsystem: msg.Subsystem,
				EventName: msg.EventName,
				SessionID: msg.SessionId,
				Payload:   msg.Payload,
			}
		}
	}()

	return events, errc
}
