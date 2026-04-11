// Package executor runs shell commands on behalf of task assignments from Core.
package executor

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"time"
)

type Result struct {
	Stdout     string
	Stderr     string
	ExitCode   int32
	DurationMs int64
}

// Run executes a shell command with the given timeout and returns the result.
func Run(ctx context.Context, command string, timeoutMs int64) Result {
	timeout := 30 * time.Second
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	durationMs := time.Since(start).Milliseconds()

	exitCode := int32(0)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = int32(exitErr.ExitCode())
		} else {
			exitCode = 1
		}
	}

	return Result{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   exitCode,
		DurationMs: durationMs,
	}
}
