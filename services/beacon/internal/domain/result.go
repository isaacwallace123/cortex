package domain

// SearchResult is a single web search result.
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// FetchResult is the content of a fetched URL.
type FetchResult struct {
	StatusCode  int
	ContentType string
	Body        string // extracted text, capped at 100 KB
	Error       string // non-empty on transport or parse failure
}
