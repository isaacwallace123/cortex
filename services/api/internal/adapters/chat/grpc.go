package chat

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	chatv1 "github.com/isaacwallace123/cortex/services/chat/gen/chat/v1"
)

// Client wraps the chat gRPC client with typed methods the gateway uses.
type Client struct {
	inner chatv1.ChatServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial chat at %s: %w", addr, err)
	}
	return &Client{inner: chatv1.NewChatServiceClient(conn)}, nil
}

type Chat struct {
	ChatID        string
	UserID        string
	WorkspaceID   string
	Title         string
	CreatedAt     int64
	LastMessageAt int64
}

type Message struct {
	MessageID string
	ChatID    string
	Role      string
	Content   string
	CreatedAt int64
}

func (c *Client) CreateChat(ctx context.Context, userID, workspaceID, title string) (Chat, error) {
	resp, err := c.inner.CreateChat(ctx, &chatv1.CreateChatRequest{
		UserId:      userID,
		WorkspaceId: workspaceID,
		Title:       title,
	})
	if err != nil {
		return Chat{}, err
	}
	return protoToChat(resp.Chat), nil
}

func (c *Client) GetChat(ctx context.Context, chatID, userID string) (Chat, error) {
	resp, err := c.inner.GetChat(ctx, &chatv1.GetChatRequest{ChatId: chatID, UserId: userID})
	if err != nil {
		return Chat{}, err
	}
	return protoToChat(resp.Chat), nil
}

func (c *Client) ListChats(ctx context.Context, userID, workspaceID string, limit int) ([]Chat, error) {
	resp, err := c.inner.ListChats(ctx, &chatv1.ListChatsRequest{
		UserId:      userID,
		WorkspaceId: workspaceID,
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, err
	}
	chats := make([]Chat, 0, len(resp.Chats))
	for _, ch := range resp.Chats {
		chats = append(chats, protoToChat(ch))
	}
	return chats, nil
}

func (c *Client) DeleteChat(ctx context.Context, chatID, userID string) (bool, error) {
	resp, err := c.inner.DeleteChat(ctx, &chatv1.DeleteChatRequest{ChatId: chatID, UserId: userID})
	if err != nil {
		return false, err
	}
	return resp.Deleted, nil
}

func (c *Client) AppendMessage(ctx context.Context, chatID, role, content string) (Message, error) {
	resp, err := c.inner.AppendMessage(ctx, &chatv1.AppendMessageRequest{
		ChatId:  chatID,
		Role:    role,
		Content: content,
	})
	if err != nil {
		return Message{}, err
	}
	return protoToMessage(resp.Message), nil
}

func (c *Client) GetMessages(ctx context.Context, chatID string, limit int) ([]Message, error) {
	resp, err := c.inner.GetMessages(ctx, &chatv1.GetMessagesRequest{
		ChatId: chatID,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, err
	}
	msgs := make([]Message, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		msgs = append(msgs, protoToMessage(m))
	}
	return msgs, nil
}

func protoToChat(p *chatv1.Chat) Chat {
	if p == nil {
		return Chat{}
	}
	return Chat{
		ChatID:        p.ChatId,
		UserID:        p.UserId,
		WorkspaceID:   p.WorkspaceId,
		Title:         p.Title,
		CreatedAt:     p.CreatedAt,
		LastMessageAt: p.LastMessageAt,
	}
}

func protoToMessage(p *chatv1.Message) Message {
	if p == nil {
		return Message{}
	}
	return Message{
		MessageID: p.MessageId,
		ChatID:    p.ChatId,
		Role:      p.Role,
		Content:   p.Content,
		CreatedAt: p.CreatedAt,
	}
}

