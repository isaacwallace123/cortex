package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
	nervav1 "github.com/isaacwallace123/cortex/services/nerva/gen/nerva/v1"
	"github.com/isaacwallace123/cortex/services/nerva/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/nerva/internal/log"
)

// Server implements the gRPC NervaService (event bus).
type Server struct {
	nervav1.UnimplementedNervaServiceServer

	publish *application.PublishUseCase
	bus     *application.Bus
	log     *slog.Logger
}

func NewServer(publish *application.PublishUseCase, bus *application.Bus) *Server {
	return &Server{publish: publish, bus: bus, log: cortexlog.Nerva()}
}

func (s *Server) Publish(ctx context.Context, req *nervav1.PublishRequest) (*nervav1.PublishResponse, error) {
	out, err := s.publish.Execute(ctx, application.PublishInput{
		Subsystem: req.GetSubsystem(),
		EventName: req.GetEventName(),
		SessionID: req.GetSessionId(),
		Payload:   req.GetPayload(),
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "nerva.Publish: %v", err)
	}
	return &nervav1.PublishResponse{
		EventId:   out.EventID,
		Timestamp: out.Timestamp,
	}, nil
}

func (s *Server) Subscribe(req *nervav1.SubscribeRequest, stream nervav1.NervaService_SubscribeServer) error {
	filter := req.GetFilter()
	subID := uuid.NewString()

	s.log.Info("[Nerva] subscriber connected",
		slog.String("sub_id", subID),
		slog.String("filter", filter),
	)

	ch, cancel := s.bus.Subscribe(subID)
	defer cancel()

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			if !application.MatchesFilter(event, filter) {
				continue
			}
			if err := stream.Send(&nervav1.EventMessage{
				EventId:   event.ID,
				Subsystem: event.Subsystem,
				EventName: event.Name,
				SessionId: event.SessionID,
				Payload:   event.Payload,
				Timestamp: event.CreatedAt.UnixMilli(),
			}); err != nil {
				s.log.Info("[Nerva] subscriber disconnected",
					slog.String("sub_id", subID),
					slog.String("error", err.Error()),
				)
				return nil
			}
		case <-stream.Context().Done():
			s.log.Info("[Nerva] subscriber disconnected",
				slog.String("sub_id", subID),
			)
			return nil
		}
	}
}
