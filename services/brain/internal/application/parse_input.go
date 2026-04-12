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

	prompt := buildParsePrompt(in.RawInput)

	const maxAttempts = 3
	var task domain.Task
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		response, err := uc.inference.Complete(ctx, prompt, 0.0)
		if err != nil {
			return ParseInputOutput{}, fmt.Errorf("inference failed during parse_input: %w", err)
		}
		task = parseInferenceResponse(in.SessionID, response)
		if task.ModeValid() {
			break
		}
		uc.log.Warn("[Prism] invalid mode in LLM output, retrying",
			slog.String("session_id", in.SessionID),
			slog.String("raw_output", response),
			slog.Int("attempt", attempt),
		)
		metrics.PrismFallbacksTotal.Inc()
	}
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
	return fmt.Sprintf(`You are a strict intent classifier. Analyze the user message below and output ONLY the three lines shown. No preamble, no explanation, no extra text.

OUTPUT FORMAT (copy exactly, fill in the blanks):
INTENT: <short phrase describing what the user wants>
ENTITIES: <comma-separated nouns/names/values, or NONE>
MODE: <one of: conversation|direct_answer|execution|tool_assisted|clarification>

MODE DEFINITIONS — pick the FIRST that matches:
  conversation   — greetings, chat, opinions, creative ideas, brainstorming, open-ended questions, "how does X work", "tell me about Y"
  direct_answer  — a closed factual question answerable from knowledge alone, no tools needed ("what is the capital of France?", "who invented TCP/IP?")
  execution      — user wants something DONE on the system: create/delete/move files, run commands, write code to disk ("create a file", "delete logs", "list running processes")
  tool_assisted  — requires searching the web OR running commands AND then explaining results ("look up X and write it to a file", "check disk usage and summarize", "research X")
  clarification  — genuinely too ambiguous to classify ("do the thing", "fix it", single words)

EXAMPLES:
  "hi there"                                                → MODE: conversation
  "what do you think about microservices?"                  → MODE: conversation
  "tell me a joke"                                          → MODE: conversation
  "what is 2 + 2?"                                          → MODE: direct_answer
  "what year was Docker released?"                          → MODE: direct_answer
  "check disk usage"                                        → MODE: execution
  "create a file called notes.txt"                          → MODE: execution
  "look up the latest Go release and write it to a file"    → MODE: tool_assisted
  "what are the minecraft versions? save them to versions.txt" → MODE: tool_assisted
  "research python async patterns"                          → MODE: tool_assisted
  "do the thing"                                            → MODE: clarification

USER MESSAGE: %s`, rawInput)
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

	// Clamp mode to the valid set. If the LLM outputs anything else (e.g. "greeting",
	// "Confirmation or agreement"), map it to conversation — the safest fallback.
	if !task.ModeValid() {
		task.Mode = "conversation"
	}

	return task
}
