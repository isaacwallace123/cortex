package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"github.com/isaacwallace123/cortex/services/policy/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/policy/internal/log"
)

// Engine evaluates plan steps against an ordered list of rules.
// First match wins. If no rule matches, the step is allowed.
type Engine struct {
	rules    []compiledRule
	log      *slog.Logger
}

type compiledRule struct {
	domain.Rule
	re *regexp.Regexp // compiled Pattern; nil if Pattern == ""
}

// NewEngine builds an Engine from a slice of domain rules.
func NewEngine(rules []domain.Rule) (*Engine, error) {
	compiled := make([]compiledRule, 0, len(rules))
	for _, r := range rules {
		cr := compiledRule{Rule: r}
		if r.Pattern != "" {
			re, err := regexp.Compile(r.Pattern)
			if err != nil {
				return nil, fmt.Errorf("rule %q has invalid pattern %q: %w", r.Name, r.Pattern, err)
			}
			cr.re = re
		}
		compiled = append(compiled, cr)
	}
	return &Engine{rules: compiled, log: cortexlog.Aegis()}, nil
}

// NewEngineFromFile loads rules from a JSON file at path.
// The file must contain a JSON array of rule objects.
func NewEngineFromFile(path string) (*Engine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules file %s: %w", path, err)
	}
	var rules []domain.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parse rules file %s: %w", path, err)
	}
	return NewEngine(rules)
}

func (e *Engine) Evaluate(_ context.Context, _, _ string, steps []domain.StepInput) ([]domain.StepDecision, error) {
	decisions := make([]domain.StepDecision, 0, len(steps))
	for _, step := range steps {
		verdict, reason, ruleName := e.evaluate(step)
		e.log.Info("[Aegis] step evaluated",
			slog.Int("step", step.Index),
			slog.String("executor", step.Executor),
			slog.String("verdict", string(verdict)),
			slog.String("rule", ruleName),
			slog.String("reason", reason),
		)
		decisions = append(decisions, domain.StepDecision{
			StepIndex: step.Index,
			Verdict:   verdict,
			Reason:    reason,
		})
	}
	return decisions, nil
}

func (e *Engine) evaluate(step domain.StepInput) (domain.Verdict, string, string) {
	for _, r := range e.rules {
		if r.matches(step) {
			return r.Action, r.Reason, r.Name
		}
	}
	return domain.VerdictAllow, "default: no matching rule", "default"
}

func (r compiledRule) matches(step domain.StepInput) bool {
	// Check role exemptions — if caller has any exempt role, skip this rule.
	for _, exempt := range r.ExceptRoles {
		for _, role := range step.Roles {
			if role == exempt {
				return false
			}
		}
	}
	if r.Executor != "" && r.Executor != step.Executor {
		return false
	}
	if r.re != nil && !r.re.MatchString(step.Command) {
		return false
	}
	return true
}

// DefaultRules returns the built-in Aegis ruleset.
// Order matters — first match wins.
// Admin users are exempt from deny rules; approval rules still apply to everyone.
func DefaultRules() []domain.Rule {
	return []domain.Rule{
		{
			Name:        "deny-recursive-delete",
			Executor:    "shell",
			Pattern:     `rm\s+.*-[^\s]*r|rm\s+--recursive|rm\s+--no-preserve-root`,
			Action:      domain.VerdictDeny,
			Reason:      "recursive deletion is not permitted",
			ExceptRoles: []string{"admin"},
		},
		{
			Name:        "deny-system-control",
			Executor:    "shell",
			Pattern:     `\b(shutdown|reboot|poweroff|halt|init\s+0)\b`,
			Action:      domain.VerdictDeny,
			Reason:      "system control commands are not permitted",
			ExceptRoles: []string{"admin"},
		},
		{
			Name:        "deny-privilege-escalation",
			Executor:    "shell",
			Pattern:     `\b(sudo|su\s|doas)\b`,
			Action:      domain.VerdictDeny,
			Reason:      "privilege escalation is not permitted",
			ExceptRoles: []string{"admin"},
		},
		{
			Name:     "deny-k8s",
			Executor: "k8s",
			Action:   domain.VerdictDeny,
			Reason:   "k8s executor is not authorized in this environment",
		},
		// Filesystem write/delete operations require explicit approval.
		// Read-only operations (ls, cat, stat) pass through automatically.
		{
			Name:     "pending-filesystem-write",
			Executor: "filesystem",
			Pattern:  `^(write|mkdir|rm|move|rename)\s`,
			Action:   domain.VerdictPending,
			Reason:   "filesystem write requires approval",
		},
		// Network operations could exfiltrate data or download untrusted content.
		{
			Name:     "pending-network-ops",
			Executor: "shell",
			Pattern:  `\b(curl|wget|ssh|scp|rsync|nc|ncat|netcat|ftp|sftp)\b`,
			Action:   domain.VerdictPending,
			Reason:   "network operations require approval",
		},
		{
			Name:     "pending-code",
			Executor: "code",
			Action:   domain.VerdictPending,
			Reason:   "code execution requires human review before running",
		},
	}
}
