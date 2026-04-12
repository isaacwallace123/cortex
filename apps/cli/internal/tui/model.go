// Package tui implements the Cortex terminal user interface using Bubble Tea.
// The TUI handles login, chat input, plan review, step approval, and result display.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/isaacwallace123/cortex/apps/cli/internal/client"
)

var spinnerFrames = []string{"-", "\\", "|", "/"}

func spinTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(_ time.Time) tea.Msg { return tickMsg{} })
}

// Internal Bubble Tea message types used to pass async results back to Update.
type (
	tickMsg    struct{}
	planMsg    struct{ resp client.ChatResponse }
	runDoneMsg struct{ results []client.StepResult }
	apiErrMsg  struct{ err error }

	loginDoneMsg struct {
		token     string
		userID    string
		username  string
		expiresAt int64
	}
	loginErrMsg struct{ msg string }
)

// appState tracks which screen or interaction mode the TUI is currently in.
type appState int

const (
	stateLogin     appState = iota // showing login form
	stateLoginBusy                 // login request in flight
	stateInput                     // user is typing a chat message
	stateThinking                  // waiting for plan from API
	statePlan                      // plan shown, awaiting [r]
	stateApproving                 // stepping through pending approvals
	stateRunning                   // executing steps
	stateSessions                  // session browser
)

const (
	headerLines = 1
	footerLines = 3 // separator + input line + status line
)

// loginField selects which login form input is focused.
type loginField int

const (
	fieldUsername loginField = iota
	fieldPassword
)

type sessionListItem struct {
	sessionID string
	title     string
	lastUsed  int64
	current   bool
}

func (i sessionListItem) FilterValue() string { return i.sessionID + " " + i.title }

func (i sessionListItem) Title() string {
	prefix := " "
	if i.current {
		prefix = "*"
	}
	label := i.title
	if label == "" {
		label = "untitled session"
	}
	return fmt.Sprintf("%s %s  %s", prefix, shortID(i.sessionID, 12), label)
}

func (i sessionListItem) Description() string {
	return "last used " + formatSessionTime(i.lastUsed)
}

// Model is the Bubble Tea model for the Cortex TUI.
type Model struct {
	api       *client.API
	sessionID string

	vp    viewport.Model
	ready bool

	input     string // text the user is currently typing
	state     appState
	plan      *client.ChatResponse
	lines     []string // accumulated rendered content for the viewport
	spinFrame int

	// approval queue populated when plan has pending steps
	pendingQueue  []client.Step // pending steps awaiting user decision
	approvedSteps []client.Step // all steps with user-overridden verdicts
	queueIdx      int           // current position in pendingQueue

	loginUsername string
	loginPassword string
	loginField    loginField // which field has focus
	loginErr      string     // non-empty when last login attempt failed

	sessionHistory []client.SessionHistoryEntry
	sessionList    list.Model

	width  int
	height int
}

// New returns an initialized TUI model.
// Pass needsLogin=true to open the login form immediately instead of chat.
func New(api *client.API, sessionID string, needsLogin bool) Model {
	state := stateInput
	if needsLogin {
		state = stateLogin
	}

	history, _ := client.LoadSessionHistory()
	m := Model{
		api:            api,
		sessionID:      sessionID,
		state:          state,
		sessionHistory: history,
	}
	m.sessionList = newSessionList(history, sessionID, 0, 0)

	if sessionID != "" {
		_ = client.RecordSession(sessionID, "")
	}

	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var vpCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpHeight := msg.Height - headerLines - footerLines
		if vpHeight < 1 {
			vpHeight = 1
		}
		if !m.ready {
			m.vp = viewport.New(msg.Width, vpHeight)
			m.vp.SetContent(m.contentStr())
			m.vp.GotoBottom()
			m.ready = true
		} else {
			m.vp.Width = msg.Width
			m.vp.Height = vpHeight
			m.syncViewport()
		}
		m.syncSessionListSize()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case loginDoneMsg:
		_ = client.SaveSession(client.SavedSession{
			Token:     msg.token,
			UserID:    msg.userID,
			Username:  msg.username,
			ExpiresAt: msg.expiresAt,
		})
		m.api = m.api.WithToken(msg.token)
		m.state = stateInput
		m.loginErr = ""
		m.loginUsername = ""
		m.loginPassword = ""
		m.lines = append(m.lines,
			styleDim.Render("  signed in as "+msg.username),
			"",
		)
		m.syncViewport()
		return m, nil

	case loginErrMsg:
		m.state = stateLogin
		m.loginErr = msg.msg
		return m, nil

	case planMsg:
		return m.onPlan(msg.resp)

	case runDoneMsg:
		m.state = stateInput
		m.plan = nil
		m.pendingQueue = nil
		m.approvedSteps = nil
		m.appendResults(msg.results)
		total, executed := len(msg.results), 0
		for _, r := range msg.results {
			if !r.Skipped {
				executed++
			}
		}
		if total > 0 {
			m.lines = append(m.lines, "")
			m.lines = append(m.lines, fmt.Sprintf("  %s  %s",
				styleGreen.Render("[ok]"),
				styleDim.Render(fmt.Sprintf("%d/%d complete", executed, total)),
			))
		}
		m.lines = append(m.lines, "")
		m.syncViewport()
		return m, nil

	case apiErrMsg:
		m.state = stateInput
		m.plan = nil
		m.pendingQueue = nil
		m.approvedSteps = nil
		for _, line := range wrapLines(friendlyErr(msg.err), m.width-4) {
			m.lines = append(m.lines, styleRed.Render("  "+line))
		}
		m.lines = append(m.lines, "")
		m.syncViewport()
		return m, nil

	case tickMsg:
		m.spinFrame = (m.spinFrame + 1) % len(spinnerFrames)
		if m.state == stateThinking || m.state == stateRunning || m.state == stateLoginBusy {
			return m, spinTick()
		}
		return m, nil
	}

	m.vp, vpCmd = m.vp.Update(msg)
	return m, vpCmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateLogin:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab, tea.KeyDown:
			if m.loginField == fieldUsername {
				m.loginField = fieldPassword
			} else {
				m.loginField = fieldUsername
			}
		case tea.KeyUp:
			if m.loginField == fieldPassword {
				m.loginField = fieldUsername
			}
		case tea.KeyEnter:
			if strings.TrimSpace(m.loginUsername) == "" {
				m.loginErr = "username is required"
				return m, nil
			}
			if strings.TrimSpace(m.loginPassword) == "" {
				m.loginErr = "password is required"
				return m, nil
			}
			m.state = stateLoginBusy
			m.loginErr = ""
			return m, tea.Batch(m.cmdDoLogin(), spinTick())
		case tea.KeyBackspace:
			if m.loginField == fieldUsername {
				runes := []rune(m.loginUsername)
				if len(runes) > 0 {
					m.loginUsername = string(runes[:len(runes)-1])
				}
			} else {
				runes := []rune(m.loginPassword)
				if len(runes) > 0 {
					m.loginPassword = string(runes[:len(runes)-1])
				}
			}
		case tea.KeyRunes:
			if m.loginField == fieldUsername {
				m.loginUsername += msg.String()
			} else {
				m.loginPassword += msg.String()
			}
		case tea.KeySpace:
			if m.loginField == fieldUsername {
				m.loginUsername += " "
			} else {
				m.loginPassword += " "
			}
		}
		return m, nil

	case stateLoginBusy:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		return m, nil

	case stateInput:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyCtrlO:
			m.openSessionBrowser()
			return m, nil
		case tea.KeyEnter:
			input := strings.TrimSpace(m.input)
			if input == "" {
				return m, nil
			}
			m.input = ""
			m.state = stateThinking
			m.lines = append(m.lines, stylePromptArrow.Render("  > ")+input)
			m.syncViewport()
			return m, tea.Batch(m.cmdFetchPlan(input), spinTick())
		case tea.KeyBackspace:
			runes := []rune(m.input)
			if len(runes) > 0 {
				m.input = string(runes[:len(runes)-1])
			}
		case tea.KeyCtrlU:
			m.input = ""
		case tea.KeyRunes:
			m.input += msg.String()
		case tea.KeySpace:
			m.input += " "
		default:
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}

	case statePlan:
		switch msg.String() {
		case "r", "R":
			if m.plan == nil {
				return m, nil
			}
			return m.startApproval()
		case "ctrl+c":
			return m, tea.Quit
		case "esc", "q":
			m.state = stateInput
			m.plan = nil
			m.lines = append(m.lines, styleDim.Render("  cancelled"))
			m.lines = append(m.lines, "")
			m.syncViewport()
		default:
			if msg.Type == tea.KeyRunes {
				m.state = stateInput
				m.plan = nil
				m.input = msg.String()
			}
		}

	case stateApproving:
		switch msg.String() {
		case "y", "Y":
			step := m.pendingQueue[m.queueIdx]
			for i := range m.approvedSteps {
				if m.approvedSteps[i].Index == step.Index {
					m.approvedSteps[i].Verdict = "allow"
					break
				}
			}
			m.lines = append(m.lines, fmt.Sprintf("  %s  step %d approved", styleGreen.Render("[ok]"), step.Index))
			return m.advanceApproval()
		case "n", "N", "esc":
			step := m.pendingQueue[m.queueIdx]
			m.lines = append(m.lines, fmt.Sprintf("  %s  step %d skipped", styleDim.Render("[-]"), step.Index))
			return m.advanceApproval()
		case "ctrl+c":
			return m, tea.Quit
		}

	case stateSessions:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

		if m.sessionList.SettingFilter() {
			var cmd tea.Cmd
			m.sessionList, cmd = m.sessionList.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "enter":
			item, ok := m.sessionList.SelectedItem().(sessionListItem)
			if !ok {
				return m, nil
			}
			m.sessionID = item.sessionID
			_ = client.RecordSession(item.sessionID, item.title)
			m.reloadSessionHistory()
			m.state = stateInput
			m.lines = append(m.lines, styleDim.Render("  resumed session "+shortID(item.sessionID, 12)))
			m.lines = append(m.lines, "")
			m.syncViewport()
			return m, nil
		case "n", "N":
			m.sessionID = ""
			m.state = stateInput
			m.lines = append(m.lines, styleDim.Render("  started a new session"))
			m.lines = append(m.lines, "")
			m.syncViewport()
			return m, nil
		case "d", "D":
			item, ok := m.sessionList.SelectedItem().(sessionListItem)
			if !ok {
				return m, nil
			}
			_ = client.DeleteSessionFromHistory(item.sessionID)
			if m.sessionID == item.sessionID {
				m.sessionID = ""
			}
			m.reloadSessionHistory()
			return m, nil
		case "esc", "q":
			m.state = stateInput
			return m, nil
		}

		var cmd tea.Cmd
		m.sessionList, cmd = m.sessionList.Update(msg)
		return m, cmd

	case stateThinking, stateRunning:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}

	return m, nil
}

// startApproval begins the step-approval flow for any pending steps in the plan.
// If there are no pending steps it jumps straight to execution.
func (m Model) startApproval() (Model, tea.Cmd) {
	m.approvedSteps = make([]client.Step, len(m.plan.Plan.Steps))
	copy(m.approvedSteps, m.plan.Plan.Steps)

	m.pendingQueue = nil
	for _, s := range m.plan.Plan.Steps {
		if s.Verdict == "pending" {
			m.pendingQueue = append(m.pendingQueue, s)
		}
	}

	if len(m.pendingQueue) == 0 {
		return m.beginRun()
	}

	m.queueIdx = 0
	m.state = stateApproving
	m.showCurrentPending()
	m.syncViewport()
	return m, nil
}

func (m Model) advanceApproval() (Model, tea.Cmd) {
	m.queueIdx++
	if m.queueIdx >= len(m.pendingQueue) {
		m.lines = append(m.lines, "")
		return m.beginRun()
	}
	m.showCurrentPending()
	m.syncViewport()
	return m, nil
}

func (m Model) beginRun() (Model, tea.Cmd) {
	m.state = stateRunning
	m.lines = append(m.lines, styleDim.Render("  executing..."))
	m.syncViewport()
	sessionID := m.plan.SessionID
	planID := m.plan.Plan.PlanID
	steps := m.approvedSteps
	return m, tea.Batch(m.cmdRunSteps(sessionID, planID, steps), spinTick())
}

func (m *Model) showCurrentPending() {
	step := m.pendingQueue[m.queueIdx]
	total := len(m.pendingQueue)
	current := m.queueIdx + 1

	descWidth := m.stepDescWidth()
	sep := styleSep.Render("  " + strings.Repeat("-", m.sepWidth()))

	m.lines = append(m.lines, "")
	m.lines = append(m.lines, fmt.Sprintf("  %s  %s",
		styleYellow.Render("[!]"),
		styleBold.Render(fmt.Sprintf("approval required (%d of %d)", current, total)),
	))
	m.lines = append(m.lines, sep)
	m.lines = append(m.lines, fmt.Sprintf("  %2d  %-*s  %-10s  %s",
		step.Index,
		descWidth,
		truncate(step.Description, descWidth),
		styleDim.Render(step.Executor),
		styleYellow.Render("pending"),
	))
	if step.Command != "" && step.Executor != "infer" {
		m.lines = append(m.lines, styleDim.Render("       $ "+step.Command))
	}
	m.lines = append(m.lines, sep)
	m.lines = append(m.lines, "")
	m.lines = append(m.lines,
		styleDim.Render("  ")+styleStatusKey.Render("[y]")+styleDim.Render(" approve  ")+
			styleStatusKey.Render("[n]")+styleDim.Render(" skip"))
	m.lines = append(m.lines, "")
}

// onPlan handles a plan received from the API. If all steps are infer steps the
// plan is auto-executed immediately; otherwise it is shown for user review.
func (m Model) onPlan(resp client.ChatResponse) (Model, tea.Cmd) {
	m.sessionID = resp.SessionID
	m.recordSession(resp.SessionID, resp.Intent)

	if allInfer(resp.Plan.Steps) {
		m.plan = &resp
		m.approvedSteps = make([]client.Step, len(resp.Plan.Steps))
		copy(m.approvedSteps, resp.Plan.Steps)
		m.state = stateRunning
		m.appendPlanHeader(resp)
		m.syncViewport()
		return m, tea.Batch(
			m.cmdRunSteps(resp.SessionID, resp.Plan.PlanID, m.approvedSteps),
			spinTick(),
		)
	}

	m.plan = &resp
	m.state = statePlan
	m.appendPlanHeader(resp)
	m.appendPlanSteps(resp)
	m.lines = append(m.lines, "")
	m.lines = append(m.lines,
		styleDim.Render("  ")+styleStatusKey.Render("[r]")+styleDim.Render(" run  ")+
			styleStatusKey.Render("[esc]")+styleDim.Render(" cancel"))
	m.lines = append(m.lines, "")
	m.syncViewport()
	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return ""
	}
	if m.state == stateLogin || m.state == stateLoginBusy {
		return m.renderLoginScreen()
	}
	if m.state == stateSessions {
		return m.renderSessionBrowser()
	}
	return strings.Join([]string{
		m.renderHeader(),
		m.vp.View(),
		m.renderSeparator(),
		m.renderInputLine(),
		m.renderStatusLine(),
	}, "\n")
}

func (m Model) renderLoginScreen() string {
	var b strings.Builder

	header := styleHeaderBar.Width(m.width).Render(" cortex  sign in")
	b.WriteString(header)
	b.WriteByte('\n')

	formHeight := 12
	padTop := (m.height - headerLines - footerLines - formHeight) / 2
	if padTop < 1 {
		padTop = 1
	}
	for i := 0; i < padTop; i++ {
		b.WriteByte('\n')
	}

	indent := "  "
	fieldWidth := 40

	b.WriteString(indent + styleBold.Render("cortex") + "  " + styleDim.Render("personal AI control plane") + "\n")
	b.WriteByte('\n')

	b.WriteString(indent + styleDim.Render("username") + "\n")
	b.WriteString(indent + renderLoginField(m.loginUsername, m.loginField == fieldUsername, fieldWidth) + "\n")
	b.WriteByte('\n')

	b.WriteString(indent + styleDim.Render("password") + "\n")
	masked := strings.Repeat("*", len([]rune(m.loginPassword)))
	b.WriteString(indent + renderLoginField(masked, m.loginField == fieldPassword, fieldWidth) + "\n")
	b.WriteByte('\n')

	if m.state == stateLoginBusy {
		b.WriteString(indent + styleDim.Render(spinnerFrames[m.spinFrame]+" signing in...") + "\n")
	} else if m.loginErr != "" {
		b.WriteString(indent + styleRed.Render("x "+m.loginErr) + "\n")
	} else {
		b.WriteByte('\n')
	}

	b.WriteString(m.renderSeparator() + "\n")
	b.WriteString(styleDim.Render("  [tab] next field  [enter] sign in  [ctrl+c] quit"))

	return b.String()
}

func renderLoginField(value string, focused bool, width int) string {
	cursor := ""
	if focused {
		cursor = "_"
	}
	runes := []rune(value)
	maxLen := width - len([]rune(cursor))
	if maxLen < 0 {
		maxLen = 0
	}
	if len(runes) > maxLen {
		runes = runes[len(runes)-maxLen:]
	}
	visible := string(runes) + cursor
	pad := width - len([]rune(visible))
	if pad < 0 {
		pad = 0
	}
	field := "[" + visible + strings.Repeat(" ", pad) + "]"
	if focused {
		return styleCyan.Render(field)
	}
	return styleSep.Render(field)
}

func (m Model) renderSessionBrowser() string {
	listView := m.sessionList.View()
	body := lipgloss.NewStyle().Padding(1, 1).Render(listView)
	return strings.Join([]string{
		styleHeaderBar.Width(m.width).Render(" cortex  sessions"),
		body,
		m.renderSeparator(),
		styleDim.Render("  [enter] open  [n] new  [d] remove local  [/] filter  [esc] back"),
	}, "\n")
}

func (m Model) renderHeader() string {
	session := "new"
	if m.sessionID != "" {
		session = shortID(m.sessionID, 12)
	}
	content := fmt.Sprintf(" cortex  session:%s", session)
	return styleHeaderBar.Width(m.width).Render(content)
}

func (m Model) renderSeparator() string {
	width := m.width
	if width < 1 {
		width = 1
	}
	return styleSep.Render(strings.Repeat("-", width))
}

func (m Model) renderInputLine() string {
	arrow := stylePromptArrow.Render(" > ")
	cursor := styleCyan.Render("_")
	return arrow + m.input + cursor
}

func (m Model) renderStatusLine() string {
	switch m.state {
	case stateThinking:
		return styleDim.Render("  " + spinnerFrames[m.spinFrame] + " thinking...")
	case stateRunning:
		return styleDim.Render("  " + spinnerFrames[m.spinFrame] + " executing...")
	case statePlan:
		return styleDim.Render("  [r] run  [esc] cancel  [ctrl+c] quit")
	case stateApproving:
		cur, total := m.queueIdx+1, len(m.pendingQueue)
		return styleDim.Render(fmt.Sprintf("  approval %d/%d  [y] approve  [n] skip  [ctrl+c] quit", cur, total))
	default:
		return styleDim.Render("  [enter] submit  [ctrl+o] sessions  [ctrl+c] quit")
	}
}

func (m *Model) appendPlanHeader(resp client.ChatResponse) {
	m.lines = append(m.lines, "")
	m.lines = append(m.lines, "  "+styleBold.Render(resp.Intent))
}

func (m *Model) appendPlanSteps(resp client.ChatResponse) {
	descWidth := m.stepDescWidth()
	sep := styleSep.Render("  " + strings.Repeat("-", m.sepWidth()))

	m.lines = append(m.lines, "")
	m.lines = append(m.lines, sep)
	for _, step := range resp.Plan.Steps {
		m.lines = append(m.lines, fmt.Sprintf("  %2d  %-*s  %-10s  %s",
			step.Index,
			descWidth,
			truncate(step.Description, descWidth),
			styleDim.Render(step.Executor),
			verdictBadge(step.Verdict),
		))
		if step.Executor == "infer" {
			m.lines = append(m.lines, styleDim.Render("       direct answer"))
		} else if step.Command != "" {
			m.lines = append(m.lines, styleDim.Render("       $ "+step.Command))
		}
	}
	m.lines = append(m.lines, sep)
}

func (m *Model) appendResults(results []client.StepResult) {
	wrapWidth := m.width - 6
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	for _, r := range results {
		if r.Skipped {
			m.lines = append(m.lines, fmt.Sprintf("  %2d  %s  %s",
				r.StepIndex,
				styleDim.Render("[-]"),
				styleDim.Render(r.SkipReason),
			))
			continue
		}

		if r.Executor == "infer" {
			m.lines = append(m.lines, "")
			if r.Stdout != "" {
				for _, raw := range strings.Split(strings.TrimRight(r.Stdout, "\n"), "\n") {
					for _, wrapped := range wrapLines(raw, wrapWidth) {
						m.lines = append(m.lines, "  "+wrapped)
					}
				}
			}
			continue
		}

		icon := styleGreen.Render("[ok]")
		if r.ExitCode != 0 {
			icon = styleRed.Render("[x]")
		}
		m.lines = append(m.lines, fmt.Sprintf("  %2d  %s  %s",
			r.StepIndex,
			icon,
			styleDim.Render(fmt.Sprintf("%dms", r.DurationMs)),
		))

		if r.Stdout != "" {
			for _, line := range strings.Split(strings.TrimRight(r.Stdout, "\n"), "\n") {
				for _, wrapped := range wrapLines(line, wrapWidth) {
					m.lines = append(m.lines, "     "+wrapped)
				}
			}
		}
		if r.Stderr != "" {
			for _, line := range strings.Split(strings.TrimRight(r.Stderr, "\n"), "\n") {
				for _, wrapped := range wrapLines(line, wrapWidth) {
					m.lines = append(m.lines, "     "+styleRed.Render(wrapped))
				}
			}
		}
	}
}

func (m *Model) syncViewport() {
	if !m.ready {
		return
	}
	m.vp.SetContent(m.contentStr())
	m.vp.GotoBottom()
}

func (m *Model) syncSessionListSize() {
	if m.width == 0 || m.height == 0 {
		return
	}
	listWidth := m.width - 4
	listHeight := m.height - headerLines - 4
	if listWidth < 20 {
		listWidth = 20
	}
	if listHeight < 6 {
		listHeight = 6
	}
	m.sessionList.SetSize(listWidth, listHeight)
}

func (m *Model) openSessionBrowser() {
	m.reloadSessionHistory()
	m.syncSessionListSize()
	m.state = stateSessions
}

func (m *Model) reloadSessionHistory() {
	entries, err := client.LoadSessionHistory()
	if err != nil {
		return
	}
	m.sessionHistory = entries
	m.sessionList = newSessionList(entries, m.sessionID, m.sessionList.Width(), m.sessionList.Height())
}

func (m *Model) recordSession(sessionID, title string) {
	if sessionID == "" {
		return
	}
	_ = client.RecordSession(sessionID, title)
	m.reloadSessionHistory()
}

func (m Model) contentStr() string {
	if len(m.lines) == 0 {
		return strings.Join([]string{
			"",
			"  " + styleBold.Render("cortex") + "  " + styleDim.Render("personal AI control plane"),
			"",
			styleDim.Render("  Type a message and press enter."),
			styleDim.Render("  Shell commands are planned and confirmed before execution."),
			styleDim.Render("  Press ctrl+o to browse recent sessions."),
			"",
		}, "\n")
	}
	return strings.Join(m.lines, "\n")
}

func newSessionList(entries []client.SessionHistoryEntry, currentSessionID string, width, height int) list.Model {
	items := make([]list.Item, 0, len(entries))
	for _, e := range entries {
		items = append(items, sessionListItem{
			sessionID: e.SessionID,
			title:     e.Title,
			lastUsed:  e.LastUsed,
			current:   e.SessionID == currentSessionID,
		})
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.Color(colorFg))
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(lipgloss.Color(colorMuted))
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color(colorBlue)).
		Bold(true)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color(colorCyan))
	delegate.Styles.FilterMatch = delegate.Styles.FilterMatch.
		Foreground(lipgloss.Color(colorYellow)).
		Bold(true)

	if width < 20 {
		width = 20
	}
	if height < 6 {
		height = 6
	}

	l := list.New(items, delegate, width, height)
	l.Title = "Recent Sessions"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowFilter(true)
	l.SetFilteringEnabled(true)

	l.Styles.TitleBar = lipgloss.NewStyle().
		Background(lipgloss.Color(colorPanel)).
		Foreground(lipgloss.Color(colorFg)).
		Padding(0, 1)
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorFg)).
		Bold(true)
	l.Styles.FilterPrompt = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorBlue)).
		Bold(true)
	l.Styles.FilterCursor = lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorBlue))
	l.Styles.NoItems = styleDim

	return l
}

func (m Model) stepDescWidth() int {
	w := m.width - 26
	if w < 20 {
		w = 20
	}
	if w > 60 {
		w = 60
	}
	return w
}

func (m Model) sepWidth() int {
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	return w
}

func shortID(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func formatSessionTime(ts int64) string {
	if ts <= 0 {
		return "unknown"
	}
	t := time.Unix(ts, 0).Local()
	return t.Format("2006-01-02 15:04")
}

// cmdDoLogin fires a login request with the current form values.
func (m Model) cmdDoLogin() tea.Cmd {
	api := m.api
	username := m.loginUsername
	password := m.loginPassword
	return func() tea.Msg {
		resp, err := api.Login(username, password, "cortex-cli")
		if err != nil {
			return loginErrMsg{msg: "invalid username or password"}
		}
		return loginDoneMsg{
			token:     resp.Session.Token,
			userID:    resp.User.UserID,
			username:  resp.User.Username,
			expiresAt: resp.Session.ExpiresAt,
		}
	}
}

func (m Model) cmdFetchPlan(input string) tea.Cmd {
	api := m.api
	sessionID := m.sessionID
	return func() tea.Msg {
		resp, err := api.Chat(sessionID, input)
		if err != nil {
			return apiErrMsg{err}
		}
		return planMsg{resp}
	}
}

// cmdRunSteps executes all steps in a single /v1/run call so Brain can chain
// prior outputs into subsequent infer steps (e.g. web results → synthesis).
func (m Model) cmdRunSteps(sessionID, planID string, steps []client.Step) tea.Cmd {
	api := m.api
	return func() tea.Msg {
		resp, err := api.Run(sessionID, planID, steps)
		if err != nil {
			return apiErrMsg{err}
		}
		return runDoneMsg{results: resp.Results}
	}
}

// allInfer reports whether every step in the plan uses the infer executor.
// These plans are auto-executed without showing the review screen.
func allInfer(steps []client.Step) bool {
	if len(steps) == 0 {
		return false
	}
	for _, s := range steps {
		if s.Executor != "infer" {
			return false
		}
	}
	return true
}

func friendlyErr(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "connectex") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "no such host") {
		return "cortex is not reachable. start the stack with: make up"
	}
	return msg
}

func wrapLines(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	if len([]rune(text)) <= width {
		return []string{text}
	}
	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	current := ""
	for _, word := range words {
		wLen := len([]rune(word))
		if current == "" {
			current = word
		} else if len([]rune(current))+1+wLen <= width {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
