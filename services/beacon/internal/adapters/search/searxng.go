// Package search provides web search backend adapters for Beacon.
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/isaacwallace123/cortex/services/beacon/internal/domain"
)

// SearXNG queries a self-hosted SearXNG instance.
// Configure BEACON_SEARXNG_URL to point at your SearXNG deployment.
type SearXNG struct {
	baseURL string
	client  *http.Client
}

func NewSearXNG(baseURL string) *SearXNG {
	return &SearXNG{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

type searxngResponse struct {
	Results []searxngResult `json:"results"`
}

type searxngResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

func (s *SearXNG) Search(ctx context.Context, query string, maxResults int) ([]domain.SearchResult, string, error) {
	if maxResults <= 0 {
		maxResults = 5
	}

	params := url.Values{
		"q":          {query},
		"format":     {"json"},
		"categories": {"general"},
	}
	endpoint := s.baseURL + "/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", fmt.Errorf("searxng: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("searxng: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, "", fmt.Errorf("searxng: read response: %w", err)
	}

	var result searxngResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", fmt.Errorf("searxng: parse response: %w", err)
	}

	out := make([]domain.SearchResult, 0, maxResults)
	for i, r := range result.Results {
		if i >= maxResults {
			break
		}
		out = append(out, domain.SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
		})
	}
	return out, "searxng", nil
}
