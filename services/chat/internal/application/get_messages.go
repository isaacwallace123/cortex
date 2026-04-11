package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/chat/internal/domain"
	"github.com/isaacwallace123/cortex/services/chat/internal/ports"
)

type GetMessagesUseCase struct{ store ports.Store }

func NewGetMessagesUseCase(s ports.Store) *GetMessagesUseCase { return &GetMessagesUseCase{store: s} }

func (uc *GetMessagesUseCase) Execute(ctx context.Context, chatID string, limit int) ([]domain.Message, error) {
	if chatID == "" {
		return nil, fmt.Errorf("chat_id required")
	}
	return uc.store.GetMessages(ctx, chatID, limit)
}
