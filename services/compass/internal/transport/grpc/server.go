package grpc

import (
	"context"
	"log/slog"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	compassv1 "github.com/isaacwallace123/cortex/services/compass/gen/compass/v1"
	"github.com/isaacwallace123/cortex/services/compass/internal/application"
	"github.com/isaacwallace123/cortex/services/compass/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/compass/internal/log"
)

// Server implements the gRPC CompassService. It is a thin transport layer that
// delegates all business logic to the application use cases.
type Server struct {
	compassv1.UnimplementedCompassServiceServer
	uc  *application.UseCases
	log *slog.Logger
}

func NewServer(uc *application.UseCases) *Server {
	return &Server{uc: uc, log: cortexlog.Compass()}
}

func (s *Server) CreateProject(ctx context.Context, req *compassv1.CreateProjectRequest) (*compassv1.CreateProjectResponse, error) {
	p, err := s.uc.CreateProject(ctx, req.GetName(), req.GetDescription())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create_project: %v", err)
	}
	s.log.Info("[Compass] project created", slog.String("id", p.ProjectID), slog.String("name", p.Name))
	return &compassv1.CreateProjectResponse{Project: domainProjectToProto(p)}, nil
}

func (s *Server) GetProject(ctx context.Context, req *compassv1.GetProjectRequest) (*compassv1.GetProjectResponse, error) {
	p, milestones, tasks, err := s.uc.GetProject(ctx, req.GetProjectId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get_project: %v", err)
	}
	if p == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetProjectId())
	}
	resp := &compassv1.GetProjectResponse{Project: domainProjectToProto(p)}
	for _, m := range milestones {
		resp.Milestones = append(resp.Milestones, domainMilestoneToProto(m))
	}
	for _, t := range tasks {
		resp.Tasks = append(resp.Tasks, domainTaskToProto(t))
	}
	return resp, nil
}

func (s *Server) ListProjects(ctx context.Context, req *compassv1.ListProjectsRequest) (*compassv1.ListProjectsResponse, error) {
	filter := protoProjectStatusToDomain(req.GetStatusFilter())
	projects, err := s.uc.ListProjects(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_projects: %v", err)
	}
	resp := &compassv1.ListProjectsResponse{}
	for _, p := range projects {
		resp.Projects = append(resp.Projects, domainProjectToProto(p))
	}
	return resp, nil
}

func (s *Server) UpdateProjectStatus(ctx context.Context, req *compassv1.UpdateProjectStatusRequest) (*compassv1.UpdateProjectStatusResponse, error) {
	p, err := s.uc.UpdateProjectStatus(ctx, req.GetProjectId(), protoProjectStatusToDomain(req.GetStatus()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update_project_status: %v", err)
	}
	return &compassv1.UpdateProjectStatusResponse{Project: domainProjectToProto(p)}, nil
}

func (s *Server) CreateMilestone(ctx context.Context, req *compassv1.CreateMilestoneRequest) (*compassv1.CreateMilestoneResponse, error) {
	m, err := s.uc.CreateMilestone(ctx, req.GetProjectId(), req.GetName(), req.GetDescription(), int(req.GetOrder()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create_milestone: %v", err)
	}
	return &compassv1.CreateMilestoneResponse{Milestone: domainMilestoneToProto(m)}, nil
}

func (s *Server) ListMilestones(ctx context.Context, req *compassv1.ListMilestonesRequest) (*compassv1.ListMilestonesResponse, error) {
	milestones, err := s.uc.ListMilestones(ctx, req.GetProjectId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_milestones: %v", err)
	}
	resp := &compassv1.ListMilestonesResponse{}
	for _, m := range milestones {
		resp.Milestones = append(resp.Milestones, domainMilestoneToProto(m))
	}
	return resp, nil
}

func (s *Server) UpdateMilestoneStatus(ctx context.Context, req *compassv1.UpdateMilestoneStatusRequest) (*compassv1.UpdateMilestoneStatusResponse, error) {
	m, err := s.uc.UpdateMilestoneStatus(ctx, req.GetMilestoneId(), protoMilestoneStatusToDomain(req.GetStatus()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update_milestone_status: %v", err)
	}
	return &compassv1.UpdateMilestoneStatusResponse{Milestone: domainMilestoneToProto(m)}, nil
}

func (s *Server) CreateTask(ctx context.Context, req *compassv1.CreateTaskRequest) (*compassv1.CreateTaskResponse, error) {
	t, err := s.uc.CreateTask(ctx, req.GetMilestoneId(), req.GetName(), req.GetDescription(), req.GetAcceptanceCriteria())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create_task: %v", err)
	}
	return &compassv1.CreateTaskResponse{Task: domainTaskToProto(t)}, nil
}

func (s *Server) ListTasks(ctx context.Context, req *compassv1.ListTasksRequest) (*compassv1.ListTasksResponse, error) {
	tasks, err := s.uc.ListTasks(ctx, req.GetMilestoneId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_tasks: %v", err)
	}
	resp := &compassv1.ListTasksResponse{}
	for _, t := range tasks {
		resp.Tasks = append(resp.Tasks, domainTaskToProto(t))
	}
	return resp, nil
}

func (s *Server) UpdateTaskStatus(ctx context.Context, req *compassv1.UpdateTaskStatusRequest) (*compassv1.UpdateTaskStatusResponse, error) {
	t, err := s.uc.UpdateTaskStatus(ctx, req.GetTaskId(), protoTaskStatusToDomain(req.GetStatus()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update_task_status: %v", err)
	}
	return &compassv1.UpdateTaskStatusResponse{Task: domainTaskToProto(t)}, nil
}

func (s *Server) AddDecision(ctx context.Context, req *compassv1.AddDecisionRequest) (*compassv1.AddDecisionResponse, error) {
	d, err := s.uc.AddDecision(ctx, req.GetProjectId(), req.GetTitle(), req.GetRationale(), req.GetAlternatives())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "add_decision: %v", err)
	}
	return &compassv1.AddDecisionResponse{Decision: domainDecisionToProto(d)}, nil
}

func (s *Server) ListDecisions(ctx context.Context, req *compassv1.ListDecisionsRequest) (*compassv1.ListDecisionsResponse, error) {
	decisions, err := s.uc.ListDecisions(ctx, req.GetProjectId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_decisions: %v", err)
	}
	resp := &compassv1.ListDecisionsResponse{}
	for _, d := range decisions {
		resp.Decisions = append(resp.Decisions, domainDecisionToProto(d))
	}
	return resp, nil
}

// domainProjectToProto converts a domain Project to its proto representation.
func domainProjectToProto(p *domain.Project) *compassv1.Project {
	if p == nil {
		return nil
	}
	return &compassv1.Project{
		ProjectId:   p.ProjectID,
		Name:        p.Name,
		Description: p.Description,
		Status:      domainProjectStatusToProto(p.Status),
		CreatedAt:   p.CreatedAt.UnixMilli(),
		UpdatedAt:   p.UpdatedAt.UnixMilli(),
	}
}

func domainMilestoneToProto(m *domain.Milestone) *compassv1.Milestone {
	if m == nil {
		return nil
	}
	return &compassv1.Milestone{
		MilestoneId: m.MilestoneID,
		ProjectId:   m.ProjectID,
		Name:        m.Name,
		Description: m.Description,
		Status:      domainMilestoneStatusToProto(m.Status),
		Order:       int32(m.Order),
	}
}

func domainTaskToProto(t *domain.Task) *compassv1.Task {
	if t == nil {
		return nil
	}
	pt := &compassv1.Task{
		TaskId:             t.TaskID,
		MilestoneId:        t.MilestoneID,
		Name:               t.Name,
		Description:        t.Description,
		Status:             domainTaskStatusToProto(t.Status),
		AcceptanceCriteria: t.AcceptanceCriteria,
		CreatedAt:          t.CreatedAt.UnixMilli(),
	}
	if t.CompletedAt != nil {
		pt.CompletedAt = t.CompletedAt.UnixMilli()
	}
	return pt
}

func domainDecisionToProto(d *domain.Decision) *compassv1.Decision {
	if d == nil {
		return nil
	}
	return &compassv1.Decision{
		DecisionId:   d.DecisionID,
		ProjectId:    d.ProjectID,
		Title:        d.Title,
		Rationale:    d.Rationale,
		Alternatives: d.Alternatives,
		DecidedAt:    d.DecidedAt.UnixMilli(),
	}
}

func domainProjectStatusToProto(s domain.ProjectStatus) compassv1.ProjectStatus {
	switch s {
	case domain.ProjectPlanning:
		return compassv1.ProjectStatus_PROJECT_STATUS_PLANNING
	case domain.ProjectActive:
		return compassv1.ProjectStatus_PROJECT_STATUS_ACTIVE
	case domain.ProjectPaused:
		return compassv1.ProjectStatus_PROJECT_STATUS_PAUSED
	case domain.ProjectCompleted:
		return compassv1.ProjectStatus_PROJECT_STATUS_COMPLETED
	default:
		return compassv1.ProjectStatus_PROJECT_STATUS_UNSPECIFIED
	}
}

func protoProjectStatusToDomain(s compassv1.ProjectStatus) domain.ProjectStatus {
	switch s {
	case compassv1.ProjectStatus_PROJECT_STATUS_PLANNING:
		return domain.ProjectPlanning
	case compassv1.ProjectStatus_PROJECT_STATUS_ACTIVE:
		return domain.ProjectActive
	case compassv1.ProjectStatus_PROJECT_STATUS_PAUSED:
		return domain.ProjectPaused
	case compassv1.ProjectStatus_PROJECT_STATUS_COMPLETED:
		return domain.ProjectCompleted
	default:
		return ""
	}
}

func domainMilestoneStatusToProto(s domain.MilestoneStatus) compassv1.MilestoneStatus {
	switch s {
	case domain.MilestonePending:
		return compassv1.MilestoneStatus_MILESTONE_STATUS_PENDING
	case domain.MilestoneInProgress:
		return compassv1.MilestoneStatus_MILESTONE_STATUS_IN_PROGRESS
	case domain.MilestoneCompleted:
		return compassv1.MilestoneStatus_MILESTONE_STATUS_COMPLETED
	default:
		return compassv1.MilestoneStatus_MILESTONE_STATUS_UNSPECIFIED
	}
}

func protoMilestoneStatusToDomain(s compassv1.MilestoneStatus) domain.MilestoneStatus {
	switch s {
	case compassv1.MilestoneStatus_MILESTONE_STATUS_PENDING:
		return domain.MilestonePending
	case compassv1.MilestoneStatus_MILESTONE_STATUS_IN_PROGRESS:
		return domain.MilestoneInProgress
	case compassv1.MilestoneStatus_MILESTONE_STATUS_COMPLETED:
		return domain.MilestoneCompleted
	default:
		return domain.MilestonePending
	}
}

func domainTaskStatusToProto(s domain.TaskStatus) compassv1.TaskStatus {
	switch s {
	case domain.TaskTodo:
		return compassv1.TaskStatus_TASK_STATUS_TODO
	case domain.TaskInProgress:
		return compassv1.TaskStatus_TASK_STATUS_IN_PROGRESS
	case domain.TaskDone:
		return compassv1.TaskStatus_TASK_STATUS_DONE
	case domain.TaskBlocked:
		return compassv1.TaskStatus_TASK_STATUS_BLOCKED
	default:
		return compassv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

func protoTaskStatusToDomain(s compassv1.TaskStatus) domain.TaskStatus {
	switch s {
	case compassv1.TaskStatus_TASK_STATUS_TODO:
		return domain.TaskTodo
	case compassv1.TaskStatus_TASK_STATUS_IN_PROGRESS:
		return domain.TaskInProgress
	case compassv1.TaskStatus_TASK_STATUS_DONE:
		return domain.TaskDone
	case compassv1.TaskStatus_TASK_STATUS_BLOCKED:
		return domain.TaskBlocked
	default:
		return domain.TaskTodo
	}
}

// splitLines splits a newline-separated string, skipping empty entries.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

