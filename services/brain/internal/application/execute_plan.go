package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/brain/internal/log"
	"github.com/isaacwallace123/cortex/services/brain/internal/metrics"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
	"github.com/isaacwallace123/cortex/services/brain/internal/sentinel"
)

// metaVal returns the first value for key from incoming gRPC metadata, or "".
func metaVal(ctx context.Context, key string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(key); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

// ExecutePlanUseCase runs approved plan steps through the appropriate executor.
// Vector owns executor selection, routing, and execution coordination.
type ExecutePlanUseCase struct {
	shell     ports.ExecutorPort
	forge     ports.ForgePort
	courier   ports.CourierPort
	crucible  ports.CruciblePort
	atlas     ports.AtlasPort
	beacon    ports.BeaconPort
	inference ports.InferencePort
	memory    ports.MemoryPort
	vault     ports.VaultPort
	telemetry ports.TelemetryPort
	sentinel  *sentinel.Sentinel
	log       *slog.Logger
}

func NewExecutePlanUseCase(shell ports.ExecutorPort, forge ports.ForgePort, courier ports.CourierPort, crucible ports.CruciblePort, atlas ports.AtlasPort, beacon ports.BeaconPort, inference ports.InferencePort, memory ports.MemoryPort, vault ports.VaultPort, telemetry ports.TelemetryPort, s *sentinel.Sentinel) *ExecutePlanUseCase {
	return &ExecutePlanUseCase{
		shell:     shell,
		forge:     forge,
		courier:   courier,
		crucible:  crucible,
		atlas:     atlas,
		beacon:    beacon,
		inference: inference,
		memory:    memory,
		vault:     vault,
		telemetry: telemetry,
		sentinel:  s,
		log:       cortexlog.Vector(),
	}
}

type ExecutePlanInput struct {
	SessionID string
	PlanID    string
	Steps     []domain.Step
}

type ExecutePlanOutput struct {
	SessionID string
	PlanID    string
	Results   []domain.ExecutionResult
}

func (uc *ExecutePlanUseCase) Execute(ctx context.Context, in ExecutePlanInput) (ExecutePlanOutput, error) {
	uc.log.Info("[Vector] executing plan",
		slog.String("session_id", in.SessionID),
		slog.String("plan_id", in.PlanID),
		slog.String("trace_id", observe.FromContext(ctx)),
		slog.Int("steps", len(in.Steps)),
	)

	results := make([]domain.ExecutionResult, 0, len(in.Steps))

	// priorOutputs accumulates stdout from executed steps so infer steps can
	// reason about real results rather than pre-written generic prompts.
	var priorOutputs []stepOutput

	for _, step := range in.Steps {
		result, skip := uc.tryExecute(ctx, in.SessionID, in.PlanID, step, priorOutputs)
		if !skip && step.Executor != "infer" && strings.TrimSpace(result.Stdout) != "" {
			priorOutputs = append(priorOutputs, stepOutput{
				index:  step.Index,
				desc:   step.Description,
				stdout: strings.TrimSpace(result.Stdout),
			})
		}
		results = append(results, result)
		if skip {
			continue
		}
		// Nerva: emit vector.step.executed with the actual executor used.
		_ = uc.telemetry.Emit(ctx, ports.Event{
			Name: "vector.step.executed",
			Payload: map[string]any{
				"session_id":  in.SessionID,
				"plan_id":     in.PlanID,
				"step_index":  step.Index,
				"executor":    step.Executor,
				"exit_code":   result.ExitCode,
				"duration_ms": result.DurationMs,
			},
		})
		stdout := result.Stdout
		if len(stdout) > 200 {
			stdout = stdout[:200]
		}
		p, _ := json.Marshal(map[string]any{
			"session_id":  in.SessionID,
			"plan_id":     in.PlanID,
			"step_index":  step.Index,
			"description": step.Description,
			"executor":    step.Executor,
			"command":     step.Command,
			"exit_code":   result.ExitCode,
			"duration_ms": result.DurationMs,
			"stdout":      stdout,
		})
		_ = uc.memory.StoreEvent(ctx, in.SessionID, "vector.step.executed", string(p))
	}

	uc.log.Info("[Vector] plan execution complete",
		slog.String("session_id", in.SessionID),
		slog.String("plan_id", in.PlanID),
		slog.Int("executed", len(results)),
	)

	return ExecutePlanOutput{SessionID: in.SessionID, PlanID: in.PlanID, Results: results}, nil
}

// stepOutput holds the stdout from a completed non-infer step so subsequent
// infer steps can be grounded in real execution results.
type stepOutput struct {
	index  int
	desc   string
	stdout string
}

// selectExecutor returns the correct executor port for the step's executor type.
// Vector owns this routing decision.
func (uc *ExecutePlanUseCase) selectExecutor(step domain.Step) (ports.StepExecutor, bool) {
	switch step.Executor {
	case "shell":
		return uc.shell, true
	case "filesystem":
		return uc.forge, true
	default:
		return nil, false
	}
}

// runInfer calls the inference service directly for infer steps.
// If prior step outputs are provided they are prepended to the prompt so the
// LLM reasons about real results rather than hypothetical ones.
func (uc *ExecutePlanUseCase) runInfer(ctx context.Context, step domain.Step, prior []stepOutput) domain.ExecutionResult {
	prompt := step.Command
	if len(prior) > 0 {
		var sb strings.Builder
		sb.WriteString("The following commands were run and produced this output:\n\n")
		for _, p := range prior {
			sb.WriteString(fmt.Sprintf("Step %d (%s):\n%s\n\n", p.index, p.desc, p.stdout))
		}
		sb.WriteString("Based on the above output, ")
		sb.WriteString(strings.ToLower(strings.TrimRight(strings.TrimSpace(step.Command), ".")))
		sb.WriteString(".")
		prompt = sb.String()
	}

	uc.log.Info("[Vector] routing step to inference",
		slog.Int("step", step.Index),
		slog.Bool("has_prior_context", len(prior) > 0),
	)
	answer, err := uc.inference.Complete(ctx, prompt)
	if err != nil {
		uc.log.Error("[Vector] inference error", slog.String("error", err.Error()))
		return domain.ExecutionResult{
			StepIndex: step.Index,
			Stderr:    err.Error(),
			ExitCode:  -1,
		}
	}
	return domain.ExecutionResult{
		StepIndex: step.Index,
		Stdout:    strings.TrimSpace(answer),
	}
}

// runAtlas executes a kubectl command against the configured Kubernetes cluster.
func (uc *ExecutePlanUseCase) runAtlas(ctx context.Context, step domain.Step) (domain.ExecutionResult, bool) {
	uc.log.Info("[Vector] routing step to atlas",
		slog.Int("step", step.Index),
		slog.String("command", step.Command),
	)
	res, err := uc.atlas.Execute(ctx, step.Command, 30)
	if err != nil {
		uc.log.Error("[Vector] atlas error",
			slog.Int("step", step.Index),
			slog.String("error", err.Error()),
		)
		return domain.ExecutionResult{StepIndex: step.Index, Stderr: err.Error(), ExitCode: -1}, false
	}
	return domain.ExecutionResult{
		StepIndex:  step.Index,
		Stdout:     res.Stdout,
		Stderr:     res.Stderr,
		ExitCode:   int(res.ExitCode),
		DurationMs: res.DurationMs,
	}, false
}

// runCrucible executes a step in an ephemeral Docker sandbox via the Crucible service.
// The step command is run as-is; the runtime tag defaults to "bash" when absent.
func (uc *ExecutePlanUseCase) runCrucible(ctx context.Context, step domain.Step) (domain.ExecutionResult, bool) {
	runtime := step.AgentID // AgentID reused as runtime tag for crucible steps (e.g. "python3")
	if runtime == "" {
		runtime = "bash"
	}
	uc.log.Info("[Vector] routing step to crucible",
		slog.Int("step", step.Index),
		slog.String("runtime", runtime),
		slog.String("command", step.Command),
	)
	res, err := uc.crucible.RunCode(ctx, runtime, step.Command, 30)
	if err != nil {
		uc.log.Error("[Vector] crucible error",
			slog.Int("step", step.Index),
			slog.String("error", err.Error()),
		)
		return domain.ExecutionResult{StepIndex: step.Index, Stderr: err.Error(), ExitCode: -1}, false
	}
	return domain.ExecutionResult{
		StepIndex:  step.Index,
		Stdout:     res.Stdout,
		Stderr:     res.Stderr,
		ExitCode:   int(res.ExitCode),
		DurationMs: res.DurationMs,
	}, false
}

// runWeb executes a "web" step via the Beacon service.
// If the command starts with http:// or https:// it fetches the URL;
// otherwise it performs a web search for the query.
func (uc *ExecutePlanUseCase) runWeb(ctx context.Context, step domain.Step) domain.ExecutionResult {
	cmd := strings.TrimSpace(step.Command)
	if strings.HasPrefix(cmd, "http://") || strings.HasPrefix(cmd, "https://") {
		uc.log.Info("[Vector] routing web step to beacon fetch",
			slog.Int("step", step.Index),
			slog.String("url", cmd),
		)
		result, err := uc.beacon.Fetch(ctx, cmd, 0)
		if err != nil {
			return domain.ExecutionResult{StepIndex: step.Index, Stderr: err.Error(), ExitCode: -1}
		}
		if result.Error != "" {
			return domain.ExecutionResult{StepIndex: step.Index, Stderr: result.Error, ExitCode: 1}
		}
		return domain.ExecutionResult{StepIndex: step.Index, Stdout: result.Body}
	}

	uc.log.Info("[Vector] routing web step to beacon search",
		slog.Int("step", step.Index),
		slog.String("query", cmd),
	)
	results, engine, err := uc.beacon.Search(ctx, cmd, 5)
	if err != nil {
		return domain.ExecutionResult{StepIndex: step.Index, Stderr: err.Error(), ExitCode: -1}
	}
	if len(results) == 0 {
		return domain.ExecutionResult{StepIndex: step.Index, Stdout: "no results found"}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results via %s for %q:\n\n", engine, cmd))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n   %s\n\n", i+1, r.Title, r.URL, r.Snippet))
	}
	return domain.ExecutionResult{StepIndex: step.Index, Stdout: strings.TrimSpace(sb.String())}
}

// runCourier dispatches a step to a remote agent via the Courier service.
func (uc *ExecutePlanUseCase) runCourier(ctx context.Context, sessionID, planID string, step domain.Step) (domain.ExecutionResult, bool) {
	if step.AgentID == "" {
		reason := "courier step missing agent target"
		uc.log.Warn("[Vector] courier step has no agent", slog.Int("step", step.Index))
		return domain.ExecutionResult{StepIndex: step.Index, Skipped: true, SkipReason: reason}, true
	}
	uc.log.Info("[Vector] routing step to courier",
		slog.Int("step", step.Index),
		slog.String("agent", step.AgentID),
		slog.String("command", step.Command),
	)
	result, err := uc.courier.Dispatch(ctx, sessionID, planID, step)
	if err != nil {
		uc.log.Error("[Vector] courier dispatch error",
			slog.Int("step", step.Index),
			slog.String("agent", step.AgentID),
			slog.String("error", err.Error()),
		)
		return domain.ExecutionResult{StepIndex: step.Index, Stderr: err.Error(), ExitCode: -1}, false
	}
	return result, false
}

func (uc *ExecutePlanUseCase) tryExecute(ctx context.Context, sessionID, planID string, step domain.Step, prior []stepOutput) (domain.ExecutionResult, bool) {
	if step.Verdict != "allow" {
		reason := fmt.Sprintf("verdict: %s", step.Verdict)
		uc.log.Info("[Vector] skipping step",
			slog.Int("step", step.Index),
			slog.String("verdict", step.Verdict),
		)
		_ = uc.telemetry.Emit(ctx, ports.Event{
			Name: "vector.step.skipped",
			Payload: map[string]any{
				"session_id": sessionID,
				"plan_id":    planID,
				"step_index": step.Index,
				"reason":     reason,
			},
		})
		return domain.ExecutionResult{
			StepIndex:  step.Index,
			Skipped:    true,
			SkipReason: reason,
		}, true
	}
	if step.Command == "" {
		reason := "no command provided"
		uc.log.Warn("[Vector] skipping step with no command", slog.Int("step", step.Index))
		_ = uc.telemetry.Emit(ctx, ports.Event{
			Name: "vector.step.skipped",
			Payload: map[string]any{
				"session_id": sessionID,
				"plan_id":    planID,
				"step_index": step.Index,
				"reason":     reason,
			},
		})
		return domain.ExecutionResult{
			StepIndex:  step.Index,
			Skipped:    true,
			SkipReason: reason,
		}, true
	}

	// Sentinel: safety-check every executable command before it reaches an executor.
	// infer steps are LLM calls, not shell commands — they bypass the safety filter.
	if step.Executor != "infer" {
		callerRoles := metadata.ValueFromIncomingContext(ctx, "x-roles")
		sv := uc.sentinel.Check(step.Command, callerRoles...)
		if !sv.Allowed {
			reason := fmt.Sprintf("sentinel blocked [%s]: %s", sv.Rule, sv.Reason)
			uc.log.Warn("[Sentinel] command blocked",
				slog.Int("step", step.Index),
				slog.String("rule", sv.Rule),
				slog.String("reason", sv.Reason),
				slog.String("command", step.Command),
			)
			_ = uc.telemetry.Emit(ctx, ports.Event{
				Name: "sentinel.blocked",
				Payload: map[string]any{
					"session_id": sessionID,
					"plan_id":    planID,
					"step_index": step.Index,
					"executor":   step.Executor,
					"rule":       sv.Rule,
					"reason":     sv.Reason,
					"command":    step.Command,
				},
			})
			return domain.ExecutionResult{
				StepIndex:  step.Index,
				Skipped:    true,
				SkipReason: reason,
				Stderr:     reason,
				ExitCode:   -1,
			}, true
		}
		if sv.Severity == sentinel.SeverityWarn {
			uc.log.Warn("[Sentinel] command allowed with warning",
				slog.Int("step", step.Index),
				slog.String("rule", sv.Rule),
				slog.String("reason", sv.Reason),
			)
			_ = uc.telemetry.Emit(ctx, ports.Event{
				Name: "sentinel.warn",
				Payload: map[string]any{
					"session_id": sessionID,
					"plan_id":    planID,
					"step_index": step.Index,
					"rule":       sv.Rule,
					"reason":     sv.Reason,
				},
			})
		}
	}

	// infer steps are answered directly by the inference service — no shell involved.
	if step.Executor == "infer" {
		result := uc.runInfer(ctx, step, prior)
		if result.Stdout != "" {
			answer := strings.TrimSpace(result.Stdout)
			// Store assistant turn for in-session conversation history.
			assistantTurn := fmt.Sprintf(`{"content":%q}`, answer)
			_ = uc.memory.StoreEvent(ctx, sessionID, "cortex.turn.assistant", assistantTurn)
			// Persist the Q&A pair to Vault for cross-session recall, scoped to the caller.
			_ = uc.vault.Store(ctx, metaVal(ctx, "x-user-id"), metaVal(ctx, "x-workspace-id"),
				fmt.Sprintf("Q: %s\nA: %s", step.Command, answer), "vector.infer")
		}
		return result, false
	}

	// courier steps are dispatched to a connected remote agent.
	if step.Executor == "courier" {
		return uc.runCourier(ctx, sessionID, planID, step)
	}

	// crucible steps run in an ephemeral Docker sandbox.
	if step.Executor == "crucible" {
		return uc.runCrucible(ctx, step)
	}

	// atlas steps run kubectl commands against the configured cluster.
	if step.Executor == "atlas" {
		return uc.runAtlas(ctx, step)
	}

	// web steps use Beacon for web search or URL fetch.
	if step.Executor == "web" {
		return uc.runWeb(ctx, step), false
	}

	executor, ok := uc.selectExecutor(step)
	if !ok {
		reason := fmt.Sprintf("executor %q not yet supported", step.Executor)
		uc.log.Info("[Vector] no executor for step",
			slog.Int("step", step.Index),
			slog.String("executor", step.Executor),
		)
		_ = uc.telemetry.Emit(ctx, ports.Event{
			Name: "vector.step.skipped",
			Payload: map[string]any{
				"session_id": sessionID,
				"plan_id":    planID,
				"step_index": step.Index,
				"reason":     reason,
			},
		})
		return domain.ExecutionResult{
			StepIndex:  step.Index,
			Skipped:    true,
			SkipReason: reason,
		}, true
	}

	uc.log.Info("[Vector] routing step",
		slog.Int("step", step.Index),
		slog.String("executor", step.Executor),
		slog.String("command", step.Command),
	)

	t0 := time.Now()
	result, err := executor.Execute(ctx, sessionID, planID, step)
	metrics.StepExecDurationSeconds.Observe(time.Since(t0).Seconds())
	if err != nil {
		uc.log.Error("[Vector] executor error",
			slog.Int("step", step.Index),
			slog.String("executor", step.Executor),
			slog.String("error", err.Error()),
		)
		return domain.ExecutionResult{
			StepIndex:  step.Index,
			Stderr:     err.Error(),
			ExitCode:   -1,
		}, false
	}
	return result, false
}
