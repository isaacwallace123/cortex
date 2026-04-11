package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/isaacwallace123/cortex/services/beacon/internal/domain"
)

const braveSearchURL = "https://api.search.brave.com/res/v1/web/search"

// Brave queries the Brave Search API.
// Requires BEACON_BRAVE_API_KEY to be set.
type Brave struct {
	apiKey string
	client *http.Client
}

func NewBrave(apiKey string) *Brave {
	return &Brave{
		apiKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

type braveResponse struct {
	Web struct {
		Results []braveResult `json:"results"`
	} `json:"web"`
}

type braveResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

func (b *Brave) Search(ctx context.Context, query string, maxResults int) ([]domain.SearchResult, string, error) {
	if maxResults <= 0 {
		maxResults = 5
	}

	params := url.Values{
		"q":     {query},
		"count": {strconv.Itoa(maxResults)},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, braveSearchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("brave: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", b.apiKey)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("brave: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("brave: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, "", fmt.Errorf("brave: read response: %w", err)
	}

	var result braveResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", fmt.Errorf("brave: parse response: %w", err)
	}

	out := make([]domain.SearchResult, 0, len(result.Web.Results))
	for _, r := range result.Web.Results {
		out = append(out, domain.SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Description,
		})
	}
	return out, "brave", nil
}
