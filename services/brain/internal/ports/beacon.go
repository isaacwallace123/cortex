package ports

import "context"

// SearchResult is a single web search result returned by Beacon.
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// FetchResult is the content retrieved from a URL by Beacon.
type FetchResult struct {
	StatusCode  int
	ContentType string
	Body        string
	Error       string
}

// BeaconPort is Brain's interface to the Beacon internet-access service.
type BeaconPort interface {
	// Search queries the web for the given query and returns ranked results.
	Search(ctx context.Context, query string, maxResults int) ([]SearchResult, string, error)

	// Fetch retrieves the text content of a URL.
	Fetch(ctx context.Context, url string, timeoutSeconds int) (FetchResult, error)
}
