package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/isaacwallace123/cortex/services/inference/internal/application"
	inferencev1 "github.com/isaacwallace123/cortex/services/inference/gen/inference/v1"
)

// Server implements the gRPC InferenceService.
type Server struct {
	inferencev1.UnimplementedInferenceServiceServer

	complete       *application.CompleteUseCase
	completeStream *application.CompleteStreamUseCase
	listModels     *application.ListModelsUseCase
}

func NewServer(complete *application.CompleteUseCase, completeStream *application.CompleteStreamUseCase, listModels *application.ListModelsUseCase) *Server {
	return &Server{complete: complete, completeStream: completeStream, listModels: listModels}
}

func (s *Server) Complete(ctx context.Context, req *inferencev1.CompleteRequest) (*inferencev1.CompleteResponse, error) {
	out, err := s.complete.Execute(ctx, application.CompleteInput{
		Model:       req.GetModel(),
		Prompt:      req.GetPrompt(),
		Temperature: req.GetTemperature(),
		MaxTokens:   req.GetMaxTokens(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "complete failed: %v", err)
	}
	return &inferencev1.CompleteResponse{
		Text:         out.Response.Text,
		Model:        out.Response.Model,
		InputTokens:  out.Response.InputTokens,
		OutputTokens: out.Response.OutputTokens,
	}, nil
}

func (s *Server) CompleteStream(req *inferencev1.CompleteRequest, stream inferencev1.InferenceService_CompleteStreamServer) error {
	ch, err := s.completeStream.Execute(stream.Context(), application.CompleteInput{
		Model:       req.GetModel(),
		Prompt:      req.GetPrompt(),
		Temperature: req.GetTemperature(),
		MaxTokens:   req.GetMaxTokens(),
	})
	if err != nil {
		return status.Errorf(codes.Internal, "complete_stream failed: %v", err)
	}
	for chunk := range ch {
		if err := stream.Send(&inferencev1.CompleteChunk{
			Token:        chunk.Token,
			Done:         chunk.Done,
			Model:        chunk.Model,
			InputTokens:  chunk.InputTokens,
			OutputTokens: chunk.OutputTokens,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) ListModels(ctx context.Context, _ *inferencev1.ListModelsRequest) (*inferencev1.ListModelsResponse, error) {
	out, err := s.listModels.Execute(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_models failed: %v", err)
	}
	models := make([]*inferencev1.Model, 0, len(out.Models))
	for _, m := range out.Models {
		models = append(models, &inferencev1.Model{
			Name:     m.Name,
			Size:     m.Size,
			Modified: m.Modified,
		})
	}
	return &inferencev1.ListModelsResponse{Models: models}, nil
}
