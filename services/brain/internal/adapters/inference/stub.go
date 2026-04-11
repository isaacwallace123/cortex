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

func (a *StubAdapter) CompleteStream(_ context.Context, prompt string, _ float32) (<-chan string, error) {
	text, err := a.Complete(context.Background(), prompt, 0)
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

func (a *StubAdapter) Complete(_ context.Context, prompt string, _ float32) (string, error) {
	// Prism (intent classification) prompt — detect by its unique header.
	if strings.Contains(prompt, "strict intent classifier") || strings.Contains(prompt, "OUTPUT FORMAT") {
		return stubParseResponse(prompt), nil
	}
	// Axiom (planning) prompt — detect by its unique header.
	if strings.Contains(prompt, "precise task planner") || strings.Contains(prompt, "STEP <n>:") {
		return stubPlanResponse(prompt), nil
	}
	// Conversation / infer fallback.
	return "I understand. How can I help you further?", nil
}

func stubParseResponse(prompt string) string {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "deploy"):
		return "INTENT: deploy application\nENTITIES: application, kubernetes\nMODE: execution"
	case strings.Contains(lower, "run") || strings.Contains(lower, "execute"):
		return "INTENT: execute script\nENTITIES: script\nMODE: execution"
	case strings.Contains(lower, "list") || strings.Contains(lower, "show"):
		return "INTENT: list resources\nENTITIES: resources\nMODE: execution"
	case strings.Contains(lower, "search") || strings.Contains(lower, "look up") || strings.Contains(lower, "find"):
		return "INTENT: search for information\nENTITIES: NONE\nMODE: tool_assisted"
	default:
		return "INTENT: general task\nENTITIES: NONE\nMODE: conversation"
	}
}

func stubPlanResponse(prompt string) string {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "deploy"):
		return `STEP 1: Build container image [executor:shell] [command:docker build -t app .]
STEP 2: Push image to registry [executor:shell] [command:docker push app]
STEP 3: Apply Kubernetes manifests [executor:atlas] [command:apply -f k8s/]
STEP 4: Verify deployment health [executor:atlas] [command:rollout status deployment/app]`
	case strings.Contains(lower, "execute script"):
		return `STEP 1: Execute script [executor:shell] [command:bash script.sh]`
	case strings.Contains(lower, "search") || strings.Contains(lower, "look up"):
		return `STEP 1: Search the web [executor:web] [command:latest information]
STEP 2: Summarise findings [executor:infer] [command:Summarise the search results clearly.]`
	default:
		return `STEP 1: Execute task [executor:shell] [command:echo "done"]`
	}
}
