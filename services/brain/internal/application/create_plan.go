package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/brain/internal/log"
	"github.com/isaacwallace123/cortex/services/brain/internal/metrics"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
	"github.com/isaacwallace123/cortex/services/brain/internal/recall"
)

// CreatePlanUseCase takes a parsed Task and produces a structured Plan.
// Axiom owns plan creation; it also gates each step through Aegis so callers
// always receive a plan with verdicts already attached.
type CreatePlanUseCase struct {
	inference ports.InferencePort
	telemetry ports.TelemetryPort
	memory    ports.MemoryPort
	policy    ports.PolicyPort
	courier   ports.CourierPort
	recall    *recall.Assembler
	arsenal   ports.ArsenalPort
	compass   ports.CompassPort
	log       *slog.Logger
}

func NewCreatePlanUseCase(inference ports.InferencePort, telemetry ports.TelemetryPort, memory ports.MemoryPort, policy ports.PolicyPort, courier ports.CourierPort, rc *recall.Assembler, arsenal ports.ArsenalPort, compass ports.CompassPort) *CreatePlanUseCase {
	return &CreatePlanUseCase{inference: inference, telemetry: telemetry, memory: memory, policy: policy, courier: courier, recall: rc, arsenal: arsenal, compass: compass, log: cortexlog.Axiom()}
}

type CreatePlanInput struct {
	Task domain.Task
}

type CreatePlanOutput struct {
	Plan domain.Plan
}

func (uc *CreatePlanUseCase) Execute(ctx context.Context, in CreatePlanInput) (CreatePlanOutput, error) {
	if in.Task.Intent == "" || in.Task.Intent == "unknown" {
		return CreatePlanOutput{}, fmt.Errorf("cannot create plan for unknown intent")
	}

	uc.log.Info("[Axiom] creating plan",
		slog.String("session_id", in.Task.SessionID),
		slog.String("trace_id", observe.FromContext(ctx)),
		slog.String("intent", in.Task.Intent),
		slog.String("mode", in.Task.Mode),
	)

	// Recall: assemble prior session context from Echo.
	// Vault knowledge is fetched later (after mode check) to avoid a search for non-execution paths.
	sessionEvents := uc.recall.Retrieve(ctx, in.Task.SessionID, "", 0).SessionEvents
	priorContext := recall.BuildSessionContext(sessionEvents)

	// conversation, direct_answer, and clarification: skip executor planning — route to infer.
	// Axiom produces a single step with executor="infer"; Vector calls inference directly.
	// The API gateway detects conversation/direct_answer mode and auto-executes inline,
	// returning the answer without requiring a separate /v1/run call.
	if in.Task.Mode == "conversation" || in.Task.Mode == "direct_answer" || in.Task.Mode == "clarification" {
		inferCmd := in.Task.RawInput
		if inferCmd == "" {
			inferCmd = in.Task.Intent
		}

		// Inject conversation history so the LLM has full session context.
		// Without this, every turn is answered cold with no memory of prior exchanges.
		convHistory := recall.BuildConversationHistory(sessionEvents)
		if convHistory != "" {
			inferCmd = "You are Cortex, a helpful and direct AI assistant. " +
				"Answer the user's latest message naturally and concisely. " +
				"Do NOT comment on the conversation itself, do NOT say things like 'you're repeating yourself' or 'I see you said X before'. " +
				"Do NOT summarise the history. Just respond to what the user said.\n\n" +
				"Conversation so far:\n" + convHistory +
				"\nUser: " + in.Task.RawInput + "\nCortex:"
		} else {
			inferCmd = "You are Cortex, a helpful and direct AI assistant. " +
				"Answer the user's message naturally and concisely. Do not add unnecessary preamble.\n\n" +
				"User: " + in.Task.RawInput + "\nCortex:"
		}

		plan := domain.Plan{
			ID:        uuid.New().String(),
			SessionID: in.Task.SessionID,
			Steps: []domain.Step{
				{
					Index:       1,
					Description: in.Task.Intent,
					Executor:    "infer",
					Command:     inferCmd,
					Verdict:     "allow", // infer steps are always allowed
				},
			},
		}
		uc.log.Info("[Axiom] direct answer plan created",
			slog.String("session_id", in.Task.SessionID),
			slog.String("plan_id", plan.ID),
		)
		metrics.PlansCreatedTotal.Inc()
		return CreatePlanOutput{Plan: plan}, nil
	}

	// project_plan: generate a structured project plan and persist it in Compass.
	if in.Task.Intent == "project_plan" || in.Task.Mode == "project_plan" {
		return uc.handleProjectPlan(ctx, in)
	}

	// Query Arsenal and filter to the tools most relevant to this task so the
	// planning prompt stays focused. All tools are fetched; only the top-N that
	// overlap with the task's entities or intent are injected.
	allTools, _ := uc.arsenal.ListTools(ctx)
	registryTools := filterRelevantTools(allTools, in.Task.Intent, in.Task.Entities, 5)

	// Query connected agents so Axiom can reference them in plans.
	agents, agentErr := uc.courier.ListAgents(ctx)
	if agentErr != nil {
		uc.log.Warn("[Axiom] could not list agents, courier steps unavailable",
			slog.String("error", agentErr.Error()),
		)
		agents = nil
	}

	// Recall: fetch relevant Vault knowledge to ground the plan in long-term memory.
	recallQuery := in.Task.RawInput
	if recallQuery == "" {
		recallQuery = in.Task.Intent
	}
	knowledge := uc.recall.Retrieve(ctx, in.Task.SessionID, recallQuery, 5).KnowledgeEntries

	response, err := uc.completeStreaming(ctx, buildPlanPrompt(in.Task, priorContext, agents, knowledge, registryTools))
	if err != nil {
		return CreatePlanOutput{}, fmt.Errorf("inference failed during create_plan: %w", err)
	}

	plan := parsePlanResponse(in.Task.SessionID, response)

	// Gate every step through Aegis before the plan leaves this use case.
	// Verdicts are attached to steps so all callers receive a fully-gated plan.
	decisions, err := uc.policy.EvaluatePlan(ctx, in.Task.SessionID, plan.ID, plan.Steps)
	if err != nil {
		uc.log.Warn("[Axiom] Aegis evaluation failed, proceeding without verdicts",
			slog.String("session_id", in.Task.SessionID),
			slog.String("plan_id", plan.ID),
			slog.String("error", err.Error()),
		)
	}
	verdictByStep := make(map[int]domain.StepDecision, len(decisions))
	for _, d := range decisions {
		verdictByStep[d.StepIndex] = d
		metrics.StepsEvaluatedTotal.WithLabelValues(d.Verdict).Inc()
	}
	for i := range plan.Steps {
		if d, ok := verdictByStep[plan.Steps[i].Index]; ok {
			plan.Steps[i].Verdict = d.Verdict
			plan.Steps[i].ApprovalID = d.ApprovalID
		}
	}

	uc.log.Info("[Axiom] plan created",
		slog.String("session_id", in.Task.SessionID),
		slog.String("plan_id", plan.ID),
		slog.Int("steps", len(plan.Steps)),
		slog.Bool("has_prior_context", priorContext != ""),
	)

	_ = uc.telemetry.Emit(ctx, ports.Event{
		Name: "axiom.plan.created",
		Payload: map[string]any{
			"session_id": in.Task.SessionID,
			"plan_id":    plan.ID,
			"step_count": len(plan.Steps),
		},
	})

	// Persist full step details including Aegis verdicts so re-runs can replay
	// the same gated plan without re-planning.
	type storedStep struct {
		Index       int    `json:"index"`
		Description string `json:"description"`
		Executor    string `json:"executor"`
		Command     string `json:"command"`
		AgentID     string `json:"agent_id,omitempty"`
		Verdict     string `json:"verdict"`
	}
	steps := make([]storedStep, 0, len(plan.Steps))
	for _, s := range plan.Steps {
		steps = append(steps, storedStep{s.Index, s.Description, s.Executor, s.Command, s.AgentID, s.Verdict})
	}
	p, _ := json.Marshal(map[string]any{
		"session_id": in.Task.SessionID,
		"plan_id":    plan.ID,
		"intent":     in.Task.Intent,
		"steps":      steps,
	})
	_ = uc.memory.StoreEvent(ctx, in.Task.SessionID, "axiom.plan.created", string(p))

	metrics.PlansCreatedTotal.Inc()

	return CreatePlanOutput{Plan: plan}, nil
}

// completeStreaming calls CompleteStream and collects all tokens into a single
// string. This keeps the use case interface simple while still benefiting from
// streaming at the transport level (tokens arrive incrementally at the brain,
// reducing time-to-first-byte pressure on the inference service).
func (uc *CreatePlanUseCase) completeStreaming(ctx context.Context, prompt string) (string, error) {
	ch, err := uc.inference.CompleteStream(ctx, prompt, 0.1)
	if err != nil {
		// Fall back to non-streaming if the adapter doesn't support it.
		return uc.inference.Complete(ctx, prompt, 0.1)
	}
	var sb strings.Builder
	for token := range ch {
		sb.WriteString(token)
	}
	if sb.Len() == 0 {
		return uc.inference.Complete(ctx, prompt, 0.1)
	}
	return sb.String(), nil
}

func buildPlanPrompt(task domain.Task, priorContext string, agents []ports.RemoteAgent, knowledge []ports.KnowledgeEntry, tools []ports.RegistryTool) string {
	entities := strings.Join(task.Entities, ", ")
	if entities == "" {
		entities = "none"
	}

	modeConstraint := ""
	switch task.Mode {
	case "tool_assisted":
		modeConstraint = "Mode: tool_assisted — gather data first (web search OR shell/filesystem commands), then add ONE infer step that explains or summarises the results. If the task involves looking something up or researching, use the web executor for the first step."
	case "execution":
		modeConstraint = "Mode: execution — shell/filesystem/web steps only. Do not add infer steps. For tasks that involve fetching information from the internet, use executor:web."
	}

	prior := ""
	if priorContext != "" {
		prior = "Prior session context:\n" + priorContext + "\n"
	}

	agentSection := ""
	if len(agents) > 0 {
		var sb strings.Builder
		sb.WriteString("REMOTE AGENTS (use executor:courier + [agent:<device_name>] to run on these machines)\n")
		for _, a := range agents {
			caps := strings.Join(a.Capabilities, ", ")
			sb.WriteString(fmt.Sprintf("  - %s (%s, %s) status:%s caps:[%s]\n",
				a.DeviceName, a.OS, a.Arch, a.Status, caps))
		}
		agentSection = sb.String() + "\n"
	}

	knowledgeSection := ""
	if len(knowledge) > 0 {
		var sb strings.Builder
		sb.WriteString("LONG-TERM MEMORY (relevant facts from past sessions — use to inform the plan)\n")
		for _, k := range knowledge {
			sb.WriteString(fmt.Sprintf("  - %s\n", k.Content))
		}
		knowledgeSection = sb.String() + "\n"
	}

	// Build the executor capability table: use Arsenal tools when available,
	// fall back to the built-in static rules when the registry is empty.
	toolSection := buildToolSection(tools)

	return fmt.Sprintf(`You are a precise task planner for an AI assistant named Cortex. Output ONLY STEP lines — no preamble, no explanation, no extra text.

TASK
Intent:   %s
Entities: %s
%s
%s%s%s%sRULES
1. Plan ONLY the steps needed for the stated intent. Never add unrelated setup steps.
2. Use the absolute minimum number of steps. Most intents need exactly 1 step.
3. Never exceed 5 steps.
4. Use real commands. Never invent command names.
5. For ANY task involving "look up", "find out", "search", "research", "what is the latest", "what are the versions" — use executor:web. The [command:...] for a web step is the search query string.
6. [command:...] is REQUIRED for every step. Never omit it.
7. For infer steps, [command:...] must be a full English instruction telling the model what to do with the prior output (e.g. "Summarise the search results and list the versions found.").
8. To write content to a file use: [executor:filesystem] [command:write /workspace/<filename> <content>]
9. [agent:...] required only for: courier (= device name) and crucible (= runtime). Omit for all others.

FORMAT — every line must match exactly (no extra text, no blank lines between steps):
STEP <n>: <short description> [executor:<type>] [command:<exact command or query>]

EXAMPLES — study these before writing the plan:
STEP 1: Say hello [executor:infer] [command:Greet the user warmly and briefly.]
---
STEP 1: Check disk usage [executor:shell] [command:df -h]
---
STEP 1: Search the web [executor:web] [command:latest stable Go release version]
STEP 2: Summarise findings [executor:infer] [command:Report the latest Go version found in the search results.]
---
STEP 1: Run Python script [executor:crucible] [agent:python3] [command:python3 -c "import math; print(math.pi)"]
---
STEP 1: List Kubernetes pods [executor:atlas] [command:get pods -A]
---
STEP 1: Write notes to file [executor:filesystem] [command:write /workspace/notes.txt Meeting notes: discussed Q3 goals]

PLAN:
`, task.Intent, entities, modeConstraint, agentSection, knowledgeSection, prior, toolSection)
}

// handleProjectPlan asks the LLM to design a project structure, persists it in
// Compass, and returns a conversational plan step that summarises the result.
func (uc *CreatePlanUseCase) handleProjectPlan(ctx context.Context, in CreatePlanInput) (CreatePlanOutput, error) {
	prompt := fmt.Sprintf(`You are a project planning assistant. The user wants to plan a project.

User request: %s

Output a structured project plan in EXACTLY this format (no other text):
PROJECT: <project name>
DESCRIPTION: <one sentence description>
MILESTONE 1: <name> | <description>
TASK 1.1: <name> | <description>
TASK 1.2: <name> | <description>
MILESTONE 2: <name> | <description>
TASK 2.1: <name> | <description>

Rules:
- 2-5 milestones
- 2-4 tasks per milestone
- Be concrete and actionable
- Names should be short (under 8 words)`, in.Task.RawInput)

	response, err := uc.completeStreaming(ctx, prompt)
	if err != nil {
		return CreatePlanOutput{}, fmt.Errorf("project plan inference failed: %w", err)
	}

	projectID, summary := uc.persistProjectPlan(ctx, response)

	answer := summary
	if projectID != "" {
		answer = fmt.Sprintf("I've created your project plan (ID: %s).\n\n%s", projectID, summary)
	}

	plan := domain.Plan{
		ID:        uuid.New().String(),
		SessionID: in.Task.SessionID,
		Steps: []domain.Step{{
			Index:       1,
			Description: "Project plan created",
			Executor:    "infer",
			Command:     answer,
			Verdict:     "allow",
		}},
	}
	metrics.PlansCreatedTotal.Inc()
	return CreatePlanOutput{Plan: plan}, nil
}

// persistProjectPlan parses the LLM project plan response and stores it in Compass.
// Returns the created projectID and a human-readable summary.
func (uc *CreatePlanUseCase) persistProjectPlan(ctx context.Context, response string) (projectID, summary string) {
	lines := strings.Split(response, "\n")

	var projectName, projectDesc string
	type milestoneEntry struct {
		name, desc string
		tasks      []struct{ name, desc string }
	}
	var milestones []milestoneEntry
	var currentMS *milestoneEntry

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PROJECT:") {
			projectName = strings.TrimSpace(strings.TrimPrefix(line, "PROJECT:"))
		} else if strings.HasPrefix(line, "DESCRIPTION:") {
			projectDesc = strings.TrimSpace(strings.TrimPrefix(line, "DESCRIPTION:"))
		} else if strings.HasPrefix(line, "MILESTONE ") {
			rest := line[strings.Index(line, ":")+1:]
			parts := strings.SplitN(rest, "|", 2)
			ms := milestoneEntry{name: strings.TrimSpace(parts[0])}
			if len(parts) > 1 {
				ms.desc = strings.TrimSpace(parts[1])
			}
			milestones = append(milestones, ms)
			currentMS = &milestones[len(milestones)-1]
		} else if strings.HasPrefix(line, "TASK ") && currentMS != nil {
			rest := line[strings.Index(line, ":")+1:]
			parts := strings.SplitN(rest, "|", 2)
			t := struct{ name, desc string }{name: strings.TrimSpace(parts[0])}
			if len(parts) > 1 {
				t.desc = strings.TrimSpace(parts[1])
			}
			currentMS.tasks = append(currentMS.tasks, t)
		}
	}

	if projectName == "" {
		projectName = "New Project"
	}

	// Persist to Compass (noop if compass is unavailable).
	projectID, err := uc.compass.CreateProject(ctx, projectName, projectDesc)
	if err != nil {
		uc.log.Warn("[Axiom] compass.CreateProject failed", slog.String("error", err.Error()))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**%s**\n%s\n\n", projectName, projectDesc))

	for i, ms := range milestones {
		msID := ""
		if projectID != "" {
			msID, _ = uc.compass.CreateMilestone(ctx, projectID, ms.name, ms.desc, i+1)
		}
		_ = msID
		sb.WriteString(fmt.Sprintf("**M%d: %s**\n%s\n", i+1, ms.name, ms.desc))
		for _, t := range ms.tasks {
			if projectID != "" && msID != "" {
				_, _ = uc.compass.CreateTask(ctx, msID, t.name, t.desc)
			}
			sb.WriteString(fmt.Sprintf("  - %s: %s\n", t.name, t.desc))
		}
		sb.WriteString("\n")
	}

	return projectID, sb.String()
}

// buildToolSection generates the AVAILABLE TOOLS block for the planning prompt.
// When Arsenal provides tools, each entry shows its description and an example
// STEP line so the LLM knows exactly how to invoke it. When no tools are
// registered the function falls back to the built-in static executor list.
func buildToolSection(tools []ports.RegistryTool) string {
	if len(tools) == 0 {
		return `AVAILABLE EXECUTORS (built-in)
  shell      — run any shell command on the local machine (date, df, ps, free, grep, cat, etc.)
  filesystem — /workspace file operations: ls, cat, mkdir, write <path> <content>, rm, stat
  web        — internet access: pass a search query OR a full https:// URL as the command
  infer      — reasoning step (tool_assisted only): summarise or explain prior step outputs
  courier    — run a command on a REMOTE AGENT; requires [agent:<device_name>]
  crucible   — run code in an isolated sandbox; requires [agent:<runtime>] (python3/node/go/bash)
  atlas      — run kubectl commands against the Kubernetes cluster

DECISION GUIDE — pick the right executor:
  "look something up" / "search" / "research" / "find out" → web
  "create/write/save a file"                               → filesystem  [command:write /workspace/<name> <content>]
  "run a command" / "check disk" / "list processes"        → shell
  "explain" / "summarise" / "what does this mean"          → infer (after other steps have run)
  "run python/node/go code"                                → crucible
  "on remote machine X"                                    → courier

REFERENCE EXAMPLES (copy the pattern exactly)
  greet                 → STEP 1: Say hello [executor:infer] [command:Say hello to the user briefly.]
  current time          → STEP 1: Print current time [executor:shell] [command:date]
  disk usage            → STEP 1: Check disk [executor:shell] [command:df -h]
  write a file          → STEP 1: Write file [executor:filesystem] [command:write /workspace/notes.txt Hello World]
  web search            → STEP 1: Search the web [executor:web] [command:latest Minecraft version list]
  fetch a page          → STEP 1: Fetch page [executor:web] [command:https://example.com]
  research + save file  → STEP 1: Search web for versions [executor:web] [command:Minecraft Java Edition version history complete list]
                          STEP 2: Write results to file [executor:filesystem] [command:write /workspace/versions.txt {results from step 1}]
  disk + explain        → STEP 1: Check disk usage [executor:shell] [command:du -sh /* 2>/dev/null | sort -rh | head -20]
                          STEP 2: Summarise findings [executor:infer] [command:Summarise what is using the most disk space.]
  run python            → STEP 1: Run script [executor:crucible] [agent:python3] [command:python3 -c "print('hello')"]
  k8s pods              → STEP 1: List pods [executor:atlas] [command:get pods -A]
  remote check          → STEP 1: Check disk on server [executor:courier] [agent:prod-server] [command:df -h]

`
	}

	var sb strings.Builder
	sb.WriteString("AVAILABLE TOOLS (from Arsenal registry)\n")
	for _, t := range tools {
		sb.WriteString(fmt.Sprintf("  %-20s [executor:%-12s] %s\n", t.Name, t.Executor, t.Description))
		if t.Example != "" {
			sb.WriteString(fmt.Sprintf("    example → %s\n", t.Example))
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

// extractTag finds the first [prefix<value>] token in body, returns the body
// with that token removed and the extracted value.
func extractTag(body, prefix string) (remaining, value string) {
	tag := "[" + prefix
	start := strings.Index(body, tag)
	if start < 0 {
		return body, ""
	}
	end := strings.Index(body[start:], "]")
	if end < 0 {
		return body, ""
	}
	value = strings.TrimSpace(body[start+len(tag) : start+end])
	remaining = strings.TrimSpace(body[:start] + body[start+end+1:])
	return remaining, value
}

// filterRelevantTools scores each tool against the task intent and entities and
// returns the top-n highest-scoring tools. Scoring is keyword-based: each match
// of an entity or intent word against a tool's name, description, or tags adds 1
// point. Tools with zero overlap are excluded. When the registry is empty or
// nothing matches, the caller falls back to the built-in static executor list.
func filterRelevantTools(tools []ports.RegistryTool, intent string, entities []string, n int) []ports.RegistryTool {
	if len(tools) == 0 {
		return nil
	}

	// Build a normalised set of query terms from intent words + entities.
	terms := make([]string, 0)
	for _, word := range strings.Fields(strings.ToLower(intent)) {
		if len(word) > 2 { // skip short stop words
			terms = append(terms, word)
		}
	}
	for _, e := range entities {
		terms = append(terms, strings.ToLower(strings.TrimSpace(e)))
	}
	if len(terms) == 0 {
		// No signal — return up to n tools unchanged.
		if len(tools) <= n {
			return tools
		}
		return tools[:n]
	}

	type scored struct {
		tool  ports.RegistryTool
		score int
	}
	var results []scored
	for _, t := range tools {
		haystack := strings.ToLower(t.Name + " " + t.Description + " " + strings.Join(t.Tags, " "))
		score := 0
		for _, term := range terms {
			if strings.Contains(haystack, term) {
				score++
			}
		}
		if score > 0 {
			results = append(results, scored{t, score})
		}
	}

	// Sort by score descending (simple insertion sort — list is small).
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	if len(results) > n {
		results = results[:n]
	}
	out := make([]ports.RegistryTool, len(results))
	for i, r := range results {
		out[i] = r.tool
	}
	return out
}

func parsePlanResponse(sessionID, response string) domain.Plan {
	plan := domain.Plan{
		ID:        uuid.New().String(),
		SessionID: sessionID,
	}

	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "STEP ") {
			continue
		}

		step := domain.Step{}

		rest := strings.TrimPrefix(line, "STEP ")
		colonIdx := strings.Index(rest, ":")
		if colonIdx < 0 {
			continue
		}

		indexStr := strings.TrimSpace(rest[:colonIdx])
		idx, err := strconv.Atoi(indexStr)
		if err != nil {
			continue
		}
		step.Index = idx

		body := strings.TrimSpace(rest[colonIdx+1:])

		// Extract tagged fields: [command:...], [executor:...], [agent:...]
		body, step.Command = extractTag(body, "command:")
		body, step.Executor = extractTag(body, "executor:")
		body, step.AgentID = extractTag(body, "agent:")

		step.Description = strings.TrimSpace(body)
		plan.Steps = append(plan.Steps, step)
	}

	return plan
}
