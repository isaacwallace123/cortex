// Package domain defines the core entities for Compass project planning.
package domain

import "time"

// ProjectStatus represents the lifecycle state of a project.
type ProjectStatus string

const (
	ProjectPlanning  ProjectStatus = "planning"
	ProjectActive    ProjectStatus = "active"
	ProjectPaused    ProjectStatus = "paused"
	ProjectCompleted ProjectStatus = "completed"
)

// MilestoneStatus represents the lifecycle state of a milestone.
type MilestoneStatus string

const (
	MilestonePending    MilestoneStatus = "pending"
	MilestoneInProgress MilestoneStatus = "in_progress"
	MilestoneCompleted  MilestoneStatus = "completed"
)

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	TaskTodo       TaskStatus = "todo"
	TaskInProgress TaskStatus = "in_progress"
	TaskDone       TaskStatus = "done"
	TaskBlocked    TaskStatus = "blocked"
)

// Project is the top-level planning entity.
type Project struct {
	ProjectID   string
	Name        string
	Description string
	Status      ProjectStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Milestone is a phase within a project.
type Milestone struct {
	MilestoneID string
	ProjectID   string
	Name        string
	Description string
	Status      MilestoneStatus
	Order       int
}

// Task is an atomic unit of work within a milestone.
type Task struct {
	TaskID              string
	MilestoneID         string
	Name                string
	Description         string
	Status              TaskStatus
	AcceptanceCriteria  []string
	CreatedAt           time.Time
	CompletedAt         *time.Time
}

// Decision records an architectural or design decision made during a project.
type Decision struct {
	DecisionID   string
	ProjectID    string
	Title        string
	Rationale    string
	Alternatives []string
	DecidedAt    time.Time
}
