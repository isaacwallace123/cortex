package domain

// OpResult is the outcome of a filesystem operation performed by Forge.
type OpResult struct {
	Output     string
	OK         bool
	DurationMs int64
}
