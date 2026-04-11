package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	memoryv1 "github.com/isaacwallace123/cortex/services/memory/gen/memory/v1"
	"github.com/isaacwallace123/cortex/services/memory/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/memory/internal/log"
)

// metaValue extracts the first value of a gRPC metadata key from the incoming context.
func metaValue(ctx context.Context, key string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(key); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

// Server implements the gRPC MemoryService (Echo).
type Server struct {
	memoryv1.UnimplementedMemoryServiceServer

	storeEvent *application.StoreEventUseCase
	getSession *application.GetSessionUseCase
	listPlans  *application.ListPlansUseCase
	log        *slog.Logger
}

func NewServer(
	storeEvent *application.StoreEventUseCase,
	getSession *application.GetSessionUseCase,
	listPlans *application.ListPlansUseCase,
) *Server {
	return &Server{
		storeEvent: storeEvent,
		getSession: getSession,
		listPlans:  listPlans,
		log:        cortexlog.Echo(),
	}
}

func (s *Server) StoreEvent(ctx context.Context, req *memoryv1.StoreEventRequest) (*memoryv1.StoreEventResponse, error) {
	out, err := s.storeEvent.Execute(ctx, application.StoreEventInput{
		SessionID:   req.GetSessionId(),
		UserID:      metaValue(ctx, "x-user-id"),
		WorkspaceID: metaValue(ctx, "x-workspace-id"),
		EventName:   req.GetEventName(),
		Payload:     req.GetPayload(),
	})
	if err != nil {
		s.log.Warn("[Echo] store_event failed",
			slog.String("session_id", req.GetSessionId()),
			slog.String("event", req.GetEventName()),
			slog.String("error", err.Error()),
		)
		return nil, status.Errorf(codes.InvalidArgument, "store_event: %v", err)
	}
	return &memoryv1.StoreEventResponse{
		EventId:  out.Event.ID,
		StoredAt: out.Event.StoredAt.Unix(),
	}, nil
}

func (s *Server) GetSession(ctx context.Context, req *memoryv1.GetSessionRequest) (*memoryv1.GetSessionResponse, error) {
	out, err := s.getSession.Execute(ctx, req.GetSessionId(), metaValue(ctx, "x-user-id"))
	if err != nil {
		s.log.Warn("[Echo] get_session failed",
			slog.String("session_id", req.GetSessionId()),
			slog.String("error", err.Error()),
		)
		return nil, status.Errorf(codes.Internal, "get_session: %v", err)
	}

	events := make([]*memoryv1.Event, 0, len(out.Events))
	for _, e := range out.Events {
		events = append(events, &memoryv1.Event{
			EventId:   e.ID,
			SessionId: e.SessionID,
			EventName: e.Name,
			Payload:   e.Payload,
			StoredAt:  e.StoredAt.Unix(),
		})
	}
	return &memoryv1.GetSessionResponse{
		SessionId: out.SessionID,
		Events:    events,
	}, nil
}

func (s *Server) ListPlans(ctx context.Context, req *memoryv1.ListPlansRequest) (*memoryv1.ListPlansResponse, error) {
	out, err := s.listPlans.Execute(ctx, req.GetSessionId(), metaValue(ctx, "x-user-id"))
	if err != nil {
		s.log.Warn("[Echo] list_plans failed",
			slog.String("session_id", req.GetSessionId()),
			slog.String("error", err.Error()),
		)
		return nil, status.Errorf(codes.Internal, "list_plans: %v", err)
	}

	plans := make([]*memoryv1.PlanSummary, 0, len(out.Plans))
	for _, p := range out.Plans {
		plans = append(plans, &memoryv1.PlanSummary{
			PlanId:    p.PlanID,
			SessionId: p.SessionID,
			Intent:    p.Intent,
			StepCount: int32(p.StepCount),
			CreatedAt: p.CreatedAt.Unix(),
		})
	}
	return &memoryv1.ListPlansResponse{Plans: plans}, nil
}
