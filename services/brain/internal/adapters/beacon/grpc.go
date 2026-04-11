package beacon

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
	beaconv1 "github.com/isaacwallace123/cortex/services/beacon/gen/beacon/v1"
)

// GRPCAdapter implements BeaconPort via the Beacon service.
type GRPCAdapter struct {
	client beaconv1.BeaconServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial beacon at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: beaconv1.NewBeaconServiceClient(conn)}, nil
}

func (a *GRPCAdapter) Search(ctx context.Context, query string, maxResults int) ([]ports.SearchResult, string, error) {
	resp, err := a.client.Search(ctx, &beaconv1.SearchRequest{
		Query:      query,
		MaxResults: int32(maxResults),
	})
	if err != nil {
		return nil, "", fmt.Errorf("beacon.Search: %w", err)
	}
	out := make([]ports.SearchResult, 0, len(resp.GetResults()))
	for _, r := range resp.GetResults() {
		out = append(out, ports.SearchResult{
			Title:   r.GetTitle(),
			URL:     r.GetUrl(),
			Snippet: r.GetSnippet(),
		})
	}
	return out, resp.GetEngine(), nil
}

func (a *GRPCAdapter) Fetch(ctx context.Context, url string, timeoutSeconds int) (ports.FetchResult, error) {
	resp, err := a.client.Fetch(ctx, &beaconv1.FetchRequest{
		Url:            url,
		TimeoutSeconds: int32(timeoutSeconds),
	})
	if err != nil {
		return ports.FetchResult{}, fmt.Errorf("beacon.Fetch: %w", err)
	}
	return ports.FetchResult{
		StatusCode:  int(resp.GetStatusCode()),
		ContentType: resp.GetContentType(),
		Body:        resp.GetBody(),
		Error:       resp.GetError(),
	}, nil
}
