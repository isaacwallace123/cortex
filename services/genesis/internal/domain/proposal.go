// Package domain defines the core types for the Genesis tool-generation pipeline.
package domain

import "time"

// GeneratedTool is a tool definition produced by the LLM.
// It matches the Arsenal tool schema so it can be registered directly.
type GeneratedTool struct {
	Name        string
	Description string
	Executor    string
	Example     string
	SafetyLevel string
	Tags        []string
	Version     string
}

// Proposal records a single generation attempt and its outcome.
type Proposal struct {
	ProposalID  string
	Description string        // original natural language request
	Tool        *GeneratedTool // nil on failure
	Registered  bool          // true if Arsenal accepted the tool
	Error       string        // non-empty on failure
	CreatedAt   time.Time
}
