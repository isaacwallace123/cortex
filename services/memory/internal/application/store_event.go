package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/isaacwallace123/cortex/services/memory/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/memory/internal/log"
	"github.com/isaacwallace123/cortex/services/memory/internal/ports"
)

type StoreEventUseCase struct {
	store ports.StorePort
	log   *slog.Logger
}

func NewStoreEventUseCase(store ports.StorePort) *StoreEventUseCase {
	return &StoreEventUseCase{store: store, log: cortexlog.Echo()}
}

type StoreEventInput struct {
	SessionID   string
	UserID      string
	WorkspaceID string
	EventName   string
	Payload     string
}

type StoreEventOutput struct {
	Event domain.Event
}

func (uc *StoreEventUseCase) Execute(ctx context.Context, in StoreEventInput) (StoreEventOutput, error) {
	if in.SessionID == "" {
		return StoreEventOutput{}, fmt.Errorf("session_id must not be empty")
	}
	if in.EventName == "" {
		return StoreEventOutput{}, fmt.Errorf("event_name must not be empty")
	}

	event := domain.Event{
		ID:          uuid.New().String(),
		SessionID:   in.SessionID,
		UserID:      in.UserID,
		WorkspaceID: in.WorkspaceID,
		Name:        in.EventName,
		Payload:     in.Payload,
		StoredAt:    time.Now().UTC(),
	}

	stored, err := uc.store.StoreEvent(ctx, event)
	if err != nil {
		return StoreEventOutput{}, fmt.Errorf("store event: %w", err)
	}

	uc.log.Info("[Echo] event stored",
		slog.String("session_id", in.SessionID),
		slog.String("event", in.EventName),
		slog.String("event_id", stored.ID),
	)

	return StoreEventOutput{Event: stored}, nil
}
