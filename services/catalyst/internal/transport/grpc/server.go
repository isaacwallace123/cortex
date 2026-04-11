package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalystv1 "github.com/isaacwallace123/cortex/services/catalyst/gen/catalyst/v1"
	"github.com/isaacwallace123/cortex/services/catalyst/internal/application"
	"github.com/isaacwallace123/cortex/services/catalyst/internal/domain"
)

// Server implements catalystv1.CatalystServiceServer.
type Server struct {
	catalystv1.UnimplementedCatalystServiceServer

	createUC  *application.CreateSkillUseCase
	getUC     *application.GetSkillUseCase
	listUC    *application.ListSkillsUseCase
	deleteUC  *application.DeleteSkillUseCase
	executeUC *application.ExecuteSkillUseCase
}

func NewServer(
	createUC *application.CreateSkillUseCase,
	getUC *application.GetSkillUseCase,
	listUC *application.ListSkillsUseCase,
	deleteUC *application.DeleteSkillUseCase,
	executeUC *application.ExecuteSkillUseCase,
) *Server {
	return &Server{
		createUC:  createUC,
		getUC:     getUC,
		listUC:    listUC,
		deleteUC:  deleteUC,
		executeUC: executeUC,
	}
}

func (s *Server) CreateSkill(ctx context.Context, req *catalystv1.CreateSkillRequest) (*catalystv1.CreateSkillResponse, error) {
	if req.GetSkill() == nil {
		return nil, status.Error(codes.InvalidArgument, "skill is required")
	}
	skill, err := s.createUC.Execute(ctx, protoToDomain(req.GetSkill()))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "create_skill: %v", err)
	}
	return &catalystv1.CreateSkillResponse{Skill: domainToProto(skill)}, nil
}

func (s *Server) GetSkill(ctx context.Context, req *catalystv1.GetSkillRequest) (*catalystv1.GetSkillResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	skill, err := s.getUC.Execute(ctx, req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get_skill: %v", err)
	}
	if skill == nil {
		return nil, status.Errorf(codes.NotFound, "skill %q not found", req.GetName())
	}
	return &catalystv1.GetSkillResponse{Skill: domainToProto(*skill)}, nil
}

func (s *Server) ListSkills(ctx context.Context, req *catalystv1.ListSkillsRequest) (*catalystv1.ListSkillsResponse, error) {
	skills, err := s.listUC.Execute(ctx, req.GetTag())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_skills: %v", err)
	}
	out := make([]*catalystv1.Skill, 0, len(skills))
	for _, sk := range skills {
		out = append(out, domainToProto(sk))
	}
	return &catalystv1.ListSkillsResponse{Skills: out}, nil
}

func (s *Server) DeleteSkill(ctx context.Context, req *catalystv1.DeleteSkillRequest) (*catalystv1.DeleteSkillResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	deleted, err := s.deleteUC.Execute(ctx, req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete_skill: %v", err)
	}
	return &catalystv1.DeleteSkillResponse{Deleted: deleted}, nil
}

func (s *Server) ExecuteSkill(ctx context.Context, req *catalystv1.ExecuteSkillRequest) (*catalystv1.ExecuteSkillResponse, error) {
	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	out, err := s.executeUC.Execute(ctx, application.ExecuteSkillInput{
		Name:      req.GetName(),
		SessionID: req.GetSessionId(),
		Params:    req.GetParams(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "execute_skill: %v", err)
	}
	results := make([]*catalystv1.SkillStepResult, 0, len(out.Results))
	for _, r := range out.Results {
		results = append(results, &catalystv1.SkillStepResult{
			StepIndex: int32(r.StepIndex),
			Stdout:    r.Stdout,
			Stderr:    r.Stderr,
			ExitCode:  r.ExitCode,
			Skipped:   r.Skipped,
		})
	}
	return &catalystv1.ExecuteSkillResponse{
		SessionId: out.SessionID,
		Results:   results,
	}, nil
}

// --- proto ↔ domain conversions ---

func protoToDomain(p *catalystv1.Skill) domain.Skill {
	steps := make([]domain.SkillStep, 0, len(p.GetSteps()))
	for _, st := range p.GetSteps() {
		steps = append(steps, domain.SkillStep{
			Index:           int(st.GetIndex()),
			Description:     st.GetDescription(),
			Executor:        st.GetExecutor(),
			CommandTemplate: st.GetCommandTemplate(),
			Agent:           st.GetAgent(),
		})
	}
	return domain.Skill{
		Name:        p.GetName(),
		Description: p.GetDescription(),
		Steps:       steps,
		Tags:        p.GetTags(),
		Version:     p.GetVersion(),
	}
}

func domainToProto(sk domain.Skill) *catalystv1.Skill {
	steps := make([]*catalystv1.SkillStep, 0, len(sk.Steps))
	for _, st := range sk.Steps {
		steps = append(steps, &catalystv1.SkillStep{
			Index:           int32(st.Index),
			Description:     st.Description,
			Executor:        st.Executor,
			CommandTemplate: st.CommandTemplate,
			Agent:           st.Agent,
		})
	}
	return &catalystv1.Skill{
		Name:        sk.Name,
		Description: sk.Description,
		Steps:       steps,
		Tags:        sk.Tags,
		Version:     sk.Version,
		CreatedAt:   sk.CreatedAt.Unix(),
	}
}
