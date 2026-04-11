// Package sentinel implements the Cortex safety layer.
// Every shell/filesystem/crucible/courier command passes through Sentinel
// before reaching an executor. Sentinel applies regex-based rules at
// configurable severity levels and returns a Verdict that Vector uses to
// allow, warn, or block the step.
//
// Safety levels (SENTINEL_LEVEL env):
//   - unrestricted â€” no checks (dev only)
//   - standard     â€” default; blocks destructive operations and remote code exec
//   - paranoid     â€” adds blocks on sudo, network ops, and all file deletions
//
// Custom rules (SENTINEL_RULES_FILE env):
//   Path to a JSON file containing a []RuleSpec array. Rules in the file are
//   prepended to the built-in rules, so they take precedence. The file is
//   re-read automatically when it changes (checked on every Check call).
package sentinel

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Level controls which rules are active.
type Level string

const (
	LevelUnrestricted Level = "unrestricted"
	LevelStandard     Level = "standard"
	LevelParanoid     Level = "paranoid"
)

// Severity of a matched rule.
type Severity string

const (
	SeverityBlock Severity = "blocked"
	SeverityWarn  Severity = "warn"
)

// Verdict is returned by Check for a single command.
type Verdict struct {
	Allowed  bool
	Severity Severity // "" when allowed with no warnings
	Reason   string
	Rule     string
}

// RuleSpec is the JSON-serializable form of a sentinel rule.
// Used for loading custom rules from SENTINEL_RULES_FILE.
type RuleSpec struct {
	Name        string   `json:"name"`
	Pattern     string   `json:"pattern"`
	Severity    Severity `json:"severity"` // "blocked" | "warn"
	Reason      string   `json:"reason"`
	MinLevel    Level    `json:"min_level"` // "standard" | "paranoid" | ""
	ExceptRoles []string `json:"except_roles,omitempty"`
}

// rule is an internal compiled safety rule.
type rule struct {
	name        string
	pattern     *regexp.Regexp
	severity    Severity
	reason      string
	minLevel    Level // minimum level at which this rule is active
	exceptRoles []string
}

// Sentinel is the safety filter. Create one with New and call Check before execution.
type Sentinel struct {
	level    Level
	mu       sync.RWMutex
	rules    []rule   // custom (file) rules prepended, then built-in rules
	filePath string   // empty if no rules file configured
	fileMod  time.Time
}

// New creates a Sentinel configured for the given safety level.
// Pass LevelStandard for the default production-safe configuration.
func New(level Level) *Sentinel {
	if level == "" {
		level = LevelStandard
	}
	return &Sentinel{
		level: level,
		rules: compiledRules(nil),
	}
}

// LoadRulesFile configures a JSON rules file for this Sentinel.
// The file is read immediately and again whenever it changes on disk.
// Returns an error if the file cannot be read or parsed on the first load.
func (s *Sentinel) LoadRulesFile(path string) error {
	custom, mod, err := readRulesFile(path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.filePath = path
	s.fileMod = mod
	s.rules = compiledRules(custom)
	s.mu.Unlock()
	return nil
}

// ReloadIfChanged re-reads the rules file if it has been modified since last load.
// Returns true when the rules were reloaded, false otherwise (including on error).
func (s *Sentinel) ReloadIfChanged() bool {
	s.mu.RLock()
	path := s.filePath
	lastMod := s.fileMod
	s.mu.RUnlock()

	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || !info.ModTime().After(lastMod) {
		return false
	}
	custom, mod, err := readRulesFile(path)
	if err != nil {
		return false
	}
	s.mu.Lock()
	s.fileMod = mod
	s.rules = compiledRules(custom)
	s.mu.Unlock()
	return true
}

// Check evaluates the command against all active rules and returns a Verdict.
// Executors should call this before running any user-supplied command string.
// Optional roles are used for role-based rule exemptions (e.g. admin bypass).
func (s *Sentinel) Check(command string, roles ...string) Verdict {
	if s.level == LevelUnrestricted {
		return Verdict{Allowed: true}
	}

	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return Verdict{Allowed: true}
	}

	// Reload rules from file if it changed since last check.
	s.ReloadIfChanged()

	s.mu.RLock()
	rules := s.rules
	s.mu.RUnlock()

	for _, r := range rules {
		if !s.levelActive(r.minLevel) {
			continue
		}
		// Role exemption: if caller holds any exempt role, skip this rule.
		if roleExempt(roles, r.exceptRoles) {
			continue
		}
		if r.pattern != nil && !r.pattern.MatchString(cmd) {
			continue
		}
		if r.severity == SeverityBlock {
			return Verdict{Allowed: false, Severity: SeverityBlock, Reason: r.reason, Rule: r.name}
		}
		// Warn â€” allow execution but surface the warning.
		return Verdict{Allowed: true, Severity: SeverityWarn, Reason: r.reason, Rule: r.name}
	}

	return Verdict{Allowed: true}
}

// Level returns the configured safety level.
func (s *Sentinel) Level() Level { return s.level }

func (s *Sentinel) levelActive(min Level) bool {
	switch min {
	case LevelStandard:
		return s.level == LevelStandard || s.level == LevelParanoid
	case LevelParanoid:
		return s.level == LevelParanoid
	default:
		return true
	}
}

// roleExempt returns true if any caller role appears in the exempt list.
func roleExempt(callerRoles, exceptRoles []string) bool {
	for _, exempt := range exceptRoles {
		for _, r := range callerRoles {
			if r == exempt {
				return true
			}
		}
	}
	return false
}

// readRulesFile reads and parses the JSON rules file.
func readRulesFile(path string) ([]RuleSpec, time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	var specs []RuleSpec
	if err := json.Unmarshal(data, &specs); err != nil {
		return nil, time.Time{}, err
	}
	return specs, info.ModTime(), nil
}

// compiledRules merges custom (file) rules with built-in rules.
// Custom rules are prepended so they take first-match precedence.
func compiledRules(custom []RuleSpec) []rule {
	var out []rule
	for _, spec := range custom {
		r := rule{
			name:        spec.Name,
			severity:    spec.Severity,
			reason:      spec.Reason,
			exceptRoles: spec.ExceptRoles,
		}
		if spec.MinLevel != "" {
			r.minLevel = spec.MinLevel
		}
		if spec.Pattern != "" {
			re, err := regexp.Compile(`(?i)` + spec.Pattern)
			if err == nil {
				r.pattern = re
			}
		}
		out = append(out, r)
	}
	out = append(out, builtinRules()...)
	return out
}

func must(pattern string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)` + pattern)
}

func builtinRules() []rule {
	return []rule{
		// standard blocking rules
		{
			name:     "recursive-root-delete",
			pattern:  must(`rm\s+(-\w*f\w*r\w*|-\w*r\w*f\w*)\s+/\s*$`),
			severity: SeverityBlock,
			reason:   "recursive deletion of root filesystem",
			minLevel: LevelStandard,
		},
		{
			name:     "recursive-wildcard-delete",
			pattern:  must(`rm\s+(-\w*f\w*r\w*|-\w*r\w*f\w*)\s+/\*`),
			severity: SeverityBlock,
			reason:   "recursive deletion of all root-level paths",
			minLevel: LevelStandard,
		},
		{
			name:     "disk-format",
			pattern:  must(`mkfs\b|dd\s+.*of=/dev/(sd|nvme|hd|vd)`),
			severity: SeverityBlock,
			reason:   "disk format or overwrite operation",
			minLevel: LevelStandard,
		},
		{
			name:     "remote-code-exec",
			pattern:  must(`(curl|wget)\s+.*\|\s*(bash|sh|python|ruby|perl)`),
			severity: SeverityBlock,
			reason:   "piping remote content directly to a shell interpreter",
			minLevel: LevelStandard,
		},
		{
			name:     "chmod-world-root",
			pattern:  must(`chmod\s+(777|a\+rwx)\s+/\s*$`),
			severity: SeverityBlock,
			reason:   "unsafe permission change on root filesystem",
			minLevel: LevelStandard,
		},
		{
			name:     "fork-bomb",
			pattern:  must(`:\s*\(\s*\)\s*\{.*:\|:.*\}`),
			severity: SeverityBlock,
			reason:   "fork bomb pattern detected",
			minLevel: LevelStandard,
		},
		// paranoid blocking rules
		{
			name:     "sudo",
			pattern:  must(`\bsudo\b`),
			severity: SeverityBlock,
			reason:   "sudo usage blocked in paranoid mode",
			minLevel: LevelParanoid,
		},
		{
			name:     "delete-paranoid",
			pattern:  must(`\brm\b`),
			severity: SeverityBlock,
			reason:   "file deletion blocked in paranoid mode",
			minLevel: LevelParanoid,
		},
		{
			name:     "network-paranoid",
			pattern:  must(`\b(curl|wget|nc|ncat|netcat|ssh|scp)\b`),
			severity: SeverityBlock,
			reason:   "network operation blocked in paranoid mode",
			minLevel: LevelParanoid,
		},
		// standard warning rules
		{
			name:     "sudo-warn",
			pattern:  must(`\bsudo\b`),
			severity: SeverityWarn,
			reason:   "command requires elevated privileges",
			minLevel: LevelStandard,
		},
		{
			name:     "reboot-warn",
			pattern:  must(`\b(shutdown|reboot|halt|poweroff|init\s+0|init\s+6)\b`),
			severity: SeverityWarn,
			reason:   "system power/reboot command",
			minLevel: LevelStandard,
		},
		{
			name:     "firewall-flush",
			pattern:  must(`iptables\s+-F`),
			severity: SeverityWarn,
			reason:   "flushing all firewall rules",
			minLevel: LevelStandard,
		},
		{
			name:     "recursive-delete-warn",
			pattern:  must(`rm\s+(-\w*f\w*r\w*|-\w*r\w*f\w*)\b`),
			severity: SeverityWarn,
			reason:   "recursive file deletion",
			minLevel: LevelStandard,
		},
	}
}

