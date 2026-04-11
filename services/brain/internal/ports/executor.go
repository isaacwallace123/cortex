package ports

// ExecutorPort is brain's interface to the Shell executor service.
// It satisfies StepExecutor — Vector dispatches to it by type.
type ExecutorPort interface {
	StepExecutor
}
