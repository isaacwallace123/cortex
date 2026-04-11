package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	genesisv1 "github.com/isaacwallace123/cortex/services/genesis/gen/genesis/v1"
	"github.com/isaacwallace123/cortex/services/genesis/internal/application"
	"github.com/isaacwallace123/cortex/services/genesis/internal/domain"
)

// Server implements genesisv1.GenesisServiceServer.
type Server struct {
	genesisv1.UnimplementedGenesisServiceServer

	generateUC *application.GenerateToolUseCase
	listUC     *application.ListProposalsUseCase
}

func NewServer(generateUC *application.GenerateToolUseCase, listUC *application.ListProposalsUseCase) *Server {
	return &Server{generateUC: generateUC, listUC: listUC}
}

func (s *Server) GenerateTool(ctx context.Context, req *genesisv1.GenerateToolRequest) (*genesisv1.GenerateToolResponse, error) {
	if req.GetDescription() == "" {
		return nil, status.Error(codes.InvalidArgument, "description is required")
	}
	proposal, err := s.generateUC.Execute(ctx, req.GetDescription())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate_tool: %v", err)
	}
	return &genesisv1.GenerateToolResponse{Proposal: proposalToProto(proposal)}, nil
}

func (s *Server) ListProposals(ctx context.Context, _ *genesisv1.ListProposalsRequest) (*genesisv1.ListProposalsResponse, error) {
	proposals, err := s.listUC.Execute(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_proposals: %v", err)
	}
	out := make([]*genesisv1.Proposal, 0, len(proposals))
	for _, p := range proposals {
		out = append(out, proposalToProto(p))
	}
	return &genesisv1.ListProposalsResponse{Proposals: out}, nil
}

func proposalToProto(p domain.Proposal) *genesisv1.Proposal {
	pp := &genesisv1.Proposal{
		ProposalId:  p.ProposalID,
		Description: p.Description,
		Registered:  p.Registered,
		Error:       p.Error,
		CreatedAt:   p.CreatedAt.Unix(),
	}
	if p.Tool != nil {
		pp.Tool = &genesisv1.GeneratedTool{
			Name:        p.Tool.Name,
			Description: p.Tool.Description,
			Executor:    p.Tool.Executor,
			Example:     p.Tool.Example,
			SafetyLevel: p.Tool.SafetyLevel,
			Tags:        p.Tool.Tags,
			Version:     p.Tool.Version,
		}
	}
	return pp
}
