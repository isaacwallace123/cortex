package subprocess

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"

	"github.com/isaacwallace123/cortex/services/shell/internal/domain"
)

// Runner executes shell commands via os/exec.
type Runner struct{}

func NewRunner() *Runner { return &Runner{} }

func (r *Runner) Run(ctx context.Context, command string) (domain.RunResult, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	durationMs := time.Since(start).Milliseconds()

	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			// Context cancelled, timeout, or exec failure — not a command error.
			return domain.RunResult{}, err
		}
	}

	return domain.RunResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ExitCode:   exitCode,
		DurationMs: durationMs,
	}, nil
}
