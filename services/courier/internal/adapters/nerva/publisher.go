// Package nerva provides an EventPublisher adapter that publishes to the Nerva event bus.
package nerva

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/isaacwallace123/cortex/pkg/observe"
	nervav1 "github.com/isaacwallace123/cortex/services/nerva/gen/nerva/v1"
)

// Publisher implements application.EventPublisher via gRPC to Nerva.
type Publisher struct {
	client nervav1.NervaServiceClient
}

func NewPublisher(addr string) (*Publisher, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial nerva at %s: %w", addr, err)
	}
	return &Publisher{client: nervav1.NewNervaServiceClient(conn)}, nil
}

func (p *Publisher) Publish(subsystem, eventName, payload string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := p.client.Publish(ctx, &nervav1.PublishRequest{
		Subsystem: subsystem,
		EventName: eventName,
		Payload:   payload,
	})
	return err
}

// NoopPublisher satisfies EventPublisher when Nerva is unavailable.
type NoopPublisher struct{}

func (NoopPublisher) Publish(_, _, _ string) error { return nil }
