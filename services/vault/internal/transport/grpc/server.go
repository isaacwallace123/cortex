package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	vaultv1 "github.com/isaacwallace123/cortex/services/vault/gen/vault/v1"
	"github.com/isaacwallace123/cortex/services/vault/internal/application"
)

type Server struct {
	vaultv1.UnimplementedVaultServiceServer
	store  *application.StoreUseCase
	search *application.SearchUseCase
	delete *application.DeleteUseCase
}

func NewServer(store *application.StoreUseCase, search *application.SearchUseCase, delete *application.DeleteUseCase) *Server {
	return &Server{store: store, search: search, delete: delete}
}

func (s *Server) Store(ctx context.Context, req *vaultv1.StoreRequest) (*vaultv1.StoreResponse, error) {
	if strings.TrimSpace(req.GetContent()) == "" {
		return nil, status.Error(codes.InvalidArgument, "content must not be empty")
	}
	e, err := s.store.Execute(ctx, application.StoreInput{
		UserID:      req.GetUserId(),
		WorkspaceID: req.GetWorkspaceId(),
		Content:     req.GetContent(),
		Source:      req.GetSource(),
		Tags:        req.GetTags(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "vault.Store: %v", err)
	}
	return &vaultv1.StoreResponse{
		EntryId:   e.ID,
		CreatedAt: e.CreatedAt.UnixMilli(),
	}, nil
}

func (s *Server) Search(ctx context.Context, req *vaultv1.SearchRequest) (*vaultv1.SearchResponse, error) {
	if strings.TrimSpace(req.GetQuery()) == "" {
		return &vaultv1.SearchResponse{}, nil
	}
	entries, err := s.search.Execute(ctx, application.SearchInput{
		UserID:      req.GetUserId(),
		WorkspaceID: req.GetWorkspaceId(),
		Query:       req.GetQuery(),
		Limit:       int(req.GetLimit()),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "vault.Search: %v", err)
	}
	out := make([]*vaultv1.Entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, &vaultv1.Entry{
			EntryId:     e.ID,
			UserId:      e.UserID,
			WorkspaceId: e.WorkspaceID,
			Content:     e.Content,
			Source:      e.Source,
			Tags:        e.Tags,
			CreatedAt:   e.CreatedAt.UnixMilli(),
		})
	}
	return &vaultv1.SearchResponse{Entries: out}, nil
}

func (s *Server) Delete(ctx context.Context, req *vaultv1.DeleteRequest) (*vaultv1.DeleteResponse, error) {
	deleted, err := s.delete.Execute(ctx, req.GetEntryId(), req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "vault.Delete: %v", err)
	}
	return &vaultv1.DeleteResponse{Deleted: deleted}, nil
}
