package application

import (
	"fmt"
	"sync"
	"time"

	courierv1 "github.com/isaacwallace123/cortex/services/courier/gen/courier/v1"
	"github.com/isaacwallace123/cortex/services/courier/internal/domain"
)

// AgentConn holds the live stream and pending task channels for a connected agent.
type AgentConn struct {
	Agent  *domain.Agent
	stream courierv1.CourierService_ConnectServer

	mu      sync.Mutex
	pending map[string]chan *domain.TaskResult // task_id → result
}

func newAgentConn(agent *domain.Agent, stream courierv1.CourierService_ConnectServer) *AgentConn {
	return &AgentConn{
		Agent:   agent,
		stream:  stream,
		pending: make(map[string]chan *domain.TaskResult),
	}
}

// Send delivers a message to the agent over its stream. Thread-safe per gRPC-Go docs.
func (c *AgentConn) Send(msg *courierv1.CoreMessage) error {
	return c.stream.Send(msg)
}

// AddPending registers a result channel for an in-flight task.
func (c *AgentConn) AddPending(taskID string, ch chan *domain.TaskResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pending[taskID] = ch
}

// RemovePending cleans up after a task completes or times out.
func (c *AgentConn) RemovePending(taskID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pending, taskID)
}

// DeliverResult routes an incoming TaskResult to the waiting Dispatch caller.
func (c *AgentConn) DeliverResult(result *domain.TaskResult) bool {
	c.mu.Lock()
	ch, ok := c.pending[result.TaskID]
	c.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case ch <- result:
	default:
	}
	return true
}

// Registry is the in-memory store of all connected agents. Thread-safe.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]*AgentConn // keyed by agent_id
}

func NewRegistry() *Registry {
	return &Registry{agents: make(map[string]*AgentConn)}
}

func (r *Registry) Register(conn *AgentConn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[conn.Agent.AgentID] = conn
}

func (r *Registry) Unregister(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, agentID)
}

func (r *Registry) Get(agentID string) (*AgentConn, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	conn, ok := r.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent %q not connected", agentID)
	}
	return conn, nil
}

func (r *Registry) List() []*domain.Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agents := make([]*domain.Agent, 0, len(r.agents))
	for _, conn := range r.agents {
		agents = append(agents, conn.Agent)
	}
	return agents
}

// UpdateHeartbeat refreshes an agent's status and last-seen timestamp.
func (r *Registry) UpdateHeartbeat(agentID string, cpu, mem, disk float32) {
	r.mu.RLock()
	conn, ok := r.agents[agentID]
	r.mu.RUnlock()
	if !ok {
		return
	}
	conn.Agent.UpdateHeartbeat(cpu, mem, disk)
}

// TouchLastSeen bumps last-seen without a full heartbeat update.
func (r *Registry) TouchLastSeen(agentID string) {
	r.mu.RLock()
	conn, ok := r.agents[agentID]
	r.mu.RUnlock()
	if ok {
		conn.Agent.LastSeen = time.Now()
	}
}
