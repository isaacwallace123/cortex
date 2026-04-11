package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/chat/internal/domain"
)

// Store is the persistence port for the Chat service.
type Store interface {
	CreateChat(ctx context.Context, chat domain.Chat) error
	GetChat(ctx context.Context, chatID, userID string) (domain.Chat, error)
	ListChats(ctx context.Context, userID, workspaceID string, limit int) ([]domain.Chat, error)
	DeleteChat(ctx context.Context, chatID, userID string) (bool, error)

	AppendMessage(ctx context.Context, msg domain.Message) error
	GetMessages(ctx context.Context, chatID string, limit int) ([]domain.Message, error)

	// UpdateLastMessageAt stamps the parent chat when a new message is appended.
	UpdateLastMessageAt(ctx context.Context, chatID string, ts int64) error
}
