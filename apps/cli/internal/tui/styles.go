package tui

import "github.com/charmbracelet/lipgloss"

const (
	colorBg     = "#1a1b26"
	colorPanel  = "#24283b"
	colorFg     = "#c0caf5"
	colorMuted  = "#565f89"
	colorBlue   = "#7aa2f7"
	colorCyan   = "#7dcfff"
	colorGreen  = "#9ece6a"
	colorRed    = "#f7768e"
	colorYellow = "#e0af68"
	colorBorder = "#3b4261"
)

var (
	styleHeader = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorFg)).
			Bold(true)

	styleHeaderBar = lipgloss.NewStyle().
			Background(lipgloss.Color(colorPanel)).
			Foreground(lipgloss.Color(colorFg)).
			Padding(0, 1)

	stylePromptArrow = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorBlue)).
				Bold(true)

	styleBold = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorFg)).
			Bold(true)

	styleDim = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))

	styleCyan = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorCyan))

	styleGreen = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGreen))

	styleRed = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorRed))

	styleYellow = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorYellow))

	styleStatusKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue)).
			Bold(true)

	styleSep = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBorder))
)

func verdictBadge(v string) string {
	switch v {
	case "allow":
		return styleGreen.Render("allow")
	case "deny":
		return styleRed.Render("deny")
	case "pending":
		return styleYellow.Render("pending")
	default:
		return styleDim.Render(v)
	}
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}
