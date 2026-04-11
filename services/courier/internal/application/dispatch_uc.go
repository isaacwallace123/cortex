package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	courierv1 "github.com/isaacwallace123/cortex/services/courier/gen/courier/v1"
	"github.com/isaacwallace123/cortex/services/courier/internal/domain"
)

const defaultTaskTimeout = 30 * time.Second

type DispatchInput struct {
	AgentID   string
	Command   string
	TimeoutMs int64
}

type DispatchOutput struct {
	Result *domain.TaskResult
}

// DispatchUseCase sends a command to a connected agent and waits for the result.
type DispatchUseCase struct {
	registry *Registry
}

func NewDispatchUseCase(registry *Registry) *DispatchUseCase {
	return &DispatchUseCase{registry: registry}
}

func (uc *DispatchUseCase) Execute(ctx context.Context, in DispatchInput) (*DispatchOutput, error) {
	conn, err := uc.registry.Get(in.AgentID)
	if err != nil {
		return nil, fmt.Errorf("dispatch: %w", err)
	}

	taskID := uuid.NewString()
	resultCh := make(chan *domain.TaskResult, 1)
	conn.AddPending(taskID, resultCh)
	defer conn.RemovePending(taskID)

	timeout := defaultTaskTimeout
	if in.TimeoutMs > 0 {
		timeout = time.Duration(in.TimeoutMs) * time.Millisecond
	}

	if err := conn.Send(&courierv1.CoreMessage{
		Payload: &courierv1.CoreMessage_TaskAssignment{
			TaskAssignment: &courierv1.TaskAssignment{
				TaskId:    taskID,
				Command:   in.Command,
				TimeoutMs: in.TimeoutMs,
			},
		},
	}); err != nil {
		return nil, fmt.Errorf("dispatch: send task assignment: %w", err)
	}

	select {
	case result := <-resultCh:
		return &DispatchOutput{Result: result}, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("dispatch: task %q timed out after %s", taskID, timeout)
	case <-ctx.Done():
		return nil, fmt.Errorf("dispatch: context canceled: %w", ctx.Err())
	}
}
