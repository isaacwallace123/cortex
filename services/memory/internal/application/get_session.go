package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/isaacwallace123/cortex/services/memory/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/memory/internal/log"
	"github.com/isaacwallace123/cortex/services/memory/internal/ports"
)

type GetSessionUseCase struct {
	store ports.StorePort
	log   *slog.Logger
}

func NewGetSessionUseCase(store ports.StorePort) *GetSessionUseCase {
	return &GetSessionUseCase{store: store, log: cortexlog.Echo()}
}

type GetSessionOutput struct {
	SessionID string
	Events    []domain.Event
}

func (uc *GetSessionUseCase) Execute(ctx context.Context, sessionID, userID string) (GetSessionOutput, error) {
	if sessionID == "" {
		return GetSessionOutput{}, fmt.Errorf("session_id must not be empty")
	}

	events, err := uc.store.GetSession(ctx, sessionID, userID)
	if err != nil {
		return GetSessionOutput{}, fmt.Errorf("get session: %w", err)
	}

	uc.log.Info("[Echo] session retrieved",
		slog.String("session_id", sessionID),
		slog.Int("events", len(events)),
	)

	return GetSessionOutput{SessionID: sessionID, Events: events}, nil
}
