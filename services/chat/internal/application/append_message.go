package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/isaacwallace123/cortex/services/chat/internal/domain"
	"github.com/isaacwallace123/cortex/services/chat/internal/ports"
)

type AppendMessageUseCase struct{ store ports.Store }

func NewAppendMessageUseCase(s ports.Store) *AppendMessageUseCase {
	return &AppendMessageUseCase{store: s}
}

func (uc *AppendMessageUseCase) Execute(ctx context.Context, chatID, role, content string) (domain.Message, error) {
	if chatID == "" {
		return domain.Message{}, fmt.Errorf("chat_id required")
	}
	if role != "user" && role != "assistant" {
		return domain.Message{}, fmt.Errorf("role must be 'user' or 'assistant', got %q", role)
	}
	if content == "" {
		return domain.Message{}, fmt.Errorf("content required")
	}
	now := time.Now().UnixMilli()
	msg := domain.Message{
		MessageID: uuid.New().String(),
		ChatID:    chatID,
		Role:      role,
		Content:   content,
		CreatedAt: now,
	}
	if err := uc.store.AppendMessage(ctx, msg); err != nil {
		return domain.Message{}, fmt.Errorf("append message: %w", err)
	}
	_ = uc.store.UpdateLastMessageAt(ctx, chatID, now)
	return msg, nil
}
