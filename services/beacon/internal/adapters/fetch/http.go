// Package fetch provides HTTP content fetching with text extraction.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/isaacwallace123/cortex/services/beacon/internal/domain"
)

const (
	maxBodyBytes    = 100 * 1024 // 100 KB text cap
	defaultTimeout  = 15
)

var (
	tagRe     = regexp.MustCompile(`<[^>]+>`)
	scriptRe  = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	spaceRe   = regexp.MustCompile(`\s{2,}`)
)

// HTTPFetcher fetches URLs and extracts readable text from the response body.
type HTTPFetcher struct{}

func NewHTTPFetcher() *HTTPFetcher { return &HTTPFetcher{} }

func (f *HTTPFetcher) Fetch(ctx context.Context, rawURL string, timeoutSeconds int) (domain.FetchResult, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultTimeout
	}

	client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return domain.FetchResult{Error: fmt.Sprintf("build request: %v", err)}, nil
	}
	req.Header.Set("User-Agent", "Cortex-Beacon/1.0 (+https://github.com/isaacwallace123/cortex)")
	req.Header.Set("Accept", "text/html,text/plain,application/json")

	resp, err := client.Do(req)
	if err != nil {
		return domain.FetchResult{Error: fmt.Sprintf("fetch %s: %v", rawURL, err)}, nil
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	result := domain.FetchResult{
		StatusCode:  resp.StatusCode,
		ContentType: ct,
	}

	// Only extract text from text/* and application/json responses.
	if !isTextContent(ct) {
		result.Error = fmt.Sprintf("unsupported content-type: %s", ct)
		return result, nil
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		result.Error = fmt.Sprintf("read body: %v", err)
		return result, nil
	}

	if strings.Contains(ct, "html") {
		result.Body = extractText(string(raw))
	} else {
		result.Body = string(raw)
	}
	return result, nil
}

func isTextContent(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.Contains(ct, "text/") ||
		strings.Contains(ct, "application/json") ||
		strings.Contains(ct, "application/xml")
}

// extractText strips HTML tags and collapses whitespace to produce readable text.
func extractText(html string) string {
	// Remove script and style blocks first.
	clean := scriptRe.ReplaceAllString(html, " ")
	// Strip remaining tags.
	clean = tagRe.ReplaceAllString(clean, " ")
	// Collapse whitespace.
	clean = spaceRe.ReplaceAllString(clean, " ")
	return strings.TrimSpace(clean)
}
