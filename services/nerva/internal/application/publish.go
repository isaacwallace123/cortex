package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/isaacwallace123/cortex/services/nerva/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/nerva/internal/log"
	"github.com/isaacwallace123/cortex/services/nerva/internal/ports"
)

// PublishUseCase accepts an event, persists it, and fans it out via the Bus.
type PublishUseCase struct {
	store ports.StorePort
	bus   *Bus
	log   *slog.Logger
}

func NewPublishUseCase(store ports.StorePort, bus *Bus) *PublishUseCase {
	return &PublishUseCase{store: store, bus: bus, log: cortexlog.Nerva()}
}

type PublishInput struct {
	Subsystem string
	EventName string
	SessionID string
	Payload   string
}

type PublishOutput struct {
	EventID   string
	Timestamp int64
}

func (uc *PublishUseCase) Execute(ctx context.Context, in PublishInput) (PublishOutput, error) {
	if in.EventName == "" {
		return PublishOutput{}, fmt.Errorf("event_name must not be empty")
	}

	event := &domain.Event{
		ID:        uuid.NewString(),
		Subsystem: in.Subsystem,
		Name:      in.EventName,
		SessionID: in.SessionID,
		Payload:   in.Payload,
		CreatedAt: time.Now().UTC(),
	}

	if err := uc.store.Save(ctx, *event); err != nil {
		uc.log.Warn("[Nerva] failed to persist event",
			slog.String("event", event.Name),
			slog.String("error", err.Error()),
		)
		// Continue — persistence failure must not drop the live event.
	}

	uc.bus.Publish(event)

	uc.log.Info("[Nerva] event published",
		slog.String("event_id", event.ID),
		slog.String("subsystem", event.Subsystem),
		slog.String("event", event.Name),
		slog.String("session_id", event.SessionID),
	)

	return PublishOutput{EventID: event.ID, Timestamp: event.CreatedAt.UnixMilli()}, nil
}
