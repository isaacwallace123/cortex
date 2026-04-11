package compass

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/isaacwallace123/cortex/pkg/observe"
	compassv1 "github.com/isaacwallace123/cortex/services/compass/gen/compass/v1"
)

// GRPCAdapter implements ports.CompassPort via the Compass gRPC service.
type GRPCAdapter struct {
	client compassv1.CompassServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial compass at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: compassv1.NewCompassServiceClient(conn)}, nil
}

func (a *GRPCAdapter) CreateProject(ctx context.Context, name, description string) (string, error) {
	resp, err := a.client.CreateProject(ctx, &compassv1.CreateProjectRequest{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return "", fmt.Errorf("compass.CreateProject: %w", err)
	}
	return resp.GetProject().GetProjectId(), nil
}

func (a *GRPCAdapter) CreateMilestone(ctx context.Context, projectID, name, description string, order int) (string, error) {
	resp, err := a.client.CreateMilestone(ctx, &compassv1.CreateMilestoneRequest{
		ProjectId:   projectID,
		Name:        name,
		Description: description,
		Order:       int32(order),
	})
	if err != nil {
		return "", fmt.Errorf("compass.CreateMilestone: %w", err)
	}
	return resp.GetMilestone().GetMilestoneId(), nil
}

func (a *GRPCAdapter) CreateTask(ctx context.Context, milestoneID, name, description string) (string, error) {
	resp, err := a.client.CreateTask(ctx, &compassv1.CreateTaskRequest{
		MilestoneId: milestoneID,
		Name:        name,
		Description: description,
	})
	if err != nil {
		return "", fmt.Errorf("compass.CreateTask: %w", err)
	}
	return resp.GetTask().GetTaskId(), nil
}

func (a *GRPCAdapter) GetProjectSummary(ctx context.Context, projectID string) (string, error) {
	resp, err := a.client.GetProject(ctx, &compassv1.GetProjectRequest{ProjectId: projectID})
	if err != nil {
		return "", fmt.Errorf("compass.GetProject: %w", err)
	}
	p := resp.GetProject()
	if p == nil {
		return "", nil
	}
	summary := fmt.Sprintf("Project: %s (%s)\n", p.GetName(), p.GetStatus())
	for _, m := range resp.GetMilestones() {
		summary += fmt.Sprintf("  Milestone %d: %s [%s]\n", m.GetOrder(), m.GetName(), m.GetStatus())
	}
	return summary, nil
}
