package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	cortexlog "github.com/isaacwallace123/cortex/services/brain/internal/log"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

// StreamChatToken is a single token or the final signal from StreamChat.
type StreamChatToken struct {
	Token  string
	Done   bool
	PlanID string // populated on the final (Done) token
}

// StreamChatUseCase orchestrates the full parse→plan→stream pipeline.
// For conversation/direct_answer mode it streams LLM tokens.
// For execution mode it runs the plan and emits stdout as a single token.
type StreamChatUseCase struct {
	parseInput  *ParseInputUseCase
	createPlan  *CreatePlanUseCase
	executePlan *ExecutePlanUseCase
	inference   ports.InferencePort
	log         *slog.Logger
}

func NewStreamChatUseCase(
	parseInput *ParseInputUseCase,
	createPlan *CreatePlanUseCase,
	executePlan *ExecutePlanUseCase,
	inference ports.InferencePort,
) *StreamChatUseCase {
	return &StreamChatUseCase{
		parseInput:  parseInput,
		createPlan:  createPlan,
		executePlan: executePlan,
		inference:   inference,
		log:         cortexlog.Brain(),
	}
}

// Stream runs the full chat pipeline and sends tokens to the returned channel.
// The channel is closed when streaming is complete (after the Done token).
// Callers must drain the channel fully.
func (uc *StreamChatUseCase) Stream(ctx context.Context, sessionID, rawInput string) (<-chan StreamChatToken, error) {
	// Parse.
	parsed, err := uc.parseInput.Execute(ctx, ParseInputInput{
		SessionID: sessionID,
		RawInput:  rawInput,
	})
	if err != nil {
		return nil, fmt.Errorf("parse_input: %w", err)
	}

	// Plan.
	planned, err := uc.createPlan.Execute(ctx, CreatePlanInput{Task: parsed.Task})
	if err != nil {
		return nil, fmt.Errorf("create_plan: %w", err)
	}

	out := make(chan StreamChatToken, 32)

	go func() {
		defer close(out)

		mode := parsed.Task.Mode
		isConversational := mode == "conversation" || mode == "direct_answer" || mode == "clarification"

		if isConversational {
			// Find the infer step — its Command is the assembled prompt.
			prompt := rawInput
			for _, step := range planned.Plan.Steps {
				if step.Executor == "infer" && step.Command != "" {
					prompt = step.Command
					break
				}
			}

			ch, err := uc.inference.CompleteStream(ctx, prompt)
			if err != nil {
				select {
				case out <- StreamChatToken{Token: fmt.Sprintf("Error: %v", err), Done: true, PlanID: planned.Plan.ID}:
				case <-ctx.Done():
				}
				return
			}

			var sb strings.Builder
			for tok := range ch {
				sb.WriteString(tok)
				select {
				case out <- StreamChatToken{Token: tok}:
				case <-ctx.Done():
					return
				}
			}

			select {
			case out <- StreamChatToken{Done: true, PlanID: planned.Plan.ID}:
			case <-ctx.Done():
			}
			return
		}

		// Execution mode: run the plan, collect stdout, emit as single token.
		execOut, err := uc.executePlan.Execute(ctx, ExecutePlanInput{
			SessionID: planned.Plan.SessionID,
			PlanID:    planned.Plan.ID,
			Steps:     planned.Plan.Steps,
		})

		answer := ""
		if err == nil {
			for _, r := range execOut.Results {
				if r.Stdout != "" {
					answer = r.Stdout
					break
				}
			}
		}

		if answer != "" {
			select {
			case out <- StreamChatToken{Token: answer}:
			case <-ctx.Done():
				return
			}
		}
		select {
		case out <- StreamChatToken{Done: true, PlanID: planned.Plan.ID}:
		case <-ctx.Done():
		}
	}()

	return out, nil
}
