// Package sandbox implements Docker-based isolated execution environments.
// Each sandbox is a Docker container with: no network access, CPU/memory limits,
// and a TTL-based auto-destroy. The Docker CLI is invoked via os/exec so the
// Crucible service itself requires only that the docker binary is on PATH and
// the Docker socket is mounted.
package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// runtimeImages maps runtime names to Docker image references.
var runtimeImages = map[string]string{
	"python3": "python:3.12-slim",
	"python":  "python:3.12-slim",
	"node":    "node:20-slim",
	"nodejs":  "node:20-slim",
	"go":      "golang:1.25-alpine",
	"golang":  "golang:1.25-alpine",
	"rust":    "rust:1.75-slim",
	"bash":    "alpine:3.19",
	"sh":      "alpine:3.19",
}

// DefaultCPULimit and DefaultMemoryLimit are applied to every sandbox container.
const (
	DefaultCPULimit    = "0.5"
	DefaultMemoryLimit = "256m"
	DefaultTimeoutSecs = 30
	DefaultSandboxTTL  = 600 // seconds (10 min)
)

// Sandbox is an active Docker container available for code execution.
type Sandbox struct {
	SandboxID   string
	Runtime     string
	ContainerID string // Docker container name (same as SandboxID prefix)
	CreatedAt   int64  // unix millis
	TimeoutSecs int
	expiresAt   time.Time
}

// Executor manages the lifecycle of Docker sandboxes.
type Executor struct {
	mu      sync.Mutex
	active  map[string]*Sandbox // sandbox_id â†’ sandbox
	stopGC  chan struct{}
}

// NewExecutor creates an Executor and starts the background TTL reaper.
func NewExecutor() *Executor {
	e := &Executor{
		active: make(map[string]*Sandbox),
		stopGC: make(chan struct{}),
	}
	go e.gcLoop()
	return e
}

// Close stops the GC loop.
func (e *Executor) Close() { close(e.stopGC) }

// Create starts a new Docker container and registers it.
func (e *Executor) Create(ctx context.Context, runtime string, timeoutSecs int) (*Sandbox, error) {
	image, ok := runtimeImages[strings.ToLower(runtime)]
	if !ok {
		return nil, fmt.Errorf("unsupported runtime %q; supported: %s", runtime, supportedRuntimes())
	}
	if timeoutSecs <= 0 {
		timeoutSecs = DefaultSandboxTTL
	}

	id := "cortex-sandbox-" + uuid.New().String()[:8]

	args := []string{
		"run", "-d",
		"--name", id,
		"--network", "none",
		"--cpus", DefaultCPULimit,
		"--memory", DefaultMemoryLimit,
		"--memory-swap", DefaultMemoryLimit, // disable swap
		"--read-only",
		"--tmpfs", "/tmp:rw,size=64m",
		"--tmpfs", "/app:rw,size=128m",
		"--workdir", "/app",
		image,
		"sleep", fmt.Sprintf("%d", timeoutSecs),
	}

	if out, err := dockerRun(ctx, args...); err != nil {
		return nil, fmt.Errorf("create sandbox %s: %w\n%s", id, err, out)
	}

	sb := &Sandbox{
		SandboxID:   id,
		Runtime:     runtime,
		ContainerID: id,
		CreatedAt:   time.Now().UnixMilli(),
		TimeoutSecs: timeoutSecs,
		expiresAt:   time.Now().Add(time.Duration(timeoutSecs) * time.Second),
	}

	e.mu.Lock()
	e.active[id] = sb
	e.mu.Unlock()

	return sb, nil
}

// Exec runs a shell command inside the sandbox and captures output.
func (e *Executor) Exec(ctx context.Context, sandboxID, command, stdinInput string, timeoutSecs int) (stdout, stderr string, exitCode int, durationMs int64, err error) {
	sb := e.get(sandboxID)
	if sb == nil {
		return "", "", -1, 0, fmt.Errorf("sandbox %s not found or expired", sandboxID)
	}
	if timeoutSecs <= 0 {
		timeoutSecs = DefaultTimeoutSecs
	}

	ctx2, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	args := []string{"exec"}
	if stdinInput != "" {
		args = append(args, "-i")
	}
	args = append(args, sb.ContainerID, "sh", "-c", command)

	start := time.Now()
	cmd := exec.CommandContext(ctx2, "docker", args...)
	if stdinInput != "" {
		cmd.Stdin = strings.NewReader(stdinInput)
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	durationMs = time.Since(start).Milliseconds()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
			err = runErr
		}
	}
	return stdout, stderr, exitCode, durationMs, err
}

// WriteFile writes content to an absolute path inside the sandbox.
// Uses `docker exec -i ... sh -c "cat > /path"` to avoid temp files on the host.
func (e *Executor) WriteFile(ctx context.Context, sandboxID, path, content string) error {
	sb := e.get(sandboxID)
	if sb == nil {
		return fmt.Errorf("sandbox %s not found or expired", sandboxID)
	}
	// Ensure parent directory exists first.
	dir := path[:strings.LastIndex(path, "/")+1]
	if dir != "" && dir != "/" {
		if _, _, _, _, err := e.Exec(ctx, sandboxID, "mkdir -p "+dir, "", 10); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", sb.ContainerID, "sh", "-c", "cat > "+path)
	cmd.Stdin = strings.NewReader(content)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("write %s: %w\n%s", path, err, errBuf.String())
	}
	return nil
}

// ReadFile reads and returns the content of a file inside the sandbox.
func (e *Executor) ReadFile(ctx context.Context, sandboxID, path string) (string, error) {
	sb := e.get(sandboxID)
	if sb == nil {
		return "", fmt.Errorf("sandbox %s not found or expired", sandboxID)
	}
	stdout, stderr, exitCode, _, err := e.Exec(ctx, sandboxID, "cat "+path, "", 10)
	if err != nil || exitCode != 0 {
		return "", fmt.Errorf("read %s (exit %d): %s", path, exitCode, stderr)
	}
	return stdout, nil
}

// Destroy stops and removes the Docker container.
func (e *Executor) Destroy(ctx context.Context, sandboxID string) (bool, error) {
	e.mu.Lock()
	sb, ok := e.active[sandboxID]
	if ok {
		delete(e.active, sandboxID)
	}
	e.mu.Unlock()

	if !ok {
		return false, nil
	}
	out, err := dockerRun(ctx, "rm", "-f", sb.ContainerID)
	if err != nil {
		return false, fmt.Errorf("destroy %s: %w\n%s", sandboxID, err, out)
	}
	return true, nil
}

// RunCode is a convenience wrapper: create â†’ exec â†’ destroy.
func (e *Executor) RunCode(ctx context.Context, runtime, command, stdinInput string, timeoutSecs int) (stdout, stderr string, exitCode int, durationMs int64, err error) {
	sb, err := e.Create(ctx, runtime, timeoutSecs+10) // sandbox lives slightly longer than exec timeout
	if err != nil {
		return "", "", -1, 0, fmt.Errorf("create sandbox: %w", err)
	}
	defer func() { _, _ = e.Destroy(context.Background(), sb.SandboxID) }()

	return e.Exec(ctx, sb.SandboxID, command, stdinInput, timeoutSecs)
}

// List returns a snapshot of all active sandboxes.
func (e *Executor) List() []*Sandbox {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]*Sandbox, 0, len(e.active))
	for _, sb := range e.active {
		out = append(out, sb)
	}
	return out
}

// get returns the active sandbox for id, or nil if it doesn't exist.
func (e *Executor) get(id string) *Sandbox {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.active[id]
}

func (e *Executor) gcLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopGC:
			return
		case <-ticker.C:
			e.reapExpired()
		}
	}
}

func (e *Executor) reapExpired() {
	e.mu.Lock()
	var expired []*Sandbox
	now := time.Now()
	for id, sb := range e.active {
		if now.After(sb.expiresAt) {
			expired = append(expired, sb)
			delete(e.active, id)
		}
	}
	e.mu.Unlock()

	for _, sb := range expired {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, _ = dockerRun(ctx, "rm", "-f", sb.ContainerID)
		cancel()
	}
}

func dockerRun(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	// Forward DOCKER_HOST if set, so the service can reach a remote daemon.
	cmd.Env = append(os.Environ())
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func supportedRuntimes() string {
	keys := make([]string, 0, len(runtimeImages))
	seen := map[string]bool{}
	for k, v := range runtimeImages {
		if !seen[v] {
			keys = append(keys, k)
			seen[v] = true
		}
	}
	return strings.Join(keys, ", ")
}

