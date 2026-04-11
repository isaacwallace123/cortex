package compass

import "context"

// NoopAdapter silently drops all Compass calls.
// Used when the Compass service is not configured.
type NoopAdapter struct{}

func (NoopAdapter) CreateProject(_ context.Context, _, _ string) (string, error)             { return "", nil }
func (NoopAdapter) CreateMilestone(_ context.Context, _, _, _ string, _ int) (string, error) { return "", nil }
func (NoopAdapter) CreateTask(_ context.Context, _, _, _ string) (string, error)             { return "", nil }
func (NoopAdapter) GetProjectSummary(_ context.Context, _ string) (string, error)            { return "", nil }
