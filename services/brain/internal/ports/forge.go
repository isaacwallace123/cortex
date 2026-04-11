package ports

// ForgePort is brain's interface to the Forge filesystem executor service.
// It satisfies StepExecutor — Vector dispatches to it by type.
type ForgePort interface {
	StepExecutor
}
