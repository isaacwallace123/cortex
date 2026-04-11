package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	workspacev1 "github.com/isaacwallace123/cortex/services/workspace/gen/workspace/v1"
	"github.com/isaacwallace123/cortex/services/workspace/internal/application"
	"github.com/isaacwallace123/cortex/services/workspace/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/workspace/internal/log"
)

type Server struct {
	workspacev1.UnimplementedWorkspaceServiceServer
	uc  *application.UseCases
	log *slog.Logger
}

func NewServer(uc *application.UseCases) *Server {
	return &Server{uc: uc, log: cortexlog.Workspace()}
}

func (s *Server) CreateWorkspace(ctx context.Context, req *workspacev1.CreateWorkspaceRequest) (*workspacev1.CreateWorkspaceResponse, error) {
	w, err := s.uc.CreateWorkspace(ctx, req.GetUserId(), req.GetName(), req.GetDescription(), req.GetRootPath())
	if err != nil {
		s.log.Warn("[Workspace] create failed", slog.String("error", err.Error()))
		return nil, status.Errorf(codes.InvalidArgument, "create_workspace: %v", err)
	}
	s.log.Info("[Workspace] created", slog.String("workspace_id", w.WorkspaceID), slog.String("user_id", w.UserID))
	return &workspacev1.CreateWorkspaceResponse{Workspace: toProto(w)}, nil
}

func (s *Server) GetWorkspace(ctx context.Context, req *workspacev1.GetWorkspaceRequest) (*workspacev1.GetWorkspaceResponse, error) {
	w, err := s.uc.GetWorkspace(ctx, req.GetWorkspaceId(), req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "get_workspace: %v", err)
	}
	return &workspacev1.GetWorkspaceResponse{Workspace: toProto(w)}, nil
}

func (s *Server) ListWorkspaces(ctx context.Context, req *workspacev1.ListWorkspacesRequest) (*workspacev1.ListWorkspacesResponse, error) {
	ws, err := s.uc.ListWorkspaces(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_workspaces: %v", err)
	}
	out := make([]*workspacev1.Workspace, 0, len(ws))
	for _, w := range ws {
		out = append(out, toProto(w))
	}
	return &workspacev1.ListWorkspacesResponse{Workspaces: out}, nil
}

func (s *Server) UpdateWorkspace(ctx context.Context, req *workspacev1.UpdateWorkspaceRequest) (*workspacev1.UpdateWorkspaceResponse, error) {
	w, err := s.uc.UpdateWorkspace(ctx, req.GetWorkspaceId(), req.GetUserId(), req.GetName(), req.GetDescription(), req.GetRootPath())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "update_workspace: %v", err)
	}
	return &workspacev1.UpdateWorkspaceResponse{Workspace: toProto(w)}, nil
}

func (s *Server) DeleteWorkspace(ctx context.Context, req *workspacev1.DeleteWorkspaceRequest) (*workspacev1.DeleteWorkspaceResponse, error) {
	deleted, err := s.uc.DeleteWorkspace(ctx, req.GetWorkspaceId(), req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete_workspace: %v", err)
	}
	return &workspacev1.DeleteWorkspaceResponse{Deleted: deleted}, nil
}

// toProto converts a domain Workspace to its proto representation.
func toProto(w domain.Workspace) *workspacev1.Workspace {
	return &workspacev1.Workspace{
		WorkspaceId: w.WorkspaceID,
		UserId:      w.UserID,
		Name:        w.Name,
		Description: w.Description,
		RootPath:    w.RootPath,
		CreatedAt:   w.CreatedAt.Unix(),
		UpdatedAt:   w.UpdatedAt.Unix(),
	}
}

