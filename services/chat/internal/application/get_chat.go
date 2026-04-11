package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/chat/internal/domain"
	"github.com/isaacwallace123/cortex/services/chat/internal/ports"
)

type GetChatUseCase struct{ store ports.Store }

func NewGetChatUseCase(s ports.Store) *GetChatUseCase { return &GetChatUseCase{store: s} }

func (uc *GetChatUseCase) Execute(ctx context.Context, chatID, userID string) (domain.Chat, error) {
	if chatID == "" {
		return domain.Chat{}, fmt.Errorf("chat_id required")
	}
	return uc.store.GetChat(ctx, chatID, userID)
}
