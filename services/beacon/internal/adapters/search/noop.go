package search

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/beacon/internal/domain"
)

// Noop is returned when no search backend is configured.
// It returns an informative error so users know what env vars to set.
type Noop struct{}

func (Noop) Search(_ context.Context, _ string, _ int) ([]domain.SearchResult, string, error) {
	return nil, "", fmt.Errorf("no search backend configured — set BEACON_SEARXNG_URL or BEACON_BRAVE_API_KEY")
}
