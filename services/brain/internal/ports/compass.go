package ports

import "context"

// ProjectPlan is a structured plan returned from Compass for a new project.
type ProjectPlan struct {
	ProjectID string
	Name      string
	Milestones []PlannedMilestone
}

// PlannedMilestone is a milestone within a newly-created project plan.
type PlannedMilestone struct {
	MilestoneID string
	Name        string
	Description string
	Order       int
	Tasks       []PlannedTask
}

// PlannedTask is a task within a planned milestone.
type PlannedTask struct {
	TaskID      string
	Name        string
	Description string
}

// CompassPort lets Brain create and query project plans.
type CompassPort interface {
	// CreateProject stores a new project and returns its ID.
	CreateProject(ctx context.Context, name, description string) (projectID string, err error)
	// CreateMilestone adds a milestone to a project.
	CreateMilestone(ctx context.Context, projectID, name, description string, order int) (milestoneID string, err error)
	// CreateTask adds a task to a milestone.
	CreateTask(ctx context.Context, milestoneID, name, description string) (taskID string, err error)
	// GetProjectSummary returns a human-readable summary of a project.
	GetProjectSummary(ctx context.Context, projectID string) (string, error)
}
