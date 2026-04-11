package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

// GetSessionUseCase retrieves the event history for a session from memory.
type GetSessionUseCase struct {
	memory ports.MemoryPort
}

func NewGetSessionUseCase(memory ports.MemoryPort) *GetSessionUseCase {
	return &GetSessionUseCase{memory: memory}
}

type GetSessionOutput struct {
	SessionID string
	Events    []ports.StoredEvent
}

func (uc *GetSessionUseCase) Execute(ctx context.Context, sessionID string) (GetSessionOutput, error) {
	if sessionID == "" {
		return GetSessionOutput{}, fmt.Errorf("session_id must not be empty")
	}
	events, err := uc.memory.GetSession(ctx, sessionID)
	if err != nil {
		return GetSessionOutput{}, fmt.Errorf("get session from memory: %w", err)
	}
	return GetSessionOutput{SessionID: sessionID, Events: events}, nil
}
