package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	brainv1 "github.com/isaacwallace123/cortex/services/brain/gen/brain/v1"
	"github.com/isaacwallace123/cortex/services/brain/internal/application"
	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/brain/internal/log"
)

// Server implements the gRPC BrainService.
// It is a thin transport layer: Prism, Axiom, and Vector logic lives in the
// application layer; this server only dispatches and maps protos.
type Server struct {
	brainv1.UnimplementedBrainServiceServer

	parseInput  *application.ParseInputUseCase
	createPlan  *application.CreatePlanUseCase
	executePlan *application.ExecutePlanUseCase
	getSession  *application.GetSessionUseCase
	streamChat  *application.StreamChatUseCase
	log         *slog.Logger
}

func NewServer(
	parseInput *application.ParseInputUseCase,
	createPlan *application.CreatePlanUseCase,
	executePlan *application.ExecutePlanUseCase,
	getSession *application.GetSessionUseCase,
	streamChat *application.StreamChatUseCase,
) *Server {
	return &Server{
		parseInput:  parseInput,
		createPlan:  createPlan,
		executePlan: executePlan,
		getSession:  getSession,
		streamChat:  streamChat,
		log:         cortexlog.Brain(),
	}
}

func (s *Server) ParseInput(ctx context.Context, req *brainv1.ParseInputRequest) (*brainv1.ParseInputResponse, error) {
	s.log.Info("[brain] handle parse_input",
		slog.String("session_id", req.GetSessionId()),
	)
	out, err := s.parseInput.Execute(ctx, application.ParseInputInput{
		SessionID: req.GetSessionId(),
		RawInput:  req.GetRawInput(),
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parse_input: %v", err)
	}
	return &brainv1.ParseInputResponse{
		SessionId: out.Task.SessionID,
		Intent:    out.Task.Intent,
		Entities:  out.Task.Entities,
		Mode:      out.Task.Mode,
		RawInput:  out.Task.RawInput,
	}, nil
}

func (s *Server) CreatePlan(ctx context.Context, req *brainv1.CreatePlanRequest) (*brainv1.CreatePlanResponse, error) {
	s.log.Info("[brain] handle create_plan",
		slog.String("session_id", req.GetSessionId()),
		slog.String("intent", req.GetIntent()),
	)
	task := domain.Task{
		SessionID: req.GetSessionId(),
		Intent:    req.GetIntent(),
		Entities:  req.GetEntities(),
		Mode:      req.GetMode(),
		RawInput:  req.GetRawInput(),
	}
	out, err := s.createPlan.Execute(ctx, application.CreatePlanInput{Task: task})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create_plan: %v", err)
	}

	// Plan steps already carry Aegis verdicts — CreatePlanUseCase gates through policy.
	steps := make([]*brainv1.Step, 0, len(out.Plan.Steps))
	for _, step := range out.Plan.Steps {
		steps = append(steps, &brainv1.Step{
			Index:       int32(step.Index),
			Description: step.Description,
			Executor:    step.Executor,
			Command:     step.Command,
			AgentId:     step.AgentID,
			Verdict:     step.Verdict,
			ApprovalId:  step.ApprovalID,
		})
	}
	return &brainv1.CreatePlanResponse{
		SessionId: out.Plan.SessionID,
		PlanId:    out.Plan.ID,
		Steps:     steps,
	}, nil
}

func (s *Server) ExecutePlan(ctx context.Context, req *brainv1.ExecutePlanRequest) (*brainv1.ExecutePlanResponse, error) {
	s.log.Info("[brain] handle execute_plan",
		slog.String("session_id", req.GetSessionId()),
		slog.String("plan_id", req.GetPlanId()),
		slog.Int("steps", len(req.GetSteps())),
	)
	steps := make([]domain.Step, 0, len(req.GetSteps()))
	for _, st := range req.GetSteps() {
		steps = append(steps, domain.Step{
			Index:       int(st.GetIndex()),
			Description: st.GetDescription(),
			Executor:    st.GetExecutor(),
			Command:     st.GetCommand(),
			AgentID:     st.GetAgentId(),
			Verdict:     st.GetVerdict(),
		})
	}

	out, err := s.executePlan.Execute(ctx, application.ExecutePlanInput{
		SessionID: req.GetSessionId(),
		PlanID:    req.GetPlanId(),
		Steps:     steps,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "execute_plan: %v", err)
	}

	results := make([]*brainv1.StepResult, 0, len(out.Results))
	for _, r := range out.Results {
		results = append(results, &brainv1.StepResult{
			StepIndex:  int32(r.StepIndex),
			Stdout:     r.Stdout,
			Stderr:     r.Stderr,
			ExitCode:   int32(r.ExitCode),
			DurationMs: r.DurationMs,
			Skipped:    r.Skipped,
			SkipReason: r.SkipReason,
		})
	}
	return &brainv1.ExecutePlanResponse{
		SessionId: out.SessionID,
		PlanId:    out.PlanID,
		Results:   results,
	}, nil
}

func (s *Server) StreamChat(req *brainv1.StreamChatRequest, stream brainv1.BrainService_StreamChatServer) error {
	s.log.Info("[brain] handle stream_chat",
		slog.String("session_id", req.GetSessionId()),
	)
	ch, err := s.streamChat.Stream(stream.Context(), req.GetSessionId(), req.GetRawInput())
	if err != nil {
		return status.Errorf(codes.Internal, "stream_chat: %v", err)
	}
	for tok := range ch {
		if err := stream.Send(&brainv1.ChatToken{
			Token:  tok.Token,
			Done:   tok.Done,
			PlanId: tok.PlanID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) GetSession(ctx context.Context, req *brainv1.GetSessionRequest) (*brainv1.GetSessionResponse, error) {
	s.log.Info("[brain] handle get_session",
		slog.String("session_id", req.GetSessionId()),
	)
	out, err := s.getSession.Execute(ctx, req.GetSessionId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get_session: %v", err)
	}

	events := make([]*brainv1.Event, 0, len(out.Events))
	for _, e := range out.Events {
		events = append(events, &brainv1.Event{
			EventId:   e.ID,
			EventName: e.Name,
			Payload:   e.Payload,
			StoredAt:  e.StoredAt.Unix(),
		})
	}
	return &brainv1.GetSessionResponse{
		SessionId: out.SessionID,
		Events:    events,
	}, nil
}
