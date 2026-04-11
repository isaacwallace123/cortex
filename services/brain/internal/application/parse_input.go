package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/brain/internal/log"
	"github.com/isaacwallace123/cortex/services/brain/internal/metrics"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

// ParseInputUseCase extracts intent, entities, and routing mode from raw user input.
// This is the Prism module — responsible for input understanding inside Brain.
type ParseInputUseCase struct {
	inference ports.InferencePort
	telemetry ports.TelemetryPort
	memory    ports.MemoryPort
	log       *slog.Logger
}

func NewParseInputUseCase(inference ports.InferencePort, telemetry ports.TelemetryPort, memory ports.MemoryPort) *ParseInputUseCase {
	return &ParseInputUseCase{
		inference: inference,
		telemetry: telemetry,
		memory:    memory,
		log:       cortexlog.Prism(),
	}
}

type ParseInputInput struct {
	SessionID string
	RawInput  string
}

type ParseInputOutput struct {
	Task domain.Task
}

func (uc *ParseInputUseCase) Execute(ctx context.Context, in ParseInputInput) (ParseInputOutput, error) {
	if strings.TrimSpace(in.RawInput) == "" {
		return ParseInputOutput{}, fmt.Errorf("raw input must not be empty")
	}

	uc.log.Info("[Prism] parsing input",
		slog.String("session_id", in.SessionID),
		slog.String("trace_id", observe.FromContext(ctx)),
		slog.Int("input_len", len(in.RawInput)),
	)

	response, err := uc.inference.Complete(ctx, buildParsePrompt(in.RawInput))
	if err != nil {
		return ParseInputOutput{}, fmt.Errorf("inference failed during parse_input: %w", err)
	}

	task := parseInferenceResponse(in.SessionID, response)
	task.RawInput = in.RawInput

	uc.log.Info("[Prism] parsed input",
		slog.String("session_id", in.SessionID),
		slog.String("intent", task.Intent),
		slog.String("mode", task.Mode),
		slog.Any("entities", task.Entities),
	)

	payload := fmt.Sprintf(`{"session_id":%q,"intent":%q,"mode":%q,"raw_input":%q}`,
		in.SessionID, task.Intent, task.Mode, in.RawInput)

	_ = uc.telemetry.Emit(ctx, ports.Event{
		Name:    "prism.input.parsed",
		Payload: map[string]any{"session_id": in.SessionID, "intent": task.Intent, "mode": task.Mode},
	})
	_ = uc.memory.StoreEvent(ctx, in.SessionID, "prism.input.parsed", payload)

	// Store the user turn for conversation history reconstruction.
	userTurn := fmt.Sprintf(`{"content":%q}`, in.RawInput)
	_ = uc.memory.StoreEvent(ctx, in.SessionID, "cortex.turn.user", userTurn)

	metrics.InputsParsedTotal.Inc()

	return ParseInputOutput{Task: task}, nil
}

func buildParsePrompt(rawInput string) string {
	return fmt.Sprintf(`Classify this user input and extract its intent and entities.

Respond in exactly this format (no extra text):
INTENT: <short intent phrase>
ENTITIES: <comma-separated list, or NONE>
MODE: <conversation|direct_answer|execution|tool_assisted|clarification>

MODE rules:
- conversation: open-ended discussion, opinions, brainstorming, abstract topics, creative thinking, or anything where a back-and-forth exchange is expected (e.g. "what do you think about microservices?", "help me brainstorm names for my project", "tell me about yourself", "how does gRPC work?")
- direct_answer: a specific factual question with a single definite answer, requiring no system commands (e.g. "what is the capital of France?", "what year was Docker released?")
- execution: requires running shell or filesystem commands where the output itself is the answer (e.g. "check disk usage", "list files", "show processes", "create a directory")
- tool_assisted: requires running commands AND then reasoning about or analyzing the results (e.g. "check disk usage and tell me what's using the most space", "show running processes and suggest what to kill")
- clarification: too ambiguous to classify without more context (e.g. "do the thing", "fix it")

When in doubt between conversation and direct_answer, choose conversation.

Input: %s`, rawInput)
}

func parseInferenceResponse(sessionID, response string) domain.Task {
	task := domain.Task{SessionID: sessionID}

	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "INTENT:"):
			task.Intent = strings.TrimSpace(strings.TrimPrefix(line, "INTENT:"))
		case strings.HasPrefix(line, "ENTITIES:"):
			raw := strings.TrimSpace(strings.TrimPrefix(line, "ENTITIES:"))
			if raw != "" && raw != "NONE" {
				for _, e := range strings.Split(raw, ",") {
					task.Entities = append(task.Entities, strings.TrimSpace(e))
				}
			}
		case strings.HasPrefix(line, "MODE:"):
			task.Mode = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "MODE:")))
		}
	}

	if task.Intent == "" {
		task.Intent = "unknown"
	}
	if task.Mode == "" {
		task.Mode = "conversation" // safe default: unknown → converse rather than risk executing commands
	}

	return task
}
