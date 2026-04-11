package telemetry

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/isaacwallace123/cortex/services/policy/internal/ports"
)

// LogAdapter emits telemetry events as structured log lines.
// This is the development-mode adapter; replace with a Nerva adapter for production.
type LogAdapter struct {
	logger *slog.Logger
}

func NewLogAdapter(logger *slog.Logger) *LogAdapter {
	return &LogAdapter{logger: logger}
}

func (a *LogAdapter) Emit(_ context.Context, event ports.Event) error {
	payloadJSON, _ := json.Marshal(event.Payload)
	a.logger.Info("[Nerva] event logged locally (no bus)",
		slog.String("event", event.Name),
		slog.String("payload", string(payloadJSON)),
	)
	return nil
}
