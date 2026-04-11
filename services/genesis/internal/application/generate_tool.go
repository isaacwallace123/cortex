// Package application contains the Genesis tool-generation pipeline.
package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/isaacwallace123/cortex/services/genesis/internal/domain"
	"github.com/isaacwallace123/cortex/services/genesis/internal/ports"
)

// GenerateToolUseCase is the core Genesis pipeline:
//
//  1. Build a structured prompt from the natural language description.
//  2. Call the inference service to generate a JSON tool definition.
//  3. Parse and validate the JSON.
//  4. Register the tool in Arsenal.
//  5. Persist the proposal record.
type GenerateToolUseCase struct {
	inference ports.InferencePort
	arsenal   ports.ArsenalPort
	store     ports.ProposalStore
	log       *slog.Logger
}

func NewGenerateToolUseCase(
	inference ports.InferencePort,
	arsenal ports.ArsenalPort,
	store ports.ProposalStore,
	log *slog.Logger,
) *GenerateToolUseCase {
	return &GenerateToolUseCase{
		inference: inference,
		arsenal:   arsenal,
		store:     store,
		log:       log,
	}
}

func (uc *GenerateToolUseCase) Execute(ctx context.Context, description string) (domain.Proposal, error) {
	if strings.TrimSpace(description) == "" {
		return domain.Proposal{}, fmt.Errorf("description is required")
	}

	proposal := domain.Proposal{Description: description}

	// Step 1 — ask the LLM to draft a tool definition.
	uc.log.Info("[Genesis] generating tool", slog.String("description", description))
	raw, err := uc.inference.Complete(ctx, buildGeneratePrompt(description), 0.3)
	if err != nil {
		proposal.Error = fmt.Sprintf("inference failed: %v", err)
		_, _ = uc.store.Save(ctx, proposal)
		return proposal, nil
	}

	// Step 2 — extract and parse the JSON block from the LLM response.
	tool, err := parseToolJSON(raw)
	if err != nil {
		uc.log.Warn("[Genesis] failed to parse tool JSON",
			slog.String("raw", raw),
			slog.String("error", err.Error()),
		)
		proposal.Error = fmt.Sprintf("parse failed: %v", err)
		_, _ = uc.store.Save(ctx, proposal)
		return proposal, nil
	}
	proposal.Tool = tool

	// Step 3 — register the tool in Arsenal.
	if regErr := uc.arsenal.RegisterTool(ctx, *tool); regErr != nil {
		uc.log.Warn("[Genesis] failed to register tool",
			slog.String("name", tool.Name),
			slog.String("error", regErr.Error()),
		)
		proposal.Error = fmt.Sprintf("register failed: %v", regErr)
	} else {
		proposal.Registered = true
		uc.log.Info("[Genesis] tool registered", slog.String("name", tool.Name))
	}

	// Step 4 — persist the proposal record.
	saved, err := uc.store.Save(ctx, proposal)
	if err != nil {
		uc.log.Warn("[Genesis] failed to save proposal", slog.String("error", err.Error()))
		return proposal, nil
	}
	return saved, nil
}

// buildGeneratePrompt constructs the LLM prompt for tool generation.
func buildGeneratePrompt(description string) string {
	return fmt.Sprintf(`You are a tool definition generator for the Cortex AI system.

Given a description of a new tool, output ONLY a valid JSON object matching the schema below.
Do not include any explanation, markdown, or code fences — just the raw JSON.

SCHEMA:
{
  "name":        "<snake_case identifier, e.g. check_disk_usage>",
  "description": "<one sentence describing what the tool does>",
  "executor":    "<one of: shell | filesystem | web | infer | crucible | atlas | courier>",
  "example":     "<a STEP line showing how Brain would use this tool, e.g. STEP 1: Check disk [executor:shell] [command:df -h]>",
  "safety_level":"<one of: safe | moderate | dangerous>",
  "tags":        ["<tag1>", "<tag2>"],
  "version":     "1.0.0"
}

EXECUTOR GUIDE:
  shell      — runs a shell command on the local machine
  filesystem — reads/writes files in the workspace
  web        — fetches URLs or performs web searches
  infer      — asks the LLM to reason about prior step outputs
  crucible   — runs code in an isolated Docker sandbox
  atlas      — runs kubectl commands against a Kubernetes cluster
  courier    — runs a command on a remote Relay agent

DESCRIPTION: %s`, description)
}

// parseToolJSON extracts and parses the JSON object from an LLM response.
// The LLM may wrap the JSON in markdown code fences, so we strip those first.
func parseToolJSON(raw string) (*domain.GeneratedTool, error) {
	// Strip markdown code fences if present.
	s := strings.TrimSpace(raw)
	if idx := strings.Index(s, "{"); idx > 0 {
		s = s[idx:]
	}
	if idx := strings.LastIndex(s, "}"); idx >= 0 {
		s = s[:idx+1]
	}

	var out struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Executor    string   `json:"executor"`
		Example     string   `json:"example"`
		SafetyLevel string   `json:"safety_level"`
		Tags        []string `json:"tags"`
		Version     string   `json:"version"`
	}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if out.Name == "" {
		return nil, fmt.Errorf("generated tool has no name")
	}
	if out.Version == "" {
		out.Version = "1.0.0"
	}
	validExecutors := map[string]bool{
		"shell": true, "filesystem": true, "web": true,
		"infer": true, "crucible": true, "atlas": true, "courier": true,
	}
	if !validExecutors[out.Executor] {
		return nil, fmt.Errorf("invalid executor %q", out.Executor)
	}
	return &domain.GeneratedTool{
		Name:        out.Name,
		Description: out.Description,
		Executor:    out.Executor,
		Example:     out.Example,
		SafetyLevel: out.SafetyLevel,
		Tags:        out.Tags,
		Version:     out.Version,
	}, nil
}
