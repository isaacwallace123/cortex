package ports

import "context"

// RegistryTool is a single tool definition retrieved from Arsenal.
type RegistryTool struct {
	Name        string
	Description string
	Executor    string
	Example     string
	SafetyLevel string
	Tags        []string
}

// ArsenalPort lets Brain query the tool registry.
// ListTools is called by Axiom before plan generation to dynamically build the
// executor capability section of the planning prompt.
type ArsenalPort interface {
	ListTools(ctx context.Context) ([]RegistryTool, error)
}
