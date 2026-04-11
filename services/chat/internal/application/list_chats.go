package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/chat/internal/domain"
	"github.com/isaacwallace123/cortex/services/chat/internal/ports"
)

type ListChatsUseCase struct{ store ports.Store }

func NewListChatsUseCase(s ports.Store) *ListChatsUseCase { return &ListChatsUseCase{store: s} }

func (uc *ListChatsUseCase) Execute(ctx context.Context, userID, workspaceID string, limit int) ([]domain.Chat, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id required")
	}
	return uc.store.ListChats(ctx, userID, workspaceID, limit)
}
