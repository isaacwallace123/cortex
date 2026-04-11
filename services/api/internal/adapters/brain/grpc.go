package brain

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	brainv1 "github.com/isaacwallace123/cortex/services/brain/gen/brain/v1"
)

type userIDKeyType struct{}

// WithUserID returns a context carrying the authenticated user_id for outgoing gRPC calls.
func WithUserID(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "x-user-id", userID)
}

// WithWorkspaceID returns a context carrying the workspace_id for outgoing gRPC calls.
func WithWorkspaceID(ctx context.Context, workspaceID string) context.Context {
	if workspaceID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "x-workspace-id", workspaceID)
}

// WithRoles returns a context carrying the user's roles as x-roles metadata for outgoing gRPC calls.
func WithRoles(ctx context.Context, roles []string) context.Context {
	if len(roles) == 0 {
		return ctx
	}
	kv := make([]string, 0, len(roles)*2)
	for _, r := range roles {
		kv = append(kv, "x-roles", r)
	}
	return metadata.AppendToOutgoingContext(ctx, kv...)
}

// Client wraps the brain gRPC client with typed methods the gateway uses.
type Client struct {
	inner brainv1.BrainServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial brain at %s: %w", addr, err)
	}
	return &Client{inner: brainv1.NewBrainServiceClient(conn)}, nil
}

type ParseInputResult struct {
	SessionID string
	Intent    string
	Entities  []string
	Mode      string
	RawInput  string
}

func (c *Client) ParseInput(ctx context.Context, sessionID, rawInput string) (ParseInputResult, error) {
	resp, err := c.inner.ParseInput(ctx, &brainv1.ParseInputRequest{
		SessionId: sessionID,
		RawInput:  rawInput,
	})
	if err != nil {
		return ParseInputResult{}, fmt.Errorf("brain.ParseInput: %w", err)
	}
	return ParseInputResult{
		SessionID: resp.GetSessionId(),
		Intent:    resp.GetIntent(),
		Entities:  resp.GetEntities(),
		Mode:      resp.GetMode(),
		RawInput:  resp.GetRawInput(),
	}, nil
}

type CreatePlanResult struct {
	SessionID string
	PlanID    string
	Steps     []Step
}

type Step struct {
	Index       int32
	Description string
	Executor    string
	Command     string
	AgentID     string // for executor:courier
	Verdict     string // “allow” | “deny” | “pending” — from Aegis
	ApprovalID  string // non-empty when Verdict == “pending”; references the Aegis approval record
}

func (c *Client) CreatePlan(ctx context.Context, sessionID, intent, mode, rawInput string, entities []string) (CreatePlanResult, error) {
	resp, err := c.inner.CreatePlan(ctx, &brainv1.CreatePlanRequest{
		SessionId: sessionID,
		Intent:    intent,
		Entities:  entities,
		Mode:      mode,
		RawInput:  rawInput,
	})
	if err != nil {
		return CreatePlanResult{}, fmt.Errorf("brain.CreatePlan: %w", err)
	}

	steps := make([]Step, 0, len(resp.GetSteps()))
	for _, s := range resp.GetSteps() {
		steps = append(steps, Step{
			Index:       s.GetIndex(),
			Description: s.GetDescription(),
			Executor:    s.GetExecutor(),
			Command:     s.GetCommand(),
			AgentID:     s.GetAgentId(),
			Verdict:     s.GetVerdict(),
			ApprovalID:  s.GetApprovalId(),
		})
	}
	return CreatePlanResult{
		SessionID: resp.GetSessionId(),
		PlanID:    resp.GetPlanId(),
		Steps:     steps,
	}, nil
}

type StepResult struct {
	StepIndex  int32
	Stdout     string
	Stderr     string
	ExitCode   int32
	DurationMs int64
	Skipped    bool
	SkipReason string
}

type ExecutePlanResult struct {
	SessionID string
	PlanID    string
	Results   []StepResult
}

func (c *Client) ExecutePlan(ctx context.Context, sessionID, planID string, steps []Step) (ExecutePlanResult, error) {
	protoSteps := make([]*brainv1.Step, 0, len(steps))
	for _, s := range steps {
		protoSteps = append(protoSteps, &brainv1.Step{
			Index:       s.Index,
			Description: s.Description,
			Executor:    s.Executor,
			Command:     s.Command,
			AgentId:     s.AgentID,
			Verdict:     s.Verdict,
		})
	}

	resp, err := c.inner.ExecutePlan(ctx, &brainv1.ExecutePlanRequest{
		SessionId: sessionID,
		PlanId:    planID,
		Steps:     protoSteps,
	})
	if err != nil {
		return ExecutePlanResult{}, fmt.Errorf("brain.ExecutePlan: %w", err)
	}

	results := make([]StepResult, 0, len(resp.GetResults()))
	for _, r := range resp.GetResults() {
		results = append(results, StepResult{
			StepIndex:  r.GetStepIndex(),
			Stdout:     r.GetStdout(),
			Stderr:     r.GetStderr(),
			ExitCode:   r.GetExitCode(),
			DurationMs: r.GetDurationMs(),
			Skipped:    r.GetSkipped(),
			SkipReason: r.GetSkipReason(),
		})
	}
	return ExecutePlanResult{
		SessionID: resp.GetSessionId(),
		PlanID:    resp.GetPlanId(),
		Results:   results,
	}, nil
}

// StreamChatToken is a single token received from Brain's StreamChat RPC.
type StreamChatToken struct {
	Token  string
	Done   bool
	PlanID string
}

// StreamChat opens a server-streaming RPC to Brain and returns a channel of tokens.
// The channel is closed when streaming completes or ctx is cancelled.
func (c *Client) StreamChat(ctx context.Context, sessionID, rawInput string) (<-chan StreamChatToken, error) {
	stream, err := c.inner.StreamChat(ctx, &brainv1.StreamChatRequest{
		SessionId: sessionID,
		RawInput:  rawInput,
	})
	if err != nil {
		return nil, fmt.Errorf("brain.StreamChat: %w", err)
	}

	out := make(chan StreamChatToken, 32)
	go func() {
		defer close(out)
		for {
			msg, err := stream.Recv()
			if err != nil {
				return
			}
			out <- StreamChatToken{
				Token:  msg.GetToken(),
				Done:   msg.GetDone(),
				PlanID: msg.GetPlanId(),
			}
			if msg.GetDone() {
				return
			}
		}
	}()
	return out, nil
}

type SessionEvent struct {
	EventID   string
	EventName string
	Payload   string
	StoredAt  time.Time
}

type GetSessionResult struct {
	SessionID string
	Events    []SessionEvent
}

func (c *Client) GetSession(ctx context.Context, sessionID string) (GetSessionResult, error) {
	resp, err := c.inner.GetSession(ctx, &brainv1.GetSessionRequest{
		SessionId: sessionID,
	})
	if err != nil {
		return GetSessionResult{}, fmt.Errorf("brain.GetSession: %w", err)
	}

	events := make([]SessionEvent, 0, len(resp.GetEvents()))
	for _, e := range resp.GetEvents() {
		events = append(events, SessionEvent{
			EventID:   e.GetEventId(),
			EventName: e.GetEventName(),
			Payload:   e.GetPayload(),
			StoredAt:  time.Unix(e.GetStoredAt(), 0).UTC(),
		})
	}
	return GetSessionResult{
		SessionID: resp.GetSessionId(),
		Events:    events,
	}, nil
}

