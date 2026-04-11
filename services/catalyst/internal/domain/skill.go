// Package domain defines the core Skill entity for the Catalyst engine.
package domain

import "time"

// SkillStep is a single step in a Skill workflow.
// CommandTemplate may contain Go template expressions ({{.param}}) that are
// expanded at execution time with caller-supplied parameters.
type SkillStep struct {
	Index           int
	Description     string
	Executor        string // "shell" | "filesystem" | "web" | "infer" | "crucible" | "atlas" | "courier"
	CommandTemplate string
	Agent           string // for courier (device name) or crucible (runtime)
}

// Skill is a named, versioned, reusable workflow composed from Arsenal tools.
type Skill struct {
	Name        string
	Description string
	Steps       []SkillStep
	Tags        []string
	Version     string
	CreatedAt   time.Time
}
