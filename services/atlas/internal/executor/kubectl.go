// Package executor wraps the kubectl CLI for Atlas.
// Atlas uses DooD (Docker-outside-of-Docker) style access to kubectl:
// the binary must be on PATH and a valid kubeconfig must be reachable
// via ATLAS_KUBECONFIG or the default ~/.kube/config location.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// KubectlExecutor runs kubectl commands with a configured kubeconfig.
type KubectlExecutor struct {
	kubeconfig string // path to kubeconfig file, "" = use default
}

// Result holds structured output from a kubectl execution.
type Result struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
}

// ClusterInfo holds metadata about the current kubeconfig context.
type ClusterInfo struct {
	Context   string
	Server    string
	Reachable bool
}

// New creates a KubectlExecutor. Pass "" for kubeconfig to use the default.
func New(kubeconfig string) *KubectlExecutor {
	return &KubectlExecutor{kubeconfig: kubeconfig}
}

// Execute runs a kubectl subcommand string (e.g. "get pods -n default").
// The "kubectl" prefix is added automatically — do not include it in command.
func (e *KubectlExecutor) Execute(ctx context.Context, command string, timeoutSecs int) (Result, error) {
	if timeoutSecs <= 0 {
		timeoutSecs = int(defaultTimeout.Seconds())
	}

	// Split command into args safely.
	args := strings.Fields(command)
	if len(args) == 0 {
		return Result{}, fmt.Errorf("empty kubectl command")
	}

	ctx2, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx2, "kubectl", args...)
	cmd.Env = e.env()

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	start := time.Now()
	runErr := cmd.Run()
	durationMs := time.Since(start).Milliseconds()

	res := Result{
		Stdout:     outBuf.String(),
		Stderr:     errBuf.String(),
		DurationMs: durationMs,
	}

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
			return res, runErr
		}
	}
	return res, nil
}

// Info returns metadata about the current cluster context.
func (e *KubectlExecutor) Info(ctx context.Context) ClusterInfo {
	info := ClusterInfo{}

	// Get current context name.
	if out, err := e.run(ctx, "config", "current-context"); err == nil {
		info.Context = strings.TrimSpace(out)
	}

	// Get API server URL for the current context.
	if out, err := e.run(ctx, "config", "view",
		"--minify", "--output=jsonpath={.clusters[0].cluster.server}"); err == nil {
		info.Server = strings.TrimSpace(out)
	}

	// Lightweight reachability check.
	if _, err := e.run(ctx, "cluster-info", "--request-timeout=5s"); err == nil {
		info.Reachable = true
	}

	return info
}

func (e *KubectlExecutor) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = e.env()
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (e *KubectlExecutor) env() []string {
	env := os.Environ()
	if e.kubeconfig != "" {
		env = append(env, "KUBECONFIG="+e.kubeconfig)
	}
	return env
}
