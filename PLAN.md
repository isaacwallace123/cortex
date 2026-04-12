# Cortex — Improvement Plan

Audit-driven. Four tracks, ordered by user impact.

---

## Track 1 — Brain Intelligence

### 1A — Axiom few-shot examples
**Problem:** The planning prompt has no examples. The LLM infers the step format entirely from the instruction text, which leads to inconsistent output — wrong executor names, malformed tags, extra prose leaking through.

**Fix:** Embed 3–4 concrete few-shot examples directly in `buildPlanPrompt()` covering each executor type (`infer`, `shell`, `filesystem`, `courier`). Examples should show the exact `STEP N:` format with all three tags.

**File:** `services/brain/internal/application/create_plan.go`

---

### 1B — Prism intent classification hardening
**Problem:** Prism falls back to `conversation` on any invalid output (line 152, `parse_input.go`). A single bad token from the LLM silently degrades every request that would have been `execution` or `tool_assisted` into a no-op chat response.

**Fix:**
- Add a retry (max 2 attempts) before falling back
- Log a warning with the raw LLM output when the fallback triggers so it's visible in Prometheus/logs
- Tighten the prompt: enumerate all five valid modes explicitly and instruct the model to output exactly one

**File:** `services/brain/internal/application/parse_input.go`

---

### 1C — Axiom temperature + determinism
**Problem:** Temperature 0.2 on planning is still noisy enough to produce occasional hallucinated executors or steps. For a task planner, near-deterministic output is preferable.

**Fix:** Lower Axiom temperature to `0.1`. Raise Prism to `0.0` (fully deterministic — it's classification, not generation). Leave `stream_chat` at 0.7 (creative responses benefit from variance).

**Files:** `create_plan.go`, `parse_input.go`

---

### 1D — Arsenal tool injection into planning prompt
**Problem:** Arsenal tools are listed in the planning prompt but the format is generic. The LLM has no signal about which tools match the current intent, so it either ignores them or picks randomly.

**Fix:** Before building the prompt, query Arsenal for tools whose tags/description semantically overlap with the parsed entities. Inject only the top-N relevant tools (max 5) with their full schema rather than dumping the entire registry.

**File:** `services/brain/internal/application/create_plan.go`

---

## Track 2 — Recall & Context Quality

### 2A — Vault result ranking
**Problem:** Vault search returns up to 5 results but they are injected in raw retrieval order with no relevance signal. The most useful facts may be truncated if they land at the end and the budget is exhausted.

**Fix:** Sort Vault results by their FTS5 rank score (already returned by SQLite FTS5) descending before assembling. Most relevant facts get injected first and survive truncation.

**File:** `services/vault/internal/` (search handler) + `services/brain/internal/recall/assembler.go`

---

### 2B — Token budget accuracy
**Problem:** The 2000-token budget is enforced by multiplying tokens × 4 characters. This is a rough average; code, JSON, and non-English content can be 2–3× denser. Context occasionally exceeds the real token window.

**Fix:** Replace the character-based estimate with a proper token count using a `tiktoken`-equivalent word-piece estimate (split on whitespace + punctuation, count pieces). Keep the 2000-token ceiling but measure it more accurately.

**File:** `services/brain/internal/recall/assembler.go`

---

### 2C — Session context window
**Problem:** `BuildSessionContext()` injects the full session history from Echo. For long sessions this can consume most of the token budget, leaving little room for Vault knowledge.

**Fix:** Cap session history at the last N turns (configurable, default 10). Older turns beyond that limit are dropped. Vault knowledge always gets at least 25% of the budget regardless of session length.

**File:** `services/brain/internal/recall/assembler.go`

---

## Track 3 — CLI Modernisation

### 3A — Migrate to the persistent chat system
**Problem:** The CLI uses `/v1/chat` (stateless, plan-only) and `/v1/run`. The new `/v1/chats` system exists — named chats, persistent message history, multi-session — but the CLI doesn't use it at all.

**Fix:**
- Add `POST /v1/chats` and `GET /v1/chats` to `client/api.go`
- Add `POST /v1/chats/{id}/messages` and `GET /v1/chats/{id}/messages`
- Replace the TUI's `cmdFetchPlan` flow with a `CreateMessage` → `stream response` flow
- Add a chat selection screen (list existing chats, create new, delete)

**Files:** `apps/cli/internal/client/api.go`, `apps/cli/internal/tui/model.go`, `apps/cli/internal/cmd/chat.go`

---

### 3B — SSE streaming in the TUI
**Problem:** The TUI calls `api.Chat()` synchronously and blocks until the full response arrives. The Brain and API both support server-sent events on `/v1/chats/{id}/stream` but nothing consumes them.

**Fix:**
- Add `StreamChat(chatID string) (<-chan string, error)` to `client/api.go` — reads `data:` lines from the SSE response and emits tokens on a channel
- In the TUI, show tokens as they arrive in a streaming `stateStreaming` state with a live-updating viewport
- Emit a `tea.Msg` per token so Bubble Tea re-renders incrementally

**Files:** `apps/cli/internal/client/api.go`, `apps/cli/internal/tui/model.go`

---

### 3C — Config file for API URL + credentials
**Problem:** The CLI requires `CORTEX_API_URL` and `CORTEX_API_KEY` env vars or `--api`/`--key` flags on every invocation. There is no persistent config file.

**Fix:** On first run (or `cortex config set`), write `~/.config/cortex/config.json` with `api_url` and `api_key`. `LoadSession()` already exists — extend it to also load/save these fields. CLI flags and env vars override the config file.

**Files:** `apps/cli/internal/client/session.go`, `apps/cli/internal/cmd/root.go`

---

## Track 4 — Service Hardening

### 4A — Kubernetes liveness + readiness probes
**Problem:** No `livenessProbe` or `readinessProbe` are defined in the Helm chart. Pods that crash silently or get into a bad state are not restarted by Kubernetes automatically.

**Fix:** Add to the deployment template:
- `livenessProbe`: `GET /healthz` on the metrics port, `initialDelaySeconds: 10`, `periodSeconds: 30`, `failureThreshold: 3`
- `readinessProbe`: same path, `initialDelaySeconds: 5`, `periodSeconds: 10` — gates traffic until the service is up
- `startupProbe` for inference and ollama only: longer `initialDelaySeconds: 60` to account for model load time

**File:** `deploy/helm/cortex/templates/deployment.yaml`, `deploy/helm/cortex/values.yaml`

---

### 4B — Per-service resource tuning
**Problem:** All 21 services share the same `defaultResources` (50m CPU / 64Mi memory). Brain, inference, and ollama need significantly more; stateless services like shell and atlas need less.

**Fix:** Add optional `resources` override per service in `values.yaml`. Services that don't specify one inherit `defaultResources`. At minimum, set higher limits for:
- `brain`: 200m CPU / 512Mi memory
- `inference`: 200m CPU / 512Mi memory  
- `ollama`: already has its own block — increase to 4Gi memory limit

**Files:** `deploy/helm/cortex/values.yaml`, `deploy/helm/cortex/templates/deployment.yaml`

---

### 4C — Graceful shutdown timeout alignment
**Problem:** Memory, inference, nerva and vault all use a 5-second HTTP shutdown timeout. Under load a gRPC stream or SQLite write in flight will be killed mid-operation.

**Fix:** Raise the shutdown timeout from 5s to 15s across all services to match the API gateway. The signal context already propagates correctly — this is a one-liner per service.

**Files:** `services/*/cmd/*/main.go` (memory, inference, nerva, vault, policy, arsenal, compass, workspace, sovereign, chat, vault)

---

## Order of Work

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | 4A — K8s probes | High — pods recover automatically | Low |
| 2 | 1A — Few-shot examples | High — immediately smarter planning | Low |
| 3 | 1C — Temperature tuning | Medium — more deterministic output | Trivial |
| 4 | 1B — Prism hardening | Medium — eliminates silent fallbacks | Low |
| 5 | 2C — Session window cap | Medium — prevents context overflow | Low |
| 6 | 2A — Vault ranking | Medium — better long-term memory recall | Low |
| 7 | 4B — Resource tuning | Medium — prevents OOM on brain/inference | Low |
| 8 | 3C — CLI config file | Medium — usability | Low |
| 9 | 2B — Token budget accuracy | Low-medium — correctness improvement | Medium |
| 10 | 1D — Arsenal tool injection | High — unlocks tool use properly | Medium |
| 11 | 4C — Shutdown timeouts | Low — correctness under load | Trivial |
| 12 | 3A — Chat system migration | High — modernises entire CLI flow | High |
| 13 | 3B — SSE streaming in TUI | High — real-time feel | High |
