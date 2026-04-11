package inference

import (
	"context"
	"strings"
)

// StubAdapter is a deterministic inference adapter used for development and testing.
type StubAdapter struct{}

func NewStubAdapter() *StubAdapter {
	return &StubAdapter{}
}

func (a *StubAdapter) CompleteStream(ctx context.Context, prompt string) (<-chan string, error) {
	text, err := a.Complete(ctx, prompt)
	ch := make(chan string, 1)
	if err != nil {
		close(ch)
		return ch, err
	}
	go func() {
		defer close(ch)
		for _, word := range strings.Split(text, " ") {
			ch <- word + " "
		}
	}()
	return ch, nil
}

func (a *StubAdapter) Complete(_ context.Context, prompt string) (string, error) {
	if strings.Contains(prompt, "Extract the intent") {
		return stubParseResponse(prompt), nil
	}
	if strings.Contains(prompt, "execution planner") || strings.Contains(prompt, "Create a step-by-step plan") {
		return stubPlanResponse(prompt), nil
	}
	return "INTENT: unknown\nENTITIES: NONE", nil
}

func stubParseResponse(prompt string) string {
	// Simple keyword-based intent detection for the stub.
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "deploy"):
		return "INTENT: deploy application\nENTITIES: application, kubernetes"
	case strings.Contains(lower, "run") || strings.Contains(lower, "execute"):
		return "INTENT: execute script\nENTITIES: script"
	case strings.Contains(lower, "list") || strings.Contains(lower, "show"):
		return "INTENT: list resources\nENTITIES: resources"
	default:
		return "INTENT: general task\nENTITIES: NONE"
	}
}

func stubPlanResponse(prompt string) string {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "deploy"):
		return `STEP 1: Build container image [executor:code]
STEP 2: Push image to registry [executor:shell]
STEP 3: Apply Kubernetes manifests [executor:k8s]
STEP 4: Verify deployment health [executor:k8s]`
	case strings.Contains(lower, "execute script"):
		return `STEP 1: Validate script file exists [executor:filesystem]
STEP 2: Execute script [executor:shell]
STEP 3: Capture and return output [executor:shell]`
	default:
		return `STEP 1: Analyze request [executor:shell]
STEP 2: Execute task [executor:shell]`
	}
}
