package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	atlasv1 "github.com/isaacwallace123/cortex/services/atlas/gen/atlas/v1"
	"github.com/isaacwallace123/cortex/services/atlas/internal/executor"
)

// Server implements the AtlasService gRPC interface.
type Server struct {
	atlasv1.UnimplementedAtlasServiceServer
	kubectl *executor.KubectlExecutor
}

func NewServer(kubectl *executor.KubectlExecutor) *Server { return &Server{kubectl: kubectl} }

func (s *Server) Execute(ctx context.Context, req *atlasv1.ExecuteRequest) (*atlasv1.ExecuteResponse, error) {
	if req.Command == "" {
		return nil, status.Error(codes.InvalidArgument, "command required")
	}
	res, err := s.kubectl.Execute(ctx, req.Command, int(req.TimeoutSecs))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "kubectl exec: %s", err)
	}
	return &atlasv1.ExecuteResponse{
		Stdout:     res.Stdout,
		Stderr:     res.Stderr,
		ExitCode:   int32(res.ExitCode),
		DurationMs: res.DurationMs,
	}, nil
}

func (s *Server) ClusterInfo(ctx context.Context, _ *atlasv1.ClusterInfoRequest) (*atlasv1.ClusterInfoResponse, error) {
	info := s.kubectl.Info(ctx)
	return &atlasv1.ClusterInfoResponse{
		Context:   info.Context,
		Server:    info.Server,
		Reachable: info.Reachable,
	}, nil
}
