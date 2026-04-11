package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	beaconv1 "github.com/isaacwallace123/cortex/services/beacon/gen/beacon/v1"
	"github.com/isaacwallace123/cortex/services/beacon/internal/application"
)

// Server implements the gRPC BeaconService.
type Server struct {
	beaconv1.UnimplementedBeaconServiceServer

	search *application.SearchUseCase
	fetch  *application.FetchUseCase
}

func NewServer(search *application.SearchUseCase, fetch *application.FetchUseCase) *Server {
	return &Server{search: search, fetch: fetch}
}

func (s *Server) Search(ctx context.Context, req *beaconv1.SearchRequest) (*beaconv1.SearchResponse, error) {
	out, err := s.search.Execute(ctx, application.SearchInput{
		Query:      req.GetQuery(),
		MaxResults: int(req.GetMaxResults()),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search: %v", err)
	}

	results := make([]*beaconv1.SearchResult, 0, len(out.Results))
	for _, r := range out.Results {
		results = append(results, &beaconv1.SearchResult{
			Title:   r.Title,
			Url:     r.URL,
			Snippet: r.Snippet,
		})
	}
	return &beaconv1.SearchResponse{Results: results, Engine: out.Engine}, nil
}

func (s *Server) Fetch(ctx context.Context, req *beaconv1.FetchRequest) (*beaconv1.FetchResponse, error) {
	out, err := s.fetch.Execute(ctx, application.FetchInput{
		URL:            req.GetUrl(),
		TimeoutSeconds: int(req.GetTimeoutSeconds()),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetch: %v", err)
	}

	return &beaconv1.FetchResponse{
		StatusCode:  int32(out.Result.StatusCode),
		ContentType: out.Result.ContentType,
		Body:        out.Result.Body,
		Error:       out.Result.Error,
	}, nil
}
