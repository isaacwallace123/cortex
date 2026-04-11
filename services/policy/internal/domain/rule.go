package domain

// Rule defines a single Aegis policy rule.
// Rules are evaluated in order; the first match wins.
// If no rule matches, the default verdict is VerdictAllow.
type Rule struct {
	Name        string   // human-readable identifier
	Executor    string   // match by executor type ("shell", "filesystem", …); empty = any
	Pattern     string   // regex matched against Command; empty = matches any command
	Action      Verdict  // verdict to issue when this rule matches
	Reason      string   // explanation surfaced to the caller
	ExceptRoles []string // roles exempt from this rule (e.g. ["admin"])
}
