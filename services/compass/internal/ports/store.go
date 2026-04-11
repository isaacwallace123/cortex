// Package ports defines the persistence interfaces for Compass.
package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/compass/internal/domain"
)

// ProjectStore persists and retrieves Compass data.
type ProjectStore interface {
	// Projects
	CreateProject(ctx context.Context, p *domain.Project) error
	GetProject(ctx context.Context, projectID string) (*domain.Project, error)
	ListProjects(ctx context.Context, statusFilter domain.ProjectStatus) ([]*domain.Project, error)
	UpdateProjectStatus(ctx context.Context, projectID string, status domain.ProjectStatus) error

	// Milestones
	CreateMilestone(ctx context.Context, m *domain.Milestone) error
	ListMilestones(ctx context.Context, projectID string) ([]*domain.Milestone, error)
	UpdateMilestoneStatus(ctx context.Context, milestoneID string, status domain.MilestoneStatus) error

	// Tasks
	CreateTask(ctx context.Context, t *domain.Task) error
	ListTasks(ctx context.Context, milestoneID string) ([]*domain.Task, error)
	UpdateTaskStatus(ctx context.Context, taskID string, status domain.TaskStatus) error

	// Decisions
	AddDecision(ctx context.Context, d *domain.Decision) error
	ListDecisions(ctx context.Context, projectID string) ([]*domain.Decision, error)
}
