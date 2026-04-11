// Package http implements the Cortex external HTTP gateway.
// It exposes a JSON REST API consumed by the CLI and any other clients.
// All /v1/ routes (except auth endpoints) require a valid session token or API key.
package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	brainadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/brain"
	chatadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/chat"
	compassadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/compass"
	memoryadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/memory"
	policyadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/policy"
	sovereignadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/sovereign"
	workspaceadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/workspace"
)

// Server is the external-facing HTTP gateway for Cortex.
type Server struct {
	brain     *brainadapter.Client
	memory    *memoryadapter.Client
	chat      *chatadapter.Client           // nil when Chat service is unavailable
	compass   *compassadapter.Client        // nil when Compass service is unavailable
	policy    *policyadapter.Client         // nil when Policy service is unavailable
	sovereign *sovereignadapter.Client      // nil when Sovereign service is unavailable
	workspace *workspaceadapter.Client      // nil when Workspace service is unavailable
	logger    *slog.Logger
	apiKey    string
}

func NewServer(brain *brainadapter.Client, memory *memoryadapter.Client, chat *chatadapter.Client, compass *compassadapter.Client, policy *policyadapter.Client, sovereign *sovereignadapter.Client, workspace *workspaceadapter.Client, logger *slog.Logger, apiKey string) *Server {
	return &Server{brain: brain, memory: memory, chat: chat, compass: compass, policy: policy, sovereign: sovereign, workspace: workspace, logger: logger, apiKey: apiKey}
}

func (s *Server) Handler() http.Handler {
	// Authenticated API routes.
	api := http.NewServeMux()
	api.HandleFunc("POST /v1/chat", s.handleChat)
	api.HandleFunc("POST /v1/run", s.handleRun)
	api.HandleFunc("GET /v1/session/{id}", s.handleGetSession)
	api.HandleFunc("GET /v1/plans", s.handleListPlans)
	api.HandleFunc("GET /v1/plans/{id}", s.handleGetPlan)

	// Compass routes — project, milestone, task, and decision management.
	api.HandleFunc("POST /v1/projects", s.handleCreateProject)
	api.HandleFunc("GET /v1/projects", s.handleListProjects)
	api.HandleFunc("GET /v1/projects/{id}", s.handleGetProject)
	api.HandleFunc("PATCH /v1/projects/{id}/status", s.handleUpdateProjectStatus)
	api.HandleFunc("POST /v1/projects/{id}/milestones", s.handleCreateMilestone)
	api.HandleFunc("GET /v1/projects/{id}/milestones", s.handleListMilestones)
	api.HandleFunc("POST /v1/milestones/{id}/tasks", s.handleCreateTask)
	api.HandleFunc("GET /v1/milestones/{id}/tasks", s.handleListTasks)
	api.HandleFunc("PATCH /v1/tasks/{id}/status", s.handleUpdateTaskStatus)
	api.HandleFunc("POST /v1/projects/{id}/decisions", s.handleAddDecision)
	api.HandleFunc("GET /v1/projects/{id}/decisions", s.handleListDecisions)

	// Workspace routes.
	api.HandleFunc("POST /v1/workspaces", s.handleCreateWorkspace)
	api.HandleFunc("GET /v1/workspaces", s.handleListWorkspaces)
	api.HandleFunc("GET /v1/workspaces/{id}", s.handleGetWorkspace)
	api.HandleFunc("PATCH /v1/workspaces/{id}", s.handleUpdateWorkspace)
	api.HandleFunc("DELETE /v1/workspaces/{id}", s.handleDeleteWorkspace)

	// Chat routes — persistent chat threads backed by the Chat service.
	api.HandleFunc("POST /v1/chats", s.handleCreateChat)
	api.HandleFunc("GET /v1/chats", s.handleListChats)
	api.HandleFunc("GET /v1/chats/{id}", s.handleGetChat)
	api.HandleFunc("DELETE /v1/chats/{id}", s.handleDeleteChat)
	api.HandleFunc("POST /v1/chats/{id}/messages", s.handleChatMessage)
	api.HandleFunc("POST /v1/chats/{id}/stream", s.handleChatStream)
	api.HandleFunc("GET /v1/chats/{id}/messages", s.handleGetMessages)

	// Approval routes — Aegis pending-step management.
	api.HandleFunc("GET /v1/approvals", s.handleListApprovals)
	api.HandleFunc("POST /v1/approvals/{id}/approve", s.handleApprove)
	api.HandleFunc("POST /v1/approvals/{id}/deny", s.handleDeny)

	// Top-level mux: healthz and auth endpoints are public; everything else gated.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	// Auth endpoints â€” public (they issue tokens, so they can't require one).
	mux.HandleFunc("POST /v1/auth/register", s.handleAuthRegister)
	mux.HandleFunc("POST /v1/auth/login", s.handleAuthLogin)
	mux.HandleFunc("POST /v1/auth/logout", s.handleAuthLogout)
	mux.HandleFunc("GET /v1/auth/me", s.handleAuthMe)
	// All other /v1/ routes require authentication.
	api.HandleFunc("GET /v1/auth/users", s.handleListUsers)
	mux.Handle("/v1/", withAuth(s.apiKey, s.sovereign, api))

	return withRequestID(withLogging(mux, s.logger))
}

// ChatRequest is the request body for POST /v1/chat.
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Input     string `json:"input"`
}

type ChatResponse struct {
	RequestID string       `json:"request_id"`
	SessionID string       `json:"session_id"`
	Intent    string       `json:"intent"`
	Entities  []string     `json:"entities"`
	Mode      string       `json:"mode"`
	// Answer is populated for conversation and direct_answer modes.
	// Clients can display this directly without calling /v1/run.
	Answer    string       `json:"answer,omitempty"`
	Plan      PlanResponse `json:"plan"`
}

type PlanResponse struct {
	PlanID string         `json:"plan_id"`
	Steps  []StepResponse `json:"steps"`
}

type StepResponse struct {
	Index       int32  `json:"index"`
	Description string `json:"description"`
	Executor    string `json:"executor"`
	Command     string `json:"command,omitempty"`
	AgentID     string `json:"agent_id,omitempty"` // for executor:courier
	Verdict     string `json:"verdict"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Input == "" {
		writeError(w, http.StatusBadRequest, "input must not be empty")
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	requestID := requestIDFromContext(r.Context())
	log := s.logger.With(slog.String("request_id", requestID), slog.String("session_id", sessionID))

	ctx := brainadapter.WithUserID(r.Context(), userIDFromContext(r.Context()))
	ctx = brainadapter.WithWorkspaceID(ctx, workspaceIDFromRequest(r))
	ctx = brainadapter.WithRoles(ctx, rolesFromContext(r.Context()))
	parsed, err := s.brain.ParseInput(ctx, sessionID, req.Input)
	if err != nil {
		log.Error("[api] parse_input failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "brain unavailable")
		return
	}

	planned, err := s.brain.CreatePlan(ctx, sessionID, parsed.Intent, parsed.Mode, parsed.RawInput, parsed.Entities)
	if err != nil {
		log.Error("[api] create_plan failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "brain unavailable")
		return
	}

	steps := make([]StepResponse, 0, len(planned.Steps))
	for _, step := range planned.Steps {
		steps = append(steps, StepResponse{
			Index:       step.Index,
			Description: step.Description,
			Executor:    step.Executor,
			Command:     step.Command,
			AgentID:     step.AgentID,
			Verdict:     step.Verdict,
		})
	}

	resp := ChatResponse{
		RequestID: requestID,
		SessionID: sessionID,
		Intent:    parsed.Intent,
		Entities:  parsed.Entities,
		Mode:      parsed.Mode,
		Plan:      PlanResponse{PlanID: planned.PlanID, Steps: steps},
	}

	// For conversational and direct_answer modes the plan is a single infer step.
	// Auto-execute it here so callers get the answer inline â€” no /v1/run needed.
	if parsed.Mode == "conversation" || parsed.Mode == "direct_answer" || parsed.Mode == "clarification" {
		brainSteps := make([]brainadapter.Step, 0, len(planned.Steps))
		for _, st := range planned.Steps {
			brainSteps = append(brainSteps, brainadapter.Step{
				Index:       st.Index,
				Description: st.Description,
				Executor:    st.Executor,
				Command:     st.Command,
				AgentID:     st.AgentID,
				Verdict:     st.Verdict,
			})
		}
		if ran, err := s.brain.ExecutePlan(ctx, sessionID, planned.PlanID, brainSteps); err == nil {
			for _, res := range ran.Results {
				if res.Stdout != "" {
					resp.Answer = res.Stdout
					break
				}
			}
		}
	}

	log.Info("[api] chat complete",
		slog.String("intent", parsed.Intent),
		slog.String("mode", parsed.Mode),
		slog.Int("steps", len(planned.Steps)),
		slog.Bool("answered", resp.Answer != ""),
	)

	writeJSON(w, http.StatusOK, resp)
}

// RunRequest is the request body for POST /v1/run. The caller provides the
// plan steps returned by /v1/chat and Brain executes them.
type RunRequest struct {
	SessionID string        `json:"session_id"`
	PlanID    string        `json:"plan_id"`
	Steps     []StepResponse `json:"steps"`
}

type StepResultResponse struct {
	StepIndex  int32  `json:"step_index"`
	Executor   string `json:"executor"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int32  `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	Skipped    bool   `json:"skipped"`
	SkipReason string `json:"skip_reason,omitempty"`
}

type RunResponse struct {
	SessionID string               `json:"session_id"`
	PlanID    string               `json:"plan_id"`
	Results   []StepResultResponse `json:"results"`
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.PlanID == "" {
		writeError(w, http.StatusBadRequest, "plan_id required")
		return
	}

	log := s.logger.With(slog.String("session_id", req.SessionID), slog.String("plan_id", req.PlanID))

	// Convert API steps to brain adapter steps.
	steps := make([]brainadapter.Step, 0, len(req.Steps))
	for _, st := range req.Steps {
		steps = append(steps, brainadapter.Step{
			Index:       st.Index,
			Description: st.Description,
			Executor:    st.Executor,
			Command:     st.Command,
			AgentID:     st.AgentID,
			Verdict:     st.Verdict,
		})
	}

	ctx := brainadapter.WithUserID(r.Context(), userIDFromContext(r.Context()))
	ctx = brainadapter.WithWorkspaceID(ctx, workspaceIDFromRequest(r))
	ctx = brainadapter.WithRoles(ctx, rolesFromContext(r.Context()))
	out, err := s.brain.ExecutePlan(ctx, req.SessionID, req.PlanID, steps)
	if err != nil {
		log.Error("[api] execute_plan failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "brain unavailable")
		return
	}

	// Build an executor index so results can carry the step's executor type.
	executorByIndex := make(map[int32]string, len(req.Steps))
	for _, st := range req.Steps {
		executorByIndex[st.Index] = st.Executor
	}

	results := make([]StepResultResponse, 0, len(out.Results))
	for _, res := range out.Results {
		results = append(results, StepResultResponse{
			StepIndex:  res.StepIndex,
			Executor:   executorByIndex[res.StepIndex],
			Stdout:     res.Stdout,
			Stderr:     res.Stderr,
			ExitCode:   res.ExitCode,
			DurationMs: res.DurationMs,
			Skipped:    res.Skipped,
			SkipReason: res.SkipReason,
		})
	}

	log.Info("[api] run complete", slog.Int("results", len(results)))
	writeJSON(w, http.StatusOK, RunResponse{
		SessionID: out.SessionID,
		PlanID:    out.PlanID,
		Results:   results,
	})
}

type SessionEventResponse struct {
	EventID   string `json:"event_id"`
	EventName string `json:"event_name"`
	Payload   string `json:"payload"`
	StoredAt  int64  `json:"stored_at"`
}

type SessionResponse struct {
	SessionID string                 `json:"session_id"`
	Events    []SessionEventResponse `json:"events"`
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id required in path")
		return
	}

	out, err := s.brain.GetSession(r.Context(), sessionID)
	if err != nil {
		s.logger.Error("[api] get_session failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "brain unavailable")
		return
	}

	events := make([]SessionEventResponse, 0, len(out.Events))
	for _, e := range out.Events {
		events = append(events, SessionEventResponse{
			EventID:   e.EventID,
			EventName: e.EventName,
			Payload:   e.Payload,
			StoredAt:  e.StoredAt.Unix(),
		})
	}

	writeJSON(w, http.StatusOK, SessionResponse{
		SessionID: out.SessionID,
		Events:    events,
	})
}

type PlanSummaryResponse struct {
	PlanID    string `json:"plan_id"`
	SessionID string `json:"session_id"`
	Intent    string `json:"intent"`
	StepCount int    `json:"step_count"`
	CreatedAt int64  `json:"created_at"`
}

type ListPlansResponse struct {
	SessionID string                `json:"session_id"`
	Plans     []PlanSummaryResponse `json:"plans"`
}

func (s *Server) handleListPlans(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id query param required")
		return
	}
	plans, err := s.memory.ListPlans(r.Context(), sessionID)
	if err != nil {
		s.logger.Error("[api] list_plans failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "memory unavailable")
		return
	}
	out := make([]PlanSummaryResponse, 0, len(plans))
	for _, p := range plans {
		out = append(out, PlanSummaryResponse{
			PlanID:    p.PlanID,
			SessionID: p.SessionID,
			Intent:    p.Intent,
			StepCount: p.StepCount,
			CreatedAt: p.CreatedAt.Unix(),
		})
	}
	writeJSON(w, http.StatusOK, ListPlansResponse{SessionID: sessionID, Plans: out})
}

type PlanStepResponse struct {
	Index       int32  `json:"index"`
	Description string `json:"description"`
	Executor    string `json:"executor"`
	Command     string `json:"command,omitempty"`
	Verdict     string `json:"verdict"`
}

type GetPlanResponse struct {
	PlanID    string             `json:"plan_id"`
	SessionID string             `json:"session_id"`
	Intent    string             `json:"intent"`
	Steps     []PlanStepResponse `json:"steps"`
}

func (s *Server) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	planID := r.PathValue("id")
	sessionID := r.URL.Query().Get("session_id")
	if planID == "" || sessionID == "" {
		writeError(w, http.StatusBadRequest, "plan_id in path and session_id query param required")
		return
	}
	detail, err := s.memory.GetPlanDetail(r.Context(), sessionID, planID)
	if err != nil {
		s.logger.Error("[api] get_plan failed", slog.String("error", err.Error()))
		writeError(w, http.StatusNotFound, "plan not found")
		return
	}
	steps := make([]PlanStepResponse, 0, len(detail.Steps))
	for _, s := range detail.Steps {
		steps = append(steps, PlanStepResponse{
			Index:       s.Index,
			Description: s.Description,
			Executor:    s.Executor,
			Command:     s.Command,
			Verdict:     s.Verdict,
		})
	}
	writeJSON(w, http.StatusOK, GetPlanResponse{
		PlanID:    detail.PlanID,
		SessionID: detail.SessionID,
		Intent:    detail.Intent,
		Steps:     steps,
	})
}

type CreateChatRequest struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
	Title       string `json:"title"`
}

type ChatMeta struct {
	ChatID        string `json:"chat_id"`
	UserID        string `json:"user_id"`
	WorkspaceID   string `json:"workspace_id"`
	Title         string `json:"title"`
	CreatedAt     int64  `json:"created_at"`
	LastMessageAt int64  `json:"last_message_at"`
}

func (s *Server) handleCreateChat(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeError(w, http.StatusServiceUnavailable, "chat service unavailable")
		return
	}
	var req CreateChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		if uid := userIDFromContext(r.Context()); uid != "" {
			req.UserID = uid
		} else {
			req.UserID = "default"
		}
	}
	chat, err := s.chat.CreateChat(r.Context(), req.UserID, req.WorkspaceID, req.Title)
	if err != nil {
		s.logger.Error("[api] create_chat failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "chat service error")
		return
	}
	writeJSON(w, http.StatusCreated, chatToMeta(chat))
}

func (s *Server) handleListChats(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeError(w, http.StatusServiceUnavailable, "chat service unavailable")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		if uid := userIDFromContext(r.Context()); uid != "" {
			userID = uid
		} else {
			userID = "default"
		}
	}
	workspaceID := r.URL.Query().Get("workspace_id")
	chats, err := s.chat.ListChats(r.Context(), userID, workspaceID, 50)
	if err != nil {
		s.logger.Error("[api] list_chats failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "chat service error")
		return
	}
	out := make([]ChatMeta, 0, len(chats))
	for _, c := range chats {
		out = append(out, chatToMeta(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"chats": out})
}

func (s *Server) handleGetChat(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeError(w, http.StatusServiceUnavailable, "chat service unavailable")
		return
	}
	chatID := r.PathValue("id")
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		if uid := userIDFromContext(r.Context()); uid != "" {
			userID = uid
		} else {
			userID = "default"
		}
	}
	chat, err := s.chat.GetChat(r.Context(), chatID, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "chat not found")
		return
	}
	writeJSON(w, http.StatusOK, chatToMeta(chat))
}

func (s *Server) handleDeleteChat(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeError(w, http.StatusServiceUnavailable, "chat service unavailable")
		return
	}
	chatID := r.PathValue("id")
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		if uid := userIDFromContext(r.Context()); uid != "" {
			userID = uid
		} else {
			userID = "default"
		}
	}
	deleted, err := s.chat.DeleteChat(r.Context(), chatID, userID)
	if err != nil {
		s.logger.Error("[api] delete_chat failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "chat service error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": deleted})
}

// ChatMessageRequest is the request body for POST /v1/chats/{id}/messages.
// Sending a message appends the user turn, runs it through Brain, persists the
// assistant reply, and returns both turns in the response.
type ChatMessageRequest struct {
	UserID  string `json:"user_id"`
	Input   string `json:"input"`
}

type ChatMessageResponse struct {
	ChatID    string          `json:"chat_id"`
	UserMsg   ChatMessageItem `json:"user_message"`
	Assistant ChatMessageItem `json:"assistant_message"`
	// Answer is the assistant's text reply (convenience copy of assistant_message.content).
	Answer string `json:"answer"`
}

type ChatMessageItem struct {
	MessageID string `json:"message_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
}

func (s *Server) handleChatMessage(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeError(w, http.StatusServiceUnavailable, "chat service unavailable")
		return
	}
	chatID := r.PathValue("id")
	var req ChatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Input == "" {
		writeError(w, http.StatusBadRequest, "input required")
		return
	}
	if req.UserID == "" {
		if uid := userIDFromContext(r.Context()); uid != "" {
			req.UserID = uid
		} else {
			req.UserID = "default"
		}
	}

	// 1. Persist the user turn.
	userMsg, err := s.chat.AppendMessage(r.Context(), chatID, "user", req.Input)
	if err != nil {
		s.logger.Error("[api] append user message failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "chat service error")
		return
	}

	// 2. Run the input through brain (use chat_id as session_id for context continuity).
	bctx := brainadapter.WithUserID(r.Context(), req.UserID)
	bctx = brainadapter.WithRoles(bctx, rolesFromContext(r.Context()))
	parsed, err := s.brain.ParseInput(bctx, chatID, req.Input)
	if err != nil {
		s.logger.Error("[api] parse_input failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "brain unavailable")
		return
	}
	planned, err := s.brain.CreatePlan(bctx, chatID, parsed.Intent, parsed.Mode, parsed.RawInput, parsed.Entities)
	if err != nil {
		s.logger.Error("[api] create_plan failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "brain unavailable")
		return
	}

	// 3. Execute â€” always auto-execute for chat messages.
	answer := ""
	brainSteps := make([]brainadapter.Step, 0, len(planned.Steps))
	for _, st := range planned.Steps {
		brainSteps = append(brainSteps, brainadapter.Step{
			Index:       st.Index,
			Description: st.Description,
			Executor:    st.Executor,
			Command:     st.Command,
			AgentID:     st.AgentID,
			Verdict:     st.Verdict,
		})
	}
	if ran, err := s.brain.ExecutePlan(bctx, chatID, planned.PlanID, brainSteps); err == nil {
		for _, res := range ran.Results {
			if res.Stdout != "" {
				answer = res.Stdout
				break
			}
		}
	}

	// 4. Persist the assistant reply.
	assistantMsg, err := s.chat.AppendMessage(r.Context(), chatID, "assistant", answer)
	if err != nil {
		s.logger.Warn("[api] append assistant message failed", slog.String("error", err.Error()))
	}

	writeJSON(w, http.StatusOK, ChatMessageResponse{
		ChatID:    chatID,
		UserMsg:   msgToItem(userMsg),
		Assistant: msgToItem(assistantMsg),
		Answer:    answer,
	})
}

// handleChatStream handles POST /v1/chats/{id}/stream.
// It runs the full parse→plan→execute pipeline and streams the response as
// Server-Sent Events (SSE). Each event has the form:
//
//	data: <token>\n\n
//
// A final event with data: [DONE] signals the end of the stream.
func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeError(w, http.StatusServiceUnavailable, "chat service unavailable")
		return
	}
	chatID := r.PathValue("id")

	var req ChatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Input == "" {
		writeError(w, http.StatusBadRequest, "input required")
		return
	}
	if req.UserID == "" {
		if uid := userIDFromContext(r.Context()); uid != "" {
			req.UserID = uid
		} else {
			req.UserID = "default"
		}
	}

	// Persist the user turn before streaming starts.
	if _, err := s.chat.AppendMessage(r.Context(), chatID, "user", req.Input); err != nil {
		s.logger.Error("[api] append user message failed (stream)", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "chat service error")
		return
	}

	bctx := brainadapter.WithUserID(r.Context(), req.UserID)
	bctx = brainadapter.WithRoles(bctx, rolesFromContext(r.Context()))

	ch, err := s.brain.StreamChat(bctx, chatID, req.Input)
	if err != nil {
		s.logger.Error("[api] stream_chat failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "brain unavailable")
		return
	}

	// Switch to SSE mode.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	var answer strings.Builder
	for tok := range ch {
		if tok.Token != "" {
			answer.WriteString(tok.Token)
			fmt.Fprintf(w, "data: %s\n\n", tok.Token)
			if canFlush {
				flusher.Flush()
			}
		}
		if tok.Done {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			if canFlush {
				flusher.Flush()
			}
		}
	}

	// Persist the full assistant reply after streaming completes.
	if reply := answer.String(); reply != "" {
		if _, err := s.chat.AppendMessage(r.Context(), chatID, "assistant", reply); err != nil {
			s.logger.Warn("[api] append assistant message failed (stream)", slog.String("error", err.Error()))
		}
	}
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeError(w, http.StatusServiceUnavailable, "chat service unavailable")
		return
	}
	chatID := r.PathValue("id")
	msgs, err := s.chat.GetMessages(r.Context(), chatID, 100)
	if err != nil {
		s.logger.Error("[api] get_messages failed", slog.String("error", err.Error()))
		writeError(w, http.StatusBadGateway, "chat service error")
		return
	}
	out := make([]ChatMessageItem, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, msgToItem(m))
	}
	writeJSON(w, http.StatusOK, map[string]any{"chat_id": chatID, "messages": out})
}

// chatToMeta converts an adapter Chat into the HTTP response shape.
func chatToMeta(c chatadapter.Chat) ChatMeta {
	return ChatMeta{
		ChatID:        c.ChatID,
		UserID:        c.UserID,
		WorkspaceID:   c.WorkspaceID,
		Title:         c.Title,
		CreatedAt:     c.CreatedAt,
		LastMessageAt: c.LastMessageAt,
	}
}

func msgToItem(m chatadapter.Message) ChatMessageItem {
	return ChatMessageItem{
		MessageID: m.MessageID,
		Role:      m.Role,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "api"})
}

// writeJSON encodes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// handleListApprovals returns all pending Aegis approvals visible to the caller.
// Admins see all; regular users see only their own.
func (s *Server) handleListApprovals(w http.ResponseWriter, r *http.Request) {
	if s.policy == nil {
		writeError(w, http.StatusServiceUnavailable, "policy service unavailable")
		return
	}
	userID := userIDFromContext(r.Context())
	approvals, err := s.policy.ListPendingApprovals(r.Context(), userID)
	if err != nil {
		s.logger.Error("policy.ListPendingApprovals failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list approvals")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"approvals": approvals})
}

// handleApprove approves a pending step.
func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	s.submitApproval(w, r, "approved")
}

// handleDeny denies a pending step.
func (s *Server) handleDeny(w http.ResponseWriter, r *http.Request) {
	s.submitApproval(w, r, "denied")
}

func (s *Server) submitApproval(w http.ResponseWriter, r *http.Request, decision string) {
	if s.policy == nil {
		writeError(w, http.StatusServiceUnavailable, "policy service unavailable")
		return
	}
	approvalID := r.PathValue("id")
	if approvalID == "" {
		writeError(w, http.StatusBadRequest, "approval id required")
		return
	}
	userID := userIDFromContext(r.Context())
	ok, verdict, err := s.policy.SubmitApproval(r.Context(), approvalID, userID, decision)
	if err != nil {
		s.logger.Error("policy.SubmitApproval failed", "error", err, "approval_id", approvalID)
		writeError(w, http.StatusInternalServerError, "failed to submit approval")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": ok, "verdict": verdict})
}

