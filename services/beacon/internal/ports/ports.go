package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/beacon/internal/domain"
)

// Searcher performs a web search and returns ranked results.
type Searcher interface {
	Search(ctx context.Context, query string, maxResults int) ([]domain.SearchResult, string, error)
}

// Fetcher retrieves the text content of a URL.
type Fetcher interface {
	Fetch(ctx context.Context, url string, timeoutSeconds int) (domain.FetchResult, error)
}
