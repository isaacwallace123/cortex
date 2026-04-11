// Package domain defines the core Tool entity for the Arsenal registry.
package domain

// SafetyLevel describes the risk associated with invoking a tool.
type SafetyLevel string

const (
	SafetyLevelSafe      SafetyLevel = "safe"
	SafetyLevelModerate  SafetyLevel = "moderate"
	SafetyLevelDangerous SafetyLevel = "dangerous"
)

// Tool represents a single registered capability in the Arsenal registry.
type Tool struct {
	Name        string      // unique identifier, e.g. "run_command"
	Description string      // what this tool does
	Executor    string      // "shell" | "filesystem" | "infer" | "crucible" | "atlas" | "courier"
	Example     string      // example STEP line for the Axiom planning prompt
	SafetyLevel SafetyLevel // risk level
	Tags        []string    // grouping labels
	Enabled     bool        // false = hidden from planners
	Version     string      // semver
}

// DefaultTools returns the built-in tool definitions pre-seeded at startup.
func DefaultTools() []Tool {
	return []Tool{
		{
			Name:        "run_command",
			Description: "Execute any standard Linux/Unix shell command on the local machine",
			Executor:    "shell",
			Example:     "STEP 1: Check disk usage [executor:shell] [command:df -h]",
			SafetyLevel: SafetyLevelModerate,
			Tags:        []string{"local", "shell"},
			Enabled:     true,
			Version:     "1.0.0",
		},
		{
			Name:        "read_file",
			Description: "Read the contents of a file in the workspace",
			Executor:    "filesystem",
			Example:     "STEP 1: Read config [executor:filesystem] [command:cat /workspace/config.yaml]",
			SafetyLevel: SafetyLevelSafe,
			Tags:        []string{"local", "filesystem"},
			Enabled:     true,
			Version:     "1.0.0",
		},
		{
			Name:        "write_file",
			Description: "Write or overwrite a file in the workspace",
			Executor:    "filesystem",
			Example:     "STEP 1: Write file [executor:filesystem] [command:write /workspace/out.txt Hello World]",
			SafetyLevel: SafetyLevelModerate,
			Tags:        []string{"local", "filesystem"},
			Enabled:     true,
			Version:     "1.0.0",
		},
		{
			Name:        "list_directory",
			Description: "List files and directories in the workspace",
			Executor:    "filesystem",
			Example:     "STEP 1: List workspace [executor:filesystem] [command:ls /workspace]",
			SafetyLevel: SafetyLevelSafe,
			Tags:        []string{"local", "filesystem"},
			Enabled:     true,
			Version:     "1.0.0",
		},
		{
			Name:        "run_code",
			Description: "Execute code in an isolated Docker sandbox (Python, Node, Go, Bash, Rust)",
			Executor:    "crucible",
			Example:     "STEP 1: Run Python [executor:crucible] [agent:python3] [command:python3 -c \"print('hello')\"]",
			SafetyLevel: SafetyLevelSafe,
			Tags:        []string{"sandbox", "code"},
			Enabled:     true,
			Version:     "1.0.0",
		},
		{
			Name:        "kubectl",
			Description: "Run kubectl commands against the configured Kubernetes cluster",
			Executor:    "atlas",
			Example:     "STEP 1: List pods [executor:atlas] [command:get pods -A]",
			SafetyLevel: SafetyLevelModerate,
			Tags:        []string{"k8s", "cluster"},
			Enabled:     true,
			Version:     "1.0.0",
		},
		{
			Name:        "remote_command",
			Description: "Execute a shell command on a connected remote Relay agent",
			Executor:    "courier",
			Example:     "STEP 1: Check remote disk [executor:courier] [agent:prod-server] [command:df -h]",
			SafetyLevel: SafetyLevelModerate,
			Tags:        []string{"remote", "agent"},
			Enabled:     true,
			Version:     "1.0.0",
		},
		{
			Name:        "ai_inference",
			Description: "Ask the LLM to analyse, explain, or summarise prior step results",
			Executor:    "infer",
			Example:     "STEP 2: Explain results [executor:infer] [command:Explain what the output above means.]",
			SafetyLevel: SafetyLevelSafe,
			Tags:        []string{"llm", "analysis"},
			Enabled:     true,
			Version:     "1.0.0",
		},
	}
}
