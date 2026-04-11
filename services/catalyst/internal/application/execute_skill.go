package application

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/google/uuid"

	"github.com/isaacwallace123/cortex/services/catalyst/internal/ports"
)

// ExecuteSkillInput holds the parameters for a skill execution request.
type ExecuteSkillInput struct {
	Name      string
	SessionID string
	Params    map[string]string
}

// ExecuteSkillOutput holds the per-step results from a skill execution.
type ExecuteSkillOutput struct {
	SessionID string
	Results   []ports.StepExecutionResult
}

// ExecuteSkillUseCase looks up a skill by name, expands parameter templates,
// and delegates execution to Brain.ExecutePlan.
type ExecuteSkillUseCase struct {
	store ports.SkillStore
	brain ports.BrainPort
}

func NewExecuteSkillUseCase(store ports.SkillStore, brain ports.BrainPort) *ExecuteSkillUseCase {
	return &ExecuteSkillUseCase{store: store, brain: brain}
}

func (uc *ExecuteSkillUseCase) Execute(ctx context.Context, in ExecuteSkillInput) (ExecuteSkillOutput, error) {
	skill, err := uc.store.Get(ctx, in.Name)
	if err != nil {
		return ExecuteSkillOutput{}, fmt.Errorf("get skill: %w", err)
	}
	if skill == nil {
		return ExecuteSkillOutput{}, fmt.Errorf("skill %q not found", in.Name)
	}

	sessionID := in.SessionID
	if sessionID == "" {
		sessionID = "skill-" + uuid.NewString()
	}
	planID := "skill-plan-" + uuid.NewString()

	steps := make([]ports.StepToExecute, 0, len(skill.Steps))
	for _, s := range skill.Steps {
		cmd, err := expandTemplate(s.CommandTemplate, in.Params)
		if err != nil {
			return ExecuteSkillOutput{}, fmt.Errorf("expand step %d template: %w", s.Index, err)
		}
		steps = append(steps, ports.StepToExecute{
			Index:       s.Index,
			Description: s.Description,
			Executor:    s.Executor,
			Command:     cmd,
			Agent:       s.Agent,
		})
	}

	results, err := uc.brain.ExecuteSteps(ctx, sessionID, planID, steps)
	if err != nil {
		return ExecuteSkillOutput{}, fmt.Errorf("execute steps: %w", err)
	}

	return ExecuteSkillOutput{SessionID: sessionID, Results: results}, nil
}

// expandTemplate renders a Go text/template string with the given params map.
// Template syntax: {{index . "key"}} or, for string map keys, {{.key}} works
// when the map is promoted to map[string]any which templates traverse by key.
func expandTemplate(tmpl string, params map[string]string) (string, error) {
	t, err := template.New("step").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	// Convert to map[string]any so {{.key}} works in templates.
	data := make(map[string]any, len(params))
	for k, v := range params {
		data[k] = v
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
