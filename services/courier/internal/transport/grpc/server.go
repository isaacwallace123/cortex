package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	courierv1 "github.com/isaacwallace123/cortex/services/courier/gen/courier/v1"
	"github.com/isaacwallace123/cortex/services/courier/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/courier/internal/log"
)

type Server struct {
	courierv1.UnimplementedCourierServiceServer

	connectUC    *application.ConnectUseCase
	dispatchUC   *application.DispatchUseCase
	listAgentsUC *application.ListAgentsUseCase
	log          *slog.Logger
}

func NewServer(
	connectUC *application.ConnectUseCase,
	dispatchUC *application.DispatchUseCase,
	listAgentsUC *application.ListAgentsUseCase,
) *Server {
	return &Server{
		connectUC:    connectUC,
		dispatchUC:   dispatchUC,
		listAgentsUC: listAgentsUC,
		log:          cortexlog.Courier(),
	}
}

// Connect handles the persistent bidirectional stream for a single agent.
func (s *Server) Connect(stream courierv1.CourierService_ConnectServer) error {
	if err := s.connectUC.Handle(stream); err != nil {
		s.log.Warn("[Courier] agent stream ended with error", slog.String("error", err.Error()))
	}
	return nil
}

// ListAgents returns all currently connected agents.
func (s *Server) ListAgents(_ context.Context, _ *courierv1.ListAgentsRequest) (*courierv1.ListAgentsResponse, error) {
	agents := s.listAgentsUC.Execute(context.Background())
	infos := make([]*courierv1.AgentInfo, 0, len(agents))
	for _, a := range agents {
		infos = append(infos, &courierv1.AgentInfo{
			AgentId:      a.AgentID,
			AgentType:    a.AgentType,
			DeviceName:   a.DeviceName,
			Hostname:     a.Hostname,
			Os:           a.OS,
			Arch:         a.Arch,
			Capabilities: a.Capabilities,
			Status:       string(a.Status),
			ConnectedAt:  a.ConnectedAt.UnixMilli(),
			LastSeen:     a.LastSeen.UnixMilli(),
		})
	}
	return &courierv1.ListAgentsResponse{Agents: infos}, nil
}

// Dispatch sends a command to a specific connected agent and returns the result.
func (s *Server) Dispatch(ctx context.Context, req *courierv1.DispatchRequest) (*courierv1.DispatchResponse, error) {
	out, err := s.dispatchUC.Execute(ctx, application.DispatchInput{
		AgentID:   req.GetAgentId(),
		Command:   req.GetCommand(),
		TimeoutMs: req.GetTimeoutMs(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "courier.Dispatch: %v", err)
	}
	return &courierv1.DispatchResponse{
		Stdout:     out.Result.Stdout,
		Stderr:     out.Result.Stderr,
		ExitCode:   out.Result.ExitCode,
		DurationMs: out.Result.DurationMs,
	}, nil
}
