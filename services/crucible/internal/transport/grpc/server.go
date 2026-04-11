package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cruciblev1 "github.com/isaacwallace123/cortex/services/crucible/gen/crucible/v1"
	"github.com/isaacwallace123/cortex/services/crucible/internal/sandbox"
)

// Server implements the CrucibleService gRPC interface backed by the Docker executor.
type Server struct {
	cruciblev1.UnimplementedCrucibleServiceServer
	exec *sandbox.Executor
}

func NewServer(exec *sandbox.Executor) *Server { return &Server{exec: exec} }

func (s *Server) RunCode(ctx context.Context, req *cruciblev1.RunCodeRequest) (*cruciblev1.RunCodeResponse, error) {
	if req.Runtime == "" {
		return nil, status.Error(codes.InvalidArgument, "runtime required")
	}
	if req.Command == "" {
		return nil, status.Error(codes.InvalidArgument, "command required")
	}
	stdout, stderr, exitCode, durationMs, err := s.exec.RunCode(ctx, req.Runtime, req.Command, req.StdinInput, int(req.TimeoutSecs))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "run_code failed: %s", err)
	}
	return &cruciblev1.RunCodeResponse{
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   int32(exitCode),
		DurationMs: durationMs,
		Runtime:    req.Runtime,
	}, nil
}

func (s *Server) CreateSandbox(ctx context.Context, req *cruciblev1.CreateSandboxRequest) (*cruciblev1.CreateSandboxResponse, error) {
	if req.Runtime == "" {
		return nil, status.Error(codes.InvalidArgument, "runtime required")
	}
	sb, err := s.exec.Create(ctx, req.Runtime, int(req.TimeoutSecs))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create sandbox: %s", err)
	}
	return &cruciblev1.CreateSandboxResponse{
		SandboxId:   sb.SandboxID,
		Runtime:     sb.Runtime,
		CreatedAt:   sb.CreatedAt,
		TimeoutSecs: int32(sb.TimeoutSecs),
	}, nil
}

func (s *Server) Exec(ctx context.Context, req *cruciblev1.ExecRequest) (*cruciblev1.ExecResponse, error) {
	if req.SandboxId == "" {
		return nil, status.Error(codes.InvalidArgument, "sandbox_id required")
	}
	stdout, stderr, exitCode, durationMs, err := s.exec.Exec(ctx, req.SandboxId, req.Command, req.StdinInput, int(req.TimeoutSecs))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "exec failed: %s", err)
	}
	return &cruciblev1.ExecResponse{
		Stdout:     stdout,
		Stderr:     stderr,
		ExitCode:   int32(exitCode),
		DurationMs: durationMs,
	}, nil
}

func (s *Server) WriteFile(ctx context.Context, req *cruciblev1.WriteFileRequest) (*cruciblev1.WriteFileResponse, error) {
	if req.SandboxId == "" || req.Path == "" {
		return nil, status.Error(codes.InvalidArgument, "sandbox_id and path required")
	}
	if err := s.exec.WriteFile(ctx, req.SandboxId, req.Path, req.Content); err != nil {
		return nil, status.Errorf(codes.Internal, "write_file: %s", err)
	}
	return &cruciblev1.WriteFileResponse{Success: true}, nil
}

func (s *Server) ReadFile(ctx context.Context, req *cruciblev1.ReadFileRequest) (*cruciblev1.ReadFileResponse, error) {
	if req.SandboxId == "" || req.Path == "" {
		return nil, status.Error(codes.InvalidArgument, "sandbox_id and path required")
	}
	content, err := s.exec.ReadFile(ctx, req.SandboxId, req.Path)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "read_file: %s", err)
	}
	return &cruciblev1.ReadFileResponse{Content: content}, nil
}

func (s *Server) DestroySandbox(ctx context.Context, req *cruciblev1.DestroySandboxRequest) (*cruciblev1.DestroySandboxResponse, error) {
	if req.SandboxId == "" {
		return nil, status.Error(codes.InvalidArgument, "sandbox_id required")
	}
	destroyed, err := s.exec.Destroy(ctx, req.SandboxId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "destroy: %s", err)
	}
	return &cruciblev1.DestroySandboxResponse{Destroyed: destroyed}, nil
}

func (s *Server) ListSandboxes(_ context.Context, _ *cruciblev1.ListSandboxesRequest) (*cruciblev1.ListSandboxesResponse, error) {
	sandboxes := s.exec.List()
	out := make([]*cruciblev1.SandboxInfo, 0, len(sandboxes))
	for _, sb := range sandboxes {
		out = append(out, &cruciblev1.SandboxInfo{
			SandboxId:   sb.SandboxID,
			Runtime:     sb.Runtime,
			CreatedAt:   sb.CreatedAt,
			TimeoutSecs: int32(sb.TimeoutSecs),
		})
	}
	return &cruciblev1.ListSandboxesResponse{Sandboxes: out}, nil
}
