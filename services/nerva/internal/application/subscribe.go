package application

import (
	"strings"

	"github.com/isaacwallace123/cortex/services/nerva/internal/domain"
)

// MatchesFilter returns true if event passes the subscriber's filter.
// Rules (case-insensitive):
//
//	""                  → all events
//	"Prism"             → subsystem match
//	"prism.input.parsed"→ exact event_name match
//	"prism."            → event_name prefix match
func MatchesFilter(event *domain.Event, filter string) bool {
	if filter == "" {
		return true
	}
	f := strings.ToLower(filter)
	// Exact event_name match.
	if strings.ToLower(event.Name) == f {
		return true
	}
	// Prefix match (e.g. "prism." matches "prism.input.parsed").
	if strings.HasPrefix(strings.ToLower(event.Name), f) {
		return true
	}
	// Subsystem match (e.g. "Prism" matches any event from Prism).
	if strings.ToLower(event.Subsystem) == f {
		return true
	}
	return false
}
