// Package application implements Compass use cases.
package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/compass/internal/domain"
	"github.com/isaacwallace123/cortex/services/compass/internal/ports"
)

// UseCases bundles all Compass operations to keep wiring simple.
type UseCases struct {
	store ports.ProjectStore
}

func New(store ports.ProjectStore) *UseCases {
	return &UseCases{store: store}
}

// CreateProject creates a new project in planning status.
func (uc *UseCases) CreateProject(ctx context.Context, name, description string) (*domain.Project, error) {
	if name == "" {
		return nil, fmt.Errorf("project name is required")
	}
	p := &domain.Project{
		Name:        name,
		Description: description,
		Status:      domain.ProjectPlanning,
	}
	if err := uc.store.CreateProject(ctx, p); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return p, nil
}

func (uc *UseCases) GetProject(ctx context.Context, projectID string) (*domain.Project, []*domain.Milestone, []*domain.Task, error) {
	p, err := uc.store.GetProject(ctx, projectID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get project: %w", err)
	}
	if p == nil {
		return nil, nil, nil, nil
	}
	milestones, err := uc.store.ListMilestones(ctx, projectID)
	if err != nil {
		return p, nil, nil, err
	}
	// Fetch tasks for all milestones.
	var allTasks []*domain.Task
	for _, m := range milestones {
		tasks, err := uc.store.ListTasks(ctx, m.MilestoneID)
		if err != nil {
			continue
		}
		allTasks = append(allTasks, tasks...)
	}
	return p, milestones, allTasks, nil
}

func (uc *UseCases) ListProjects(ctx context.Context, statusFilter domain.ProjectStatus) ([]*domain.Project, error) {
	return uc.store.ListProjects(ctx, statusFilter)
}

func (uc *UseCases) UpdateProjectStatus(ctx context.Context, projectID string, status domain.ProjectStatus) (*domain.Project, error) {
	if err := uc.store.UpdateProjectStatus(ctx, projectID, status); err != nil {
		return nil, fmt.Errorf("update project status: %w", err)
	}
	return uc.store.GetProject(ctx, projectID)
}

// CreateMilestone adds a milestone to the given project.
func (uc *UseCases) CreateMilestone(ctx context.Context, projectID, name, description string, order int) (*domain.Milestone, error) {
	m := &domain.Milestone{
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		Status:      domain.MilestonePending,
		Order:       order,
	}
	if err := uc.store.CreateMilestone(ctx, m); err != nil {
		return nil, fmt.Errorf("create milestone: %w", err)
	}
	return m, nil
}

func (uc *UseCases) ListMilestones(ctx context.Context, projectID string) ([]*domain.Milestone, error) {
	return uc.store.ListMilestones(ctx, projectID)
}

func (uc *UseCases) UpdateMilestoneStatus(ctx context.Context, milestoneID string, status domain.MilestoneStatus) (*domain.Milestone, error) {
	if err := uc.store.UpdateMilestoneStatus(ctx, milestoneID, status); err != nil {
		return nil, fmt.Errorf("update milestone status: %w", err)
	}
	milestones, err := uc.store.ListMilestones(ctx, "")
	if err != nil {
		return nil, err
	}
	for _, m := range milestones {
		if m.MilestoneID == milestoneID {
			return m, nil
		}
	}
	return &domain.Milestone{MilestoneID: milestoneID, Status: status}, nil
}

// CreateTask adds a task to the given milestone.
func (uc *UseCases) CreateTask(ctx context.Context, milestoneID, name, description string, criteria []string) (*domain.Task, error) {
	t := &domain.Task{
		MilestoneID:        milestoneID,
		Name:               name,
		Description:        description,
		Status:             domain.TaskTodo,
		AcceptanceCriteria: criteria,
	}
	if err := uc.store.CreateTask(ctx, t); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return t, nil
}

func (uc *UseCases) ListTasks(ctx context.Context, milestoneID string) ([]*domain.Task, error) {
	return uc.store.ListTasks(ctx, milestoneID)
}

func (uc *UseCases) UpdateTaskStatus(ctx context.Context, taskID string, status domain.TaskStatus) (*domain.Task, error) {
	if err := uc.store.UpdateTaskStatus(ctx, taskID, status); err != nil {
		return nil, fmt.Errorf("update task status: %w", err)
	}
	return &domain.Task{TaskID: taskID, Status: status}, nil
}

// AddDecision records an architectural decision log entry for the given project.
func (uc *UseCases) AddDecision(ctx context.Context, projectID, title, rationale string, alternatives []string) (*domain.Decision, error) {
	d := &domain.Decision{
		ProjectID:    projectID,
		Title:        title,
		Rationale:    rationale,
		Alternatives: alternatives,
	}
	if err := uc.store.AddDecision(ctx, d); err != nil {
		return nil, fmt.Errorf("add decision: %w", err)
	}
	return d, nil
}

func (uc *UseCases) ListDecisions(ctx context.Context, projectID string) ([]*domain.Decision, error) {
	return uc.store.ListDecisions(ctx, projectID)
}

