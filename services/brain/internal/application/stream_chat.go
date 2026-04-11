package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	cortexlog "github.com/isaacwallace123/cortex/services/brain/internal/log"
	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
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

			ch, err := uc.inference.CompleteStream(ctx, prompt, 0.7)
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

		// Execution / tool_assisted mode: run the plan, collect results.
		execOut, err := uc.executePlan.Execute(ctx, ExecutePlanInput{
			SessionID: planned.Plan.SessionID,
			PlanID:    planned.Plan.ID,
			Steps:     planned.Plan.Steps,
		})

		answer := collectAnswer(execOut, err, planned.Plan.Steps)

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

// collectAnswer extracts the best human-readable answer from plan execution results.
// Priority order:
//  1. Last infer step stdout — the LLM's analysis of tool outputs (best for tool_assisted)
//  2. Last non-empty stdout from any step — raw command output
//  3. A generic confirmation when steps ran but produced no output (e.g. file writes)
func collectAnswer(execOut ExecutePlanOutput, execErr error, steps []domain.Step) string {
	if execErr != nil {
		return fmt.Sprintf("Error running plan: %v", execErr)
	}

	// Build a map from step index to result for fast lookup.
	resultByIndex := make(map[int]domain.ExecutionResult, len(execOut.Results))
	for _, r := range execOut.Results {
		resultByIndex[r.StepIndex] = r
	}

	// 1. Look for the last infer step with a non-empty answer.
	for i := len(steps) - 1; i >= 0; i-- {
		s := steps[i]
		if s.Executor != "infer" {
			continue
		}
		if r, ok := resultByIndex[s.Index]; ok && r.Stdout != "" {
			return r.Stdout
		}
	}

	// 2. Last non-empty stdout from any step.
	for i := len(execOut.Results) - 1; i >= 0; i-- {
		if execOut.Results[i].Stdout != "" {
			return execOut.Results[i].Stdout
		}
	}

	// 3. Generic confirmation — plan ran but produced no printable output.
	if len(execOut.Results) > 0 {
		allSkipped := true
		for _, r := range execOut.Results {
			if !r.Skipped {
				allSkipped = false
				break
			}
		}
		if !allSkipped {
			return "Done."
		}
	}
	return ""
}
