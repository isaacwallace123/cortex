# CORTEX — Master Implementation Prompt

> **What this document is:** A single, self-contained specification you paste into Claude Code (or any agentic coding assistant) so it can implement the entire Cortex system milestone-by-milestone. It combines the project vision, naming conventions, subsystem definitions, current repo state, all 16 epics, data contracts, and execution rules into one reference.

---

## 1 — Project Identity

**Cortex** is a personal AI control plane, agentic assistant platform, and conversational AI companion.

It enables:

- **Conversation** — Talk to Cortex about anything: brainstorm ideas, discuss topics, ask questions, think out loud
- **Project planning** — Plan, track, and manage projects with structured milestones, tasks, and notes
- **Sandboxed code execution** — Spin up isolated environments to write, run, test, and iterate on code safely
- **Local machine control** — Execute commands, manage files, automate workflows on your machine
- **Cluster orchestration** — Manage Kubernetes clusters, deployments, and infrastructure
- **Workspace automation** — Per-project isolated environments with scoped context
- **Internet access** — Search the web, call APIs, fetch documentation
- **Persistent memory & long-term learning** — Remembers past sessions, learns your preferences, builds knowledge over time
- **Intelligent recommendations** — Proactive insights based on system telemetry and past experience
- **Multi-device coordination** — Edge agents on every machine, orchestrated centrally
- **Multi-user environments** — Separate identities, permissions, and contexts

### What Cortex IS

| ✅ It is                                        | ❌ It is NOT                       |
| ----------------------------------------------- | ---------------------------------- |
| A personal AI you can talk to about _anything_  | Just an LLM wrapper                |
| A system operator that executes on your behalf  | A dumb terminal                    |
| A distributed AI runtime with agents everywhere | A single-machine script            |
| A project planner that tracks your work         | A glorified to-do app              |
| A code sandbox that tests before it suggests    | An AI that only talks but can't do |
| An AI operating system                          | A monolith                         |

### The Goal Feel

When complete, Cortex should feel like:

1. A **personal AI companion** — you can talk to it about anything: random topics, ideas, debugging, life. It remembers you.
2. An **AI operating system** — terminal-first, always aware of your infra, managing your machines.
3. A **DevOps controller** — managing clusters, nodes, containers autonomously.
4. A **project planning partner** — helps you plan, break down, and track complex projects with real structure.
5. A **code laboratory** — spins up sandboxed environments to run and test code, iterates on solutions before presenting them.
6. A **distributed agent system** — edge agents on every device, coordinated centrally.
7. A **personal assistant with memory** — it remembers past sessions, learns context, and improves over time.

---

## 2 — Hard Rules (Read First)

1. **DO NOT overbuild.** Every line of code must justify itself.
2. **DO NOT create unused abstractions.** No speculative interfaces, no "just in case" layers.
3. **ALWAYS work in milestones.** Each milestone must be a vertical slice that produces working, testable behavior.
4. **EVERY milestone must produce working behavior.** No "scaffold-only" milestones.
5. **KEEP architecture modular and replaceable.** Hexagonal / ports-and-adapters. No service should hard-depend on another's internals.
6. **Minimal steps only.** No unnecessary actions, no hallucinated capabilities.
7. **Respect naming.** All subsystem names below are **mandatory** — do not rename them.
8. **Proto-first contracts.** When adding a new service or expanding an existing one, define the `.proto` file first, generate, then implement. The proto IS the contract.
9. **Errors must be structured.** Every service returns domain errors with error codes, not raw strings. Use gRPC status codes properly.
10. **Every service must support graceful shutdown.** Handle `SIGTERM`/`SIGINT`, drain connections, flush state.
11. **Correlation IDs on everything.** Every request entering the system gets a trace/correlation ID that propagates through all service calls and events for end-to-end tracing.

---

## 3 — Subsystem Glossary (Mandatory Naming)

Every module in Cortex has a codename. These names are non-negotiable and must be used in code, logs, configs, and documentation.

### 3.1 — Brain (Decision Layer)

| Module     | Role             | Responsibilities                                                  | Example                                      |
| ---------- | ---------------- | ----------------------------------------------------------------- | -------------------------------------------- |
| **Prism**  | Intent Parsing   | Understands user intent, structures raw input into a typed intent | `"restart my server"` → `cluster_action`     |
| **Axiom**  | Reasoning Engine | Thinks, plans, decides whether to execute or ask follow-up        | Generates a multi-step execution plan        |
| **Vector** | Routing Engine   | Decides **where** execution goes (which executor)                 | local → Shell, k8s → Atlas, remote → Courier |

### 3.2 — Memory Layer

Memory is split into three cooperating modules. No single module handles everything.

| Module     | Role                          | Responsibilities                                                                                                        | Backing Store                                                  | Example                                                                                              |
| ---------- | ----------------------------- | ----------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| **Echo**   | Session / conversation memory | Stores the event stream for active and recent sessions. Short-to-medium term. Fast reads, append-only writes.           | SQLite (exists today)                                          | "What did I ask 3 messages ago?", restoring conversation context on reconnect                        |
| **Vault**  | Long-term knowledge store     | Persists facts, learned patterns, user preferences, past solutions, project-specific knowledge. Semantic/vector search. | Vector DB (e.g. Qdrant, Milvus, or embedded SQLite + pgvector) | "How did I fix the OOM issue last month?", "What's my preferred k8s namespace?"                      |
| **Recall** | Retrieval & context assembly  | Queries both Echo and Vault, ranks/deduplicates results, assembles a context window that gets injected into Prism/Axiom | Stateless (queries Echo + Vault)                               | Before Axiom plans, Recall fetches the 5 most relevant past interactions and injects them as context |

**How they interact:**

```
User message arrives
       │
       ▼
    Recall ──────► Echo   (fetch recent session context)
       │
       ├────────► Vault   (semantic search for relevant long-term knowledge)
       │
       ▼
  Assembled context window
       │
       ▼
  Prism / Axiom  (intent parsing + planning with full context)
       │
       ▼
  After execution, results are:
       ├────────► Echo   (append to session stream)
       └────────► Vault  (extract & store any learnable facts)
```

**Isolation:** All three modules are scoped by `user_id` and `workspace_id`. A user's Vault knowledge in workspace `infra` is invisible in workspace `dev` unless explicitly shared.

**Why three modules instead of one:**

- **Echo** is optimized for sequential, append-only writes and time-ordered reads (session log). It stays fast and small.
- **Vault** is optimized for semantic similarity search across a large, growing knowledge base. Different storage engine, different access patterns.
- **Recall** is stateless glue — it owns the _strategy_ of what context to assemble (recency bias, relevance ranking, token budget management) without owning any data itself. This keeps the retrieval logic testable and swappable without touching storage.

### 3.3 — Execution Layer

| Module       | Role                              | Example                                                                           |
| ------------ | --------------------------------- | --------------------------------------------------------------------------------- |
| **Shell**    | Local command execution           | `run_command("docker ps")`                                                        |
| **Forge**    | Workspace / code / file execution | Reading code, editing files, scanning projects                                    |
| **Atlas**    | Kubernetes / cluster execution    | `kubectl get pods`, restart deployment                                            |
| **Courier**  | Remote execution via agents       | Run command on a remote node, deploy to server                                    |
| **Crucible** | Sandboxed code execution          | Spin up an isolated container, run Python/Go/Node code, capture output, tear down |

### 3.4 — Intelligence Layer

| Module        | Role                                               | Example                                                                           |
| ------------- | -------------------------------------------------- | --------------------------------------------------------------------------------- |
| **Pulse**     | Telemetry / health / heartbeat / metrics           | CPU usage, memory usage, agent online/offline, GPU usage                          |
| **Overwatch** | Analysis & recommendations (interprets Pulse data) | "Node is under memory pressure", "Pod is crash-looping", "Disk nearing capacity"  |
| **Compass**   | Project planning & management                      | Create project plan, break into milestones/tasks, track progress, store decisions |

### 3.5 — System Layer

| Module        | Role                                     | Concept                        | Example                                                    |
| ------------- | ---------------------------------------- | ------------------------------ | ---------------------------------------------------------- |
| **Nerva**     | Event system (pub/sub, internal signals) | The nervous system             | `agent.registered`, `task.started`, `approval.requested`   |
| **Sovereign** | Authentication & identity                | **Who you are**                | Login, device binding, session tokens, identity validation |
| **Aegis**     | Policy & approval system                 | **What you are allowed to do** | "Requires approval to delete file", "admin-only operation" |
| **Sentinel**  | Safety enforcement                       | **What is safe to do**         | Blocking `rm -rf /`, preventing destructive commands       |

### 3.6 — Capability Layer

| Module       | Role                                         | Example                                                                                 |
| ------------ | -------------------------------------------- | --------------------------------------------------------------------------------------- |
| **Arsenal**  | Tool registry — stores all available tools   | `run_command`, `read_file`, `kubectl_get_pods`                                          |
| **Catalyst** | Skill engine — composes tools into workflows | `diagnose_cluster_issue`, `analyze_logs`                                                |
| **Genesis**  | Tool & skill auto-generation                 | Creates new tools/skills — **must** go through Aegis, **must** be validated by Sentinel |

### 3.7 — External

| Module     | Role                                           | Example                              |
| ---------- | ---------------------------------------------- | ------------------------------------ |
| **Beacon** | Internet access — web search, APIs, HTTP calls | Fetch API data, search documentation |

### 3.8 — Interfaces / Agents

| Module          | Role                                       | Stack / Runs On                                                                                        | Repository                                                  |
| --------------- | ------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------- |
| **Halo**        | Desktop interface (Tauri app)              | React + TypeScript (UI), Tauri, Rust                                                                   | **Separate repo** (not in this monorepo)                    |
| **Halo Edge**   | Embedded edge runtime (runs _inside_ Halo) | Rust — tool execution, filesystem access, terminal bridge, device registration, comms with Cortex Core | **Inside Halo repo**                                        |
| **Relay Agent** | Headless edge agent                        | Go — runs on servers/nodes/infrastructure                                                              | **Inside this monorepo** (`apps/relay/` or `agents/relay/`) |

> **Important:** Halo (the desktop app) lives in its own repository. This monorepo defines the **API contract** (protobuf / OpenAPI) that Halo consumes, but does not contain Halo source code. The Cortex Core API must be stable and well-documented enough for Halo to develop against independently.

### 3.9 — Module Deep Dives

#### Crucible (Sandboxed Execution Environments)

Crucible is Cortex's **code laboratory**. It lets the AI (or the user) spin up fully isolated, ephemeral environments to write, run, test, and iterate on code — without touching the host system.

**Core concept:** When Cortex needs to test code, validate a fix, prototype a solution, or run untrusted code, it creates a Crucible sandbox rather than executing on the host.

**How it works:**

```
User: "Write me a Python script that scrapes HN and shows the top 10 posts"
       │
       ▼
  Prism: intent = code_execute
  Axiom: plan = [write code → create sandbox → run → capture output → iterate if errors → present result]
       │
       ▼
  Crucible:
    1. Create sandbox (Python 3.12 container, with pip)
    2. Write script to sandbox filesystem
    3. Install dependencies (pip install requests beautifulsoup4)
    4. Run script, capture stdout/stderr
    5. If error → Axiom analyzes → modifies code → re-run (up to N retries)
    6. Return final output + working code to user
    7. Tear down sandbox (or keep alive if user wants to iterate)
```

**Sandbox types:**

| Type        | Backing                     | Use Case                                  |
| ----------- | --------------------------- | ----------------------------------------- |
| `container` | Docker (default)            | General-purpose, any language. Ephemeral. |
| `vm`        | Firecracker / QEMU (future) | High-isolation for untrusted code         |
| `wasm`      | Wasm runtime (future)       | Ultra-lightweight, sub-second startup     |

**Sandbox lifecycle:**

- `Create(config) → sandbox_id` — spins up environment with specified runtime, packages, resource limits
- `Exec(sandbox_id, command) → {stdout, stderr, exit_code}` — run a command inside the sandbox
- `WriteFile(sandbox_id, path, content)` — inject files into the sandbox
- `ReadFile(sandbox_id, path) → content` — extract files from the sandbox
- `Destroy(sandbox_id)` — tear down and clean up
- `Snapshot(sandbox_id) → image` — (future) save sandbox state for replay

**Resource limits (per sandbox):**

- CPU: configurable (default 1 core)
- Memory: configurable (default 512MB)
- Disk: configurable (default 1GB)
- Network: disabled by default, opt-in with Sentinel approval
- Timeout: configurable (default 5 minutes, max 30 minutes)

**Integration with Brain:**

- Axiom can autonomously create sandboxes to test solutions before presenting them
- The AI **iterates** inside the sandbox: write code → run → read errors → fix → re-run
- Only presents the final working result to the user (unless the user wants to see the process)
- Sandbox results are stored in Echo (session) and optionally Vault (learned solutions)

**Safety:** Crucible sandboxes are fully isolated. Network access is off by default. Sentinel must approve any sandbox with network access enabled. The host filesystem is never mounted.

---

#### Compass (Project Planning & Management)

Compass is Cortex's **project brain**. Use it to plan projects, break them into milestones and tasks, track progress, store architectural decisions, and get AI-assisted planning.

**Core concept:** When you're starting a project or have a complex goal, Cortex helps you think through it, creates a structured plan, and tracks your progress over time.

**How it works:**

```
User: "I want to build a REST API for my recipe app"
       │
       ▼
  Prism: intent = project_plan
  Axiom: plan = [discuss requirements → create project → define milestones → break into tasks]
       │
       ▼
  Compass:
    1. Create project: "Recipe API"
    2. AI-assisted milestone breakdown:
       - M1: Data model & DB schema
       - M2: CRUD endpoints
       - M3: Auth & users
       - M4: Search & filtering
       - M5: Deployment
    3. Each milestone → tasks with descriptions, acceptance criteria
    4. Store in workspace-scoped project storage
    5. User can ask "what's next?" → Compass checks progress → suggests next task
```

**Data model:**

```yaml
project:
  project_id: string
  user_id: string
  workspace_id: string
  name: string
  description: string
  status: "planning" | "active" | "paused" | "completed"
  created_at: timestamp
  updated_at: timestamp

milestone:
  milestone_id: string
  project_id: string
  name: string
  description: string
  status: "pending" | "in_progress" | "completed"
  order: int
  tasks: [task]

task:
  task_id: string
  milestone_id: string
  name: string
  description: string
  status: "todo" | "in_progress" | "done" | "blocked"
  acceptance_criteria: [string]
  notes: [string]         # AI or user can append notes
  dependencies: [task_id] # tasks that must complete first
  created_at: timestamp
  completed_at: timestamp

decision:
  decision_id: string
  project_id: string
  title: string           # e.g., "Use PostgreSQL over MongoDB"
  rationale: string       # why this decision was made
  alternatives: [string]  # what else was considered
  decided_at: timestamp
```

**Capabilities:**

- **AI-assisted planning:** Describe what you want to build → Compass + Axiom break it into structured milestones and tasks
- **Progress tracking:** Ask "what's my progress on X?" → Compass shows status across milestones
- **Next step suggestions:** "What should I work on next?" → Compass evaluates dependencies and priorities
- **Decision logging:** Record architectural decisions with rationale so you remember _why_ you chose something
- **Integration with Crucible:** Tasks can trigger sandbox execution (e.g., "implement this task" → Crucible prototypes it)
- **Integration with Vault:** Project knowledge (decisions, solutions, patterns) persists in Vault for long-term retrieval

**Storage:** Compass data is stored per-user, per-workspace. Projects in workspace `dev` are invisible in workspace `infra`.

---

## 4 — Current Repository State

The repo is a **Go monorepo** using `go.work` (Go 1.24). Module path: `github.com/isaacwallace123/cortex`. Architecture follows hexagonal / ports-and-adapters per service.

```
cortex/
├── apps/
│   └── cli/                  # CLI client (Cobra-based, TUI with Bubble Tea)
│       ├── cmd/cortex/
│       └── internal/
│           ├── client/       # gRPC/HTTP client wrappers
│           ├── render/       # output formatting
│           └── tui/          # terminal UI
├── services/
│   ├── api/                  # API Gateway   — HTTP → gRPC translation (port 8000)
│   ├── brain/                # Brain core    — Prism + Axiom + Vector (ports 8080/8090)
│   │   └── internal/
│   │       ├── adapters/     # inference, memory, nerva, policy, shell, forge, telemetry
│   │       ├── application/  # parse_input, create_plan, execute_plan, get_session
│   │       ├── domain/       # plan.go, session.go, task.go
│   │       ├── ports/        # interfaces
│   │       └── transport/    # HTTP + gRPC handlers
│   ├── forge/                # Forge executor — file operations (port 5050)
│   ├── inference/            # Inference      — LLM abstraction, Ollama adapter (port 9090)
│   ├── memory/               # Memory (Echo)  — session events, SQLite (port 7070)
│   ├── nerva/                # Nerva          — event bus, pub/sub, SQLite (port 3030)
│   ├── policy/               # Policy (Aegis) — approval gate (port 6060)
│   └── shell/                # Shell executor — local commands (port 4040)
├── pkg/
│   └── observe/              # Shared observability (tracing)
├── deploy/
│   ├── docker/               # Ollama init script etc.
│   └── prometheus/           # Prometheus config
├── docker-compose.yml        # Full stack: ollama, memory, inference, shell, forge, nerva, policy, brain, prometheus, api
├── Makefile                  # up/down/build/rebuild/proto/cli/test/tidy/vet/chat/health
└── go.work                   # Workspace: cli, observe, api, brain, forge, inference, memory, nerva, policy, shell
```

### What Already Works

- **Brain (Prism)**: LLM-based intent parsing. Classifies into 4 modes: `direct_answer`, `execution`, `tool_assisted`, `clarification`. Extracts intent phrase and entities. Emits `prism.input.parsed` events to both Nerva and Echo. Task domain model has `IntentType`, `ExecutionStyle`, `Mode`, and `RawInput` fields.
- **Brain (Axiom)**: LLM-based plan creation. **Already has a no-plan path**: when mode is `direct_answer` or `clarification`, Axiom skips LLM planning and creates a single `infer` step that passes the raw question directly to the inference service. For `execution` and `tool_assisted` modes, generates multi-step plans with `STEP N: description [executor:type] [command:cmd]` format. Retrieves prior session context from Echo to give the LLM history. Gates every plan through Aegis (policy). Uses `CompleteStream` with fallback to `Complete`. Plans and verdicts are persisted to Echo as `axiom.plan.created` events.
- **Brain (Vector)**: Executes plan steps. Has a `StepExecutor` interface (`ports/step_executor.go`) with `Execute(ctx, sessionID, planID, step) → ExecutionResult`. `ExecutorPort` (Shell) and `ForgePort` (Forge) both implement `StepExecutor`. Routing via `selectExecutor()` switch on `step.Executor`: `"shell"` → Shell, `"filesystem"` → Forge, `"infer"` → direct inference call. Accumulates prior step stdout so `infer` steps can reason about real execution results. Emits `vector.step.executed` and `vector.step.skipped` telemetry events.
- **Brain (Metrics)**: Custom Prometheus metrics: `cortex_inputs_parsed_total` (Prism counter), `cortex_plans_created_total` (Axiom counter), `cortex_steps_evaluated_total` (Aegis verdict counter, labeled by verdict), `cortex_step_exec_duration_seconds` (Vector histogram).
- **Brain (Logging)**: Structured JSON logging via `slog` with per-subsystem loggers: `Prism`, `Axiom`, `Vector`, `Nerva`, `brain`. Each logger tagged with `"subsystem"` field.
- **Brain (Bootstrap)**: Clean DI wiring in `bootstrap/wire.go`. Every adapter has a noop/stub fallback — brain starts even if downstream services are unavailable.
- **Inference**: Ollama adapter with gRPC interface. Has both `Complete` (unary) and `CompleteStream` (token streaming via channel). Supports `list_models`. Domain model: `GenerationRequest` with model, prompt, temperature, max_tokens. Default model configurable via `INFERENCE_DEFAULT_MODEL`.
- **Memory (Echo)**: SQLite session event store with gRPC interface. Three use cases: `StoreEvent`, `GetSession` (returns all events for a session), `ListPlans` (extracts plan summaries from session events). JSON payload parsing for plan detail extraction. No user/workspace scoping yet.
- **Shell**: Subprocess executor (`adapters/subprocess/runner.go`). Runs commands via `exec.Command`. Returns structured `ExecutionResult` with stdout, stderr, exit_code, duration.
- **Forge**: Filesystem executor (`adapters/fs/executor.go`). Operates within a configurable root path (`FORGE_ROOT`). Supports: `ls`, `cat`, `stat`, `mkdir`, `write`, `rm`, `move/rename`.
- **Nerva**: Event bus with in-memory fan-out (`Bus`) + SQLite persistence. gRPC Publish/Subscribe. Server-side streaming for subscribers. Filter support (subsystem or exact event name match). Non-blocking send to slow consumers (drops rather than blocks).
- **Policy (Aegis)**: Full regex-based rule engine with ordered rule evaluation (first match wins). Default rules include:
  - `deny-recursive-delete`: blocks `rm -r` / `rm --recursive`
  - `deny-system-control`: blocks `shutdown`, `reboot`, `poweroff`, `halt`
  - `deny-privilege-escalation`: blocks `sudo`, `su`, `doas`
  - `deny-k8s`: blocks all k8s executor steps
  - `pending-filesystem-write`: requires approval for `write`, `mkdir`, `rm` in filesystem executor
  - `pending-network-ops`: requires approval for `curl`, `wget`, `ssh`, `scp`, etc.
  - `pending-code`: requires approval for code executor steps
  - Supports loading custom rules from JSON file at runtime. Has its own telemetry adapter.
- **API Gateway**: HTTP server with 6 endpoints:
  - `POST /v1/chat` — parse input + create plan (returns intent, entities, plan with steps and Aegis verdicts)
  - `POST /v1/run` — execute a plan's steps (returns per-step results with stdout/stderr/exit_code/duration)
  - `GET /v1/session/{id}` — retrieve session event history
  - `GET /v1/plans?session_id=X` — list plan summaries for a session
  - `GET /v1/plans/{id}?session_id=X` — get detailed plan with steps
  - `GET /healthz` — health check
  - **Middleware**: `withRequestID` (generates/propagates `X-Request-ID`), `withAuth` (API key via `X-Api-Key` or `Authorization: Bearer`), `withLogging` (structured request logging with method, path, status, duration, request_id).
- **CLI**: Cobra + Bubble Tea TUI. API client with typed methods for all 6 API endpoints. Has a **Nerva streaming client** (`client/nerva.go`) that subscribes to event streams via gRPC and returns events through a Go channel. ANSI rendering utilities.
- **Observability**: Prometheus scraping brain `/metrics`. OpenTelemetry tracing via `pkg/observe` (trace context propagation).
- **Proto**: 7 proto files — `brain/v1`, `forge/v1`, `inference/v1`, `memory/v1`, `nerva/v1`, `policy/v1`, `shell/v1`. All use `buf` for generation.
- **Docker Compose**: Full stack orchestration (ollama, memory, inference, shell, forge, nerva, policy, brain, prometheus, api). All inter-service communication internal via Docker network; only API (:8000), Nerva (:3030), and Prometheus (:9090) exposed to host.

### What Does NOT Exist Yet

- **True conversational mode** — Brain has `direct_answer` mode where Axiom creates an `infer` step, but Prism does NOT have a proper `conversation` intent. All inputs are still classified as intent + mode and routed through the plan pipeline. Needs: a `conversation` intent that bypasses planning entirely and generates a free-form response.
- **Vault** — No long-term knowledge store, no vector search, no semantic retrieval
- **Recall** — No context assembly / retrieval-augmented pipeline (Axiom only gets prior session events, no cross-session or semantic context)
- **Crucible** — No sandboxed code execution
- **Compass** — No project planning or management
- **Sovereign** — No user identity system. API has API-key auth only (shared key, no per-user identity). No sessions, no device binding.
- **Sentinel** — No dedicated safety service. Policy/Aegis has safety-like rules (deny destructive commands) but Sentinel as a separate pre-execution safety layer does not exist.
- **Atlas** — No Kubernetes executor. Policy has a `deny-k8s` rule that blocks k8s steps since the executor doesn't exist yet.
- **Courier** — No remote execution
- **Arsenal** — No formal tool registry. Tools are implicit in the LLM planning prompt (shell, filesystem, infer executors) and hardcoded in Vector's `selectExecutor()` switch.
- **Catalyst** — No skill/workflow engine
- **Genesis** — No tool auto-generation
- **Beacon** — No internet access
- **Overwatch** — No analysis/recommendation engine (only raw Prometheus metrics + Brain-level custom counters)
- **Halo** — No desktop app (will be separate repo)
- **Halo Edge** — No embedded edge runtime (will be in Halo repo)
- **Relay Agent** — No headless agent
- **Workspace isolation** — No per-user workspace scoping. Forge operates on a single shared `/workspace` root.
- **Chat system** — No multi-chat, no persistent chat history. Sessions are ephemerally scoped by `session_id` but there's no chat model.
- **Client-facing streaming** — Inference has `CompleteStream` internally, but the API Gateway does not expose streaming to clients (no SSE, no WebSocket). CLI gets batch responses.
- **WebSocket support** — No real-time channels for agent heartbeats, live logs, or streaming chat

---

## 5 — Epics (Complete Breakdown)

Each epic maps to a subsystem. They are designed to be **buildable in isolation** and turn into milestones directly.

> **Total: 16 epics** covering the full Cortex system.

---

### Epic 1 — Brain Core (Prism / Axiom / Vector)

**Goal:** Harden the core intelligence pipeline: `intent → reasoning → routing`. Support both action-oriented intents AND proper conversational intents.

**Deliverables:**

- Structured intent model with typed enum variants (not free-form strings from the LLM)
- **Proper conversational mode:** Prism should classify a `conversation` intent (distinct from `direct_answer`). When classified as `conversation`, Axiom generates a direct conversational response without the plan pipeline.
- Execution planner that produces validated, multi-step plans with dependency ordering
- Routing logic that dispatches to the correct executor based on a routing table
- Extend the existing `StepExecutor` interface to support new executors cleanly

**Flow (action):** `User: "restart my pod"` → Prism: `cluster_action` → Axiom: plan restart → Vector: route to Atlas

**Flow (conversation):** `User: "what do you think about Rust vs Go?"` → Prism: `conversation` → Axiom: generate response (with Recall context) → return to user. No executor involved.

**Status:** Partially implemented. Prism classifies into 4 modes (`direct_answer`, `execution`, `tool_assisted`, `clarification`) via LLM. Axiom **already has a no-plan path** for `direct_answer` and `clarification` — it creates a single `infer` step that calls inference directly with the raw user input. Vector has a `StepExecutor` interface and routes `shell` → Shell, `filesystem` → Forge, `infer` → Inference. Session context from Echo is injected into Axiom's planning prompt. Plans are gated through Aegis and persisted to Echo.

**What's missing / needs improvement:**

- Prism's classification is via free-form LLM output (parses `INTENT:` / `MODE:` / `ENTITIES:` lines). Needs: a formal typed intent enum instead of whatever the LLM produces. The `IntentType` and `ExecutionStyle` fields exist in the Task domain but Prism doesn't populate them from a fixed vocabulary.
- `direct_answer` mode still goes through the plan pipeline (creates a 1-step plan). True `conversation` intent should skip planning entirely and go straight to inference.
- Prism should output a confidence score. Unknown/ambiguous inputs should default to `conversation`, not `direct_answer`.
- **Prism must distinguish** between "I need information to act" (`ask_question`) and "the user wants to have a conversation" (`conversation`). Currently these are conflated into `direct_answer`.
- Axiom plans should be serializable and diffable (so users can review before execution)
- Vector's `selectExecutor()` is a simple switch. Needs a registry pattern so new executors can be added without modifying Vector source code.

---

### Epic 2 — Execution Layer (Shell / Forge / Atlas / Courier / Crucible)

**Goal:** Enable Cortex to actually _do things_ — reliably and extensibly.

**Deliverables:**

- Unified `Executor` interface: `Execute(ctx, step) → Result`
- All executors implement the same port — Shell, Forge, Atlas, Courier, Crucible
- Structured input/output for every execution step (not raw strings)
- Execution timeout and cancellation support
- Output streaming for long-running commands

**Status:** Shell and Forge exist and both implement the `StepExecutor` interface (via `ExecutorPort` and `ForgePort` which embed `StepExecutor`). However, the interface is defined in brain's ports — not in a shared `pkg/`. Atlas (k8s), Courier (remote), and Crucible (sandbox) are missing entirely.

**Improvements needed:**

- The `StepExecutor` interface exists in `services/brain/internal/ports/step_executor.go` but it's brain-private. Move it to `pkg/executor/` so all executor services share the same interface without importing brain internals.
- Shell and Forge already implement the interface but need to be verified against the shared version
- Every executor must report structured results: `{status, stdout, stderr, exit_code, duration}`

---

### Epic 3 — Capability System (Arsenal / Catalyst / Genesis)

**Goal:** Create a scalable, discoverable tool + workflow system.

**Deliverables:**

- **Arsenal:** Tool registry with schema validation. Each tool has: name, description, parameters schema (JSON Schema), required permissions, safety level
- **Catalyst:** Skill composition engine — DAG-based workflows built from Arsenal tools. Support conditionals, loops, error handling
- **Genesis:** Auto-generation pipeline. LLM proposes tool → Aegis approves → Sentinel validates safety → Arsenal registers

**Status:** Not implemented. Tools are currently hardcoded in brain adapters.

**Improvements needed:**

- Tools should be declarative (YAML/JSON definitions loadable at runtime), not Go code
- Arsenal should support tool versioning — agents may run different tool versions
- Catalyst skills should be shareable across users and workspaces

---

### Epic 4 — Sovereign (Auth & Identity)

**Goal:** Enable multi-user, multi-device Cortex.

**Deliverables:**

- User model: `{user_id, username, email, roles[], created_at, last_active}`
- Roles: `admin`, `user`, `restricted` (RBAC base layer)
- Session model: `{session_id, user_id, device_id, token, expires_at, created_at}`
- Device binding: sessions are tied to specific devices
- Token issuance and validation (JWT or opaque tokens)
- Middleware for all services to validate identity from request context

**Flow:** User logs in → Sovereign issues session → binds to device → all downstream requests carry `user_id` + `session_id` in metadata

**Status:** Not implemented. No auth of any kind exists. All requests are anonymous.

**Implementation note:** Sovereign should be a **new service** (`services/sovereign/`) with its own gRPC API. The API Gateway should call Sovereign to validate tokens on every inbound request (middleware pattern). Don't bake auth logic into the API Gateway itself.

---

### Epic 5 — Aegis (Policy & Approvals)

**Goal:** Control what actions are allowed, per-user and per-role.

**Deliverables:**

- Permission rules engine: define rules as `{role, action, resource, effect: allow|deny|require_approval}`
- Approval workflows: when an action `require_approval`, Cortex pauses, notifies the user, and waits for confirmation
- Approval persistence: pending approvals survive service restarts
- Policy evaluation: `Aegis.Evaluate(user, action, resource) → {allowed, denied, pending_approval}`

**Flow:** Delete file → Aegis evaluates → requires approval → user confirms → Aegis marks approved → executor proceeds

**Status:** More advanced than a basic gate. Has a full regex-based rule engine (`adapters/rules/engine.go`) with 7 default rules covering destructive commands, privilege escalation, network ops, filesystem writes, and code execution. Supports loading rules from JSON files. First-match-wins evaluation. But: no user-awareness (no concept of who is requesting), no role-based rules, no approval state persistence, no integration with Sovereign.

---

### Epic 6 — Sentinel (Safety Layer)

**Goal:** Prevent unsafe or destructive behavior at the execution boundary.

**Deliverables:**

- Safety filter rules: blocklist patterns (regex), allowlist overrides, severity levels
- Pre-execution hook: every executor calls Sentinel before running
- Configurable safety levels: `unrestricted` (dev), `standard` (default), `paranoid` (production)
- Audit log: every blocked action is logged with reason

**Flow:** `"rm -rf /"` → Sentinel pattern match → `BLOCKED: destructive filesystem operation` → logged → user notified

**Status:** Not implemented.

**Improvements needed:**

- Sentinel rules should be hot-reloadable (don't require restart to update safety rules)
- Should support both static rules (pattern matching) and dynamic rules (LLM-assessed risk)
- Must have an escape hatch: admin users + explicit confirmation can override (logged)

---

### Epic 7 — Edge Agents (Relay Agent)

**Goal:** Make Cortex distributed — run agents on every server and node.

**Deliverables:**

- **Relay Agent runtime**: lightweight, standalone Go binary (`apps/relay/` or `agents/relay/`)
- Registration system: agents self-register with Cortex Core via gRPC
- Capability declaration: each agent advertises what it can do
- Task execution pipeline: receive task → execute → stream results back
- Heartbeat / liveness: periodic health reports to Pulse
- Auto-reconnect: agents reconnect if Core goes down temporarily

**Flow:** `Cortex Core → Relay Agent on server-03 → run command → stream output → return result`

**Status:** Not implemented. This is the **recommended first real milestone** because it unlocks distributed execution and real system behavior.

**Architecture decision:** Use **bidirectional gRPC streaming** for agent ↔ Core communication. Agent initiates connection (NAT-friendly), Core pushes tasks down the stream, agent pushes results back up.

> **Note on Halo Edge:** Halo Edge is the Rust-based edge runtime embedded in the Halo desktop app. Since Halo lives in a **separate repository**, Halo Edge is NOT built in this monorepo. However, the protocol/contract that Halo Edge uses to communicate with Cortex Core is **identical** to the Relay Agent protocol — define the proto once, both agents implement it. The proto definitions for agent ↔ Core communication should live in this monorepo (e.g., `proto/agent/v1/`) and be published or vendored into the Halo repo.

---

### Epic 8 — Halo Desktop (Tauri App) — SEPARATE REPOSITORY

**Goal:** Build the primary user interface and embed the Halo Edge runtime.

**Stack:** React + TypeScript (UI), Tauri, Rust (Halo Edge runtime)

**Deliverables:**

- Working desktop app: chat UI, dashboards, logs, workspace UI
- Embedded Halo Edge agent: Rust command bridge (implements the same agent protocol as Relay)
- UI → Cortex Core integration via the public HTTP/WebSocket API
- Local-first: works offline with Halo Edge, syncs when Core is reachable

**Repository:** This lives in a **separate repository** (e.g., `halo` or `cortex-halo`). This monorepo (`cortex/`) is responsible for:

1. Defining stable proto/API contracts that Halo consumes
2. Publishing those contracts (proto files, OpenAPI specs) so the Halo repo can generate clients
3. Ensuring backward compatibility — API changes must not break Halo

**What this monorepo must provide for Halo:**

- Proto definitions for agent ↔ Core communication (shared with Relay Agent)
- OpenAPI spec or proto for the HTTP API Gateway (`/v1/chat`, `/v1/sessions`, etc.)
- WebSocket contract for streaming (chat responses, live logs, agent events)
- Authentication flow documentation (Sovereign token exchange)

**Status:** Not implemented. Depends on Epic 7 (edge agent protocol), Epic 4 (Sovereign auth), and a stable API surface.

---

### Epic 9 — Memory System (Echo / Vault / Recall)

**Goal:** Give Cortex long-term intelligence, context-aware reasoning, and per-user knowledge isolation.

**This epic is split into three sub-milestones:**

#### 9A — Echo Hardening (Session Memory)

**Goal:** Make the existing session memory production-ready.

**Deliverables:**

- Add `user_id` and `workspace_id` scoping to all session queries
- Session TTL / expiry (auto-archive old sessions)
- Efficient pagination for long sessions
- Event schema versioning (so storage format can evolve)

**Status:** Partially exists. Current `services/memory/` stores session events but has no user/workspace awareness.

#### 9B — Vault (Long-Term Knowledge Store)

**Goal:** Persistent, searchable knowledge that survives across sessions.

**Deliverables:**

- New service: `services/vault/` (or extend `services/memory/` with a separate domain)
- Vector embedding pipeline: text → embeddings (via Inference service or dedicated embedding model)
- Storage: vector DB adapter (start with SQLite + cosine similarity, graduate to Qdrant/pgvector)
- CRUD for knowledge entries: `{id, user_id, workspace_id, content, embedding, source, tags, created_at}`
- Semantic search: `Search(query, user_id, workspace_id, top_k) → []KnowledgeEntry`
- Ingestion hooks: after plan execution, extract learnable facts and store them

**Status:** Not implemented.

#### 9C — Recall (Context Assembly)

**Goal:** Intelligent retrieval layer that feeds Prism/Axiom the right context.

**Deliverables:**

- Stateless service or library (can be a module inside Brain, not a separate service)
- Assembles context by querying Echo (recent session) + Vault (relevant knowledge)
- Token budget management: fits assembled context within the model's context window
- Relevance ranking: combines recency, semantic similarity, and source reliability
- Configurable retrieval strategy: per-workspace or per-user preferences

**Status:** Not implemented. Currently Brain has no retrieval-augmented context at all.

---

### Epic 10 — Observability (Pulse + Overwatch)

**Goal:** Understand system health and surface actionable insights.

**Deliverables:**

- **Pulse:** Standardize metrics across all services (not just Brain). Every service exposes `/metrics`. Agent heartbeats feed into Pulse.
- **Overwatch:** Analysis engine that watches Pulse data, detects anomalies, and surfaces recommendations via Nerva events.

**Status:** Prometheus scraping exists for Brain only. No Overwatch.

**Improvements needed:**

- Every service should expose Prometheus metrics (request count, latency histograms, error rates)
- Structured logging format should be standardized (`pkg/log/` conventions)
- Agent telemetry (Pulse heartbeats) should include system resource metrics

---

### Epic 11 — Internet Layer (Beacon)

**Goal:** Allow Cortex to interact with the outside world.

**Deliverables:**

- Web search tool (pluggable: SearXNG, Google, Brave Search)
- Generic HTTP call tool (fetch arbitrary URLs with safety checks via Sentinel)
- API integration framework: define external APIs as Arsenal tools
- Rate limiting and caching for external calls

**Status:** Not implemented.

---

### Epic 12 — Workspace System

**Goal:** Enable per-user isolated environments.

**Deliverables:**

- Workspace model: `{workspace_id, user_id, name, description, root_path, created_at}`
- Workspace switching: user can switch active workspace, all context updates
- Scoped execution: Shell/Forge commands run within workspace `root_path`
- Scoped memory: Echo and Vault queries are filtered by workspace
- Scoped tools: workspaces can restrict which Arsenal tools are available

**Example:** `dev` workspace (code repos, read/write tools), `infra` workspace (cluster configs, Atlas/kubectl tools)

**Status:** Not implemented.

---

### Epic 13 — Chat System

**Goal:** Persistent, structured communication layer.

**Deliverables:**

- Chat model: `{chat_id, user_id, workspace_id, title, created_at, last_message_at}`
- Multi-chat: multiple concurrent conversations per user
- Persistent history: messages survive server restarts
- Workspace-linked context: chat inherits workspace scope
- Streaming responses: server-sent events or WebSocket for real-time token streaming

**Status:** Not implemented. Current system is single-session, ephemeral.

**Improvement:** The API Gateway needs WebSocket or SSE support. Chat responses from LLMs should stream token-by-token to the client, not batch the entire response.

---

### Epic 14 — Deployment & Infrastructure

**Goal:** Make Cortex runnable anywhere.

**Deliverables:**

- **Local mode:** single machine, docker-compose (**already exists**)
- **Server mode:** central Cortex Core on a server, Relay Agents on other machines
- **Cluster mode:** Kubernetes-native deployment (Helm chart or Kustomize)
- Environment setup automation (one-command bootstrap)
- Scaling strategy: which services can be horizontally scaled, which are singletons

**Status:** Docker Compose local mode works. Server and cluster modes do not exist.

---

### Epic 15 — Crucible (Sandboxed Code Execution)

**Goal:** Give Cortex the ability to write, run, test, and iterate on code in fully isolated environments.

**Deliverables:**

- New service: `services/crucible/` with gRPC API
- Docker-based sandbox lifecycle: Create → WriteFile → Exec → ReadFile → Destroy
- Pre-built runtime images: Python 3.12, Node 20, Go 1.24, Rust (add more as needed)
- Resource limits: CPU, memory, disk, timeout per sandbox
- Network isolation: disabled by default, opt-in requires Sentinel approval
- Streaming output: long-running sandbox commands stream stdout/stderr back to Brain
- **AI iteration loop:** Brain can autonomously create sandbox → write code → run → read errors → fix → re-run (up to configurable retry limit)
- Result extraction: Brain reads the final output and files from the sandbox, presents to user

**Flow:**

```
User: "Write a Go program that generates Fibonacci numbers"
  → Prism: code_execute
  → Axiom: plan [write code, create sandbox, execute, verify output]
  → Crucible: create sandbox (Go 1.24) → write main.go → go run main.go
  → Output: "0 1 1 2 3 5 8 13 21 34"
  → Axiom: output looks correct → present to user with code
```

**Safety:**

- Sandboxes run as unprivileged containers with no host mounts
- Network access off by default (Sentinel must approve)
- Resource limits enforced (OOM kill, CPU throttle, timeout kill)
- Maximum concurrent sandboxes per user (configurable, default 3)
- Auto-cleanup: idle sandboxes destroyed after configurable TTL (default 10 min)

**Integration with Forge:** Forge handles file operations on the _host workspace_. Crucible handles file operations _inside isolated containers_. They are complementary, not overlapping. For code that's safe and belongs to the user's project, use Forge. For untrusted, experimental, or throwaway code, use Crucible.

**Status:** Not implemented.

---

### Epic 16 — Compass (Project Planning & Management)

**Goal:** Give Cortex the ability to help users plan, scope, and track projects with structured milestones and tasks.

**Deliverables:**

- New service or module: `services/compass/` (or embed in Brain if lightweight enough)
- Project CRUD: create, list, update, archive projects
- Milestone & task management: create, reorder, update status, track dependencies
- Decision log: record architectural/design decisions with rationale and alternatives
- **AI-assisted planning:** User describes a goal → Axiom + Compass produce a structured project plan with milestones and tasks
- **Progress queries:** "What's my progress on X?" → Compass reports milestone/task completion percentages
- **Next step suggestions:** "What should I work on next?" → Compass evaluates task dependencies, priorities, and blockers
- Workspace-scoped: each project belongs to a workspace
- Storage: SQLite initially (consistent with other services)

**Flow:**

```
User: "I want to build a CLI tool for managing my homelab"
  → Prism: project_plan
  → Axiom: gather requirements via conversation, then call Compass
  → Compass: create project "Homelab CLI"
    → M1: Core CLI framework (cobra, config loading)
    → M2: Server inventory management
    → M3: SSH connection & command execution
    → M4: Service health monitoring
    → M5: Deployment automation
  → Each milestone auto-decomposed into tasks
  → User can later ask: "What's next on the homelab CLI?" → Compass responds with current status + next task
```

**Integration with other modules:**

- **Vault:** Project decisions and learned patterns are stored in Vault for long-term retrieval
- **Echo:** Planning conversations are stored in session memory
- **Crucible:** Tasks can trigger sandbox execution ("implement this task" → Crucible prototypes it)
- **Forge:** Tasks can reference workspace files ("this task modifies api/routes.go")

**Status:** Not implemented.

---

## 6 — Agent Registration Model

Every agent (Halo Edge or Relay Agent) **must** register with Cortex Core using this schema:

```yaml
identity:
  agent_id: string # unique agent ID (UUID, generated on first boot, persisted)
  agent_type: string # "halo_edge" | "relay"
  version: string # agent binary version (semver)

device:
  device_id: string # unique device fingerprint (hardware-derived or generated UUID)
  device_name: string # human-readable name (e.g., "isaac-desktop", "prod-node-03")
  hostname: string
  os: string # "linux" | "darwin" | "windows"
  architecture: string # "amd64" | "arm64"

network:
  local_ips: [string]
  public_ip: string # optional, resolved if possible
  interfaces: [string]

session:
  user_id: string # bound user
  roles: [string] # ["admin", "user", "restricted"]

capabilities: # what this agent CAN do (hardware/software)
  - terminal
  - filesystem
  - docker
  - kubernetes
  - network
  - gpu # has GPU available for local inference
  - local_models # can run local LLM models

permissions: # what this agent is ALLOWED to do (policy, set by Aegis)
  allow: [string]
  deny: [string]
  approval_required: [string]

tools: # structured tool definitions exposed by this agent
  - name: string
    description: string
    parameters: object # JSON Schema
    returns: object # JSON Schema
    safety_level: string # "safe" | "moderate" | "dangerous"

telemetry:
  status: string # "online" | "offline" | "degraded"
  last_seen: timestamp
  system_resources: # reported on heartbeat
    cpu_percent: float
    memory_percent: float
    disk_percent: float
    gpu_percent: float # if applicable
```

---

## 7 — Intent System (Prism Contract)

### Supported Intents

Intents are grouped by category for clarity:

**Conversational (no execution):**

| Intent         | Description                                                     | Handler                                                                |
| -------------- | --------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `conversation` | Free-form chat — discuss ideas, brainstorm, talk about anything | Axiom generates response directly via Inference (no plan, no executor) |
| `ask_question` | User needs specific information to proceed with a task          | Brain (Axiom generates answer, may query Recall for context)           |

**File & Workspace:**

| Intent             | Description                   | Executor Target |
| ------------------ | ----------------------------- | --------------- |
| `read_file`        | Read file contents            | Forge           |
| `write_file`       | Write or modify a file        | Forge           |
| `list_directory`   | List directory contents       | Forge           |
| `search_workspace` | Search within workspace files | Forge           |

**Execution:**

| Intent           | Description                        | Executor Target                   |
| ---------------- | ---------------------------------- | --------------------------------- |
| `run_command`    | Execute a shell command            | Shell (local) or Courier (remote) |
| `code_execute`   | Write and run code in a sandbox    | Crucible                          |
| `system_inspect` | Inspect local system state         | Shell                             |
| `agent_command`  | Send a command to a specific agent | Courier → target agent            |

**Kubernetes:**

| Intent            | Description                 | Executor Target |
| ----------------- | --------------------------- | --------------- |
| `cluster_inspect` | Inspect Kubernetes cluster  | Atlas           |
| `cluster_action`  | Perform a Kubernetes action | Atlas           |

**Planning & Memory:**

| Intent           | Description                           | Handler                         |
| ---------------- | ------------------------------------- | ------------------------------- |
| `project_plan`   | Create or update a project plan       | Compass                         |
| `project_status` | Check project/milestone/task progress | Compass                         |
| `plan_only`      | Plan an action but do not execute     | Brain (returns plan for review) |
| `memory_query`   | Search past context/knowledge         | Recall → Echo + Vault           |

**External:**

| Intent       | Description                     | Executor Target    |
| ------------ | ------------------------------- | ------------------ |
| `web_search` | Search the internet             | Beacon             |
| `diagnose`   | Diagnose a problem (multi-step) | Multiple executors |

### How Prism Distinguishes Conversation vs. Action

This is critical. Prism must correctly identify when the user just wants to **talk** vs. when they want Cortex to **do something**:

| User says                                | Intent                                            | Why                                  |
| ---------------------------------------- | ------------------------------------------------- | ------------------------------------ |
| "What do you think about microservices?" | `conversation`                                    | Opinion/discussion, no action needed |
| "Explain how gRPC streaming works"       | `conversation`                                    | Educational, no action needed        |
| "What's the weather like?"               | `conversation` (or `web_search` if Beacon exists) | Casual question                      |
| "What's wrong with my cluster?"          | `diagnose`                                        | Implies inspection and action        |
| "Read the main.go file"                  | `read_file`                                       | Explicit action                      |
| "Let's plan out my new project"          | `project_plan`                                    | Explicit planning intent             |
| "Write a Python script to sort a list"   | `code_execute`                                    | Wants working code, use Crucible     |

**Rule:** When in doubt, default to `conversation`. It's better to have a chat turn than to accidentally execute something.

### Request Contract

Every parsed request **must** include:

```json
{
  "intent": "<intent_type>",
  "execution_style": "minimal | exploratory | diagnostic | multi_step | conversational",
  "entities": {
    "target": "string | null",
    "parameters": {}
  },
  "confidence": 0.0,
  "requires_context": true
}
```

- When `execution_style` is `conversational`, the Brain bypasses planning and executor dispatch entirely.
- When `requires_context` is true, Recall is invoked before Axiom to assemble relevant memory.
- `confidence` below a configurable threshold (default 0.7) should trigger a clarification question before proceeding.

---

## 8 — Auth & Multi-User System (Sovereign)

### Requirements

- **Users**: identities + roles (admin, user, restricted)
- **Devices**: uniquely identified machines, bound to users
- **Sessions**: user + device binding with tokens, expiry, refresh
- **Per-user isolation**: separate chats, separate memory (Echo + Vault), separate workspaces, separate permissions

### Security Chain

```
Request arrives at API Gateway
       │
       ▼
  Sovereign: validate token → extract user_id, roles, device_id
       │
       ▼
  Aegis: evaluate permissions for this user + action
       │
       ▼
  Sentinel: check safety rules for this action
       │
       ▼
  Execute (Shell / Forge / Atlas / Courier / Crucible)

  OR (if conversational):
       │
       ▼
  Axiom: generate response directly via Inference (no executor)
```

### Workspace Model

Each workspace has:

- Unique ID + human-readable name
- Owner (user_id)
- File scope (root path)
- Memory scope (Echo + Vault queries filtered)
- Tool access scope (which Arsenal tools are enabled)

### Chat Model

Each user supports:

- Multiple concurrent chats, each optionally linked to a workspace
- Persistent, retrievable history
- Workspace-linked context (chat inherits workspace scope)

---

## 9 — Execution Rules

1. **Minimal steps only** — do the least work necessary.
2. **No unnecessary actions** — if it doesn't serve the goal, don't do it.
3. **No hallucinated capabilities** — if a subsystem doesn't exist yet, don't pretend it does.
4. **Every tool call goes through Aegis** (policy check) before execution.
5. **Every destructive action goes through Sentinel** (safety check) before execution.
6. **Every execution has a timeout** — no infinite hangs. Default 30s, configurable per tool.
7. **Every execution result is structured** — `{status, output, error, duration_ms}`, never raw strings.
8. **Every execution is idempotent where possible** — re-running the same plan step should be safe.

---

## 10 — Suggested Milestone Order

The epics above can be implemented in any order, but here is the recommended dependency-aware sequence:

| Priority | Epic                             | Rationale                                                                                             |
| -------- | -------------------------------- | ----------------------------------------------------------------------------------------------------- |
| 🥇 1     | **Epic 1 — Brain Core**          | Harden intent parsing + add conversational mode. This makes Cortex usable for daily chat immediately. |
| 2        | **Epic 13 — Chat System**        | Multi-chat, persistent history, streaming. Makes Cortex feel like a real AI companion.                |
| 3        | **Epic 15 — Crucible**           | Sandboxed code execution. Massive capability unlock — Cortex can write and test code.                 |
| 4        | **Epic 7 — Edge Agents (Relay)** | Unlocks distributed execution and real system behavior.                                               |
| 5        | **Epic 2 — Execution Layer**     | Add Atlas (k8s) and Courier (remote). Refactor Shell/Forge to shared interface.                       |
| 6        | **Epic 3 — Capability System**   | Build Arsenal (tool registry) so tools aren't hardcoded.                                              |
| 7        | **Epic 16 — Compass**            | Project planning. Once chat + code execution work, add structured planning.                           |
| 8        | **Epic 4 — Sovereign**           | Auth, users, sessions, device binding. Prerequisite for all multi-user features.                      |
| 9        | **Epic 6 — Sentinel**            | Safety layer — must exist before any untrusted execution.                                             |
| 10       | **Epic 5 — Aegis**               | Upgrade policy from basic gate to full RBAC-aware approval engine.                                    |
| 11       | **Epic 9A — Echo Hardening**     | Add user/workspace scoping to existing session memory.                                                |
| 12       | **Epic 9B — Vault**              | Long-term knowledge store with vector search.                                                         |
| 13       | **Epic 9C — Recall**             | Context assembly — connects Echo + Vault to Brain.                                                    |
| 14       | **Epic 12 — Workspaces**         | Per-user isolated environments.                                                                       |
| 15       | **Epic 11 — Beacon**             | Internet access.                                                                                      |
| 16       | **Epic 10 — Observability**      | Overwatch recommendations on top of existing Prometheus.                                              |
| 17       | **Epic 8 — Halo Desktop**        | Separate repo — built against the stable API/proto contracts defined here.                            |
| 18       | **Epic 14 — Deployment**         | Server mode, cluster mode, scaling.                                                                   |

---

## 11 — Workflow for Each Milestone

When implementing any milestone, follow this exact workflow:

1. **Inspect the repo** — understand what exists. Read `capabilities.yaml` files, domain models, existing adapters, and proto definitions.
2. **Choose ONE milestone** — pick the next epic or a vertical slice of an epic.
3. **Plan** — write out:
   - What you will build (new services, new files, modified files)
   - What proto/interfaces you will define (proto-first)
   - What the test plan is (how to verify it works)
4. **Implement** — write the code. Follow hexagonal architecture:
   - `domain/` — types, entities, value objects (no dependencies)
   - `ports/` — interfaces (driven + driving)
   - `adapters/` — implementations of ports (DB, gRPC clients, etc.)
   - `application/` — use cases / business logic (depends on ports only)
   - `transport/` — HTTP/gRPC handlers (driving adapters)
   - `bootstrap/` — wiring / dependency injection
5. **Test** — run `make test`, verify the new behavior works end-to-end.
6. **Report** — summarize what was built, what works, what's next.

---

## 12 — Architecture Conventions

### Stack

- **Language:** Go (services, Relay Agent, CLI), Rust (Halo Edge runtime — separate repo), React+TypeScript (Halo UI — separate repo)
- **Go module path:** `github.com/isaacwallace123/cortex`
- **Monorepo:** `go.work` workspace
- **Service structure:** Hexagonal — `internal/{domain, ports, adapters, application, transport, bootstrap, log, metrics}`
- **Total source files:** ~98 Go files across 8 services + 1 CLI + 1 shared package

### Communication

- **Service ↔ Service:** gRPC (protobuf)
- **Agent ↔ Core:** Bidirectional gRPC streaming (planned)
- **Client ↔ API Gateway:** HTTP/JSON (WebSocket planned for streaming)
- **Proto tooling:** `buf` for protobuf generation
- **Existing protos:** `brain/v1`, `forge/v1`, `inference/v1`, `memory/v1`, `nerva/v1`, `policy/v1`, `shell/v1`
- **API versioning:** All endpoints prefixed with `/v1/`. Breaking changes get a new version.

### Storage

- **Default:** SQLite for local/dev persistence
- **Production path:** PostgreSQL + pgvector (for Vault)
- **Vector search:** Start embedded (SQLite cosine sim), graduate to Qdrant/Milvus

### Events

- **Internal events:** Nerva pub/sub
- **Event schema:** All events carry `{event_type, timestamp, correlation_id, source, payload}`

### Observability

- **Metrics:** Prometheus. Brain exposes custom metrics (`cortex_inputs_parsed_total`, `cortex_plans_created_total`, `cortex_steps_evaluated_total`, `cortex_step_exec_duration_seconds`). Other services should add their own.
- **Tracing:** OpenTelemetry (`pkg/observe`) — trace ID extraction and context propagation
- **Logging:** Structured JSON via `slog` (Go stdlib). Each service has a `internal/log/` package with subsystem-tagged loggers (e.g., `Prism()`, `Axiom()`, `Vector()`). Pattern: `slog.NewJSONHandler` with `subsystem` field.

### Infrastructure

- **Dev:** Docker + docker-compose
- **CI/CD:** (future) Gitea Actions or GitHub Actions
- **Build:** Makefile targets

### Error Handling

- Use gRPC status codes properly: `NotFound`, `InvalidArgument`, `PermissionDenied`, `Internal`, etc.
- Domain errors are typed: `ErrNotFound`, `ErrUnauthorized`, `ErrBlocked`, etc.
- Never return raw error strings to clients — always wrap in structured error responses.

### Existing API Endpoints (API Gateway :8000)

- `POST /v1/chat` — parse input + create plan → `{request_id, session_id, intent, entities, plan{plan_id, steps[]}}`
- `POST /v1/run` — execute plan steps → `{session_id, plan_id, results[{stdout, stderr, exit_code, duration_ms}]}`
- `GET /v1/session/{id}` — session event history
- `GET /v1/plans?session_id=X` — plan summaries
- `GET /v1/plans/{id}?session_id=X` — plan detail with steps
- `GET /healthz` — health check
- Auth via `X-Api-Key` header or `Authorization: Bearer` (disabled if `CORTEX_API_KEY` env var unset)

### Configuration

- All config via environment variables (12-factor)
- Each service documents its env vars in `capabilities.yaml`
- Sensible defaults for local dev (no config file required to start)

---

## 13 — Inter-Service Communication Patterns

Understanding when to use each communication pattern is critical:

| Pattern            | When to Use                                                  | Example                                                    |
| ------------------ | ------------------------------------------------------------ | ---------------------------------------------------------- |
| **gRPC unary**     | Synchronous request/response between services                | Brain → Inference: `Complete(prompt)`                      |
| **gRPC streaming** | Long-running or real-time bidirectional communication        | Agent ↔ Core: task streaming, heartbeats                   |
| **Nerva events**   | Fire-and-forget notifications, audit logging, system signals | `task.completed`, `agent.registered`, `approval.requested` |
| **HTTP/JSON**      | External client → API Gateway                                | CLI → API: `POST /v1/chat`                                 |
| **WebSocket**      | Real-time streaming to external clients                      | Halo → API: streaming chat responses, live logs            |

**Rule of thumb:** If you need a response, use gRPC. If you're broadcasting a fact, use Nerva. If you're talking to an external client, use HTTP/WebSocket via the API Gateway.

---

## 14 — Testing Strategy

Every epic should include tests at these levels:

| Level           | What                                                  | Where                                 | Tool                                            |
| --------------- | ----------------------------------------------------- | ------------------------------------- | ----------------------------------------------- |
| **Unit**        | Domain logic, pure functions                          | `*_test.go` next to source            | `go test`                                       |
| **Integration** | Port ↔ Adapter (e.g., SQLite adapter works correctly) | `tests/` directory per service        | `go test` with test containers or in-memory DBs |
| **Service**     | Full service behavior via transport (gRPC/HTTP)       | `tests/` directory per service        | `go test` with actual server startup            |
| **End-to-end**  | Full pipeline: CLI → API → Brain → Executor → result  | Top-level `tests/` or use `make chat` | Manual or scripted                              |

**Minimum requirement per milestone:** Every new service must have at least one integration test that proves the happy path works end-to-end.

---

## 15 — Architectural Recommendations

Based on constraints and patterns currently in the codebase, the following architectural fixes should be prioritized to harden the core:

1. **Pluggable Executor Framework (`pkg/executor`)**: `Vector` currently hardcodes the routing of `shell` and `filesystem` steps in a `switch` statement, and the `StepExecutor` interface is tightly coupled inside `brain/internal/ports`. Move `StepExecutor` to a shared `pkg/executor` library. Update `Vector` to use a dynamic registry where executors register themselves (via gRPC Service Discovery or explicit init). This allows adding executors like `atlas` (k8s) or `crucible` without ever touching `brain` source code.
2. **Nerva Backpressure & Durability**: The in-memory `Bus` drops events (`default:` case in the select channel) if a subscriber is slow. While this is acceptable for raw logs/telemetry, if `Nerva` is used for event-driven orchestration (e.g. triggering an agent workflow), dropping events silently is catastrophic. Nerva should support durable queues or at minimum, offset tracking in SQLite.
3. **Gateway-Level WebSockets**: `Inference` supports `CompleteStream` internally, but the `API Gateway` only provides synchronous HTTP POST endpoints. For a true real-time chat feel and live feedback on execution logs, the Gateway MUST expose WebSockets or Server-Sent Events (SSE). Without this, the UI (Halo desktop app) will feel sluggish and unresponsive.
4. **Context Synthesis vs Raw Events (Recall)**: `Axiom` currently receives an unfiltered dump of previous `Echo` session events to generate plans. As sessions grow, this will blow up context windows. Cortex needs `Recall` to implement intelligent RAG: storing vector embeddings of past plans, shell outputs, and chat messages to synthesize _only_ the relevant context before feeding the LLM.

---

## 16 — Ideas for "Cool & Powerful Features"

If Cortex is going to be the ultimate AI control plane for personal infrastructure, here are killer features to make it stand out:

1. **Autonomic Healing / Background Watcher**: Right now, Cortex is purely request-response (you ask, it acts). What if Cortex could be _proactive and event-driven_? You tell Cortex: _"Monitor my homelab Kubernetes cluster. If a pod crashes due to OOM, bump the memory limit by 10% and restart it."_ Cortex subscribes to the event via `Nerva`, wakes up when it fires, reasons about it (using `Axiom`), and executes the fix autonomously in the background, pinging your phone to say "I fixed it".
2. **Cross-Device Swarm Execution**: Using the headless `Relay Agent`, Cortex could orchestrate actions across your entire fleet simultaneously. You could say: _"Update packages on my laptop, my Raspberry Pi, and my remote VPS."_ Vector decomposes the plan and routes the Linux `shell` commands through `Courier` to hit all three edge devices in parallel.
3. **"Shadow Mode" (Dry Run Execution)**: A feature where Cortex simulates the execution of a risky shell command or k8s deployment inside a `Crucible` ephemeral sandbox first. If the sandbox validates the output and exit codes, Cortex automatically approves the real run. This gives you the speed of auto-execution with the safety of Aegis approval.
4. **Catalyst Workflow Graphs**: For repetitive tasks (e.g., _"Deploy my portfolio app"_), LLMs can be slow and non-deterministic. Cortex should notice frequent routines, compile them into a deterministic DAG (Directed Acyclic Graph) of operations (a "Skill"), and execute them _without_ invoking the LLM model again, saving tokens, time, and eliminating flakiness.
5. **Dynamic Personas / Masks**: Cortex shouldn't be a monolithic personality. You should be able to equip different "Masks" based on the task:
   - **Domain-Specific Expertise**: Switching to the "Financial Analyst" persona alters the system prompt to focus on market analysis and explicitly loads real-time stock tools into the toolbelt. Switching to "Senior Staff Engineer" loads strict code review standards and grants access to `Crucible` and GitHub.
   - **Intent-Driven or Explicit**: `Prism` could automatically detect a topic shift (e.g., `intent: discuss_politics`) and implicitly swap to a neutral, fact-checking persona, or you could manually select an avatar in the `Halo` UI.
   - **Scoped Memory**: Personas instruct `Recall` on how to query the `Vault`. When in the "Politics" persona, `Recall` retrieves previous debates and facts you've logged, ignoring your past coding questions.

---

## 17 — Final Summary

Cortex is designed to be:

- ✅ **Conversational** — Talk to it about anything. It's your AI companion, not just a tool runner.
- ✅ **A code laboratory** — Crucible sandboxes let it write, run, test, and iterate on code autonomously.
- ✅ **A project partner** — Compass helps plan, scope, and track projects with AI-assisted breakdown.
- ✅ **Multi-user** — Sovereign handles identity, sessions, device binding
- ✅ **Multi-device** — Relay Agent (Go, this repo) + Halo Edge (Rust, Halo repo)
- ✅ **Agent-based** — distributed task execution with capability declarations
- ✅ **Secure by design** — Sovereign (identity) → Aegis (authorization) → Sentinel (safety)
- ✅ **Extensible** — Arsenal (tools) + Catalyst (skills) + Genesis (auto-generation)
- ✅ **Memory-rich** — Echo (sessions) + Vault (knowledge) + Recall (context assembly)
- ✅ **Observable** — Pulse (metrics) + Overwatch (analysis) + structured tracing
- ✅ **Production-grade** — hexagonal architecture, proto-first contracts, structured errors, graceful shutdown

---

> **START:** Inspect the repo, identify the most impactful missing piece, propose a milestone, get approval, implement it.
