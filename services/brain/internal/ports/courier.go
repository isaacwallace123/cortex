package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
)

// RemoteAgent is a summary of a connected agent, used by Axiom to build planning context.
type RemoteAgent struct {
	AgentID      string
	DeviceName   string
	Hostname     string
	OS           string
	Arch         string
	Capabilities []string
	Status       string
}

// CourierPort is used by both Axiom (to query available agents for planning) and
// Vector (to dispatch steps to a specific connected agent).
type CourierPort interface {
	// ListAgents returns all currently connected agents.
	ListAgents(ctx context.Context) ([]RemoteAgent, error)

	// Dispatch sends a shell command to the named agent and returns the result.
	// agentRef may be a device_name or agent_id — Courier resolves either.
	Dispatch(ctx context.Context, sessionID, planID string, step domain.Step) (domain.ExecutionResult, error)
}
