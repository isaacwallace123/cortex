package domain

// Task is a unit of parsed user intent produced by Prism (input understanding).
type Task struct {
	SessionID string
	Intent    string
	Entities  []string

	// IntentType is Prism's typed classification — the primary routing signal for Axiom.
	//   ask_question | read_file | write_file | list_directory | search_workspace |
	//   run_command   | system_inspect | cluster_inspect | cluster_action |
	//   web_search    | get_weather | diagnose | plan_only
	IntentType string

	// ExecutionStyle controls how Axiom scopes the plan.
	//   minimal    — single action, no exploration (default)
	//   exploratory — investigate broadly
	//   diagnostic  — gather data then explain
	//   multi_step  — complex sequential workflow
	ExecutionStyle string

	// Mode is the legacy routing field kept for backward compatibility.
	//   "direct_answer" | "execution" | "tool_assisted" | "clarification"
	Mode string

	// RawInput is the original unmodified user message used by Axiom for infer prompts.
	RawInput string
}
