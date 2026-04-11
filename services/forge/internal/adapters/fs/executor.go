// Package fs implements sandboxed filesystem operations for Forge.
// All paths are resolved under the configured root — no escapes allowed.
package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/isaacwallace123/cortex/services/forge/internal/domain"
)

// Executor performs filesystem operations sandboxed under Root.
type Executor struct {
	Root string
}

func NewExecutor(root string) (*Executor, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve forge root %q: %w", root, err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("create forge root %q: %w", abs, err)
	}
	return &Executor{Root: abs}, nil
}

func (e *Executor) Execute(_ context.Context, command string) (domain.OpResult, error) {
	start := time.Now()
	output, err := e.dispatch(command)
	ms := time.Since(start).Milliseconds()
	if err != nil {
		return domain.OpResult{Output: err.Error(), OK: false, DurationMs: ms}, nil
	}
	return domain.OpResult{Output: output, OK: true, DurationMs: ms}, nil
}

func (e *Executor) dispatch(command string) (string, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}
	op := strings.ToLower(parts[0])

	switch op {
	case "ls", "list":
		return e.list(argPath(parts, 1, "."))
	case "cat", "read":
		return e.read(argPath(parts, 1, ""))
	case "mkdir":
		return e.mkdir(argPath(parts, 1, ""))
	case "rm", "delete":
		return e.remove(argPath(parts, 1, ""))
	case "touch", "create":
		return e.touch(argPath(parts, 1, ""))
	case "write":
		if len(parts) < 3 {
			return "", fmt.Errorf("write requires: write <path> <content>")
		}
		content := strings.Join(parts[2:], " ")
		return e.write(parts[1], content)
	case "stat":
		return e.stat(argPath(parts, 1, ""))
	default:
		return "", fmt.Errorf("unknown operation %q — supported: ls, cat, mkdir, rm, touch, write, stat", op)
	}
}

// safe resolves path under Root and rejects any attempt to escape it.
// Absolute paths that already begin with Root are used as-is (the LLM knows the
// sandbox path). Relative paths are joined under Root.
func (e *Executor) safe(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("path must not be empty")
	}
	var abs string
	if filepath.IsAbs(p) {
		abs = filepath.Clean(p)
	} else {
		abs = filepath.Join(e.Root, p)
	}
	// Ensure the resolved path lives under Root.
	rootWithSlash := e.Root + string(filepath.Separator)
	if abs != e.Root && !strings.HasPrefix(abs, rootWithSlash) {
		return "", fmt.Errorf("path %q escapes sandbox root", p)
	}
	return abs, nil
}

func (e *Executor) list(rel string) (string, error) {
	abs, err := e.safe(rel)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return "", fmt.Errorf("list %q: %w", rel, err)
	}
	var sb strings.Builder
	for _, entry := range entries {
		info, _ := entry.Info()
		if entry.IsDir() {
			fmt.Fprintf(&sb, "d  %s/\n", entry.Name())
		} else {
			fmt.Fprintf(&sb, "f  %-8d  %s\n", info.Size(), entry.Name())
		}
	}
	if sb.Len() == 0 {
		return "(empty)", nil
	}
	return sb.String(), nil
}

func (e *Executor) read(rel string) (string, error) {
	abs, err := e.safe(rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("read %q: %w", rel, err)
	}
	return string(data), nil
}

func (e *Executor) mkdir(rel string) (string, error) {
	abs, err := e.safe(rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %q: %w", rel, err)
	}
	return fmt.Sprintf("created %s", rel), nil
}

func (e *Executor) remove(rel string) (string, error) {
	abs, err := e.safe(rel)
	if err != nil {
		return "", err
	}
	if err := os.Remove(abs); err != nil {
		return "", fmt.Errorf("remove %q: %w", rel, err)
	}
	return fmt.Sprintf("removed %s", rel), nil
}

func (e *Executor) touch(rel string) (string, error) {
	abs, err := e.safe(rel)
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(abs, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("touch %q: %w", rel, err)
	}
	f.Close()
	return fmt.Sprintf("created %s", rel), nil
}

func (e *Executor) write(rel, content string) (string, error) {
	abs, err := e.safe(rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", fmt.Errorf("create parent dirs for %q: %w", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write %q: %w", rel, err)
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(content), rel), nil
}

func (e *Executor) stat(rel string) (string, error) {
	abs, err := e.safe(rel)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", rel, err)
	}
	kind := "file"
	if info.IsDir() {
		kind = "directory"
	}
	return fmt.Sprintf("%s  %s  %d bytes  %s", info.Name(), kind, info.Size(), info.ModTime().Format(time.RFC3339)), nil
}

func argPath(parts []string, idx int, fallback string) string {
	if idx < len(parts) {
		return parts[idx]
	}
	return fallback
}
