package render

// ANSI escape sequences. Only emitted to terminal output — no external deps.
const (
	aReset  = "\033[0m"
	aBold   = "\033[1m"
	aDim    = "\033[2m"
	aGreen  = "\033[32m"
	aRed    = "\033[31m"
	aYellow = "\033[33m"
	aCyan   = "\033[36m"
	aGray   = "\033[90m"
	aClear  = "\033[K" // erase from cursor to end of line
)

func bold(s string) string   { return aBold + s + aReset }
func dim(s string) string    { return aDim + s + aReset }
func green(s string) string  { return aGreen + s + aReset }
func red(s string) string    { return aRed + s + aReset }
func yellow(s string) string { return aYellow + s + aReset }
func cyan(s string) string   { return aCyan + s + aReset }
func gray(s string) string   { return aGray + s + aReset }

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

func verdictStr(v string) string {
	switch v {
	case "allow":
		return aGreen + "allow" + aReset
	case "deny":
		return aRed + "deny" + aReset
	case "pending":
		return aYellow + "pending" + aReset
	case "":
		return aDim + "—" + aReset
	default:
		return aDim + v + aReset
	}
}
