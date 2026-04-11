package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/chat/internal/ports"
)

type DeleteChatUseCase struct{ store ports.Store }

func NewDeleteChatUseCase(s ports.Store) *DeleteChatUseCase { return &DeleteChatUseCase{store: s} }

func (uc *DeleteChatUseCase) Execute(ctx context.Context, chatID, userID string) (bool, error) {
	if chatID == "" {
		return false, fmt.Errorf("chat_id required")
	}
	return uc.store.DeleteChat(ctx, chatID, userID)
}
