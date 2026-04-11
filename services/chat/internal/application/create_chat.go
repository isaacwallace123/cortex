package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/isaacwallace123/cortex/services/chat/internal/domain"
	"github.com/isaacwallace123/cortex/services/chat/internal/ports"
)

type CreateChatUseCase struct{ store ports.Store }

func NewCreateChatUseCase(s ports.Store) *CreateChatUseCase { return &CreateChatUseCase{store: s} }

func (uc *CreateChatUseCase) Execute(ctx context.Context, userID, workspaceID, title string) (domain.Chat, error) {
	if userID == "" {
		return domain.Chat{}, fmt.Errorf("user_id required")
	}
	if title == "" {
		title = "New Chat"
	}
	chat := domain.Chat{
		ChatID:      uuid.New().String(),
		UserID:      userID,
		WorkspaceID: workspaceID,
		Title:       title,
		CreatedAt:   time.Now().UnixMilli(),
	}
	if err := uc.store.CreateChat(ctx, chat); err != nil {
		return domain.Chat{}, fmt.Errorf("create chat: %w", err)
	}
	return chat, nil
}
