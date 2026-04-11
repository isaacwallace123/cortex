// Package grpc exposes the ArsenalService gRPC server.
package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	arsenalv1 "github.com/isaacwallace123/cortex/services/arsenal/gen/arsenal/v1"
	"github.com/isaacwallace123/cortex/services/arsenal/internal/application"
	"github.com/isaacwallace123/cortex/services/arsenal/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/arsenal/internal/log"
)

// Server implements arsenalv1.ArsenalServiceServer.
type Server struct {
	arsenalv1.UnimplementedArsenalServiceServer

	registerUC *application.RegisterToolUseCase
	getUC      *application.GetToolUseCase
	listUC     *application.ListToolsUseCase
	deleteUC   *application.DeleteToolUseCase
	log        *slog.Logger
}

func NewServer(
	registerUC *application.RegisterToolUseCase,
	getUC *application.GetToolUseCase,
	listUC *application.ListToolsUseCase,
	deleteUC *application.DeleteToolUseCase,
) *Server {
	return &Server{
		registerUC: registerUC,
		getUC:      getUC,
		listUC:     listUC,
		deleteUC:   deleteUC,
		log:        cortexlog.Arsenal(),
	}
}

func (s *Server) RegisterTool(ctx context.Context, req *arsenalv1.RegisterToolRequest) (*arsenalv1.RegisterToolResponse, error) {
	if req.GetTool() == nil {
		return nil, status.Error(codes.InvalidArgument, "tool is required")
	}
	t := protoToDomain(req.GetTool())
	out, err := s.registerUC.Execute(ctx, t)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "register_tool: %v", err)
	}
	s.log.Info("[Arsenal] tool registered", slog.String("name", out.Name), slog.String("executor", out.Executor))
	return &arsenalv1.RegisterToolResponse{Tool: domainToProto(out)}, nil
}

func (s *Server) GetTool(ctx context.Context, req *arsenalv1.GetToolRequest) (*arsenalv1.GetToolResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	t, err := s.getUC.Execute(ctx, req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get_tool: %v", err)
	}
	if t == nil {
		return nil, status.Errorf(codes.NotFound, "tool %q not found", req.GetName())
	}
	return &arsenalv1.GetToolResponse{Tool: domainToProto(t)}, nil
}

func (s *Server) ListTools(ctx context.Context, req *arsenalv1.ListToolsRequest) (*arsenalv1.ListToolsResponse, error) {
	tools, err := s.listUC.Execute(ctx, req.GetExecutor(), req.GetTag())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_tools: %v", err)
	}
	out := make([]*arsenalv1.Tool, 0, len(tools))
	for _, t := range tools {
		out = append(out, domainToProto(t))
	}
	return &arsenalv1.ListToolsResponse{Tools: out}, nil
}

func (s *Server) DeleteTool(ctx context.Context, req *arsenalv1.DeleteToolRequest) (*arsenalv1.DeleteToolResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	deleted, err := s.deleteUC.Execute(ctx, req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete_tool: %v", err)
	}
	return &arsenalv1.DeleteToolResponse{Deleted: deleted}, nil
}

// protoToDomain converts a proto Tool message to the domain type.
func protoToDomain(p *arsenalv1.Tool) *domain.Tool {
	return &domain.Tool{
		Name:        p.GetName(),
		Description: p.GetDescription(),
		Executor:    p.GetExecutor(),
		Example:     p.GetExample(),
		SafetyLevel: domain.SafetyLevel(p.GetSafetyLevel()),
		Tags:        p.GetTags(),
		Enabled:     p.GetEnabled(),
		Version:     p.GetVersion(),
	}
}

func domainToProto(t *domain.Tool) *arsenalv1.Tool {
	return &arsenalv1.Tool{
		Name:        t.Name,
		Description: t.Description,
		Executor:    t.Executor,
		Example:     t.Example,
		SafetyLevel: string(t.SafetyLevel),
		Tags:        t.Tags,
		Enabled:     t.Enabled,
		Version:     t.Version,
	}
}

