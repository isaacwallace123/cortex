package compass

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	compassv1 "github.com/isaacwallace123/cortex/services/compass/gen/compass/v1"
)

// Client wraps the Compass gRPC client with typed methods the gateway uses.
type Client struct {
	inner compassv1.CompassServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial compass at %s: %w", addr, err)
	}
	return &Client{inner: compassv1.NewCompassServiceClient(conn)}, nil
}

type Project struct {
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type Milestone struct {
	MilestoneID string `json:"milestone_id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Order       int32  `json:"order"`
}

type Task struct {
	TaskID             string   `json:"task_id"`
	MilestoneID        string   `json:"milestone_id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Status             string   `json:"status"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	CreatedAt          int64    `json:"created_at"`
	CompletedAt        int64    `json:"completed_at,omitempty"`
}

type Decision struct {
	DecisionID   string   `json:"decision_id"`
	ProjectID    string   `json:"project_id"`
	Title        string   `json:"title"`
	Rationale    string   `json:"rationale"`
	Alternatives []string `json:"alternatives,omitempty"`
	DecidedAt    int64    `json:"decided_at"`
}

type ProjectDetail struct {
	Project    Project     `json:"project"`
	Milestones []Milestone `json:"milestones"`
	Tasks      []Task      `json:"tasks"`
}

func (c *Client) CreateProject(ctx context.Context, name, description string) (Project, error) {
	resp, err := c.inner.CreateProject(ctx, &compassv1.CreateProjectRequest{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return Project{}, err
	}
	return protoToProject(resp.GetProject()), nil
}

func (c *Client) GetProject(ctx context.Context, projectID string) (ProjectDetail, error) {
	resp, err := c.inner.GetProject(ctx, &compassv1.GetProjectRequest{ProjectId: projectID})
	if err != nil {
		return ProjectDetail{}, err
	}
	detail := ProjectDetail{Project: protoToProject(resp.GetProject())}
	for _, m := range resp.GetMilestones() {
		detail.Milestones = append(detail.Milestones, protoToMilestone(m))
	}
	for _, t := range resp.GetTasks() {
		detail.Tasks = append(detail.Tasks, protoToTask(t))
	}
	return detail, nil
}

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	resp, err := c.inner.ListProjects(ctx, &compassv1.ListProjectsRequest{})
	if err != nil {
		return nil, err
	}
	out := make([]Project, 0, len(resp.GetProjects()))
	for _, p := range resp.GetProjects() {
		out = append(out, protoToProject(p))
	}
	return out, nil
}

func (c *Client) UpdateProjectStatus(ctx context.Context, projectID, status string) (Project, error) {
	resp, err := c.inner.UpdateProjectStatus(ctx, &compassv1.UpdateProjectStatusRequest{
		ProjectId: projectID,
		Status:    stringToProjectStatus(status),
	})
	if err != nil {
		return Project{}, err
	}
	return protoToProject(resp.GetProject()), nil
}

func (c *Client) CreateMilestone(ctx context.Context, projectID, name, description string, order int32) (Milestone, error) {
	resp, err := c.inner.CreateMilestone(ctx, &compassv1.CreateMilestoneRequest{
		ProjectId:   projectID,
		Name:        name,
		Description: description,
		Order:       order,
	})
	if err != nil {
		return Milestone{}, err
	}
	return protoToMilestone(resp.GetMilestone()), nil
}

func (c *Client) ListMilestones(ctx context.Context, projectID string) ([]Milestone, error) {
	resp, err := c.inner.ListMilestones(ctx, &compassv1.ListMilestonesRequest{ProjectId: projectID})
	if err != nil {
		return nil, err
	}
	out := make([]Milestone, 0, len(resp.GetMilestones()))
	for _, m := range resp.GetMilestones() {
		out = append(out, protoToMilestone(m))
	}
	return out, nil
}

func (c *Client) CreateTask(ctx context.Context, milestoneID, name, description string, criteria []string) (Task, error) {
	resp, err := c.inner.CreateTask(ctx, &compassv1.CreateTaskRequest{
		MilestoneId:        milestoneID,
		Name:               name,
		Description:        description,
		AcceptanceCriteria: criteria,
	})
	if err != nil {
		return Task{}, err
	}
	return protoToTask(resp.GetTask()), nil
}

func (c *Client) ListTasks(ctx context.Context, milestoneID string) ([]Task, error) {
	resp, err := c.inner.ListTasks(ctx, &compassv1.ListTasksRequest{MilestoneId: milestoneID})
	if err != nil {
		return nil, err
	}
	out := make([]Task, 0, len(resp.GetTasks()))
	for _, t := range resp.GetTasks() {
		out = append(out, protoToTask(t))
	}
	return out, nil
}

func (c *Client) UpdateTaskStatus(ctx context.Context, taskID, status string) (Task, error) {
	resp, err := c.inner.UpdateTaskStatus(ctx, &compassv1.UpdateTaskStatusRequest{
		TaskId: taskID,
		Status: stringToTaskStatus(status),
	})
	if err != nil {
		return Task{}, err
	}
	return protoToTask(resp.GetTask()), nil
}

func (c *Client) AddDecision(ctx context.Context, projectID, title, rationale string, alternatives []string) (Decision, error) {
	resp, err := c.inner.AddDecision(ctx, &compassv1.AddDecisionRequest{
		ProjectId:    projectID,
		Title:        title,
		Rationale:    rationale,
		Alternatives: alternatives,
	})
	if err != nil {
		return Decision{}, err
	}
	return protoToDecision(resp.GetDecision()), nil
}

func (c *Client) ListDecisions(ctx context.Context, projectID string) ([]Decision, error) {
	resp, err := c.inner.ListDecisions(ctx, &compassv1.ListDecisionsRequest{ProjectId: projectID})
	if err != nil {
		return nil, err
	}
	out := make([]Decision, 0, len(resp.GetDecisions()))
	for _, d := range resp.GetDecisions() {
		out = append(out, protoToDecision(d))
	}
	return out, nil
}

func protoToProject(p *compassv1.Project) Project {
	if p == nil {
		return Project{}
	}
	return Project{
		ProjectID:   p.GetProjectId(),
		Name:        p.GetName(),
		Description: p.GetDescription(),
		Status:      p.GetStatus().String(),
		CreatedAt:   p.GetCreatedAt(),
		UpdatedAt:   p.GetUpdatedAt(),
	}
}

func protoToMilestone(m *compassv1.Milestone) Milestone {
	if m == nil {
		return Milestone{}
	}
	return Milestone{
		MilestoneID: m.GetMilestoneId(),
		ProjectID:   m.GetProjectId(),
		Name:        m.GetName(),
		Description: m.GetDescription(),
		Status:      m.GetStatus().String(),
		Order:       m.GetOrder(),
	}
}

func protoToTask(t *compassv1.Task) Task {
	if t == nil {
		return Task{}
	}
	return Task{
		TaskID:             t.GetTaskId(),
		MilestoneID:        t.GetMilestoneId(),
		Name:               t.GetName(),
		Description:        t.GetDescription(),
		Status:             t.GetStatus().String(),
		AcceptanceCriteria: t.GetAcceptanceCriteria(),
		CreatedAt:          t.GetCreatedAt(),
		CompletedAt:        t.GetCompletedAt(),
	}
}

func protoToDecision(d *compassv1.Decision) Decision {
	if d == nil {
		return Decision{}
	}
	return Decision{
		DecisionID:   d.GetDecisionId(),
		ProjectID:    d.GetProjectId(),
		Title:        d.GetTitle(),
		Rationale:    d.GetRationale(),
		Alternatives: d.GetAlternatives(),
		DecidedAt:    d.GetDecidedAt(),
	}
}

func stringToProjectStatus(s string) compassv1.ProjectStatus {
	switch s {
	case "planning":
		return compassv1.ProjectStatus_PROJECT_STATUS_PLANNING
	case "active":
		return compassv1.ProjectStatus_PROJECT_STATUS_ACTIVE
	case "paused":
		return compassv1.ProjectStatus_PROJECT_STATUS_PAUSED
	case "completed":
		return compassv1.ProjectStatus_PROJECT_STATUS_COMPLETED
	default:
		return compassv1.ProjectStatus_PROJECT_STATUS_UNSPECIFIED
	}
}

func stringToTaskStatus(s string) compassv1.TaskStatus {
	switch s {
	case "todo":
		return compassv1.TaskStatus_TASK_STATUS_TODO
	case "in_progress":
		return compassv1.TaskStatus_TASK_STATUS_IN_PROGRESS
	case "done":
		return compassv1.TaskStatus_TASK_STATUS_DONE
	case "blocked":
		return compassv1.TaskStatus_TASK_STATUS_BLOCKED
	default:
		return compassv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

