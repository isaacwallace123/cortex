// Package nerva provides a TelemetryPort adapter that publishes events to the Nerva event bus.
package nerva

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
	nervav1 "github.com/isaacwallace123/cortex/services/nerva/gen/nerva/v1"
)

// GRPCAdapter implements TelemetryPort by publishing to the Nerva event bus over gRPC.
type GRPCAdapter struct {
	client nervav1.NervaServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial nerva at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: nervav1.NewNervaServiceClient(conn)}, nil
}

// Emit sends the event to Nerva. The subsystem is inferred from the event name prefix.
// "prism.input.parsed" → subsystem = "Prism"
func (a *GRPCAdapter) Emit(ctx context.Context, event ports.Event) error {
	subsystem := subsystemFromName(event.Name)
	sessionID, _ := event.Payload["session_id"].(string)

	payloadJSON, _ := json.Marshal(event.Payload)

	_, err := a.client.Publish(ctx, &nervav1.PublishRequest{
		Subsystem: subsystem,
		EventName: event.Name,
		SessionId: sessionID,
		Payload:   string(payloadJSON),
	})
	if err != nil {
		return fmt.Errorf("nerva.Publish: %w", err)
	}
	return nil
}

// subsystemFromName extracts the subsystem display name from a dot-separated event name.
// "prism.input.parsed" → "Prism", "vector.step.executed" → "Vector"
func subsystemFromName(name string) string {
	if idx := strings.Index(name, "."); idx > 0 {
		part := name[:idx]
		if len(part) > 0 {
			return strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return "Brain"
}
