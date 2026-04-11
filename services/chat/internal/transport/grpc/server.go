package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/isaacwallace123/cortex/services/chat/gen/chat/v1"
	"github.com/isaacwallace123/cortex/services/chat/internal/application"
	"github.com/isaacwallace123/cortex/services/chat/internal/domain"
)

// Server implements the ChatService gRPC interface.
type Server struct {
	chatv1.UnimplementedChatServiceServer
	createChat    *application.CreateChatUseCase
	getChat       *application.GetChatUseCase
	listChats     *application.ListChatsUseCase
	deleteChat    *application.DeleteChatUseCase
	appendMessage *application.AppendMessageUseCase
	getMessages   *application.GetMessagesUseCase
}

func NewServer(
	create *application.CreateChatUseCase,
	get *application.GetChatUseCase,
	list *application.ListChatsUseCase,
	del *application.DeleteChatUseCase,
	append_ *application.AppendMessageUseCase,
	getMsgs *application.GetMessagesUseCase,
) *Server {
	return &Server{
		createChat:    create,
		getChat:       get,
		listChats:     list,
		deleteChat:    del,
		appendMessage: append_,
		getMessages:   getMsgs,
	}
}

func (s *Server) CreateChat(ctx context.Context, req *chatv1.CreateChatRequest) (*chatv1.CreateChatResponse, error) {
	chat, err := s.createChat.Execute(ctx, req.UserId, req.WorkspaceId, req.Title)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", err)
	}
	return &chatv1.CreateChatResponse{Chat: domainChatToProto(chat)}, nil
}

func (s *Server) GetChat(ctx context.Context, req *chatv1.GetChatRequest) (*chatv1.GetChatResponse, error) {
	chat, err := s.getChat.Execute(ctx, req.ChatId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%s", err)
	}
	return &chatv1.GetChatResponse{Chat: domainChatToProto(chat)}, nil
}

func (s *Server) ListChats(ctx context.Context, req *chatv1.ListChatsRequest) (*chatv1.ListChatsResponse, error) {
	chats, err := s.listChats.Execute(ctx, req.UserId, req.WorkspaceId, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err)
	}
	out := make([]*chatv1.Chat, 0, len(chats))
	for _, c := range chats {
		out = append(out, domainChatToProto(c))
	}
	return &chatv1.ListChatsResponse{Chats: out}, nil
}

func (s *Server) DeleteChat(ctx context.Context, req *chatv1.DeleteChatRequest) (*chatv1.DeleteChatResponse, error) {
	deleted, err := s.deleteChat.Execute(ctx, req.ChatId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err)
	}
	return &chatv1.DeleteChatResponse{Deleted: deleted}, nil
}

func (s *Server) AppendMessage(ctx context.Context, req *chatv1.AppendMessageRequest) (*chatv1.AppendMessageResponse, error) {
	msg, err := s.appendMessage.Execute(ctx, req.ChatId, req.Role, req.Content)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", err)
	}
	return &chatv1.AppendMessageResponse{Message: domainMsgToProto(msg)}, nil
}

func (s *Server) GetMessages(ctx context.Context, req *chatv1.GetMessagesRequest) (*chatv1.GetMessagesResponse, error) {
	msgs, err := s.getMessages.Execute(ctx, req.ChatId, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err)
	}
	out := make([]*chatv1.Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, domainMsgToProto(m))
	}
	return &chatv1.GetMessagesResponse{Messages: out}, nil
}

func domainChatToProto(c domain.Chat) *chatv1.Chat {
	return &chatv1.Chat{
		ChatId:        c.ChatID,
		UserId:        c.UserID,
		WorkspaceId:   c.WorkspaceID,
		Title:         c.Title,
		CreatedAt:     c.CreatedAt,
		LastMessageAt: c.LastMessageAt,
	}
}

func domainMsgToProto(m domain.Message) *chatv1.Message {
	return &chatv1.Message{
		MessageId: m.MessageID,
		ChatId:    m.ChatID,
		Role:      m.Role,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
	}
}
